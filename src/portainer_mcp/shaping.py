"""Universal response shaping for every tool the server exposes.

Two cooperating layers:

1. `ResponseCapMiddleware` — caps every tool result's text content at
   `max_chars`. The final safety valve.

2. `SelectArgTransform` — wraps every tool with an optional JMESPath
   `select` parameter, so the model can project noisy Portainer
   responses server-side. Tools that already declare `select` are passed
   through unchanged.

`select` narrows first (cheaper bodies); the cap catches whatever slips
through (model omitted `select`, or the post-projection body is still
genuinely big).
"""

from __future__ import annotations

import json
import logging
from collections.abc import Sequence
from typing import Annotated, Any

import jmespath
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.server.transforms import Transform, VersionSpec
from fastmcp.tools import Tool, forward
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent
from pydantic import Field

logger = logging.getLogger("portainer_mcp")

# Must fire before Claude Code's MCP output cap (~25k tokens, ~62k chars
# for dense Portainer JSON at ~2.5 chars/token) so our truncation hint —
# which names `select` and shows an example — reaches the model instead
# of Claude Code's generic "saved to file, use offset/limit/jq" message
# (which steers the model toward jq against the spilled file rather than
# retrying with a server-side projection). 50k chars leaves ~12k headroom
# below that ceiling plus room for the hint itself. Override via
# PORTAINER_MAX_RESPONSE_CHARS.
DEFAULT_MAX_RESPONSE_CHARS = 50_000

SELECT_DESCRIPTION = (
    "Optional JMESPath expression to project the response server-side. "
    "Use it on noisy endpoints to drop fields you don't need — e.g. "
    "`[].{id:Id,name:Name,type:Type}` for a list of environments, or "
    "`{kind:Kind,name:metadata.name,phase:status.phase}` for a single K8s object. "
    "Omit to receive the full response (subject to the global size cap)."
)


def project(data: Any, select: str) -> Any:
    """Apply a JMESPath expression to `data`, or raise `ValueError`."""
    try:
        return jmespath.search(select, data)
    except jmespath.exceptions.JMESPathError as exc:
        raise ValueError(f"invalid JMESPath expression {select!r}: {exc}") from exc


class ResponseCapMiddleware(Middleware):
    """Truncate oversized tool results with a hint to narrow `select`.

    Applied uniformly to every tool. When truncation fires, the
    `structured_content` field is also cleared so the model can't read
    around the cap by inspecting the structured copy of the same payload.
    """

    def __init__(self, max_chars: int) -> None:
        super().__init__()
        self.max_chars = max_chars

    async def on_call_tool(
        self,
        context: MiddlewareContext,
        call_next: CallNext,
    ) -> ToolResult:
        result = await call_next(context)
        truncated = False
        for item in result.content:
            text = getattr(item, "text", None)
            if isinstance(text, str) and len(text) > self.max_chars:
                item.text = (
                    text[: self.max_chars]
                    + f"\n\n[truncated: response was {len(text)} chars, "
                    + f"capped at {self.max_chars}. Retry with a JMESPath "
                    + "`select` to project just the fields you need — e.g. "
                    + '`select="[].{id:Id,name:Name}"` for a list response, '
                    + 'or `select="{name:metadata.name,phase:status.phase}"` '
                    + "for a single object.]"
                )
                truncated = True
        if truncated:
            result.structured_content = None
        return result


async def _select_wrapper(
    select: Annotated[str | None, Field(description=SELECT_DESCRIPTION)] = None,
    **kwargs: Any,
) -> ToolResult:
    """Call the parent tool, then project its result via JMESPath."""
    result = await forward(**kwargs)
    if not select:
        return result

    data: Any = result.structured_content
    if data is None:
        text_blocks = [
            getattr(c, "text", None) for c in result.content if hasattr(c, "text")
        ]
        candidate = next(
            (t for t in text_blocks if isinstance(t, str) and t), None
        )
        if candidate is None:
            return result  # nothing projectable (no text, no structured)
        try:
            data = json.loads(candidate)
        except json.JSONDecodeError as exc:
            raise ValueError(
                f"response was not JSON; cannot apply `select`: {exc}"
            ) from exc

    # FastMCP wraps non-dict OpenAPI responses as `{"result": ...}` so they
    # fit MCP's structured_content schema (which must be an object). Unwrap
    # before applying JMESPath so callers write against the natural API
    # shape — e.g. `[].Id` against a list endpoint — rather than
    # `result[].Id`.
    if isinstance(data, dict) and set(data.keys()) == {"result"}:
        data = data["result"]

    projected = project(data, select)
    return ToolResult(
        content=[TextContent(type="text", text=json.dumps(projected))],
        # MCP structured_content must be a dict; drop it for lists/scalars.
        structured_content=projected if isinstance(projected, dict) else None,
    )


def _has_select(tool: Tool) -> bool:
    props = (tool.parameters or {}).get("properties") or {}
    return "select" in props


class SelectArgTransform(Transform):
    """Wrap every tool with an optional JMESPath `select` argument.

    Tools that already declare `select` are passed through unchanged.
    """

    async def list_tools(self, tools: Sequence[Tool]) -> Sequence[Tool]:
        return [
            t if _has_select(t) else Tool.from_tool(t, transform_fn=_select_wrapper)
            for t in tools
        ]

    async def get_tool(
        self,
        name: str,
        call_next: Any,
        *,
        version: VersionSpec | None = None,
    ) -> Tool | None:
        tool = await call_next(name, version=version)
        if tool is None or _has_select(tool):
            return tool
        return Tool.from_tool(tool, transform_fn=_select_wrapper)
