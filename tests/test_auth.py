"""Unit tests for `src/portainer_mcp/auth.py`.

Covers the pure-data surface: token validation rules, fingerprint formatting,
and the constant-time verifier outcome. The live HTTP-handler integration
(401 + WWW-Authenticate on a real FastMCP server) needs a running server and
is not exercised here.
"""

from __future__ import annotations

import json
import logging

import httpx
import pytest
from fastmcp.server.http import set_http_request
from starlette.requests import Request

from portainer_mcp import auth, passthrough


# --- require_token -----------------------------------------------------------


def test_require_token_accepts_32_char_token():
    raw = "a" * 32
    assert auth.require_token(raw) == raw


def test_require_token_accepts_typical_hex_secret():
    raw = "deadbeef" * 8  # 64 hex chars, what `openssl rand -hex 32` emits
    assert auth.require_token(raw) == raw


@pytest.mark.parametrize("missing", [None, ""])
def test_require_token_rejects_missing(missing):
    with pytest.raises(SystemExit, match="PORTAINER_MCP_AUTH_TOKEN is required"):
        auth.require_token(missing)


def test_require_token_rejects_too_short():
    with pytest.raises(SystemExit, match="at least 32 characters"):
        auth.require_token("a" * 31)


@pytest.mark.parametrize(
    "raw",
    [
        "a" * 31 + " ",            # trailing space
        " " + "a" * 31,            # leading space
        "a" * 16 + "\n" + "a" * 15,  # embedded newline
        "a" * 16 + "\t" + "a" * 15,  # embedded tab
    ],
)
def test_require_token_rejects_whitespace(raw):
    with pytest.raises(SystemExit, match="must not contain whitespace"):
        auth.require_token(raw)


@pytest.mark.parametrize(
    "raw",
    [
        "a" * 31 + "​",       # zero-width space
        "a" * 31 + "﻿",       # BOM / zero-width no-break space
        "a" * 31 + "‮",       # right-to-left override
        "a" * 31 + "é",            # non-ASCII printable
    ],
)
def test_require_token_rejects_non_ascii_or_nonprintable(raw):
    with pytest.raises(SystemExit, match="ASCII printable"):
        auth.require_token(raw)


# --- fingerprint -------------------------------------------------------------


def test_fingerprint_shows_first_and_last_four():
    assert auth.fingerprint("abcdefghijkl") == "abcd…ijkl"


# --- StaticBearerVerifier ----------------------------------------------------


async def test_verifier_accepts_matching_token():
    token = "a" * 64
    verifier = auth.StaticBearerVerifier(token)
    access = await verifier.verify_token(token)
    assert access is not None
    assert access.client_id == "portainer-mcp"


async def test_verifier_redacts_token_in_access_object():
    # The bearer secret must not survive into AccessToken.token, since the
    # model's default repr would dump it into any downstream log line.
    token = "abcd" + "z" * 56 + "wxyz"
    verifier = auth.StaticBearerVerifier(token)
    access = await verifier.verify_token(token)
    assert access is not None
    assert access.token == "abcd…wxyz"
    assert token not in access.token


async def test_verifier_rejects_wrong_token():
    verifier = auth.StaticBearerVerifier("a" * 64)
    assert await verifier.verify_token("b" * 64) is None


async def test_verifier_rejects_empty_input():
    verifier = auth.StaticBearerVerifier("a" * 64)
    assert await verifier.verify_token("") is None


# --- audit log ---------------------------------------------------------------


async def test_verifier_audit_logs_ok_without_request_context(caplog):
    # With no ASGI request in flight (and thus no contextvars set), the
    # record is just the event + outcome — token_fp is intentionally
    # omitted since a single shared secret produces a constant value.
    token = "abcd" + "z" * 56 + "wxyz"
    verifier = auth.StaticBearerVerifier(token)
    with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
        await verifier.verify_token(token)
    records = [r for r in caplog.records if r.name == "portainer_mcp.audit"]
    assert len(records) == 1
    assert records[0].levelno == logging.INFO
    payload = json.loads(records[0].message)
    assert payload == {"event": "auth", "outcome": "ok"}


async def test_verifier_audit_logs_mismatch_without_request_context(caplog):
    verifier = auth.StaticBearerVerifier("a" * 64)
    with caplog.at_level(logging.WARNING, logger="portainer_mcp.audit"):
        await verifier.verify_token("b" * 64)
    records = [r for r in caplog.records if r.name == "portainer_mcp.audit"]
    assert len(records) == 1
    assert records[0].levelno == logging.WARNING
    payload = json.loads(records[0].message)
    assert payload == {"event": "auth", "outcome": "mismatch"}


async def test_verifier_audit_includes_request_context_when_set(caplog):
    from fastmcp.server.http import set_http_request
    from starlette.requests import Request

    token = "a" * 64
    verifier = auth.StaticBearerVerifier(token)
    request = Request(
        {
            "type": "http",
            "method": "POST",
            "path": "/mcp",
            "raw_path": b"/mcp",
            "query_string": b"",
            "client": ("203.0.113.7", 51234),
            "headers": [
                (b"user-agent", b"Inspector/0.1"),
                (b"mcp-session-id", b"sess-abc"),
            ],
        }
    )
    with set_http_request(request):
        with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
            await verifier.verify_token(token)
            await verifier.verify_token("wrong-" + "x" * 58)

    records = [r for r in caplog.records if r.name == "portainer_mcp.audit"]
    assert len(records) == 2
    ok = json.loads(records[0].message)
    bad = json.loads(records[1].message)
    for payload in (ok, bad):
        assert payload["client_ip"] == "203.0.113.7"
        assert payload["user_agent"] == "Inspector/0.1"
        assert payload["session_id"] == "sess-abc"
    assert ok["outcome"] == "ok"
    assert bad["outcome"] == "mismatch"


