"""Portainer Docker/Kubernetes proxy tools.

Each proxied response is run through two optional shapers, in order:
1. JMESPath projection via the caller-supplied `select` expression.
2. Hard char-count cap, to bound model context spend.

See `docs/proxy-tools.md` for design rationale and the planned evolution
to resource-spillover if filtering alone proves insufficient.
"""

from __future__ import annotations

import json
import logging
from typing import Annotated

import httpx
import jmespath
from fastmcp import FastMCP
from pydantic import Field

logger = logging.getLogger("portainer_mcp")

# Target ~25k tokens. Dense JSON (Docker/K8s payloads with IDs, hashes,
# nested structure) packs at ~3 chars/token, so 75k chars is a deliberately
# conservative cap. This is a safety valve, not a precise meter — exact
# token counts vary with content. Override via PORTAINER_PROXY_MAX_CHARS.
DEFAULT_MAX_RESPONSE_CHARS = 75_000

_ALLOWED_METHODS = frozenset({"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"})


def _apply_select(text: str, select: str | None) -> str:
    if not select:
        return text
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return text  # not JSON (plain text, binary, error page); pass through
    try:
        data = jmespath.search(select, data)
    except jmespath.exceptions.JMESPathError as e:
        return json.dumps(
            {
                "error": "invalid JMESPath expression",
                "expression": select,
                "detail": str(e),
            }
        )
    return json.dumps(data, indent=2)


def _enforce_budget(text: str, max_chars: int) -> str:
    if len(text) <= max_chars:
        return text
    return (
        text[:max_chars]
        + f"\n\n[truncated: response was {len(text)} chars, capped at {max_chars}. "
        + "Narrow the `select` JMESPath expression or refine `path`/`query_params`.]"
    )


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
    if not path.startswith("/"):
        raise ValueError("path must start with a leading slash")
    method = method.upper()
    if method not in _ALLOWED_METHODS:
        raise ValueError(f"invalid HTTP method: {method}")

    url = f"/endpoints/{environment_id}/{kind}{path}"
    response = await client.request(
        method,
        url,
        params=query_params or None,
        headers=headers or None,
        content=body if body else None,
    )
    return response.text


def register(
    mcp: FastMCP,
    client: httpx.AsyncClient,
    *,
    read_only: bool,
    max_response_chars: int = DEFAULT_MAX_RESPONSE_CHARS,
) -> None:
    """Register the proxy tools on `mcp`.

    In read-only mode, non-GET methods are rejected at tool-invocation time.
    `max_response_chars` caps the post-filter body size; oversized responses
    are truncated with a hint to narrow `select`.
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
            str | None,
            Field(
                description="JMESPath expression to project the response server-side"
            ),
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
        return _enforce_budget(_apply_select(text, select), max_response_chars)

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
            str | None,
            Field(
                description="JMESPath expression to project the response server-side"
            ),
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
        return _enforce_budget(_apply_select(text, select), max_response_chars)

    logger.info(
        "proxy tools registered (read_only=%s, max_response_chars=%d)",
        read_only,
        max_response_chars,
    )
