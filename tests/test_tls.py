"""Unit tests for `src/portainer_mcp/tls.py` and its auth wiring.

Covers the boot-time posture resolution (loud-fail on every broken
declaration, refuse-to-boot on an undeclared non-loopback bind), the
self-signed cert warning, the runtime scheme-enforcement middleware, and the
two auth touch-points: the pre-auth middleware hook and the `insecure_transport`
audit mark.
"""

from __future__ import annotations

import datetime
import json
import logging

import httpx
import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.x509.oid import NameOID
from starlette.middleware import Middleware

from portainer_mcp import auth, passthrough, tls

_TLS_ENV = [
    tls.CERT_ENV,
    tls.KEY_ENV,
    tls.TRUST_PROXY_ENV,
    tls.FORWARDED_IPS_ENV,
    tls.ALLOW_PLAINTEXT_ENV,
]


@pytest.fixture(autouse=True)
def clean_tls_env(monkeypatch):
    """Start every test from an undeclared posture so the host's real env
    can't leak a cert path or the plaintext flag into the assertions.
    """
    for name in _TLS_ENV:
        monkeypatch.delenv(name, raising=False)


# --- cert fixtures -----------------------------------------------------------


def _write_cert(tmp_path, name, *, self_signed):
    key = ec.generate_private_key(ec.SECP256R1())
    subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, name)])
    if self_signed:
        issuer_name, issuer_key = subject, key
    else:
        ca_key = ec.generate_private_key(ec.SECP256R1())
        issuer_name = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Test CA")])
        issuer_key = ca_key
    cert = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(issuer_name)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.datetime(2020, 1, 1))
        .not_valid_after(datetime.datetime(2035, 1, 1))
        .sign(issuer_key, hashes.SHA256())
    )
    cert_path = tmp_path / f"{name}-cert.pem"
    key_path = tmp_path / f"{name}-key.pem"
    cert_path.write_bytes(cert.public_bytes(serialization.Encoding.PEM))
    key_path.write_bytes(
        key.private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.PKCS8,
            serialization.NoEncryption(),
        )
    )
    return str(cert_path), str(key_path)


# --- is_self_signed ----------------------------------------------------------


def test_is_self_signed_true_for_self_signed(tmp_path):
    cert, _ = _write_cert(tmp_path, "self", self_signed=True)
    assert tls.is_self_signed(cert) is True


def test_is_self_signed_false_for_ca_signed(tmp_path):
    cert, _ = _write_cert(tmp_path, "ca", self_signed=False)
    assert tls.is_self_signed(cert) is False


def test_is_self_signed_loud_fails_on_malformed_pem(tmp_path):
    bad = tmp_path / "garbage.pem"
    bad.write_text("this is not a certificate")
    with pytest.raises(SystemExit, match="not a valid PEM certificate"):
        tls.is_self_signed(str(bad))


# --- resolve_posture: hard fails ---------------------------------------------


@pytest.mark.parametrize("present", [tls.CERT_ENV, tls.KEY_ENV])
def test_cert_without_key_or_key_without_cert_fails(monkeypatch, present):
    monkeypatch.setenv(present, "/tmp/whatever.pem")
    with pytest.raises(SystemExit, match="must be set together"):
        tls.resolve_posture("0.0.0.0")


def test_unreadable_cert_fails(monkeypatch, tmp_path):
    monkeypatch.setenv(tls.CERT_ENV, str(tmp_path / "missing-cert.pem"))
    monkeypatch.setenv(tls.KEY_ENV, str(tmp_path / "missing-key.pem"))
    with pytest.raises(SystemExit, match="not a readable file"):
        tls.resolve_posture("0.0.0.0")


def test_trust_proxy_without_forwarded_ips_fails(monkeypatch):
    monkeypatch.setenv(tls.TRUST_PROXY_ENV, "1")
    with pytest.raises(SystemExit, match="requires PORTAINER_MCP_FORWARDED_ALLOW_IPS"):
        tls.resolve_posture("0.0.0.0")


