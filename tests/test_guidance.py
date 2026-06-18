"""Tests for the deterministic guidance gate.

The gate forces `get_guidance` to be called once per session before any other
tool runs. Scoping is transport-aware: stdio uses a single synthetic key, HTTP
keys on `Mcp-Session-Id`, and a sessionless HTTP client fails open.
"""

from __future__ import annotations

from types import SimpleNamespace

import pytest
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent

from portainer_mcp import guidance, request_context

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
