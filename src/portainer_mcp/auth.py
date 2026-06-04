"""HTTP bearer-token auth for the streamable-HTTP transport.

A shared secret gates the HTTP `/mcp` endpoint (`StaticBearerVerifier`).
Over HTTP that gate is layered with per-user key validation
(`PassthroughVerifier`): the gate admits the request, then the caller's own
Portainer key is validated before the request is trusted. Stdio transport is
unaffected (no auth).
"""

from __future__ import annotations

import hmac
import json
import logging

import httpx
from fastmcp.server.auth import AccessToken, TokenVerifier
from starlette.middleware import Middleware

from . import passthrough, request_context

ENV_VAR = "PORTAINER_MCP_AUTH_TOKEN"
MIN_TOKEN_LENGTH = 32

# Dedicated sub-logger so operators can route audit events to a separate
# sink via standard logging config without touching the rest of the
# server's output.
audit_logger = logging.getLogger("portainer_mcp.audit")

# Process-wide posture set once at boot. When the operator opts into plaintext
# (PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP) every audit record carries
# `insecure_transport: true` so the log itself records that the credentials it
# admitted crossed the wire unencrypted.
_insecure_transport = False


def mark_insecure_transport() -> None:
    global _insecure_transport
    _insecure_transport = True


def _audit(outcome: str, level: int, **extra: object) -> None:
    """Emit one auth audit record. The per-request context is read here so
    every call site stays a single line; `extra` carries outcome-specific
    fields (e.g. the validated identity on `ok`) and must never include the
    token or per-user key.
    """
    record: dict[str, object] = {"event": "auth", "outcome": outcome}
    if _insecure_transport:
        record["insecure_transport"] = True
    audit_logger.log(
        level, json.dumps({**record, **request_context.snapshot(), **extra})
    )


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

    def _matches(self, token: str) -> bool:
        return hmac.compare_digest(token.encode("utf-8"), self._expected)

    def _access_token(self, token: str) -> AccessToken:
        # Store the fingerprint, not the raw secret. AccessToken is a Pydantic
        # model whose default __repr__ dumps all fields, so any downstream log
        # of request.user.access_token would leak the bearer. client_id carries
        # identity; this field isn't used for re-verification downstream.
        return AccessToken(
            token=fingerprint(token),
            client_id="portainer-mcp",
            scopes=[],
            expires_at=None,
        )

    async def verify_token(self, token: str) -> AccessToken | None:
        # With a single shared secret, fingerprinting the *expected* token
        # would just emit the same constant on every record — useless as a
        # correlator. The request-context fields (client_ip, user_agent,
        # session_id) are what actually distinguish callers here.
        if self._matches(token):
            _audit("ok", logging.INFO)
            return self._access_token(token)
        # Mismatch — don't fingerprint the attempted token. The forensic
        # signal worth keeping is "auth failed at time T from <ip/ua>";
        # the attacker's supplied bytes are noise.
        _audit("mismatch", logging.WARNING)
        return None


class PassthroughVerifier(StaticBearerVerifier):
    """Gate + per-user validation for the HTTP transport.

    Layer 1: the shared gate bearer in `Authorization` is constant-time
    compared (parent). Layer 2: the caller's own Portainer key in
    `X-Portainer-API-Key` is validated against `/users/me` (cached) before the
    request is admitted. Either failure → 401, with no fallback to a shared
    upstream key. The gate runs first so credential-less floods die at a cheap
    local 401 without ever reaching Portainer.
    """

    def __init__(
        self,
        token: str,
        client: httpx.AsyncClient,
        cache: passthrough.ValidationCache,
    ) -> None:
        super().__init__(token)
        self._client = client
        self._cache = cache
        self._pre_auth_middleware: list[Middleware] = []

    def add_pre_auth_middleware(self, middleware: Middleware) -> None:
        """Stack ASGI middleware ahead of the bearer-auth backend.

        FastMCP appends the `server.run(middleware=…)` list *after* the auth
        backend, but the provider's own `get_middleware()` is extended first —
        so anything returned here runs before `verify_token`. The TLS check
        uses this to reject a plaintext request before the per-user key is
        validated upstream (otherwise the no-TLS reject would land only after
        the key had already crossed the wire to Portainer).
        """
        self._pre_auth_middleware.append(middleware)

    def get_middleware(self) -> list:
        return [*self._pre_auth_middleware, *super().get_middleware()]

    async def verify_token(self, token: str) -> AccessToken | None:
        if not self._matches(token):
            _audit("mismatch", logging.WARNING)
            return None
        key = passthrough.key_from_request()
        if not key:
            # Gate passed but no per-user key — the request can't act as anyone.
            _audit("no_user_key", logging.WARNING)
            return None
        identity, validated_now = await passthrough.validate(
            self._client, self._cache, key
        )
        if identity is None:
            _audit("invalid_user_key", logging.WARNING)
            return None
        # Audit `ok` marks a *validation* event, not every admitted request:
        # it fires only when the key was actually checked against Portainer (a
        # cache miss). Cache hits admit silently — the per-request structured
        # log still records them with identity. Attribute by the Portainer
        # identity the key resolves to; the key itself is never logged.
        if validated_now:
            _audit("ok", logging.INFO, **identity.audit_fields())
        return self._access_token(token)
