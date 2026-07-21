"""Unit tests for `src/portainer_mcp/auth_posture.py`.

The resolution fail matrix is the security property here: no combination of
env vars may boot an ungated HTTP server, and every ambiguous declaration
must stop the boot with an actionable message.
"""

from __future__ import annotations

import pytest

from portainer_mcp import auth, auth_posture, http_security, tls

ALL_VARS = (
    auth.ENV_VAR,
    auth_posture.TRUST_PROXY_AUTH_ENV,
    auth_posture.PEER_IPS_ENV,
    tls.TRUST_PROXY_ENV,
    tls.FORWARDED_IPS_ENV,
    tls.CERT_ENV,
    tls.KEY_ENV,
    tls.ALLOW_PLAINTEXT_ENV,
    http_security.ALLOWED_HOSTS_ENV,
)


@pytest.fixture(autouse=True)
def clean_env(monkeypatch):
    for var in ALL_VARS:
        monkeypatch.delenv(var, raising=False)
    return monkeypatch


def _inherited_env(monkeypatch):
    """The issue-#76 deployment: TLS-terminating identity-aware proxy."""
    monkeypatch.setenv(auth_posture.TRUST_PROXY_AUTH_ENV, "1")
    monkeypatch.setenv(tls.TRUST_PROXY_ENV, "1")
    monkeypatch.setenv(tls.FORWARDED_IPS_ENV, "10.0.0.5")
    monkeypatch.setenv(http_security.ALLOWED_HOSTS_ENV, "mcp.example.com")


def _peer_ips_env(monkeypatch):
    """Server-terminated TLS with an auth proxy in front."""
    monkeypatch.setenv(auth_posture.TRUST_PROXY_AUTH_ENV, "1")
    monkeypatch.setenv(auth_posture.PEER_IPS_ENV, "10.0.0.5")
    monkeypatch.setenv(tls.CERT_ENV, "/tls/cert.pem")
    monkeypatch.setenv(tls.KEY_ENV, "/tls/key.pem")
    monkeypatch.setenv(http_security.ALLOWED_HOSTS_ENV, "mcp.example.com:17717")


# --- gate mode (default) ------------------------------------------------------


def test_gate_mode_when_trust_flag_unset(clean_env):
    posture = auth_posture.resolve("0.0.0.0")
    assert posture.mode == "gate"
    assert posture.peer_matcher is None
    assert posture.uvicorn_kwargs == {}


def test_gate_mode_ignores_token_value(clean_env):
    # Token presence/strength is require_token's job; resolve only picks a mode.
    clean_env.setenv(auth.ENV_VAR, "x" * 64)
    assert auth_posture.resolve("0.0.0.0").mode == "gate"


def test_gate_mode_rejects_dangling_peer_allowlist(clean_env):
    # A peer allowlist without the trust flag would be silently dead security
    # config — loud-fail like every other half-configured posture.
    clean_env.setenv(auth.ENV_VAR, "x" * 64)
    clean_env.setenv(auth_posture.PEER_IPS_ENV, "10.0.0.5")
    with pytest.raises(SystemExit, match="silently ignored"):
        auth_posture.resolve("0.0.0.0")


# --- mutual exclusion ---------------------------------------------------------


def test_both_token_and_trust_proxy_refuse_to_boot(clean_env):
    _inherited_env(clean_env)
    clean_env.setenv(auth.ENV_VAR, "x" * 64)
    with pytest.raises(SystemExit, match="ambiguous auth posture"):
        auth_posture.resolve("0.0.0.0")


def test_trust_proxy_plus_plaintext_refuses_to_boot(clean_env):
    _inherited_env(clean_env)
    clean_env.setenv(tls.ALLOW_PLAINTEXT_ENV, "1")
    with pytest.raises(SystemExit, match="plaintext"):
        auth_posture.resolve("0.0.0.0")


# --- ALLOWED_HOSTS requirement ------------------------------------------------


def test_trust_proxy_requires_allowed_hosts_on_non_loopback(clean_env):
    _inherited_env(clean_env)
    clean_env.delenv(http_security.ALLOWED_HOSTS_ENV)
    with pytest.raises(SystemExit, match="PORTAINER_MCP_ALLOWED_HOSTS"):
        auth_posture.resolve("0.0.0.0")


def test_trust_proxy_loopback_bind_exempt_from_allowed_hosts(clean_env):
    _inherited_env(clean_env)
    clean_env.delenv(http_security.ALLOWED_HOSTS_ENV)
    assert auth_posture.resolve("127.0.0.1").mode == "trust_proxy"


# --- inherited shape (TLS-terminating proxy) ----------------------------------


def test_inherited_shape_resolves(clean_env):
    _inherited_env(clean_env)
    posture = auth_posture.resolve("0.0.0.0")
    assert posture.mode == "trust_proxy"
    assert posture.peer_matcher is None
    assert posture.uvicorn_kwargs == {}
    assert "10.0.0.5" in posture.description


def test_inherited_shape_requires_trust_proxy_tls(clean_env):
    clean_env.setenv(auth_posture.TRUST_PROXY_AUTH_ENV, "1")
    clean_env.setenv(http_security.ALLOWED_HOSTS_ENV, "mcp.example.com")
    with pytest.raises(SystemExit, match="per-request proxy attestation"):
        auth_posture.resolve("0.0.0.0")


def test_inherited_shape_requires_forwarded_ips(clean_env):
    _inherited_env(clean_env)
    clean_env.delenv(tls.FORWARDED_IPS_ENV)
    with pytest.raises(SystemExit, match="PORTAINER_MCP_FORWARDED_ALLOW_IPS"):
        auth_posture.resolve("0.0.0.0")


