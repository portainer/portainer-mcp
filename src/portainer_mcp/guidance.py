"""A meta-tool that serves the server's operating guidance on demand.

The full hygiene guide is far larger than the MCP `instructions` field can
carry (clients truncate instructions at ~2KB), so the short instructions point
here and the model fetches the full doc through a tool call when it needs it —
progressive disclosure rebuilt inside MCP.
"""

from __future__ import annotations

import json
import logging
import os
import time
from typing import Annotated

from fastmcp import FastMCP
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.tools.tool import ToolResult
from mcp.types import TextContent, ToolAnnotations
from pydantic import Field

from portainer_mcp import passthrough, request_context
from portainer_mcp.shaping import SELECT_DESCRIPTION

logger = logging.getLogger("portainer_mcp")

GUIDANCE_TOOL_NAME = "get_guidance"

TTL_ENV_VAR = "PORTAINER_MCP_GUIDANCE_TTL"
DEFAULT_TTL = 1800
DISABLE_ENV_VAR = "PORTAINER_MCP_DISABLE_GUIDANCE_GATE"

# Synthetic caller key for stdio, where there is no HTTP request and the
# process serves a single client.
_STDIO_KEY = "__stdio__"


def resolve_ttl() -> int:
    raw = os.environ.get(TTL_ENV_VAR)
    if raw is None:
        return DEFAULT_TTL
    try:
        ttl = int(raw)
    except ValueError:
        raise SystemExit(
            f"{TTL_ENV_VAR} must be an integer number of seconds (got {raw!r})"
        )
    if ttl <= 0:
        # ttl=0 would bounce the retry too — the exact lockout loop of #75.
        # "No gate" is spelled PORTAINER_MCP_DISABLE_GUIDANCE_GATE=1.
        raise SystemExit(f"{TTL_ENV_VAR} must be > 0 (got {ttl})")
    return ttl


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


def _toll_notice(guide: str, tool: str) -> ToolResult:
    target = f"`{tool}`"
    return ToolResult(
        content=[
            TextContent(
                type="text",
                text=(
                    "This Portainer MCP server delivers its operating guide "
                    f"before the first tool call of a conversation. {target} was "
                    "NOT executed. Read the guide below, then retry "
                    f"{target} with the same arguments — you won't be "
                    "interrupted again this conversation.\n\n"
                    f"{guide}\n\n"
                    f"[End of guide. Now retry {target} with the same arguments.]"
                ),
            )
        ]
    )


class GuidanceGateMiddleware(Middleware):
    """Deliver the operating guide in-band, once per caller per idle window.

    A toll booth, not a lock: the first tool call from a caller whose window
    has lapsed is answered with the guide itself plus a retry instruction, and
    the caller is marked guided immediately — delivery is the proof, so there
    is nothing to correlate across requests and no way to get locked out.

    Callers are keyed on the authenticated principal (SHA-256 of the per-user
    `X-Portainer-API-Key` over HTTP, a process-wide sentinel over stdio) and
    never on `Mcp-Session-Id`: clients and bridges mint a fresh session id per
    request — documented ecosystem behaviour (SEP-2567) — which is what made
    the session-keyed gate lock callers out permanently (#75).

    The TTL slides: every admitted call refreshes it, so re-delivery happens
    only after `ttl` seconds of idleness — the closest observable proxy for
    "a new conversation". A long active task is never interrupted mid-flow.
    """

    def __init__(self, guide: str, *, ttl: float, is_http: bool = False) -> None:
        super().__init__()
        self._guide = guide
        self._ttl = ttl
        self._is_http = is_http
        self._last_seen: dict[str, float] = {}
        self._warned_keyless = False

    def _caller_key(self) -> str:
        # Over HTTP the verifier guarantees the per-user key header is present
        # before any tool dispatch; no key means no HTTP request, i.e. stdio.
        key = passthrough.key_from_request()
        if key:
            return passthrough.digest(key)
        if self._is_http and not self._warned_keyless:
            # Reachable only if a FastMCP refactor breaks get_http_request()
            # inside the dispatch task (see request_context.py) — callers
            # would silently share one guidance bucket, so say it out loud.
            logger.warning(
                "guidance gate: HTTP tool call carries no per-user key in the "
                "request context; all callers are sharing one guidance bucket"
            )
            self._warned_keyless = True
        return _STDIO_KEY

    async def on_call_tool(
        self,
        context: MiddlewareContext,
        call_next: CallNext,
    ) -> ToolResult:
        now = time.monotonic()
        caller = self._caller_key()
        tool = context.message.name
        last = self._last_seen.get(caller)
        if tool == GUIDANCE_TOOL_NAME or (last is not None and now - last < self._ttl):
            self._last_seen[caller] = now
            return await call_next(context)
        self._prune(now)
        # Marked before the retry arrives: the guide is in the caller's context
        # from this very response, so there is nothing left to verify.
        self._last_seen[caller] = now
        # The outer request log records this call as a normal success; without
        # this record a bounced mutation would be indistinguishable from an
        # executed one in the audit trail. Context fields only — never the key.
        logger.info(
            json.dumps(
                {"event": "guidance_bounce", "tool": tool}
                | request_context.snapshot()
            )
        )
        return _toll_notice(self._guide, tool)

    def _prune(self, now: float) -> None:
        # On the bounce path only (rare): keeps the table bounded to callers
        # active within the last window.
        expired = [k for k, t in self._last_seen.items() if now - t >= self._ttl]
        for k in expired:
            del self._last_seen[k]
