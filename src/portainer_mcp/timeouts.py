"""Upstream Portainer HTTP timeout: one knob, self-diagnosing failures.

`PORTAINER_TIMEOUT` governs how long the server waits on Portainer
(read/write/pool). Connect is capped separately at 10s so a generous read
allowance never turns "Portainer is unreachable" into a minutes-long hang.
The default is well above the old hardcoded 30s because Portainer deploys
stacks synchronously — the create request holds open through image pull
and compose up (issue #80).

A timed-out write is *ambiguous*, not failed: Portainer keeps processing
after the client gives up. `TimeoutHintMiddleware` rewrites post-send
timeouts so the model verifies state before retrying — FastMCP's stock
"please retry" message is exactly the reflex that turned two timed-out
stack creates into duplicate stack records in #80.
"""

from __future__ import annotations

import math
import os

import httpx
from fastmcp.exceptions import ToolError
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.tools.tool import ToolResult

ENV_VAR = "PORTAINER_TIMEOUT"
DEFAULT_SECONDS = 120.0
CONNECT_SECONDS = 10.0

# ConnectTimeout / PoolTimeout mean the request never reached Portainer —
# no server-side state to be ambiguous about, so those keep the stock error.
_POST_SEND_TIMEOUTS = (httpx.ReadTimeout, httpx.WriteTimeout)


def resolve() -> httpx.Timeout:
    raw = os.environ.get(ENV_VAR)
    if raw is None:
        seconds = DEFAULT_SECONDS
    else:
        try:
            seconds = float(raw)
        except ValueError:
            raise SystemExit(
                f"{ENV_VAR} must be a number of seconds (got {raw!r})"
            )
        if not math.isfinite(seconds) or seconds <= 0:
            raise SystemExit(
                f"{ENV_VAR} must be a positive number of seconds (got {raw!r})"
            )
    return httpx.Timeout(seconds, connect=min(CONNECT_SECONDS, seconds))


def _post_send_timeout(exc: BaseException) -> httpx.TimeoutException | None:
    """Find a post-send timeout on the exception's cause chain.

    By the time an upstream timeout reaches middleware it is wrapped —
    OpenAPI tools raise ToolError -> ValueError -> ReadTimeout, the proxy
    tools ToolError -> ReadTimeout — so walk the chain rather than match
    the outermost type. Only explicit `raise ... from` links (`__cause__`)
    are followed: `__context__` could surface a timeout that some handler
    already caught and moved past, mislabelling the real failure.
    """
    seen: set[int] = set()
    cur: BaseException | None = exc
    while cur is not None and id(cur) not in seen:
        seen.add(id(cur))
        if isinstance(cur, _POST_SEND_TIMEOUTS):
            return cur
        cur = cur.__cause__
    return None


class TimeoutHintMiddleware(Middleware):
    """Rewrite post-send upstream timeouts into an actionable error.

    Names the ambiguity (Portainer may have completed the request), the
    safe next step (verify state before retrying), and the knob.
    """

    def __init__(self, read_seconds: float) -> None:
        super().__init__()
        self.read_seconds = read_seconds

    async def on_call_tool(
        self,
        context: MiddlewareContext,
        call_next: CallNext,
    ) -> ToolResult:
        try:
            return await call_next(context)
        except Exception as exc:
            cause = _post_send_timeout(exc)
            if cause is None:
                raise
            raise ToolError(
                f"Tool {context.message.name!r} timed out waiting for Portainer "
                f"({type(cause).__name__} after {self.read_seconds:g}s). The "
                "request may still have completed server-side — Portainer keeps "
                "processing after the client gives up. If this was a write, "
                "verify current state first (e.g. StackList after a stack "
                "create) instead of retrying blindly; a blind retry can create "
                "duplicates. If the operation legitimately needs longer (e.g. "
                "a stack deploy pulling large images), raise the "
                f"{ENV_VAR} environment variable (seconds)."
            ) from exc
