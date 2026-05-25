"""HTTP bearer-token auth for the streamable-HTTP transport.

A single shared secret gates the HTTP `/mcp` endpoint. Stdio transport
is unaffected.
"""

from __future__ import annotations

import hmac
import json
import logging

from fastmcp.server.auth import AccessToken, TokenVerifier

from . import request_context

ENV_VAR = "PORTAINER_MCP_AUTH_TOKEN"
MIN_TOKEN_LENGTH = 32

# Dedicated sub-logger so operators can route audit events to a separate
# sink via standard logging config without touching the rest of the
# server's output.
audit_logger = logging.getLogger("portainer_mcp.audit")


def require_token(raw: str | None) -> str:
    """Validate the operator-supplied token. Raise SystemExit on any defect.

    Booting an unauthenticated HTTP server is the footgun this is meant to
    eliminate, so any defect fails loudly at startup.
    """
    if not raw:
        raise SystemExit(
            f"{ENV_VAR} is required when PORTAINER_MCP_TRANSPORT=http "
            f"(generate one with `openssl rand -hex 32`)"
        )
    if len(raw) < MIN_TOKEN_LENGTH:
        raise SystemExit(
            f"{ENV_VAR} must be at least {MIN_TOKEN_LENGTH} characters "
            f"(generate one with `openssl rand -hex 32`)"
        )
    if any(ch.isspace() for ch in raw):
        raise SystemExit(f"{ENV_VAR} must not contain whitespace")
    if not raw.isascii() or not raw.isprintable():
        # Zero-width chars, directional marks, and control bytes survive
        # the isspace check but mismatch the expected token at compare
        # time — reject up front so the operator sees a clear startup
        # error rather than a confusing 401.
        raise SystemExit(
            f"{ENV_VAR} must be ASCII printable "
            f"(generate one with `openssl rand -hex 32`)"
        )
    return raw


def fingerprint(token: str) -> str:
    """Masked form for logging: first4…last4. Never log the full token."""
    return f"{token[:4]}…{token[-4:]}"


class StaticBearerVerifier(TokenVerifier):
    """Constant-time bearer-token verifier for shared-secret HTTP deployments.

    FastMCP's bundled `StaticTokenVerifier` does a `dict.get(token)` lookup
    (not constant-time) and carries a "never use in production" warning;
    this subclass uses `hmac.compare_digest` on a single expected token.
    """

    def __init__(self, token: str) -> None:
        super().__init__()
        self._expected = token.encode("utf-8")

    async def verify_token(self, token: str) -> AccessToken | None:
        # With a single shared secret, fingerprinting the *expected* token
        # would just emit the same constant on every record — useless as a
        # correlator. The request-context fields (client_ip, user_agent,
        # session_id) are what actually distinguish callers here.
        context = request_context.snapshot()
        if hmac.compare_digest(token.encode("utf-8"), self._expected):
            audit_logger.info(json.dumps({"event": "auth", "outcome": "ok", **context}))
            # Store the fingerprint, not the raw secret. AccessToken is a
            # Pydantic model whose default __repr__ dumps all fields, so
            # any downstream log of request.user.access_token would leak
            # the bearer. client_id carries identity; this field isn't
            # used for re-verification downstream.
            return AccessToken(
                token=fingerprint(token),
                client_id="portainer-mcp",
                scopes=[],
                expires_at=None,
            )
        # Mismatch — don't fingerprint the attempted token. The forensic
        # signal worth keeping is "auth failed at time T from <ip/ua>";
        # the attacker's supplied bytes are noise.
        audit_logger.warning(
            json.dumps({"event": "auth", "outcome": "mismatch", **context})
        )
        return None
