"""End-to-end checks that `readOnlyHint` reaches `list_tools()`.

The non-obvious bit is that `SelectArgTransform` re-wraps every tool, so
these assert the method-derived hint survives that wrap. The proxy tools are
checked separately since they carry the flag via decorator, not the method.
"""

from __future__ import annotations

import httpx
import pytest
from fastmcp import FastMCP
from fastmcp.server.providers.openapi import MCPType, RouteMap

from portainer_mcp import proxy, shaping
from portainer_mcp.server import _annotate_read_only

_SPEC = {
    "openapi": "3.0.0",
    "info": {"title": "t", "version": "1"},
    "servers": [{"url": "http://test"}],
    "paths": {
        "/things": {
            "get": {
                "operationId": "listThings",
                "responses": {"200": {"description": "ok"}},
            },
            "post": {
                "operationId": "createThing",
                "responses": {"200": {"description": "ok"}},
            },
        },
    },
}


def _client() -> httpx.AsyncClient:
    return httpx.AsyncClient(base_url="http://test")


async def test_read_only_hint_survives_select_wrap():
    mcp = FastMCP.from_openapi(
        openapi_spec=_SPEC,
        client=_client(),
        route_maps=[RouteMap(methods="*", mcp_type=MCPType.TOOL)],
        mcp_component_fn=_annotate_read_only,
        validate_output=False,
    )
    mcp.add_transform(shaping.SelectArgTransform())

    tools = {t.name: t for t in await mcp.list_tools()}

    assert tools["listThings"].annotations.readOnlyHint is True
    assert tools["createThing"].annotations.readOnlyHint is False
    # The wrap that injects `select` must not drop the inherited annotation.
    assert shaping._has_select(tools["listThings"])


@pytest.mark.parametrize("read_only", [True, False])
async def test_proxy_tools_track_read_only(read_only: bool):
    mcp = FastMCP(name="t")
    proxy.register(mcp, _client(), read_only=read_only)

    tools = {t.name: t for t in await mcp.list_tools()}

    for name in ("docker_proxy", "kubernetes_proxy"):
        assert tools[name].annotations.readOnlyHint is read_only
