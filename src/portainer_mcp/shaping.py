"""Universal response shaping for every tool the server exposes.

Two cooperating layers, applied at server build time:

1. `ResponseCapMiddleware` — caps every tool result's text content at
   `max_chars`, regardless of which tool produced it. The final safety
   valve, replaces the inline cap that used to live in `proxy.py`.

2. `inject_select_arg()` — wraps every OpenAPI-generated tool with an
   optional JMESPath `select` parameter, so the model can project noisy
   Portainer responses server-side. Same affordance the hand-written
   proxy tools already expose, applied uniformly across the auto-
   generated surface.

The two layers cooperate: `select` narrows first (cheaper bodies), the
cap catches whatever slips through (model omitted `select`, or the
post-projection body is still genuinely big).
"""

from __future__ import annotations

import json
import logging
from typing import Annotated, Any

import jmespath
from fastmcp import FastMCP
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.server.providers.openapi import OpenAPIProvider
from fastmcp.tools import Tool, forward
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent
from pydantic import Field

logger = logging.getLogger("portainer_mcp")

# Target ~25k tokens. Dense JSON (Docker/K8s payloads with IDs, hashes,
# nested structure) packs at ~3 chars/token, so 75k chars is deliberately
# conservative. The actual returned text may slightly exceed this cap by
# the length of the truncation hint — the value is a target, not an exact
# ceiling. Override via PORTAINER_MAX_RESPONSE_CHARS.
DEFAULT_MAX_RESPONSE_CHARS = 75_000

SELECT_DESCRIPTION = (
    "Optional JMESPath expression to project the response server-side. "
    "Use it on noisy endpoints to drop fields you don't need — e.g. "
    "`[].{id:Id,name:Name,type:Type}` for a list of environments, or "
    "`{kind:Kind,name:metadata.name,phase:status.phase}` for a single K8s object. "
    "Omit to receive the full response (subject to the global size cap)."
)


def project(data: Any, select: str) -> Any:
    """Apply a JMESPath expression to `data`, or raise `ValueError`.

    Shared by the proxy tools' inline path and the universal wrapper so the
    JMESPath core lives in one place with a single error convention.
    """
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
                    + f"capped at {self.max_chars}. Narrow the `select` "
                    + "JMESPath expression or refine other parameters.]"
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


def inject_select_arg(mcp: FastMCP) -> int:
    """Wrap every OpenAPI-generated tool with an optional `select` argument.

    The wrapped version is registered on the local provider and the
    original is removed from the OpenAPI provider, so `list_tools()`
    returns exactly one entry per tool. Tools that already declare
    `select` (e.g. our hand-written proxy tools) are skipped.

    Returns the number of tools wrapped.
    """

    # OpenAPIProvider populates `_tools` synchronously during from_openapi,
    # so we read (and pop) it directly. The async `list_tools()` API would
    # need a running event loop, which we don't have here.
    wrapped = 0
    for provider in mcp.providers:
        if not isinstance(provider, OpenAPIProvider):
            continue
        for tool in list(provider._tools.values()):
            props = (tool.parameters or {}).get("properties") or {}
            if "select" in props:
                continue
            new_tool = Tool.from_tool(tool, transform_fn=_select_wrapper)
            mcp.add_tool(new_tool)
            provider._tools.pop(tool.name, None)
            wrapped += 1
    logger.info("injected `select` arg on %d OpenAPI tools", wrapped)
    return wrapped
