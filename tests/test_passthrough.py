"""Unit tests for `src/portainer_mcp/passthrough.py`.

Covers the validation cache (positive-only, TTL), the `/users/me` probe, the
per-request key reader, and the upstream-injection hook's fail-closed
isolation. The live FastMCP integration (verifier → tool → upstream) needs a
running server and is not exercised here.
"""

from __future__ import annotations

import httpx
import pytest
from fastmcp.server.http import set_http_request
from starlette.requests import Request

from portainer_mcp import passthrough
from portainer_mcp.passthrough import (
    Identity,
    ValidationCache,
    identity_audit_fields,
    inject_api_key,
    key_from_request,
    resolve_ttl,
    validate,
)

ALICE = {"Id": 7, "Username": "alice", "Role": 1}


def _request(headers: dict[str, str] | None = None) -> Request:
    raw = [(k.lower().encode(), v.encode()) for k, v in (headers or {}).items()]
    return Request(
        {
            "type": "http",
            "method": "POST",
            "path": "/mcp",
            "raw_path": b"/mcp",
            "query_string": b"",
            "client": ("203.0.113.7", 51234),
            "headers": raw,
        }
    )


def _client(handler) -> httpx.AsyncClient:
    return httpx.AsyncClient(
        base_url="http://portainer/api", transport=httpx.MockTransport(handler)
    )


# --- resolve_ttl ------------------------------------------------------------


def test_resolve_ttl_default(monkeypatch):
    monkeypatch.delenv(passthrough.TTL_ENV_VAR, raising=False)
    assert resolve_ttl() == passthrough.DEFAULT_CACHE_TTL


def test_resolve_ttl_reads_env(monkeypatch):
    monkeypatch.setenv(passthrough.TTL_ENV_VAR, "120")
    assert resolve_ttl() == 120


def test_resolve_ttl_zero_allowed(monkeypatch):
    monkeypatch.setenv(passthrough.TTL_ENV_VAR, "0")
    assert resolve_ttl() == 0


@pytest.mark.parametrize("bad", ["-1", "abc", "1.5"])
def test_resolve_ttl_rejects_invalid(monkeypatch, bad):
    monkeypatch.setenv(passthrough.TTL_ENV_VAR, bad)
    with pytest.raises(SystemExit):
        resolve_ttl()


# --- ValidationCache --------------------------------------------------------


def test_cache_roundtrip():
    cache = ValidationCache(ttl=60)
    identity = Identity(user_id=7, username="alice")
    assert cache.get("ptr_key") is None
    cache.put("ptr_key", identity)
    assert cache.get("ptr_key") == identity


def test_cache_ttl_zero_disables(monkeypatch):
    cache = ValidationCache(ttl=0)
    cache.put("ptr_key", Identity(7, "alice"))
    assert cache.get("ptr_key") is None


def test_cache_expires(monkeypatch):
    clock = {"t": 1000.0}
    monkeypatch.setattr(passthrough.time, "monotonic", lambda: clock["t"])
    cache = ValidationCache(ttl=60)
    cache.put("ptr_key", Identity(7, "alice"))
    clock["t"] += 59
    assert cache.get("ptr_key") is not None
    clock["t"] += 2  # now 61s elapsed
    assert cache.get("ptr_key") is None


def test_cache_does_not_store_raw_key():
    # The raw key must never appear in the cache's internal state, not even
    # as a dict key — only its digest.
    cache = ValidationCache(ttl=60)
    cache.put("ptr_super_secret", Identity(7, "alice"))
    assert "ptr_super_secret" not in cache._entries
    assert all("ptr_super_secret" not in k for k in cache._entries)


# --- validate ---------------------------------------------------------------


async def test_validate_success_caches():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        assert request.headers["X-API-KEY"] == "ptr_alice"
        assert request.url.path == "/api/users/me"
        assert request.url.params["noEndpointAuthorizations"] == "true"
        return httpx.Response(200, json=ALICE)

    cache = ValidationCache(ttl=60)
    client = _client(handler)
    identity, validated_now = await validate(client, cache, "ptr_alice")
    assert identity == Identity(user_id=7, username="alice")
    assert validated_now is True  # cache miss → upstream call made
    # Second call is served from cache — no second upstream hit, and it reports
    # that it did not re-validate.
    again, validated_now = await validate(client, cache, "ptr_alice")
    assert again == identity
    assert validated_now is False
    assert calls["n"] == 1


async def test_validate_rejects_non_200_and_does_not_cache():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, json={"message": "invalid token"})

    cache = ValidationCache(ttl=60)
    assert await validate(_client(handler), cache, "ptr_bad") == (None, True)
    assert cache.get("ptr_bad") is None  # negatives are never cached


async def test_validate_handles_unreachable_portainer():
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("refused")

    cache = ValidationCache(ttl=60)
    assert await validate(_client(handler), cache, "ptr_alice") == (None, True)


# --- key_from_request -------------------------------------------------------


def test_key_from_request_reads_header():
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        assert key_from_request() == "ptr_alice"


def test_key_from_request_none_without_header():
    with set_http_request(_request({})):
        assert key_from_request() is None


def test_key_from_request_none_outside_request():
    assert key_from_request() is None


# --- identity_audit_fields --------------------------------------------------


def test_identity_audit_fields_from_cache():
    cache = ValidationCache(ttl=60)
    cache.put("ptr_alice", Identity(user_id=7, username="alice"))
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        assert identity_audit_fields(cache) == {
            "portainer_user_id": 7,
            "portainer_username": "alice",
        }


def test_identity_audit_fields_empty_when_uncached():
    cache = ValidationCache(ttl=60)
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        assert identity_audit_fields(cache) == {}


def test_identity_audit_fields_empty_without_cache():
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        assert identity_audit_fields(None) == {}


# --- inject_api_key (the upstream hook) -------------------------------------


async def test_inject_sets_upstream_key_from_request():
    request = httpx.Request("GET", "http://portainer/api/endpoints")
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        await inject_api_key(request)
    assert request.headers["X-API-KEY"] == "ptr_alice"


async def test_inject_leaves_existing_upstream_key():
    # The validation probe sets X-API-KEY explicitly; the hook must not clobber
    # it with the inbound header (they're the same key, but the probe owns it).
    request = httpx.Request(
        "GET", "http://portainer/api/users/me", headers={"X-API-KEY": "ptr_probe"}
    )
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_other"})):
        await inject_api_key(request)
    assert request.headers["X-API-KEY"] == "ptr_probe"


async def test_inject_fails_closed_without_request_context():
    # No in-flight request → never send a keyless upstream call.
    request = httpx.Request("GET", "http://portainer/api/endpoints")
    with pytest.raises(RuntimeError, match="no per-user Portainer API key"):
        await inject_api_key(request)


async def test_inject_fails_closed_without_header():
    request = httpx.Request("GET", "http://portainer/api/endpoints")
    with set_http_request(_request({})):
        with pytest.raises(RuntimeError, match="no per-user Portainer API key"):
            await inject_api_key(request)


async def test_inject_isolation_one_request_cannot_borrow_another():
    # Two concurrent-style requests with different keys: each injection sees
    # only its own in-flight request's header, never the other's.
    req_a = httpx.Request("GET", "http://portainer/api/endpoints")
    req_b = httpx.Request("GET", "http://portainer/api/endpoints")
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        await inject_api_key(req_a)
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_bob"})):
        await inject_api_key(req_b)
    assert req_a.headers["X-API-KEY"] == "ptr_alice"
    assert req_b.headers["X-API-KEY"] == "ptr_bob"