def test_non_loopback_bind_without_posture_refuses_to_boot(monkeypatch):
    with pytest.raises(SystemExit, match="no transport posture is declared"):
        tls.resolve_posture("0.0.0.0")


# --- resolve_posture: accepted postures --------------------------------------


@pytest.mark.parametrize("host", ["127.0.0.1", "localhost", "::1"])
def test_loopback_bind_needs_no_posture(host):
    posture = tls.resolve_posture(host)
    assert posture.uvicorn_kwargs == {}
    assert posture.enforce_https is False
    assert posture.insecure_transport is False
    assert posture.warnings == ()


def test_server_terminated_tls_ca_cert(monkeypatch, tmp_path):
    cert, key = _write_cert(tmp_path, "leaf", self_signed=False)
    monkeypatch.setenv(tls.CERT_ENV, cert)
    monkeypatch.setenv(tls.KEY_ENV, key)
    posture = tls.resolve_posture("0.0.0.0")
    assert posture.uvicorn_kwargs == {"ssl_certfile": cert, "ssl_keyfile": key}
    assert posture.enforce_https is True
    assert posture.insecure_transport is False
    assert posture.warnings == ()


def test_server_terminated_tls_self_signed_warns(monkeypatch, tmp_path):
    cert, key = _write_cert(tmp_path, "selfsigned", self_signed=True)
    monkeypatch.setenv(tls.CERT_ENV, cert)
    monkeypatch.setenv(tls.KEY_ENV, key)
    posture = tls.resolve_posture("0.0.0.0")
    assert posture.enforce_https is True
    assert any("self-signed" in w for w in posture.warnings)


def test_proxy_attestation_posture(monkeypatch):
    monkeypatch.setenv(tls.TRUST_PROXY_ENV, "1")
    monkeypatch.setenv(tls.FORWARDED_IPS_ENV, "10.0.0.0/8")
    posture = tls.resolve_posture("0.0.0.0")
    assert posture.uvicorn_kwargs == {"forwarded_allow_ips": "10.0.0.0/8"}
    assert posture.enforce_https is True
    assert posture.insecure_transport is False


def test_forwarded_ips_alone_is_not_a_posture(monkeypatch):
    # forwarded_allow_ips is functional (audit attribution); without the
    # explicit attestation flag it doesn't satisfy the TLS gate.
    monkeypatch.setenv(tls.FORWARDED_IPS_ENV, "10.0.0.0/8")
    with pytest.raises(SystemExit, match="no transport posture is declared"):
        tls.resolve_posture("0.0.0.0")


def test_plaintext_opt_out(monkeypatch):
    monkeypatch.setenv(tls.ALLOW_PLAINTEXT_ENV, "1")
    posture = tls.resolve_posture("0.0.0.0")
    assert posture.uvicorn_kwargs == {}
    assert posture.enforce_https is False
    assert posture.insecure_transport is True
    assert any("unencrypted" in w for w in posture.warnings)


def test_plaintext_flag_ignored_when_tls_configured(monkeypatch, tmp_path):
    cert, key = _write_cert(tmp_path, "leaf", self_signed=False)
    monkeypatch.setenv(tls.CERT_ENV, cert)
    monkeypatch.setenv(tls.KEY_ENV, key)
    monkeypatch.setenv(tls.ALLOW_PLAINTEXT_ENV, "1")
    posture = tls.resolve_posture("0.0.0.0")
    assert posture.insecure_transport is False
    assert posture.enforce_https is True
    assert any("no effect" in w for w in posture.warnings)


# --- TLSRequiredMiddleware ---------------------------------------------------


