"""`JsonBodyDirector`: a declared JSON body goes out as `{}` when no body
arguments are supplied; routes without a body keep sending nothing."""

from __future__ import annotations

import json

import httpx
from fastmcp import FastMCP
from fastmcp.server.providers.openapi import MCPType, RouteMap

from portainer_mcp import json_body

_ID = {"name": "id", "in": "path", "required": True, "schema": {"type": "integer"}}

_SPEC = {
    "openapi": "3.0.0",
    "info": {"title": "t", "version": "1"},
    "servers": [{"url": "http://test"}],
    "paths": {
        "/stacks/{id}/git/redeploy": {
            "put": {
                "operationId": "redeploy",
                "parameters": [_ID],
                "requestBody": {
                    "required": True,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {"Prune": {"type": "boolean"}},
                            }
                        }
                    },
                },
                "responses": {"200": {"description": "ok"}},
            },
        },
        "/stacks/{id}/start": {
            "post": {
                "operationId": "start",
                "parameters": [_ID],
                "responses": {"200": {"description": "ok"}},
            },
        },
    },
}


async def _call(name: str, args: dict) -> httpx.Request:
    seen: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append(request)
        return httpx.Response(200, json={"ok": True})

    client = httpx.AsyncClient(
        base_url="http://test", transport=httpx.MockTransport(handler)
    )
    mcp = FastMCP.from_openapi(
        openapi_spec=_SPEC,
        client=client,
        route_maps=[RouteMap(methods="*", mcp_type=MCPType.TOOL)],
        validate_output=False,
    )
    assert await json_body.install(mcp) == 2
    await mcp.call_tool(name, args)
    (request,) = seen
    return request


async def test_no_body_args_sends_empty_json_object():
    request = await _call("redeploy", {"id": 1})
    assert request.content == b"{}"
    assert request.headers["content-type"] == "application/json"


async def test_supplied_body_args_are_sent_unchanged():
    request = await _call("redeploy", {"id": 1, "Prune": False})
    assert json.loads(request.content) == {"Prune": False}


async def test_route_without_request_body_still_sends_nothing():
    request = await _call("start", {"id": 1})
    assert request.content == b""
    assert "content-type" not in request.headers
