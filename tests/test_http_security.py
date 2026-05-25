"""Unit tests for `src/portainer_mcp/http_security.py`.

The middleware is small and pure-data; tests exercise it via a tiny
Starlette app driven by `httpx.AsyncClient` against `ASGITransport`.
"""

from __future__ import annotations

import httpx
import pytest
from starlette.applications import Starlette
from starlette.middleware import Middleware
from starlette.responses import PlainTextResponse
from starlette.routing import Route

from portainer_mcp import http_security


def _build_app(*, hosts: str | None = None) -> Starlette:
    async def ok(_request):
        return PlainTextResponse("ok")

    settings = http_security.build_settings(hosts=hosts)
    return Starlette(
        routes=[Route("/mcp", ok, methods=["GET", "POST"])],
        middleware=[
            Middleware(http_security.DNSRebindingMiddleware, settings=settings)
        ],
    )


async def _get(app, *, host: str, origin: str | None = None) -> httpx.Response:
    headers: dict[str, str] = {}
    if origin is not None:
        headers["origin"] = origin
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(
        transport=transport, base_url=f"http://{host}"
    ) as client:
        return await client.get("/mcp", headers=headers)


# --- build_settings ----------------------------------------------------------


def test_build_settings_splits_hosts_csv_and_strips():
    s = http_security.build_settings(hosts="a, b ,c")
    assert s.allowed_hosts == ["a", "b", "c"]
    assert s.enable_dns_rebinding_protection is True


def test_build_settings_falls_back_to_localhost_default_hosts():
    s = http_security.build_settings()
    assert s.allowed_hosts == http_security.DEFAULT_ALLOWED_HOSTS


def test_build_settings_pins_origins_to_localhost_set():
    # Origins are not configurable — the localhost set is the contract.
    # Any future change here is a behavior change worth catching in review.
    assert (
        http_security.build_settings().allowed_origins
        == http_security.ALLOWED_ORIGINS
    )


# --- DNSRebindingMiddleware --------------------------------------------------


@pytest.mark.parametrize(
    "host", ["127.0.0.1:8000", "localhost:8000", "[::1]:8000"]
)
async def test_allows_default_localhost_hosts(host):
    response = await _get(_build_app(), host=host)
    assert response.status_code == 200


async def test_rejects_unknown_host():
    response = await _get(_build_app(), host="evil.example.com")
    assert response.status_code == 421
    # Body must name the env var so the operator can self-diagnose from
    # the client's 421 without grepping docs.
    assert "PORTAINER_MCP_ALLOWED_HOSTS" in response.text


async def test_allows_request_without_origin():
    # Programmatic MCP clients don't send Origin; the absence must not
    # block them. Browser-pivot attacks send Origin and are caught below.
    response = await _get(_build_app(), host="localhost:8000")
    assert response.status_code == 200


async def test_rejects_unknown_origin():
    # Origin allowlist is hardcoded to the localhost set, so a browser
    # at a non-loopback origin gets a 403 with no operator-facing knob
    # to extend — the 403 body deliberately stays terse.
    response = await _get(
        _build_app(),
        host="localhost:8000",
        origin="http://evil.example.com",
    )
    assert response.status_code == 403


async def test_operator_host_override_applies():
    app = _build_app(hosts="mcp.example.com")
    response = await _get(app, host="mcp.example.com")
    assert response.status_code == 200


# --- misconfig_warning -------------------------------------------------------


@pytest.mark.parametrize("bind", ["127.0.0.1", "localhost", "::1"])
def test_misconfig_warning_silent_on_loopback_bind(bind):
    settings = http_security.build_settings()
    assert http_security.misconfig_warning(bind, settings) is None


def test_misconfig_warning_silent_when_allowlist_extended():
    settings = http_security.build_settings(hosts="mcp.example.com")
    assert http_security.misconfig_warning("0.0.0.0", settings) is None


@pytest.mark.parametrize("bind", ["0.0.0.0", "::", "10.0.0.5"])
def test_misconfig_warning_fires_on_public_bind_with_default_allowlist(bind):
    settings = http_security.build_settings()
    msg = http_security.misconfig_warning(bind, settings)
    assert msg is not None
    assert "PORTAINER_MCP_ALLOWED_HOSTS" in msg
    assert bind in msg
