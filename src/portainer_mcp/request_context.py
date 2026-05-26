"""Per-request context for audit and structured-log records.

Two layers need the same handful of fields (`client_ip`, `user_agent`,
`session_id`):

- The bearer audit log, which fires inside `verify_token` at the ASGI
  auth-middleware layer.
- The FastMCP-layer structured request log, which fires inside the
  streamable-HTTP session task.

Custom outer ContextVars don't survive the second boundary: MCP's
session manager dispatches each JSON-RPC message into a long-lived task
that captured ContextVars at session-creation time, so subsequent
requests would log the stale `initialize`-time values. Reading directly
from the live HTTP request via `get_http_request()` avoids that — MCP
SDK's `request_ctx` is set per-message, not per-session.

FastMCP's own `RequestContextMiddleware` is inserted at position 0 of
the middleware stack (see `fastmcp.server.http.create_base_app`), so it
runs outside the bearer-auth middleware. `get_http_request()` is
already populated by the time `verify_token` executes — no extra
middleware needed here.
"""

from __future__ import annotations

from fastmcp.server.dependencies import get_http_request


def snapshot() -> dict[str, str]:
    """Per-request fields drawn from the active HTTP request, or `{}` when
    none is in flight (e.g. a unit test calling the verifier directly).
    """
    try:
        request = get_http_request()
    except RuntimeError:
        return {}

    out: dict[str, str] = {}
    if request.client is not None:
        out["client_ip"] = request.client.host
    ua = request.headers.get("user-agent")
    if ua:
        out["user_agent"] = ua
    # `Mcp-Session-Id` is assigned on the response to `initialize` and
    # echoed back on every subsequent request; absent on the first call.
    sid = request.headers.get("mcp-session-id")
    if sid:
        out["session_id"] = sid
    return out
