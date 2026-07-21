"""Tests for the deterministic guidance gate.

The gate forces `get_guidance` to be called once per session before any other
tool runs. Scoping is transport-aware: stdio uses a single synthetic key, HTTP
keys on `Mcp-Session-Id`, and a sessionless HTTP client fails open. A caller
whose session ids churn (a bridge re-initializing per request, #75) is
detected per user-key digest and failed open on the `_CHURN_THRESHOLD`-th
distinct post-guidance session inside `_CHURN_WINDOW`; the flag decays after
`_UNSTABLE_TTL`.
"""

from __future__ import annotations

from types import SimpleNamespace

import pytest
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent

from portainer_mcp import guidance, passthrough, request_context

_REAL = ToolResult(content=[TextContent(type="text", text="REAL")])


def _ctx(tool: str):
    return SimpleNamespace(message=SimpleNamespace(name=tool))


async def _run(mw: guidance.GuidanceGateMiddleware, tool: str) -> tuple[bool, ToolResult]:
    """Returns (passed_through, result). passed_through is True when the gate
    let the call reach the underlying tool."""
    called = False

    async def call_next(_ctx):
        nonlocal called
        called = True
        return _REAL

    result = await mw.on_call_tool(context=_ctx(tool), call_next=call_next)
    return called, result


def _set_session(monkeypatch, sid: str | None) -> None:
    monkeypatch.setattr(
        request_context, "snapshot", lambda: {"session_id": sid} if sid else {}
    )


def _set_caller(monkeypatch, key: str | None) -> None:
    monkeypatch.setattr(passthrough, "key_from_request", lambda: key)


def _is_notice(result: ToolResult) -> bool:
    return guidance.GUIDANCE_TOOL_NAME in result.content[0].text


# --- stdio ------------------------------------------------------------------


async def test_stdio_bounces_first_call_then_admits_after_guidance(monkeypatch):
    _set_session(monkeypatch, None)
    mw = guidance.GuidanceGateMiddleware(is_http=False)

    passed, result = await _run(mw, "EndpointList")
    assert not passed
    assert _is_notice(result)

    # get_guidance always passes through and unlocks the session.
    passed, _ = await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    assert passed

    passed, result = await _run(mw, "EndpointList")
    assert passed
    assert result is _REAL


async def test_stdio_guidance_never_bounced(monkeypatch):
    _set_session(monkeypatch, None)
    mw = guidance.GuidanceGateMiddleware(is_http=False)

    passed, _ = await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    assert passed


# --- HTTP, per-session scoping ----------------------------------------------


async def test_http_scopes_per_session(monkeypatch):
    mw = guidance.GuidanceGateMiddleware(is_http=True)

    # Session A reads the guide.
    _set_session(monkeypatch, "sess-A")
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    passed, _ = await _run(mw, "EndpointList")
    assert passed

    # Session B is still gated — A's guidance call doesn't unlock it.
    _set_session(monkeypatch, "sess-B")
    passed, result = await _run(mw, "EndpointList")
    assert not passed
    assert _is_notice(result)

    # B reads the guide and is now admitted.
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    passed, _ = await _run(mw, "EndpointList")
    assert passed


# --- HTTP, session-id churn (bridge re-initializing per request, #75) --------


async def test_http_churning_bridge_fails_open_after_threshold(monkeypatch, caplog):
    mw = guidance.GuidanceGateMiddleware(is_http=True)
    _set_caller(monkeypatch, "user-key-1")

    # Every request arrives in a fresh session, so get_guidance can never
    # unlock the session the retry lands in — the #75 lockout.
    _set_session(monkeypatch, "sess-1")
    passed, _ = await _run(mw, "StackList")
    assert not passed  # pre-guidance bounce: normal first contact, not counted

    _set_session(monkeypatch, "sess-2")
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)

    with caplog.at_level("WARNING", logger="portainer_mcp"):
        _set_session(monkeypatch, "sess-3")
        passed, _ = await _run(mw, "StackList")
        assert not passed
        _set_session(monkeypatch, "sess-4")
        passed, _ = await _run(mw, "StackList")
        assert not passed
        # The third distinct post-guidance session trips the detector; the
        # tripping call itself is admitted.
        _set_session(monkeypatch, "sess-5")
        passed, result = await _run(mw, "StackList")
        assert passed
        assert result is _REAL

        # Caller stays admitted in yet another fresh session.
        _set_session(monkeypatch, "sess-6")
        passed, _ = await _run(mw, "StackList")
        assert passed

    warnings = [r for r in caplog.records if "unstable" in r.message]
    assert len(warnings) == 1
    # The raw key never appears in the log; only its digest prefix does.
    assert "user-key-1" not in warnings[0].getMessage()
    assert passthrough.digest("user-key-1")[:12] in warnings[0].getMessage()