async def _drive(scheme):
    sent: list[dict] = []
    passed = False

    async def app(scope, receive, send):
        nonlocal passed
        passed = True

    async def receive():
        return {"type": "http.request", "body": b"", "more_body": False}

    async def send(message):
        sent.append(message)

    scope = {"type": "http", "scheme": scheme, "method": "POST", "headers": []}
    await tls.TLSRequiredMiddleware(app)(scope, receive, send)
    return passed, sent


async def test_middleware_passes_https():
    passed, sent = await _drive("https")
    assert passed is True
    assert sent == []


async def test_middleware_rejects_http_with_426():
    passed, sent = await _drive("http")
    assert passed is False
    start = next(m for m in sent if m["type"] == "http.response.start")
    assert start["status"] == 426
    body = b"".join(m.get("body", b"") for m in sent if m["type"] == "http.response.body")
    assert b"requires TLS" in body


# --- auth wiring -------------------------------------------------------------

GATE = "g" * 64


def _passthrough_verifier():
    client = httpx.AsyncClient(base_url="http://portainer/api")
    return auth.PassthroughVerifier(GATE, client, passthrough.ValidationCache(ttl=60))


def test_pre_auth_middleware_runs_before_bearer_backend():
    verifier = _passthrough_verifier()
    baseline = len(verifier.get_middleware())
    mw = Middleware(tls.TLSRequiredMiddleware)
    verifier.add_pre_auth_middleware(mw)
    stacked = verifier.get_middleware()
    assert stacked[0] is mw  # outermost — ahead of AuthenticationMiddleware
    assert len(stacked) == baseline + 1


def test_plaintext_request_rejected_before_auth_backend():
    # The security invariant, exercised through the live ASGI chain (not just
    # the middleware-list order above): with the TLS check stacked ahead of the
    # bearer backend, a plaintext request 426s before `verify_token` runs — so
    # the per-user key is never validated against / forwarded to Portainer over
    # cleartext. Guards against a FastMCP refactor silently reordering the stack.
    from starlette.applications import Starlette
    from starlette.responses import PlainTextResponse
    from starlette.routing import Route
    from starlette.testclient import TestClient

    seen: list[str] = []

    class SpyVerifier(auth.PassthroughVerifier):
        async def verify_token(self, token):
            seen.append(token)
            return await super().verify_token(token)

    client = httpx.AsyncClient(base_url="http://portainer/api")
    verifier = SpyVerifier(GATE, client, passthrough.ValidationCache(ttl=60))
    verifier.add_pre_auth_middleware(Middleware(tls.TLSRequiredMiddleware))

    async def endpoint(request):
        return PlainTextResponse("ok")

    app = Starlette(
        routes=[Route("/mcp", endpoint, methods=["POST"])],
        middleware=verifier.get_middleware(),
    )
    headers = {"Authorization": f"Bearer {GATE}"}

    with TestClient(app, base_url="http://testserver") as tc:
        plaintext = tc.post("/mcp", headers=headers)
    assert plaintext.status_code == 426
    assert seen == []  # auth backend never reached on plaintext

    with TestClient(app, base_url="https://testserver") as tc:
        tc.post("/mcp", headers=headers)
    assert seen  # an https request did reach verify_token


async def test_audit_marks_insecure_transport(monkeypatch, caplog):
    monkeypatch.setattr(auth, "_insecure_transport", True)
    verifier = auth.StaticBearerVerifier("a" * 64)
    with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
        await verifier.verify_token("a" * 64)
    record = next(r for r in caplog.records if r.name == "portainer_mcp.audit")
    assert json.loads(record.message)["insecure_transport"] is True


async def test_audit_omits_insecure_transport_when_secure(caplog):
    verifier = auth.StaticBearerVerifier("a" * 64)
    with caplog.at_level(logging.INFO, logger="portainer_mcp.audit"):
        await verifier.verify_token("a" * 64)
    record = next(r for r in caplog.records if r.name == "portainer_mcp.audit")
    assert "insecure_transport" not in json.loads(record.message)
