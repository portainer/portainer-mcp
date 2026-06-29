"""Portainer Docker/Kubernetes proxy tools.

Each proxied response is run through an optional caller-supplied JMESPath
`select` projection. The global response-size cap (see `shaping.py`) is
applied by middleware after this returns, so it's not duplicated here.
"""

from __future__ import annotations

import json
import logging
from typing import Annotated

import httpx
from fastmcp import FastMCP
from fastmcp.exceptions import ToolError
from mcp.types import ToolAnnotations
from pydantic import BeforeValidator, Field

from portainer_mcp import redaction
from portainer_mcp.shaping import SELECT_DESCRIPTION, project

logger = logging.getLogger("portainer_mcp")

# Headers the model is not allowed to set via the proxy tools. X-API-KEY
# is the operator's Portainer credential; the others are common bypass /
# auth-confusion vectors that have no realistic Docker/K8s use case.
_BLOCKED_HEADERS = frozenset({"x-api-key", "authorization", "cookie", "host"})


def _apply_select(text: str, select: str | None) -> str:
    expose = redaction.is_expose_enabled()
    if expose and not select:
        return text
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return text  # not JSON (plain text, binary, error page); pass through

    if not expose:
        data, _ = redaction.redact_envs(data)
    if select:
        data = project(data, select)

    out = json.dumps(data)
    # Count sentinels left in the projected body, not what was redacted
    # upstream — a `select` that drops every env field must not still claim
    # values were redacted from a response that no longer contains them.
    redaction_count = 0 if expose else redaction.count_in(out)
    if redaction_count:
        out = f"{out}\n\n{redaction.hint(redaction_count)}"
    return out


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


def _coerce_param_map(value: object) -> object:
    """Normalize a query-params / headers argument into `dict[str, str]`.

    MCP clients differ in how they serialize object-typed arguments: some send
    a native object, others (notably Claude Desktop) send the whole value as a
    JSON string. Parse both, then stringify each value to its wire form so the
    model isn't punished for sending native bools/numbers — and so a nested
    object becomes the JSON string Docker's `filters` query expects. Keys whose
    value is None are dropped: an unset optional, not the literal "null".
    """
    if value is None:
        return None
    if isinstance(value, str):
        try:
            value = json.loads(value)
        except json.JSONDecodeError as exc:
            raise ValueError(
                f"expected a JSON object, got invalid JSON: {exc}"
            ) from exc
    if not isinstance(value, dict):
        raise ValueError(f"expected a JSON object, got {type(value).__name__}")
    return {
        str(k): v if isinstance(v, str) else json.dumps(v)
        for k, v in value.items()
        if v is not None
    }


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
    if response.is_error:
        # Surface 4xx/5xx as a tool error so the model can't mistake a failed
        # call for empty data. The body is passed through as-is (not env-
        # redacted like success bodies): error bodies overwhelmingly echo the
        # caller's own request payload, already in the model's context, rather
        # than server-side env state. Truncated because the response cap
        # middleware doesn't apply to raised exceptions.
        raise ToolError(
            f"{kind} API request failed (HTTP {response.status_code}) for "
            f"{method.upper()} {path}: {response.text[:2000]}"
        )
    return response.text


def register(mcp: FastMCP, client: httpx.AsyncClient, *, read_only: bool) -> None:
    """Register the proxy tools on `mcp`.

    In read-only mode, non-GET methods are rejected at tool-invocation time.
    The response-size cap is enforced by `ResponseCapMiddleware` (see
    `shaping.py`), not here.
    """

    # Honest because read-only mode hard-rejects non-GET below; when writes
    # are allowed the proxy can forward arbitrary DELETE/POST, so it is not
    # read-only and inherits the spec's destructiveHint default.
    proxy_annotations = ToolAnnotations(readOnlyHint=read_only)

    @mcp.tool(
        name="docker_proxy",
        annotations=proxy_annotations,
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
            dict[str, str] | None,
            BeforeValidator(_coerce_param_map),
            Field(description="Query parameters"),
        ] = None,
        headers: Annotated[
            dict[str, str] | None,
            BeforeValidator(_coerce_param_map),
            Field(description="Extra request headers"),
        ] = None,
        body: Annotated[
            str | None,
            Field(description="Raw request body string (e.g. JSON for POST)"),
        ] = None,
        select: Annotated[str | None, Field(description=SELECT_DESCRIPTION)] = None,
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
        annotations=proxy_annotations,
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
            BeforeValidator(_coerce_param_map),
            Field(description="Query parameters (labelSelector, fieldSelector, ...)"),
        ] = None,
        headers: Annotated[
            dict[str, str] | None,
            BeforeValidator(_coerce_param_map),
            Field(description="Extra request headers"),
        ] = None,
        body: Annotated[
            str | None,
            Field(description="Raw request body string (e.g. JSON manifest for POST)"),
        ] = None,
        select: Annotated[str | None, Field(description=SELECT_DESCRIPTION)] = None,
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
