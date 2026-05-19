"""Unit tests for `src/portainer_mcp/shaping.py`.

Covers the pure-data layers: `project()` and `ResponseCapMiddleware`.
`_select_wrapper` and `SelectArgTransform` need a live FastMCP runtime
and aren't covered here.
"""

from __future__ import annotations

import pytest
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent

from portainer_mcp.shaping import ResponseCapMiddleware, project


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
