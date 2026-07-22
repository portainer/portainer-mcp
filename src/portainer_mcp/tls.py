"""TLS posture enforcement for the HTTP transport.

Over HTTP this server carries two secrets on the wire — the shared gate
token and each caller's own Portainer API key (which never expires, so a
captured one is usable until manually revoked). Plaintext is therefore never
served on a non-loopback bind by accident: the operator must declare one of
three postures or the server refuses to boot.

  - server-terminated TLS: `PORTAINER_MCP_TLS_CERT` / `_TLS_KEY` are threaded
    into uvicorn's `ssl_certfile` / `ssl_keyfile`, so the process speaks HTTPS
    directly and no plaintext hop exists.
  - TLS-terminating proxy: `PORTAINER_MCP_TRUST_PROXY_TLS=1` (an explicit
    attestation) plus `PORTAINER_MCP_FORWARDED_ALLOW_IPS` tell uvicorn to trust
    `X-Forwarded-Proto` from the proxy, which rewrites the request scheme.
  - plaintext opt-out: `PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP=1` — the
    one loud, explicit escape hatch for trusted private networks.

Both encrypted shapes converge on a single runtime signal — `scheme ==
"https"` — which `TLSRequiredMiddleware` enforces as a backstop, installed
before bearer-auth so a plaintext request is rejected before the per-user key
is ever validated upstream. Loopback binds are exempt so `make dev` works
unconfigured. There is deliberately no auto self-signed posture: real MCP
clients reject self-signed certs by default and none support fingerprint
pinning, so it would be "encrypted but unconnectable". A homelab operator can
still mount their own self-signed cert (warned, not blocked) and install its
CA on each client.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import NamedTuple

from cryptography import x509
from cryptography.exceptions import InvalidSignature
from starlette.responses import PlainTextResponse
from starlette.types import ASGIApp, Receive, Scope, Send

from .http_security import LOOPBACK_BINDS

CERT_ENV = "PORTAINER_MCP_TLS_CERT"
KEY_ENV = "PORTAINER_MCP_TLS_KEY"
TRUST_PROXY_ENV = "PORTAINER_MCP_TRUST_PROXY_TLS"
FORWARDED_IPS_ENV = "PORTAINER_MCP_FORWARDED_ALLOW_IPS"
ALLOW_PLAINTEXT_ENV = "PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP"


# Mirrors server.py's `_env_flag` (not importable from there — `server`
# imports this module, so importing back would be circular). Public because
# `auth_posture` shares it.
def flag(name: str) -> bool:
    raw = os.environ.get(name)
    return raw is not None and raw not in {"0", "false", "False"}


# Raised from two resolvers (`resolve_posture` here, `auth_posture.resolve`
# earlier in boot) — one constant so the same misconfiguration can't produce
# two divergent messages.
TRUST_PROXY_REQUIRES_IPS = (
    f"{TRUST_PROXY_ENV}=1 requires {FORWARDED_IPS_ENV}=<proxy ip/subnet> "
    f"so the runtime scheme check only trusts X-Forwarded-Proto from "
    f"the proxy"
)


class Posture(NamedTuple):
    """Resolved transport posture. `uvicorn_kwargs` is merged into the
    `uvicorn_config` dict; `enforce_https` decides whether the runtime scheme
    check is installed; `insecure_transport` marks the audit log; `warnings`
    are operator-facing lines for `main()` to log.
    """

    uvicorn_kwargs: dict[str, str]
    enforce_https: bool
    insecure_transport: bool
    warnings: tuple[str, ...]


def is_self_signed(cert_path: str) -> bool:
    """True if the leaf cert is self-signed (issuer name matches subject *and*
    the signature verifies against its own public key). A cert signed by a
    private internal CA is not flagged — the server only holds the leaf and
    can't see the chain — so this catches the literal self-signed case only.

    Loud-fails (SystemExit) on a cert that's readable but not parseable as PEM,
    rather than letting the raw cryptography error surface — the readability
    check ran upstream, so reaching here with garbage is a malformed cert.
    """
    try:
        cert = x509.load_pem_x509_certificate(Path(cert_path).read_bytes())
    except (OSError, ValueError) as exc:
        raise SystemExit(
            f"{CERT_ENV} is not a valid PEM certificate: {cert_path}"
        ) from exc
    if cert.issuer != cert.subject:
        return False
    try:
        cert.verify_directly_issued_by(cert)
    except (ValueError, TypeError, InvalidSignature):
        return False
    return True


_PLAINTEXT_WARNING = (
    f"{ALLOW_PLAINTEXT_ENV} is set — serving plain HTTP. The gate token and "
    "every caller's Portainer API key cross the wire unencrypted; anyone on "
    "the network path can capture them, and a captured Portainer key works "
    "until manually revoked. Acceptable only on a trusted private network you "
    "fully control."
)


def resolve_posture(bind_host: str) -> Posture:
    """Validate the declared posture and return the resolved transport config.

    Raises `SystemExit` (loud-fail, like the auth-token check) on any broken
    declaration so the server never silently downgrades. On a non-loopback
    bind it refuses to boot unless an encrypted posture or the explicit
    plaintext opt-out is declared.
    """
    cert = os.environ.get(CERT_ENV)
    key = os.environ.get(KEY_ENV)
    trust_proxy = flag(TRUST_PROXY_ENV)
    forwarded_ips = os.environ.get(FORWARDED_IPS_ENV)
    allow_plaintext = flag(ALLOW_PLAINTEXT_ENV)
    bind_is_loopback = bind_host in LOOPBACK_BINDS

    uvicorn_kwargs: dict[str, str] = {}
    warnings: list[str] = []

    if bool(cert) != bool(key):
        raise SystemExit(
            f"{CERT_ENV} and {KEY_ENV} must be set together "
            f"(one without the other is a half-configured TLS posture)"
        )
    if cert and key:
        for label, path in ((CERT_ENV, cert), (KEY_ENV, key)):
            if not os.access(path, os.R_OK):
                raise SystemExit(f"{label} is not a readable file: {path}")
        uvicorn_kwargs["ssl_certfile"] = cert
        uvicorn_kwargs["ssl_keyfile"] = key
        if is_self_signed(cert):
            warnings.append(
                f"{CERT_ENV} ({cert}) appears self-signed — most MCP clients "
                "reject self-signed certs by default. For a homelab, install "
                "this cert's CA on each client; otherwise use a CA-signed cert."
            )

    if trust_proxy and not forwarded_ips:
        raise SystemExit(TRUST_PROXY_REQUIRES_IPS)
    if forwarded_ips:
        uvicorn_kwargs["forwarded_allow_ips"] = forwarded_ips

    encrypted = bool(cert and key) or trust_proxy

    if not bind_is_loopback and not encrypted and not allow_plaintext:
        raise SystemExit(
            f"bind host {bind_host!r} is non-loopback but no transport posture "
            f"is declared. Set one of: {CERT_ENV} + {KEY_ENV} (server-terminated "
            f"TLS), {TRUST_PROXY_ENV}=1 + {FORWARDED_IPS_ENV} (TLS-terminating "
            f"proxy), or {ALLOW_PLAINTEXT_ENV}=1 (serve plaintext — unencrypted, "
            f"trusted networks only)."
        )

    insecure_transport = allow_plaintext and not encrypted
    if insecure_transport:
        warnings.append(_PLAINTEXT_WARNING)
    elif allow_plaintext and encrypted:
        warnings.append(
            f"{ALLOW_PLAINTEXT_ENV} is set but a TLS posture is also "
            f"configured — serving TLS, the plaintext opt-out has no effect."
        )

    return Posture(
        uvicorn_kwargs=uvicorn_kwargs,
        enforce_https=encrypted and not bind_is_loopback,
        insecure_transport=insecure_transport,
        warnings=tuple(warnings),
    )


_UPGRADE_HINT = (
    "This server requires TLS. The request arrived over plain HTTP. Terminate "
    f"TLS in the server ({CERT_ENV}/{KEY_ENV}) or at a trusted proxy "
    f"({TRUST_PROXY_ENV}=1 + {FORWARDED_IPS_ENV}); to serve plaintext on a "
    f"trusted network set {ALLOW_PLAINTEXT_ENV}=1."
)


class TLSRequiredMiddleware:
    """Reject any request whose scheme isn't `https`.

    A backstop below the boot-time posture check: uvicorn sets the scheme from
    real TLS (server-terminated) or from a trusted `X-Forwarded-Proto` (proxy),
    so this one check is shape-agnostic. Installed before the bearer-auth
    backend (via the verifier's `get_middleware`) so a plaintext request is
    rejected before the per-user key is touched or forwarded upstream — the
    loopback exemption is keyed on the bind host at install time, never on the
    per-request client IP.
    """

    def __init__(self, app: ASGIApp) -> None:
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] == "http" and scope.get("scheme") != "https":
            await PlainTextResponse(
                _UPGRADE_HINT, status_code=426, headers={"Upgrade": "TLS/1.2"}
            )(scope, receive, send)
            return
        await self.app(scope, receive, send)
