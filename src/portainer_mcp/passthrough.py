"""Per-user Portainer API-key passthrough for the HTTP transport.

Under HTTP there is no shared upstream key: each client sends its own
Portainer key in `X-Portainer-API-Key`. The gate bearer in `Authorization`
admits the request (`auth.PassthroughVerifier`); the per-user key is then
validated against Portainer's `/users/me` (cached, positive-only) before the
request is trusted, and injected upstream as `X-API-KEY` by an httpx request
hook that reads *only* the in-flight request — so one caller can never borrow
another's key (fails closed if no request is in flight).

The three credentials are deliberately distinct headers so the verified
credential and the forwarded credential are never the same value: the gate
token lives in `Authorization` (verified, never forwarded), the per-user key
in `X-Portainer-API-Key` (validated, then forwarded as upstream `X-API-KEY`).
"""

from __future__ import annotations

import hashlib
import logging
import os
import time
from dataclasses import dataclass

import httpx
from fastmcp.server.dependencies import get_http_request

logger = logging.getLogger("portainer_mcp")

# The client carries its own Portainer key here. Distinct from the gate bearer
# (`Authorization`) and from the upstream header (`X-API-KEY`).
USER_KEY_HEADER = "X-Portainer-API-Key"
UPSTREAM_KEY_HEADER = "X-API-KEY"

TTL_ENV_VAR = "PORTAINER_MCP_AUTH_CACHE_TTL"
DEFAULT_CACHE_TTL = 60

# CurrentUserInspect: one cheap user read that accepts an API key, returns
# id/username/role, and skips the heavy per-environment authorization map.
_VALIDATE_PATH = "/users/me"
_VALIDATE_PARAMS = {"noEndpointAuthorizations": "true"}
# Shorter than the shared client's 30s default: this runs inside the per-request
# auth path, so a slow/down Portainer must not stall every cache-miss for 30s.
_VALIDATE_TIMEOUT = 10


@dataclass(frozen=True)
class Identity:
    """The fields `/users/me` returns that are worth attributing in logs.

    Never holds the API key — only the upstream's view of who that key is.
    """

    user_id: int | None
    username: str | None

    def audit_fields(self) -> dict[str, object]:
        out: dict[str, object] = {}
        if self.user_id is not None:
            out["portainer_user_id"] = self.user_id
        if self.username is not None:
            out["portainer_username"] = self.username
        return out


def resolve_ttl() -> int:
    raw = os.environ.get(TTL_ENV_VAR)
    if raw is None:
        return DEFAULT_CACHE_TTL
    try:
        ttl = int(raw)
    except ValueError:
        raise SystemExit(
            f"{TTL_ENV_VAR} must be an integer number of seconds (got {raw!r})"
        )
    if ttl < 0:
        raise SystemExit(f"{TTL_ENV_VAR} must be >= 0 (got {ttl})")
    return ttl


def _digest(key: str) -> str:
    return hashlib.sha256(key.encode("utf-8")).hexdigest()


class ValidationCache:
    """Positive-only TTL cache keyed by the SHA-256 of the API key.

    Caching only *valid* keys is deliberate: a negative cache would lock out a
    freshly minted key for the TTL window. The TTL is a performance / DoS knob,
    not the authorization boundary — Portainer still rejects a revoked key on
    every real upstream call, so a stale entry lets a dead key pass the MCP
    front door for at most one window but never *act*. The raw key is never
    stored, not even as a dict key. `ttl == 0` disables caching (validate every
    request).
    """

    def __init__(self, ttl: int) -> None:
        self._ttl = ttl
        self._entries: dict[str, tuple[float, Identity]] = {}

    def get(self, key: str) -> Identity | None:
        if self._ttl == 0:
            return None
        entry = self._entries.get(_digest(key))
        if entry is None:
            return None
        expiry, identity = entry
        if time.monotonic() >= expiry:
            self._entries.pop(_digest(key), None)
            return None
        return identity

    def put(self, key: str, identity: Identity) -> None:
        if self._ttl == 0:
            return
        self._entries[_digest(key)] = (time.monotonic() + self._ttl, identity)


def _parse_identity(response: httpx.Response) -> Identity:
    try:
        data = response.json()
    except ValueError:
        data = {}
    if not isinstance(data, dict):
        data = {}
    return Identity(user_id=data.get("Id"), username=data.get("Username"))


async def validate(
    client: httpx.AsyncClient, cache: ValidationCache, key: str
) -> tuple[Identity | None, bool]:
    """Validate `key`, returning `(identity, validated_now)`.

    `validated_now` is True when this call made the upstream `/users/me`
    request (a cache miss), and False when the result was served from cache.
    Callers use it to log a validation as an event without emitting one on
    every cache hit. A cache hit returns immediately; a miss round-trips
    `/users/me` once and caches a positive result. Any non-200 (or an
    unreachable Portainer) is a failed validation returning `(None, True)` —
    the cache is left untouched so a transient error or a just-created key
    isn't pinned as invalid.
    """
    cached = cache.get(key)
    if cached is not None:
        return cached, False
    try:
        response = await client.get(
            _VALIDATE_PATH,
            params=_VALIDATE_PARAMS,
            headers={UPSTREAM_KEY_HEADER: key},
            timeout=_VALIDATE_TIMEOUT,
        )
    except httpx.HTTPError as exc:
        logger.warning("per-user key validation could not reach Portainer: %s", exc)
        return None, True
    if response.status_code != 200:
        return None, True
    identity = _parse_identity(response)
    cache.put(key, identity)
    return identity, True


def key_from_request() -> str | None:
    """The per-user key from the in-flight HTTP request, or None outside one."""
    try:
        request = get_http_request()
    except RuntimeError:
        return None
    return request.headers.get(USER_KEY_HEADER)


def identity_audit_fields(cache: ValidationCache | None) -> dict[str, object]:
    """Best-effort identity for the in-flight request, read from `cache` only.

    Never probes Portainer and never returns the key — purely for enriching
    the structured request log with whoever the verifier already validated.
    """
    if cache is None:
        return {}
    key = key_from_request()
    if not key:
        return {}
    identity = cache.get(key)
    return identity.audit_fields() if identity is not None else {}


async def inject_api_key(request: httpx.Request) -> None:
    """httpx request hook: forward the caller's key upstream as `X-API-KEY`.

    Reads only the in-flight MCP request, so a request can never inject another
    request's key (RA-7). The validation probe sets `X-API-KEY` explicitly and
    is left untouched. Missing key / no request context → raise: never send an
    upstream call without the caller's credential.
    """
    if request.headers.get(UPSTREAM_KEY_HEADER):
        return
    key = key_from_request()
    if not key:
        raise RuntimeError(
            "no per-user Portainer API key in the current request context"
        )
    request.headers[UPSTREAM_KEY_HEADER] = key
