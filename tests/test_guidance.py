"""Tests for the guidance toll booth.

The gate delivers the operating guide in-band: the first tool call from a
caller whose idle window has lapsed is answered with the guide itself (the
tool is not executed), and the caller is marked guided immediately — so the
retry, and everything after it, passes. Callers are keyed on the
authenticated principal (per-user API-key digest over HTTP, a process
sentinel over stdio), never on `Mcp-Session-Id`, so session-id churn (#75)
cannot lock anyone out by construction.
"""

from __future__ import annotations

from types import SimpleNamespace

import pytest
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent

from portainer_mcp import guidance, passthrough

GUIDE = "# Operating guide\n\nAlways project with `select`."
TTL = 1800

_REAL = ToolResult(content=[TextContent(type="text", text="REAL")])


class _Clock:
    def __init__(self) -> None:
        self.now = 1000.0

    def __call__(self) -> float:
        return self.now


@pytest.fixture
def clock(monkeypatch) -> _Clock:
    c = _Clock()
    monkeypatch.setattr(guidance.time, "monotonic", c)
    return c


def _set_caller(monkeypatch, key: str | None) -> None:
    """Simulate the transport: `key` is the per-user API key from the live
    HTTP request, None means no HTTP request in flight (stdio)."""
    monkeypatch.setattr(passthrough, "key_from_request", lambda: key)


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


def _mw() -> guidance.GuidanceGateMiddleware:
    return guidance.GuidanceGateMiddleware(GUIDE, ttl=TTL)


# --- toll booth core ----------------------------------------------------------


async def test_first_call_bounced_with_guide_then_retry_admitted(monkeypatch, clock):
    _set_caller(monkeypatch, None)  # stdio
    mw = _mw()

    passed, result = await _run(mw, "EndpointList")
    assert not passed
    text = result.content[0].text
    assert GUIDE in text
    assert "`EndpointList`" in text
    assert "NOT executed" in text

    # The bounce itself delivered the guide — the retry needs no other action.
    passed, result = await _run(mw, "EndpointList")
    assert passed
    assert result is _REAL


async def test_get_guidance_never_bounced_and_marks_caller(monkeypatch, clock):
    _set_caller(monkeypatch, None)
    mw = _mw()

    passed, _ = await _run(mw, guidance.GUIDANCE_TOOL_NAME)
    assert passed

    # A proactive get_guidance call satisfies the booth: no bounce afterwards.
    passed, result = await _run(mw, "EndpointList")
    assert passed
    assert result is _REAL


async def test_http_callers_isolated(monkeypatch, clock):
    mw = _mw()

    _set_caller(monkeypatch, "ptr_key_A")
    await _run(mw, "EndpointList")  # bounce marks A
    passed, _ = await _run(mw, "EndpointList")
    assert passed

    # B presents a different key: still owed its own delivery.
    _set_caller(monkeypatch, "ptr_key_B")
    passed, result = await _run(mw, "StackList")
    assert not passed
    assert GUIDE in result.content[0].text

    # ...and A is unaffected by B's bounce.
    _set_caller(monkeypatch, "ptr_key_A")
    passed, _ = await _run(mw, "EndpointList")
    assert passed


async def test_session_churn_cannot_lock_out(monkeypatch, clock):
    # The #75 reproduction: a bridge minting a fresh Mcp-Session-Id per
    # request. The booth never reads the session — a stable caller key is
    # bounced exactly once, then admitted, whatever the transport does.
    _set_caller(monkeypatch, "ptr_stable_key")
    mw = _mw()

    passed, _ = await _run(mw, "EndpointList")
    assert not passed
    for tool in ("EndpointList", "StackList", "EndpointList"):
        passed, _ = await _run(mw, tool)
        assert passed


# --- TTL semantics ------------------------------------------------------------


async def test_idle_expiry_rearms_booth(monkeypatch, clock):
    _set_caller(monkeypatch, None)
    mw = _mw()

    await _run(mw, "EndpointList")  # bounce
    passed, _ = await _run(mw, "EndpointList")
    assert passed

    clock.now += TTL + 1
    passed, result = await _run(mw, "EndpointList")
    assert not passed
    assert GUIDE in result.content[0].text


async def test_ttl_slides_with_activity(monkeypatch, clock):
    _set_caller(monkeypatch, None)
    mw = _mw()

    await _run(mw, "EndpointList")  # bounce at t=0
    # An active conversation spanning several windows is never re-bounced:
    # each admitted call refreshes the window.
    for _ in range(4):
        clock.now += TTL - 100
        passed, _ = await _run(mw, "StackList")
        assert passed

    # Only idleness re-arms.
    clock.now += TTL + 1
    passed, _ = await _run(mw, "StackList")
    assert not passed


async def test_expired_callers_pruned_on_bounce(monkeypatch, clock):
    mw = _mw()

    _set_caller(monkeypatch, "ptr_key_A")
    await _run(mw, "EndpointList")
    clock.now += TTL + 1

    _set_caller(monkeypatch, "ptr_key_B")
    await _run(mw, "EndpointList")
    assert passthrough.digest("ptr_key_A") not in mw._last_seen


# --- resolve_ttl --------------------------------------------------------------


def test_resolve_ttl_default(monkeypatch):
    monkeypatch.delenv(guidance.TTL_ENV_VAR, raising=False)
    assert guidance.resolve_ttl() == guidance.DEFAULT_TTL


def test_resolve_ttl_reads_env(monkeypatch):
    monkeypatch.setenv(guidance.TTL_ENV_VAR, "600")
    assert guidance.resolve_ttl() == 600


@pytest.mark.parametrize("raw", ["0", "-5", "ten"])
def test_resolve_ttl_rejects_invalid(monkeypatch, raw):
    # 0 would bounce the retry too — the exact #75 lockout loop.
    monkeypatch.setenv(guidance.TTL_ENV_VAR, raw)
    with pytest.raises(SystemExit):
        guidance.resolve_ttl()
