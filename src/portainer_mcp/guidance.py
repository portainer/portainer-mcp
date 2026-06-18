"""A meta-tool that serves the server's operating guidance on demand.

The full hygiene guide is far larger than the MCP `instructions` field can
carry (clients truncate instructions at ~2KB), so the short instructions point
here and the model fetches the full doc through a tool call when it needs it —
progressive disclosure rebuilt inside MCP.
"""

from __future__ import annotations

import logging
from typing import Annotated

from fastmcp import FastMCP
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent, ToolAnnotations
from pydantic import Field

from portainer_mcp import request_context
from portainer_mcp.shaping import SELECT_DESCRIPTION

logger = logging.getLogger("portainer_mcp")

GUIDANCE_TOOL_NAME = "get_guidance"

# Synthetic session key for stdio, where there is no HTTP request and the
# process lifetime *is* the session (one client connection per process).
_STDIO_KEY = "__stdio__"


def register(mcp: FastMCP, guide: str) -> None:
    """Register the `get_guidance` tool returning the hygiene guide verbatim."""

    @mcp.tool(
        name="get_guidance",
        annotations=ToolAnnotations(readOnlyHint=True),
        description=(
            "Operating guide for this Portainer MCP server: projecting responses "
            "with `select`, where the heavy fields live, results that are easy to "
            "misread (e.g. an edge environment's health comes from its heartbeat, "
            "not its `Status` field; typed K8s tools use different field names than "
            "the raw proxy), and how to deploy / scale / delete safely. Call it "
            "once at the start of any Portainer task, before interpreting responses "
            "or planning multi-step changes — it materially improves correctness "
            "and saves context."
        ),
    )
    async def get_guidance(
        # `select` is declared so the tool satisfies the universal-select
        # invariant and SelectArgTransform skips it (it would otherwise re-encode
        # the markdown body). The guide isn't a projectable JSON payload, so the
        # parameter is a no-op here.
        select: Annotated[str | None, Field(description=SELECT_DESCRIPTION)] = None,
    ) -> str:
        return guide

    logger.info("guidance tool registered (%d chars)", len(guide))


def _gate_notice(tool: str | None) -> ToolResult:
    target = f"`{tool}`" if tool else "this tool"
    return ToolResult(
        content=[
            TextContent(
                type="text",
                text=(
                    "This Portainer MCP server requires its operating guide to be "
                    "read once per session before any other tool runs. Call the "
                    f"`{GUIDANCE_TOOL_NAME}` tool now, then retry {target} with the "
                    "same arguments. (You'll only see this once per session.)"
                ),
            )
        ]
    )


class GuidanceGateMiddleware(Middleware):
    """Force `get_guidance` to be called once per session before any other tool.

    A deterministic gate: the first non-`get_guidance` tool call in a session is
    short-circuited with a notice instead of executing, until that session has
    called `get_guidance`. This guarantees the hygiene guide lands in the model's
    context rather than relying on the (truncatable, ignorable) `instructions`
    field.

    Session scoping is transport-aware. Over stdio the process *is* the session,
    so a single synthetic key suffices. Over HTTP many sessions multiplex one
    process, so the gate keys on the `Mcp-Session-Id` echoed on every post-
    `initialize` request. A stateless HTTP client that sends no session id can't
    be scoped (and can't be marked seen without bouncing forever), so the gate
    fails open and admits it — logged once. Stateless streamable-HTTP clients are
    uncommon in practice.
    """

    def __init__(self, *, is_http: bool) -> None:
        super().__init__()
        self._is_http = is_http
        self._seen: set[str] = set()
        self._warned_sessionless = False

    def _session_key(self) -> str | None:
        sid = request_context.snapshot().get("session_id")
        if sid:
            return sid
        # No session id: stdio is legitimately keyless (gate via the synthetic
        # key); HTTP means a stateless client we can't scope (fail open).
        return None if self._is_http else _STDIO_KEY

    async def on_call_tool(
        self,
        context: MiddlewareContext,
        call_next: CallNext,
    ) -> ToolResult:
        tool = getattr(context.message, "name", None)
        key = self._session_key()

        if tool == GUIDANCE_TOOL_NAME:
            if key is not None:
                self._seen.add(key)
            return await call_next(context)

        if key is None:
            if not self._warned_sessionless:
                logger.warning(
                    "guidance gate: HTTP request without Mcp-Session-Id; "
                    "admitting ungated (stateless client)"
                )
                self._warned_sessionless = True
            return await call_next(context)

        if key in self._seen:
            return await call_next(context)

        return _gate_notice(tool)
