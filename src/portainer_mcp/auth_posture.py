"""Auth posture resolution for the HTTP transport.

Exactly one of two auth postures must be declared, mirroring the TLS posture
pattern (`tls.resolve_posture`): the server never boots with an ambiguous or
absent auth boundary.

  - gate token (default): `PORTAINER_MCP_AUTH_TOKEN` — a shared bearer in
    `Authorization` admits the request (`auth.PassthroughVerifier`).
  - trust-proxy auth: `PORTAINER_MCP_TRUST_PROXY_AUTH=1` — an identity-aware
    proxy in front (e.g. Pomerium in MCP server mode) performs the MCP OAuth
    flow and *owns* the `Authorization` header, so a static gate token cannot
    survive the hop (issue #76). The gate compare is replaced by per-request
    proof that the request transited the proxy.

That proof takes one of two shapes, because uvicorn's proxy-header handling
rewrites `scope["client"]` from `X-Forwarded-For` for connections arriving
from `PORTAINER_MCP_FORWARDED_ALLOW_IPS` — so "check the socket peer" and
"trust the proxy's forwarded headers" are mutually exclusive signals:

  - inherited (TLS-terminating proxy, the common shape): requires
    `PORTAINER_MCP_TRUST_PROXY_TLS=1` + `PORTAINER_MCP_FORWARDED_ALLOW_IPS`.
    The attestation is the request scheme: uvicorn sets `https` only from a
    trusted peer's `X-Forwarded-Proto`, and this shape holds no cert, so no
    direct connection can present `https`. The allowlist is inherited — no
    extra variable to configure.
  - socket peer (server-terminated TLS, the end-to-end-encrypted shape):
    `PORTAINER_MCP_TRUSTED_PROXY_AUTH_IPS=<ip/cidr,...>` names the proxy
    directly and uvicorn's proxy-header rewrite is disabled so
    `scope["client"]` stays the raw transport peer.

Whichever shape, `*` is a startup error — a wildcard peer allowlist would
admit any host that can reach the bind address — and the per-user
`X-Portainer-API-Key` validation floor is untouched: trust-proxy auth drops
the gate, never authentication.
"""

from __future__ import annotations

import ipaddress
import os
from typing import NamedTuple

from . import http_security, tls
from .auth import ENV_VAR as TOKEN_ENV

TRUST_PROXY_AUTH_ENV = "PORTAINER_MCP_TRUST_PROXY_AUTH"
PEER_IPS_ENV = "PORTAINER_MCP_TRUSTED_PROXY_AUTH_IPS"


class PeerMatcher:
    """Validated IP/CIDR allowlist matched against the socket peer.

    Loud-fails at construction on a wildcard, an empty list, or an entry
    that isn't an IP address or CIDR network — this list is the auth
    boundary, so a typo must stop the boot, not silently never match.
    """

    def __init__(self, raw: str) -> None:
        entries = [e.strip() for e in raw.split(",") if e.strip()]
        if not entries:
            raise SystemExit(f"{PEER_IPS_ENV} is set but empty")
        self._networks: list[ipaddress.IPv4Network | ipaddress.IPv6Network] = []
        for entry in entries:
            if entry == "*":
                raise SystemExit(
                    f"{PEER_IPS_ENV} must not contain '*' — this allowlist is "
                    f"the auth boundary, and a wildcard would admit any host "
                    f"that can reach the bind address. Pin the proxy's IP or "
                    f"subnet."
                )
            try:
                self._networks.append(ipaddress.ip_network(entry, strict=False))
            except ValueError:
                raise SystemExit(
                    f"{PEER_IPS_ENV} entry {entry!r} is not an IP address or "
                    f"CIDR network"
                )

    def matches(self, host: str) -> bool:
        try:
            ip = ipaddress.ip_address(host)
        except ValueError:
            return False
        return any(ip in net for net in self._networks)


class AuthPosture(NamedTuple):
    """Resolved auth posture. `peer_matcher` is set only for the socket-peer
    shape; `uvicorn_kwargs` is merged into the uvicorn config (it disables
    proxy-header rewriting when the socket peer is the attestation);
    `description` is the operator-facing startup log fragment.
    """

    mode: str  # "gate" | "trust_proxy"
    peer_matcher: PeerMatcher | None
    uvicorn_kwargs: dict[str, object]
    description: str