async def test_verifier_audit_never_logs_token_bytes(caplog):
    # Forensic value vs. secret hygiene: the audit log records that an
    # attempt happened, never what was attempted. Assert the structural
    # property — the only allowed keys are the documented audit/context
    # set — rather than substring-checking the message, which can
    # coincidentally pass even if a token leaks.
    expected = "expected-bearer-token-stays-secret" * 2  # 68 chars
    attempted = "attacker-supplied-token-stays-secret" * 2  # 72 chars
    verifier = auth.StaticBearerVerifier(expected)
    with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
        await verifier.verify_token(expected)
        await verifier.verify_token(attempted)
    allowed_keys = {
        "event",
        "outcome",
        "client_ip",
        "user_agent",
        "session_id",
        "insecure_transport",
    }
    for record in caplog.records:
        payload = json.loads(record.message)
        assert set(payload).issubset(allowed_keys), (
            f"unexpected keys in audit payload: {set(payload) - allowed_keys}"
        )


# --- PassthroughVerifier -----------------------------------------------------

GATE = "g" * 64
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


def _verifier(handler=None) -> auth.PassthroughVerifier:
    if handler is None:

        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json=ALICE)

    client = httpx.AsyncClient(
        base_url="http://portainer/api", transport=httpx.MockTransport(handler)
    )
    return auth.PassthroughVerifier(GATE, client, passthrough.ValidationCache(ttl=60))


async def test_passthrough_rejects_gate_mismatch():
    verifier = _verifier()
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        assert await verifier.verify_token("b" * 64) is None


async def test_passthrough_rejects_missing_user_key(caplog):
    verifier = _verifier()
    with set_http_request(_request({})):  # gate ok, but no per-user key
        with caplog.at_level(logging.WARNING, logger="portainer_mcp.audit"):
            assert await verifier.verify_token(GATE) is None
    outcomes = [json.loads(r.message)["outcome"] for r in caplog.records]
    assert outcomes == ["no_user_key"]


async def test_passthrough_rejects_invalid_user_key(caplog):
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, json={"message": "invalid token"})

    verifier = _verifier(handler)
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_bad"})):
        with caplog.at_level(logging.WARNING, logger="portainer_mcp.audit"):
            assert await verifier.verify_token(GATE) is None
    outcomes = [json.loads(r.message)["outcome"] for r in caplog.records]
    assert outcomes == ["invalid_user_key"]


async def test_passthrough_accepts_and_attributes_identity(caplog):
    verifier = _verifier()
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
            access = await verifier.verify_token(GATE)
    assert access is not None
    assert access.client_id == "portainer-mcp"
    assert access.token == auth.fingerprint(GATE)
    payload = json.loads(caplog.records[0].message)
    assert payload["outcome"] == "ok"
    assert payload["portainer_user_id"] == 7
    assert payload["portainer_username"] == "alice"
    assert payload["client_ip"] == "203.0.113.7"


async def test_passthrough_ok_fires_only_on_validation_not_cache_hits(caplog):
    # The audit `ok` marks a validation event (cache miss → upstream call), not
    # every admitted request. The first request validates and logs; subsequent
    # cache hits admit silently.
    handler_calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        handler_calls["n"] += 1
        return httpx.Response(200, json=ALICE)

    verifier = _verifier(handler)
    with set_http_request(_request({"X-Portainer-API-Key": "ptr_alice"})):
        with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
            assert await verifier.verify_token(GATE) is not None  # miss → ok
            assert await verifier.verify_token(GATE) is not None  # hit → silent
            assert await verifier.verify_token(GATE) is not None  # hit → silent
    oks = [r for r in caplog.records if r.name == "portainer_mcp.audit"]
    assert len(oks) == 1
    assert json.loads(oks[0].message)["outcome"] == "ok"
    assert handler_calls["n"] == 1  # only the cache miss hit Portainer


async def test_passthrough_never_logs_the_user_key(caplog):
    # The forwarded credential must never reach any audit record — only the
    # identity it resolves to.
    secret = "ptr_super_secret_value_that_must_not_leak"
    verifier = _verifier()
    with set_http_request(_request({"X-Portainer-API-Key": secret})):
        with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
            await verifier.verify_token(GATE)  # ok
            await verifier.verify_token("b" * 64)  # gate mismatch
    for record in caplog.records:
        assert secret not in record.message
    allowed_keys = {
        "event",
        "outcome",
        "client_ip",
        "user_agent",
        "session_id",
        "portainer_user_id",
        "portainer_username",
    }
    for record in caplog.records:
        payload = json.loads(record.message)
        assert set(payload).issubset(allowed_keys), (
            f"unexpected keys in audit payload: {set(payload) - allowed_keys}"
        )