@pytest.mark.parametrize(
    "forwarded",
    ["*", "10.0.0.5,*", " * ", "0.0.0.0/0", "::/0", "10.0.0.5,0.0.0.0/0"],
)
def test_inherited_shape_rejects_wildcard_forwarded_ips(clean_env, forwarded):
    # Both the literal '*' and zero-prefix networks admit every peer once the
    # list becomes the auth boundary.
    _inherited_env(clean_env)
    clean_env.setenv(tls.FORWARDED_IPS_ENV, forwarded)
    with pytest.raises(SystemExit, match="wildcard"):
        auth_posture.resolve("0.0.0.0")


def test_inherited_shape_rejects_server_held_cert(clean_env):
    # With a server-held cert, every direct TLS connection presents
    # scheme=https and the inherited attestation attests nothing.
    _inherited_env(clean_env)
    clean_env.setenv(tls.CERT_ENV, "/tls/cert.pem")
    clean_env.setenv(tls.KEY_ENV, "/tls/key.pem")
    with pytest.raises(SystemExit, match="attestation is void"):
        auth_posture.resolve("0.0.0.0")


# --- socket-peer shape (server-terminated TLS) --------------------------------


def test_peer_ips_shape_resolves_and_disables_proxy_headers(clean_env):
    _peer_ips_env(clean_env)
    posture = auth_posture.resolve("0.0.0.0")
    assert posture.mode == "trust_proxy"
    assert posture.peer_matcher is not None
    assert posture.peer_matcher.matches("10.0.0.5")
    assert posture.uvicorn_kwargs == {"proxy_headers": False}


def test_peer_ips_conflicts_with_trust_proxy_tls(clean_env):
    _peer_ips_env(clean_env)
    clean_env.setenv(tls.TRUST_PROXY_ENV, "1")
    with pytest.raises(SystemExit, match="incompatible"):
        auth_posture.resolve("0.0.0.0")


def test_peer_ips_conflicts_with_forwarded_ips(clean_env):
    _peer_ips_env(clean_env)
    clean_env.setenv(tls.FORWARDED_IPS_ENV, "10.0.0.5")
    with pytest.raises(SystemExit, match="incompatible"):
        auth_posture.resolve("0.0.0.0")


def test_peer_ips_requires_cert_on_non_loopback(clean_env):
    _peer_ips_env(clean_env)
    clean_env.delenv(tls.CERT_ENV)
    with pytest.raises(SystemExit, match="server-terminated-TLS"):
        auth_posture.resolve("0.0.0.0")


def test_peer_ips_on_loopback_needs_no_cert(clean_env):
    clean_env.setenv(auth_posture.TRUST_PROXY_AUTH_ENV, "1")
    clean_env.setenv(auth_posture.PEER_IPS_ENV, "127.0.0.1")
    posture = auth_posture.resolve("127.0.0.1")
    assert posture.mode == "trust_proxy"
    assert posture.peer_matcher.matches("127.0.0.1")


# --- PeerMatcher --------------------------------------------------------------


def test_peer_matcher_single_ip():
    matcher = auth_posture.PeerMatcher("10.0.0.5")
    assert matcher.matches("10.0.0.5")
    assert not matcher.matches("10.0.0.6")


def test_peer_matcher_cidr_and_multiple_entries():
    matcher = auth_posture.PeerMatcher("172.18.0.0/16, 10.0.0.5")
    assert matcher.matches("172.18.4.2")
    assert matcher.matches("10.0.0.5")
    assert not matcher.matches("192.168.1.1")


def test_peer_matcher_ipv6():
    matcher = auth_posture.PeerMatcher("fd00::/8,::1")
    assert matcher.matches("fd00::1")
    assert matcher.matches("::1")
    assert not matcher.matches("2001:db8::1")


def test_peer_matcher_non_ip_peer_never_matches():
    # A unix-socket peer or unparsable host must fail closed.
    matcher = auth_posture.PeerMatcher("10.0.0.5")
    assert not matcher.matches("proxy.internal")


def test_peer_matcher_unmaps_ipv4_mapped_ipv6_peers():
    # A dual-stack bind surfaces IPv4 peers as ::ffff:a.b.c.d — they must
    # match their IPv4 allowlist entry.
    matcher = auth_posture.PeerMatcher("10.0.0.5")
    assert matcher.matches("::ffff:10.0.0.5")
    assert not matcher.matches("::ffff:10.0.0.6")


def test_peer_matcher_rejects_wildcard():
    with pytest.raises(SystemExit, match="must not contain '\\*'"):
        auth_posture.PeerMatcher("10.0.0.5,*")


@pytest.mark.parametrize("entry", ["0.0.0.0/0", "::/0", "10.0.0.5/0"])
def test_peer_matcher_rejects_zero_prefix_networks(entry):
    # Wildcards in CIDR spelling ("10.0.0.5/0" normalizes to 0.0.0.0/0 under
    # strict=False) must stop the boot like the literal '*'.
    with pytest.raises(SystemExit, match="zero-prefix"):
        auth_posture.PeerMatcher(entry)


def test_peer_matcher_rejects_garbage_entry():
    with pytest.raises(SystemExit, match="not an IP address or CIDR"):
        auth_posture.PeerMatcher("proxy.internal")


def test_peer_matcher_rejects_empty():
    with pytest.raises(SystemExit, match="empty"):
        auth_posture.PeerMatcher(" , ")