def resolve(bind_host: str) -> AuthPosture:
    """Validate the declared auth posture and return the resolved config.

    Raises `SystemExit` (loud-fail, like `tls.resolve_posture`) on every
    ambiguous or degenerate combination, so no flag combination boots an
    ungated server.
    """
    trust = tls._flag(TRUST_PROXY_AUTH_ENV)
    if not trust:
        # Gate mode. A missing token still hard-fails, in auth.require_token —
        # its error names this posture as the alternative.
        return AuthPosture("gate", None, {}, "shared gate token")

    if os.environ.get(TOKEN_ENV):
        raise SystemExit(
            f"{TOKEN_ENV} and {TRUST_PROXY_AUTH_ENV}=1 are both set — ambiguous "
            f"auth posture. The gate token cannot survive a proxy that owns "
            f"the Authorization header; declare exactly one."
        )
    if tls._flag(tls.ALLOW_PLAINTEXT_ENV):
        raise SystemExit(
            f"{TRUST_PROXY_AUTH_ENV}=1 makes each caller's X-Portainer-API-Key "
            f"the only credential this server sees — it must never cross the "
            f"wire in plaintext. Remove {tls.ALLOW_PLAINTEXT_ENV}."
        )
    bind_is_loopback = bind_host in http_security.LOOPBACK_BINDS
    if not bind_is_loopback and not os.environ.get(http_security.ALLOWED_HOSTS_ENV):
        raise SystemExit(
            f"{TRUST_PROXY_AUTH_ENV}=1 requires {http_security.ALLOWED_HOSTS_ENV} "
            f"to name the hostname the proxy fronts this server with (the "
            f"localhost defaults would 421-reject every proxied request)."
        )

    peer_ips = os.environ.get(PEER_IPS_ENV)
    trust_proxy_tls = tls._flag(tls.TRUST_PROXY_ENV)
    forwarded = os.environ.get(tls.FORWARDED_IPS_ENV)

    if peer_ips:
        if trust_proxy_tls or forwarded:
            raise SystemExit(
                f"{PEER_IPS_ENV} attests the socket peer, but "
                f"{tls.TRUST_PROXY_ENV}/{tls.FORWARDED_IPS_ENV} make uvicorn "
                f"rewrite the peer from X-Forwarded-For — the two attestations "
                f"are incompatible. Behind a TLS-terminating proxy, drop "
                f"{PEER_IPS_ENV}: {TRUST_PROXY_AUTH_ENV}=1 inherits "
                f"{tls.FORWARDED_IPS_ENV} as the peer allowlist."
            )
        if not bind_is_loopback and not os.environ.get(tls.CERT_ENV):
            raise SystemExit(
                f"{PEER_IPS_ENV} is the server-terminated-TLS shape (auth proxy "
                f"in front, this server holds the cert) — set {tls.CERT_ENV} + "
                f"{tls.KEY_ENV}, or drop {PEER_IPS_ENV} and set "
                f"{tls.TRUST_PROXY_ENV}=1 to inherit the proxy attestation."
            )
        return AuthPosture(
            "trust_proxy",
            PeerMatcher(peer_ips),
            # The socket peer is the attestation, so uvicorn must not rewrite
            # scope["client"] from X-Forwarded-For (its default trusts
            # 127.0.0.1, which would corrupt the signal for a local proxy).
            {"proxy_headers": False},
            f"socket-peer allowlist {peer_ips}",
        )

    if not trust_proxy_tls:
        raise SystemExit(
            f"{TRUST_PROXY_AUTH_ENV}=1 needs a per-request proxy attestation: "
            f"either {tls.TRUST_PROXY_ENV}=1 + {tls.FORWARDED_IPS_ENV} "
            f"(TLS-terminating proxy — the allowlist is inherited) or "
            f"{PEER_IPS_ENV}=<proxy ip/cidr> (server-terminated TLS)."
        )
    if not forwarded:
        # tls.resolve_posture repeats this check, but it runs after the
        # verifier is built — fail at the earliest read.
        raise SystemExit(
            f"{tls.TRUST_PROXY_ENV}=1 requires {tls.FORWARDED_IPS_ENV}="
            f"<proxy ip/subnet>"
        )
    if any(e.strip() == "*" for e in forwarded.split(",")):
        raise SystemExit(
            f"{tls.FORWARDED_IPS_ENV}='*' cannot back {TRUST_PROXY_AUTH_ENV}=1 "
            f"— the inherited allowlist becomes the auth boundary, and a "
            f"wildcard would admit any peer that can reach the bind address. "
            f"Pin the proxy's IP or subnet."
        )
    return AuthPosture(
        "trust_proxy",
        None,
        {},
        f"attested TLS proxy, peers inherited from {tls.FORWARDED_IPS_ENV}"
        f"={forwarded}",
    )
