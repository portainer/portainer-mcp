"""Unit tests for `src/portainer_mcp/auth.py`.

Covers the pure-data surface: token validation rules, fingerprint formatting,
and the constant-time verifier outcome. The live HTTP-handler integration
(401 + WWW-Authenticate on a real FastMCP server) needs a running server and
is not exercised here.
"""

from __future__ import annotations

import pytest

from portainer_mcp import auth


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
