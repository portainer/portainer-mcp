"""Remote authorization support — per-client Portainer API key passthrough.

When PORTAINER_REMOTE_AUTH=true (HTTP transport only), each MCP client
provides its own Portainer API key via the Authorization header:

    Authorization: Bearer ptr_xxxxxxxxxxxxxxxx

The server extracts this token from the incoming HTTP request and uses it
as the X-API-KEY for upstream Portainer calls, instead of a global
PORTAINER_API_KEY env var.

This enables multi-user deployments where each caller has their own
Portainer permissions.
"""

from __future__ import annotations

import json
import logging

import httpx
from fastmcp.server.auth import AccessToken, TokenVerifier
from fastmcp.server.dependencies import get_http_request

from . import request_context

ENV_VAR = "PORTAINER_REMOTE_AUTH"

audit_logger = logging.getLogger("portainer_mcp.audit")


def get_client_token() -> str | None:
    """Extract the Portainer API key from the current HTTP request headers.

    Priority: X-Portainer-API-Key > Authorization: Bearer
    Returns None when no HTTP request is in flight (stdio mode).
    """
    try:
        request = get_http_request()
    except RuntimeError:
        return None

    # Prefer explicit custom header
    token = request.headers.get("x-portainer-api-key")
    if token:
        return token

    # Fall back to Authorization: Bearer
    auth_header = request.headers.get("authorization")
    if auth_header and auth_header.lower().startswith("bearer "):
        return auth_header[7:].strip()

    return None


class RemoteAuthVerifier(TokenVerifier):
    """Verifier that accepts any non-empty bearer token as a Portainer API key.

    Unlike StaticBearerVerifier, this does NOT compare against a fixed secret.
    The token is the user's Portainer API key — validity is checked by
    Portainer itself on the first upstream call.
    """

    async def verify_token(self, token: str) -> AccessToken | None:
        context = request_context.snapshot()
        if not token or not token.strip():
            audit_logger.warning(
                json.dumps({"event": "auth", "outcome": "empty_token", **context})
            )
            return None

        audit_logger.info(
            json.dumps({"event": "auth", "outcome": "ok", "mode": "remote", **context})
        )
        return AccessToken(
            token="remote",
            client_id="remote-user",
            scopes=[],
            expires_at=None,
        )


def inject_token_hook(request: httpx.Request) -> None:
    """httpx event hook that replaces X-API-KEY with the client's token.

    Attached to the httpx.AsyncClient via event_hooks={"request": [...]}.
    """
    token = get_client_token()
    if token:
        request.headers["x-api-key"] = token
