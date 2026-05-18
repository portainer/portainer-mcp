"""Portainer Docker/Kubernetes proxy tools.

Each proxied response is run through an optional caller-supplied JMESPath
`select` projection. The global response-size cap (see `shaping.py`) is
applied by middleware after this returns, so it's not duplicated here.

See `docs/proxy-tools.md` for design rationale and the planned evolution
to resource-spillover if filtering alone proves insufficient.
"""

from __future__ import annotations

import json
import logging
from typing import Annotated

import httpx
from fastmcp import FastMCP
from pydantic import Field

from portainer_mcp.shaping import SELECT_DESCRIPTION, project

logger = logging.getLogger("portainer_mcp")

# Headers the model is not allowed to set via the proxy tools. X-API-KEY
# is the operator's Portainer credential; the others are common bypass /
# auth-confusion vectors that have no realistic Docker/K8s use case.
_BLOCKED_HEADERS = frozenset({"x-api-key", "authorization", "cookie", "host"})


def _apply_select(text: str, select: str | None) -> str:
    if not select:
        return text
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return text  # not JSON (plain text, binary, error page); pass through
    return json.dumps(project(data, select))


def _validate_path(path: str) -> None:
    if not path.startswith("/"):
        raise ValueError("path must start with a leading slash")
    if any(c in path for c in "?#"):
        raise ValueError(
            "path must not contain '?' or '#'; use `query_params` for query strings"
        )
    if ".." in path.split("/"):
        raise ValueError("path must not contain '..' segments")


def _validate_headers(headers: dict[str, str] | None) -> None:
    if not headers:
        return
    for key in headers:
        if key.lower() in _BLOCKED_HEADERS:
            raise ValueError(f"header {key!r} is not allowed")


async def _call(
    client: httpx.AsyncClient,
    *,
    kind: str,
    environment_id: int,
    method: str,
    path: str,
    query_params: dict[str, str] | None,
    headers: dict[str, str] | None,
    body: str | None,
) -> str:
    _validate_path(path)
    _validate_headers(headers)
    url = f"/endpoints/{environment_id}/{kind}{path}"
    response = await client.request(
        method.upper(),
        url,
        params=query_params or None,
        headers=headers or None,
        content=body if body else None,
    )
    return response.text


def register(mcp: FastMCP, client: httpx.AsyncClient, *, read_only: bool) -> None:
    """Register the proxy tools on `mcp`.

    In read-only mode, non-GET methods are rejected at tool-invocation time.
    The response-size cap is enforced by `ResponseCapMiddleware` (see
    `shaping.py`), not here.
    """

    @mcp.tool(
        name="docker_proxy",
        description=(
            "Proxy a Docker Engine API request through Portainer for a given environment. "
            "The `path` is forwarded as-is to the Docker daemon (e.g. /containers/json, "
            "/images/json, /containers/{id}/json). Pass a JMESPath `select` to project just "
            "the fields you need — raw Docker payloads are large. "
            "Example for /containers/json: `[].{id:Id,name:Names[0],state:State,image:Image}`."
        ),
    )
    async def docker_proxy(
        environment_id: Annotated[int, Field(description="Portainer environment ID")],
        path: Annotated[
            str,
            Field(
                description="Docker API path with leading slash, e.g. /containers/json"
            ),
        ],
        method: Annotated[str, Field(description="HTTP method")] = "GET",
        query_params: Annotated[
            dict[str, str] | None, Field(description="Query parameters")
        ] = None,
        headers: Annotated[
            dict[str, str] | None, Field(description="Extra request headers")
        ] = None,
        body: Annotated[
            str | None,
            Field(description="Raw request body string (e.g. JSON for POST)"),
        ] = None,
        select: Annotated[
            str | None, Field(description=SELECT_DESCRIPTION)
        ] = None,
    ) -> str:
        if read_only and method.upper() != "GET":
            raise ValueError("only GET requests are allowed in read-only mode")
        text = await _call(
            client,
            kind="docker",
            environment_id=environment_id,
            method=method,
            path=path,
            query_params=query_params,
            headers=headers,
            body=body,
        )
        return _apply_select(text, select)

    @mcp.tool(
        name="kubernetes_proxy",
        description=(
            "Proxy a Kubernetes API request through Portainer for a given environment. "
            "The `path` is forwarded as-is to the K8s API server (e.g. /api/v1/pods, "
            "/apis/apps/v1/namespaces/default/deployments). K8s payloads are noisy — "
            "`metadata.managedFields` alone is often 30-70% of an object — so pass a "
            "JMESPath `select` to project just what you need. Example for /api/v1/pods: "
            "`items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,node:spec.nodeName}`."
        ),
    )
    async def kubernetes_proxy(
        environment_id: Annotated[int, Field(description="Portainer environment ID")],
        path: Annotated[
            str, Field(description="Kubernetes API path with leading slash")
        ],
        method: Annotated[str, Field(description="HTTP method")] = "GET",
        query_params: Annotated[
            dict[str, str] | None,
            Field(description="Query parameters (labelSelector, fieldSelector, ...)"),
        ] = None,
        headers: Annotated[
            dict[str, str] | None, Field(description="Extra request headers")
        ] = None,
        body: Annotated[
            str | None,
            Field(description="Raw request body string (e.g. JSON manifest for POST)"),
        ] = None,
        select: Annotated[
            str | None, Field(description=SELECT_DESCRIPTION)
        ] = None,
    ) -> str:
        if read_only and method.upper() != "GET":
            raise ValueError("only GET requests are allowed in read-only mode")
        text = await _call(
            client,
            kind="kubernetes",
            environment_id=environment_id,
            method=method,
            path=path,
            query_params=query_params,
            headers=headers,
            body=body,
        )
        return _apply_select(text, select)

    logger.info("proxy tools registered (read_only=%s)", read_only)