async def test_http_multi_window_user_never_flagged(monkeypatch, caplog):
    """A session-preserving client opening many conversations is bounced once
    per session (by design) and never trips the detector: fetching guidance in
    the same session it was bounced in clears the suspicion each time."""
    mw = guidance.GuidanceGateMiddleware(is_http=True)
    _set_caller(monkeypatch, "user-key-1")

    with caplog.at_level("WARNING", logger="portainer_mcp"):
        for n in range(6):
            _set_session(monkeypatch, f"sess-{n}")
            passed, result = await _run(mw, "StackList")
            assert not passed
            assert _is_notice(result)
            await _run(mw, guidance.GUIDANCE_TOOL_NAME)
            passed, _ = await _run(mw, "StackList")
            assert passed

    assert not [r for r in caplog.records if "unstable" in r.message]
    # Per-session semantics intact: the next window is still gated.
    _set_session(monkeypatch, "sess-final")
    passed, result = await _run(mw, "StackList")
    assert not passed
    assert _is_notice(result)


async def test_http_pre_guidance_churn_never_fails_open(monkeypatch, caplog):
    """A caller that never fetches guidance keeps getting bounced — fresh
    sessions alone are not churn evidence."""
    mw = guidance.GuidanceGateMiddleware(is_http=True)
    _set_caller(monkeypatch, "user-key-1")

    with caplog.at_level("WARNING", logger="portainer_mcp"):
        for n in range(6):
            _set_session(monkeypatch, f"sess-{n}")
            passed, result = await _run(mw, "StackList")
            assert not passed
            assert _is_notice(result)

    assert not [r for r in caplog.records if "unstable" in r.message]


async def test_http_churn_detector_scoped_per_caller(monkeypatch):
    mw = guidance.GuidanceGateMiddleware(is_http=True)

    # Caller A churns: guidance in sess-1, bounces in sess-2 and sess-3, then
    # the third post-guidance session trips the detector and is admitted.
    _set_caller(monkeypatch, "key-A")
    _set_session(monkeypatch, "sess-1")
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    for n in (2, 3):
        _set_session(monkeypatch, f"sess-{n}")
        passed, _ = await _run(mw, "StackList")
        assert not passed
    _set_session(monkeypatch, "sess-4")
    passed, _ = await _run(mw, "StackList")
    assert passed

    # Caller B is unaffected by A's flag.
    _set_caller(monkeypatch, "key-B")
    _set_session(monkeypatch, "sess-5")
    passed, result = await _run(mw, "StackList")
    assert not passed
    assert _is_notice(result)


async def test_http_stale_bounces_age_out(monkeypatch, caplog):
    """Bounced-then-abandoned sessions spread over a long process lifetime are
    not churn evidence: only bounces inside _CHURN_WINDOW count."""
    mw = guidance.GuidanceGateMiddleware(is_http=True)
    _set_caller(monkeypatch, "user-key-1")
    clock = 0.0
    mw._now = lambda: clock

    _set_session(monkeypatch, "sess-0")
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)

    with caplog.at_level("WARNING", logger="portainer_mcp"):
        # Two abandoned bounces, then a long quiet stretch.
        for sid, t in (("sess-1", 10.0), ("sess-2", 20.0)):
            clock = t
            _set_session(monkeypatch, sid)
            passed, _ = await _run(mw, "StackList")
            assert not passed
        # Well past the window: the old evidence has aged out, so this is
        # bounce #1 again, not the tripping third session.
        clock = 20.0 + guidance._CHURN_WINDOW
        _set_session(monkeypatch, "sess-3")
        passed, result = await _run(mw, "StackList")
        assert not passed
        assert _is_notice(result)

        # A real bridge inside the window still trips: two more fresh
        # sessions in quick succession reach the threshold and are admitted.
        for sid in ("sess-4", "sess-5"):
            clock += 1.0
            _set_session(monkeypatch, sid)
            passed, _ = await _run(mw, "StackList")
        assert passed

    assert len([r for r in caplog.records if "unstable" in r.message]) == 1


async def test_http_unstable_flag_expires(monkeypatch):
    """A flagged caller returns to normal per-session gating after
    _UNSTABLE_TTL — a false positive decays; a real bridge just re-trips."""
    mw = guidance.GuidanceGateMiddleware(is_http=True)
    _set_caller(monkeypatch, "user-key-1")
    clock = 0.0
    mw._now = lambda: clock

    _set_session(monkeypatch, "sess-0")
    await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    for n in (1, 2, 3):
        _set_session(monkeypatch, f"sess-{n}")
        passed, _ = await _run(mw, "StackList")
    assert passed  # flagged on the third session

    clock = guidance._UNSTABLE_TTL + 1.0
    _set_session(monkeypatch, "sess-4")
    passed, result = await _run(mw, "StackList")
    assert not passed
    assert _is_notice(result)


# --- HTTP, stateless (no session id) ----------------------------------------


async def test_http_sessionless_fails_open_and_warns_once(monkeypatch, caplog):
    _set_session(monkeypatch, None)
    mw = guidance.GuidanceGateMiddleware(is_http=True)

    with caplog.at_level("WARNING", logger="portainer_mcp"):
        passed, result = await _run(mw, "EndpointList")
        assert passed
        assert result is _REAL
        # Second sessionless call also admitted, but only one warning total.
        passed, _ = await _run(mw, "StackList")
        assert passed

    warnings = [r for r in caplog.records if "without Mcp-Session-Id" in r.message]
    assert len(warnings) == 1
