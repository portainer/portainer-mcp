"""A meta-tool that serves the server's operating guidance on demand.

The full hygiene guide is far larger than the MCP `instructions` field can
carry (clients truncate instructions at ~2KB), so the short instructions point
here and the model fetches the full doc through a tool call when it needs it —
progressive disclosure rebuilt inside MCP.
"""

from __future__ import annotations

import logging
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

# Synthetic session key for stdio, where there is no HTTP request and the
# process lifetime *is* the session (one client connection per process).
_STDIO_KEY = "__stdio__"

# Distinct fresh post-guidance sessions within _CHURN_WINDOW before the gate
# concludes a caller's session ids are unstable and fails open for it. The
# tripping call is admitted, so a caller eats threshold-1 bounced round-trips.
# A real re-initializing bridge accumulates these in seconds; the window keeps
# occasional bounced-then-abandoned sessions from adding up into a false
# positive over a long process lifetime, and the flag TTL lets a false
# positive decay (a true bridge just re-trips cheaply after expiry).
_CHURN_THRESHOLD = 3
_CHURN_WINDOW = 300.0
_UNSTABLE_TTL = 3600.0


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

    A degenerate-but-legal variant of statelessness is a bridge that
    re-initializes per request: every call presents a fresh, *valid*
    `Mcp-Session-Id`, so `get_guidance` can never unlock the session the retry
    arrives in and the caller is locked out of every tool (#75). A single such
    request is indistinguishable from a genuinely new conversation, so the gate
    detects the pattern instead, scoped per caller by the SHA-256 of the
    per-user `X-Portainer-API-Key` the HTTP transport already requires. The
    discriminating signature: a session-preserving client that gets bounced
    fetches guidance *in that same session*, which proves ids are stable and
    clears the caller's suspicion; an id-churning bridge fetches guidance in
    yet another fresh session, so its post-guidance bounces accumulate. On the
    `_CHURN_THRESHOLD`-th distinct session within `_CHURN_WINDOW` the gate
    fails open for that caller — the tripping call is admitted (the caller
    eats threshold-1 bounced round-trips), logged once with the likely cause.
    The window keeps rare bounced-then-abandoned sessions from accumulating
    into a false positive over a long process lifetime, and the flag expires
    after `_UNSTABLE_TTL` so a false positive decays. The caller key scopes
    only this detector, never the gate itself: per-session semantics are
    unchanged for every session-preserving client, including a second
    concurrent conversation by the same user.
    """

    # Seam for tests; also the clock ValidationCache already uses.
    _now = staticmethod(time.monotonic)

    def __init__(self, *, is_http: bool) -> None:
        super().__init__()
        self._is_http = is_http
        self._seen: set[str] = set()
        self._warned_sessionless = False
        # Churn-detector state, keyed by caller digest (never the raw key).
        self._guided_callers: set[str] = set()
        self._bounced_sessions: dict[str, dict[str, float]] = {}
        self._unstable_callers: dict[str, float] = {}

    def _session_key(self) -> str | None:
        sid = request_context.snapshot().get("session_id")
        if sid:
            return sid
        # No session id: stdio is legitimately keyless (gate via the synthetic
        # key); HTTP means a stateless client we can't scope (fail open).
        return None if self._is_http else _STDIO_KEY

    @staticmethod
    def _caller_key() -> str | None:
        key = passthrough.key_from_request()
        return passthrough.digest(key) if key else None

    def _register_bounce(self, caller: str, sid: str) -> bool:
        """Record a gated bounce for `caller` in session `sid`; True admits.

        Only post-guidance bounces count — a bounce before the caller has ever
        fetched guidance is the normal first-contact flow, and fresh sessions
        alone are not churn evidence.
        """
        now = self._now()
        flagged_at = self._unstable_callers.get(caller)
        if flagged_at is not None:
            if now - flagged_at < _UNSTABLE_TTL:
                return True
            del self._unstable_callers[caller]
        if caller not in self._guided_callers:
            return False
        bounced = self._bounced_sessions.setdefault(caller, {})
        for stale in [s for s, t in bounced.items() if now - t >= _CHURN_WINDOW]:
            del bounced[stale]
        bounced[sid] = now
        if len(bounced) < _CHURN_THRESHOLD:
            return False
        del self._bounced_sessions[caller]
        self._unstable_callers[caller] = now
        logger.warning(
            "guidance gate: caller %s… seen in %d distinct fresh sessions "
            "within %ds of fetching guidance — Mcp-Session-Id appears unstable "
            "(bridge re-initializing per request?); admitting this caller "
            "ungated",
            caller[:12],
            _CHURN_THRESHOLD,
            int(_CHURN_WINDOW),
        )
        return True

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
            caller = self._caller_key()
            if caller is not None:
                self._guided_callers.add(caller)
                bounced = self._bounced_sessions.get(caller)
                if bounced is not None and key in bounced:
                    # Guidance arrived in a session we previously bounced:
                    # this caller's session ids are stable after all.
                    del self._bounced_sessions[caller]
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

        caller = self._caller_key()
        if caller is not None and self._register_bounce(caller, key):
            return await call_next(context)

        return _gate_notice(tool)
