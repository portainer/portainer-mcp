"""DNS rebinding protection for the HTTP transport.

Third-party `fastmcp` doesn't plumb `TransportSecuritySettings` through to
its streamable-HTTP manager, so the MCP SDK's built-in Host/Origin
validation is silently off in our build. This module installs the same
check as a Starlette ASGI middleware in front of the MCP routes.
"""

from __future__ import annotations

from mcp.server.transport_security import (
    TransportSecurityMiddleware,
    TransportSecuritySettings,
)
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp, Receive, Scope, Send

ALLOWED_HOSTS_ENV = "PORTAINER_MCP_ALLOWED_HOSTS"

# Mirror MCP SDK's localhost defaults so `make dev` and the documented
# `docker run -p 17717:17717` example work without extra config.
DEFAULT_ALLOWED_HOSTS = ["127.0.0.1:*", "localhost:*", "[::1]:*"]

# Origin allowlist is hardcoded — programmatic MCP clients (Claude Code,
# Desktop, SDKs) omit the Origin header and pass through; the only thing
# this list gates is browser-hosted clients, and the MCP Inspector on
# localhost is the only one in scope today. A future non-loopback browser
# client would need this knob re-introduced with a real use case.
ALLOWED_ORIGINS = [
    "http://127.0.0.1:*",
    "http://localhost:*",
    "http://[::1]:*",
]

# Treated as "the server is only listening locally" for the misconfig
# warning. `0.0.0.0` and `::` are explicitly excluded.
LOOPBACK_BINDS = frozenset({"127.0.0.1", "localhost", "::1"})


def _split_csv(raw: str | None) -> list[str]:
    return [v.strip() for v in (raw or "").split(",") if v.strip()]


def build_settings(hosts: str | None = None) -> TransportSecuritySettings:
    return TransportSecuritySettings(
        enable_dns_rebinding_protection=True,
        allowed_hosts=_split_csv(hosts) or DEFAULT_ALLOWED_HOSTS,
        allowed_origins=ALLOWED_ORIGINS,
    )


# SDK 421 body is intentionally terse ("Invalid Host header"); without a
# pointer to the knob, the operator's only signal is the stderr warning
# from the SDK plus a 421 the client sees. Hoist the env var name into the
# response body so it's actionable from either side. The 403 (Origin) case
# is deliberately not enriched: there's no operator-facing fix to point to.
_HOST_HINT = (
    f"This server's {ALLOWED_HOSTS_ENV} allowlist does not include this "
    "Host. The server operator must add this hostname to the env var "
    "(comma-separated)."
)


def _enrich(error: Response) -> Response:
    if error.status_code != 421:
        return error
    body = error.body.decode("utf-8", errors="replace") if error.body else ""
    return Response(
        f"{body}. {_HOST_HINT}" if body else _HOST_HINT,
        status_code=error.status_code,
    )


class DNSRebindingMiddleware:
    """Reject requests whose Host/Origin isn't in the configured allowlist."""

    def __init__(self, app: ASGIApp, settings: TransportSecuritySettings) -> None:
        self.app = app
        self._sec = TransportSecurityMiddleware(settings)

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        request = Request(scope, receive=receive)
        error = await self._sec.validate_request(
            request, is_post=(request.method == "POST")
        )
        if error is not None:
            await _enrich(error)(scope, receive, send)
            return
        await self.app(scope, receive, send)


def misconfig_warning(
    bind_host: str, settings: TransportSecuritySettings
) -> str | None:
    """Detect the "bound publicly, allowlist still localhost" combo.

    Returns the warning text to log, or None if the configuration is
    self-consistent. The bind half is what makes the misconfig observable
    by clients — a loopback bind keeps the localhost defaults useful.
    """
    if bind_host in LOOPBACK_BINDS:
        return None
    if list(settings.allowed_hosts) != DEFAULT_ALLOWED_HOSTS:
        return None
    return (
        f"bind host is {bind_host!r} but {ALLOWED_HOSTS_ENV} is still the "
        f"localhost defaults — non-local clients will be 421-rejected. Set "
        f"{ALLOWED_HOSTS_ENV} to the hostname(s) clients will use to reach "
        f"this server (comma-separated)."
    )
