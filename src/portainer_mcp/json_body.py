"""Send `{}` instead of no body when a route declares a JSON request body.

FastMCP's `RequestDirector` builds the request body only from body-located
tool arguments, so a write tool whose body fields are all optional sends *no*
request body when the model supplies none. Portainer decodes payloads with a
bare `json.Decoder`, which rejects an empty body with `400 Invalid request
payload: EOF` — so a "redeploy as configured" `StackGitRedeploy` could never
succeed without a decoy field. Routes without a declared body are untouched.
"""

from __future__ import annotations

from typing import Any

from fastmcp import FastMCP
from fastmcp.server.providers.openapi import OpenAPITool
from fastmcp.utilities.openapi.director import RequestDirector
from fastmcp.utilities.openapi.models import HTTPRoute


class JsonBodyDirector(RequestDirector):
    def _unflatten_arguments(
        self, route: HTTPRoute, flat_args: dict[str, Any]
    ) -> tuple[dict[str, Any], dict[str, Any], dict[str, Any], dict[str, Any], Any]:
        path, query, headers, cookies, body = super()._unflatten_arguments(
            route, flat_args
        )
        if body is None and _declares_json_body(route):
            body = {}
        return path, query, headers, cookies, body


def _declares_json_body(route: HTTPRoute) -> bool:
    content = route.request_body.content_schema if route.request_body else {}
    return any(
        media.split(";")[0].strip().lower() == "application/json"
        for media in content
    )


async def install(mcp: FastMCP) -> int:
    """Point every OpenAPI tool at a `JsonBodyDirector`; returns how many.

    FastMCP creates the tools eagerly inside `from_openapi` and hands each one
    the provider's director at construction, so there is no hook to supply a
    director up front — swap it on the tools afterwards. Must run before
    `SelectArgTransform` is added: the transform wraps the tools, and the
    wrappers are what `list_tools()` returns from then on.
    """
    tools = [t for t in await mcp.list_tools() if isinstance(t, OpenAPITool)]
    if not tools:
        return 0
    director = JsonBodyDirector(tools[0]._director._spec)
    for tool in tools:
        tool._director = director
    return len(tools)
