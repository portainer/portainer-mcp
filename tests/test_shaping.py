"""Unit tests for `src/portainer_mcp/shaping.py`.

Covers the pure-data layers: `project()` and `ResponseCapMiddleware`.
`SelectArgTransform` needs a live FastMCP runtime and isn't covered here;
`_select_wrapper` is exercised by stubbing `fastmcp.tools.forward`.
"""

from __future__ import annotations

import json

import pytest
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent

from portainer_mcp import shaping
from portainer_mcp.redaction import EXPOSE_ENV_VAR, SENTINEL
from portainer_mcp.shaping import ResponseCapMiddleware, project


@pytest.fixture(autouse=True)
def _redact_by_default(monkeypatch):
    monkeypatch.delenv(EXPOSE_ENV_VAR, raising=False)


# --- project() --------------------------------------------------------------


def test_project_applies_expression():
    data = [{"Id": "a", "Name": "x"}, {"Id": "b", "Name": "y"}]
    assert project(data, "[].Id") == ["a", "b"]


def test_project_raises_value_error_on_invalid_expression():
    with pytest.raises(ValueError, match="invalid JMESPath"):
        project({}, "foo[")


# --- ResponseCapMiddleware --------------------------------------------------


def _result(text: str, structured: dict | None = None) -> ToolResult:
    return ToolResult(
        content=[TextContent(type="text", text=text)],
        structured_content=structured,
    )


async def _run(middleware: ResponseCapMiddleware, result: ToolResult) -> ToolResult:
    async def call_next(_ctx):
        return result

    return await middleware.on_call_tool(context=None, call_next=call_next)


async def test_cap_passes_through_when_under_limit():
    middleware = ResponseCapMiddleware(max_chars=100)
    structured = {"k": "v"}
    out = await _run(middleware, _result("hello", structured=structured))
    assert out.content[0].text == "hello"
    assert out.structured_content == structured


async def test_cap_truncates_and_clears_structured():
    middleware = ResponseCapMiddleware(max_chars=10)
    out = await _run(middleware, _result("x" * 50, structured={"k": "v"}))
    text = out.content[0].text
    assert text.startswith("x" * 10)
    assert "truncated: response was 50 chars" in text
    assert "capped at 10" in text
    assert out.structured_content is None


# --- _select_wrapper redaction ---------------------------------------------


def _stub_forward(monkeypatch, payload):
    """Replace `forward` so calling `_select_wrapper` returns `payload`."""

    async def fake_forward(**kwargs):
        return ToolResult(
            content=[TextContent(type="text", text=json.dumps(payload))],
            structured_content=payload if isinstance(payload, dict) else None,
        )

    monkeypatch.setattr(shaping, "forward", fake_forward)


async def test_wrapper_redacts_without_select_and_emits_hint(monkeypatch):
    _stub_forward(monkeypatch, {"Env": [{"name": "DB", "value": "secret"}]})
    result = await shaping._select_wrapper()
    body = json.loads(result.content[0].text)
    assert body == {"Env": [{"name": "DB", "value": SENTINEL}]}
    assert "1 env value(s) redacted" in result.content[1].text
    assert result.structured_content == body


async def test_wrapper_redacts_before_select(monkeypatch):
    # `select="Env[0].value"` must land on the sentinel — the env walker
    # runs before JMESPath projection, so callers can't bypass redaction.
    _stub_forward(monkeypatch, {"Env": [{"name": "DB", "value": "secret"}]})
    result = await shaping._select_wrapper(select="Env[0].value")
    assert json.loads(result.content[0].text) == SENTINEL
    # Hint is still present.
    assert "redacted" in result.content[1].text


async def test_wrapper_exposes_when_toggle_set(monkeypatch):
    monkeypatch.setenv(EXPOSE_ENV_VAR, "1")
    payload = {"Env": [{"name": "DB", "value": "secret"}]}
    _stub_forward(monkeypatch, payload)
    # Without select, no shaping happens at all — the upstream ToolResult
    # is returned verbatim, with the real value.
    result = await shaping._select_wrapper()
    assert result.structured_content == payload
    assert len(result.content) == 1


async def test_wrapper_no_hint_when_no_env(monkeypatch):
    _stub_forward(monkeypatch, {"Id": 1, "Name": "alpha"})
    result = await shaping._select_wrapper(select="Name")
    assert json.loads(result.content[0].text) == "alpha"
    # Only the projected body — no hint TextContent.
    assert len(result.content) == 1


async def test_wrapper_unwraps_result_envelope_then_redacts(monkeypatch):
    # FastMCP wraps list responses as `{"result": [...]}` for the schema.
    # The walker must reach into the unwrapped list.
    _stub_forward(monkeypatch, {"result": [{"Env": ["FOO=bar"]}]})
    result = await shaping._select_wrapper()
    body = json.loads(result.content[0].text)
    assert body == [{"Env": [f"FOO={SENTINEL}"]}]
    assert "1 env value(s) redacted" in result.content[1].text
