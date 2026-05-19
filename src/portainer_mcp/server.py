"""Portainer MCP server, bootstrapped from the Portainer OpenAPI spec via FastMCP.

Requires PORTAINER_URL and PORTAINER_API_KEY. Tunables:

- PORTAINER_PROFILES (default: BASE,DOCKER,KUBERNETES) — see `docs/profiles.md`.
- PORTAINER_TAGS_EXTRA — comma-separated tags to append, escape hatch for
  surfaces no profile covers.
- PORTAINER_READ_ONLY=1 — strict: registers GET/HEAD operations only.
- PORTAINER_NO_PROXY=1 — skip `docker_proxy` / `kubernetes_proxy` registration.
- PORTAINER_TLS_VERIFY=0 — skip TLS verification (self-signed certs).
- PORTAINER_MCP_LOG — override the log file path.
"""

from __future__ import annotations

import asyncio
import logging
import os
from pathlib import Path

import httpx
import yaml
from fastmcp import FastMCP
from fastmcp.server.providers.openapi import MCPType, RouteMap

from portainer_mcp import profiles, proxy, shaping

_ROOT = Path(__file__).resolve().parents[2]
SPEC_PATH = _ROOT / "spec" / "portainer-patched.yaml"
LOG_PATH = _ROOT / "logs" / "portainer-mcp.log"

logger = logging.getLogger("portainer_mcp")


def _env_flag(name: str, *, default: bool) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw not in {"0", "false", "False"}


def _spec_tags(spec: dict) -> set[str]:
    return {
        tag
        for path in spec.get("paths", {}).values()
        if isinstance(path, dict)
        for op in path.values()
        if isinstance(op, dict)
        for tag in op.get("tags", []) or ()
    }


def _setup_logging() -> None:
    log_path = Path(os.environ.get("PORTAINER_MCP_LOG") or LOG_PATH)
    log_path.parent.mkdir(parents=True, exist_ok=True)
    handler = logging.FileHandler(log_path, mode="a")
    handler.setFormatter(
        logging.Formatter("%(asctime)s %(levelname)s %(name)s %(message)s")
    )
    for name in ("portainer_mcp", "fastmcp", "httpx"):
        log = logging.getLogger(name)
        log.setLevel(logging.DEBUG)
        log.addHandler(handler)
    logger.info("log file: %s", log_path)


async def _log_response(response: httpx.Response) -> None:
    await response.aread()
    body = response.text[:2000].replace("\n", " ")
    logger.info(
        "%s %s -> %d | %s",
        response.request.method,
        response.request.url,
        response.status_code,
        body,
    )


def build_server() -> FastMCP:
    _setup_logging()

    base = os.environ["PORTAINER_URL"].rstrip("/") + "/api"
    verify = _env_flag("PORTAINER_TLS_VERIFY", default=True)
    client = httpx.AsyncClient(
        base_url=base,
        headers={"X-API-KEY": os.environ["PORTAINER_API_KEY"]},
        verify=verify,
        timeout=30,
        event_hooks={"response": [_log_response]},
    )
    with SPEC_PATH.open() as f:
        spec = yaml.safe_load(f)

    read_only = _env_flag("PORTAINER_READ_ONLY", default=False)
    no_proxy = _env_flag("PORTAINER_NO_PROXY", default=False)
    methods = ["GET", "HEAD"] if read_only else "*"
    if read_only:
        logger.info("read-only mode: exposing GET/HEAD operations only")

    allowed_tags = profiles.resolve(
        os.environ.get("PORTAINER_PROFILES") or profiles.DEFAULT_PROFILES,
        os.environ.get("PORTAINER_TAGS_EXTRA", ""),
        known_tags=_spec_tags(spec),
    )
    if allowed_tags is None:
        route_maps = [RouteMap(methods=methods, mcp_type=MCPType.TOOL)]
        logger.info("profiles: ALL (tag filter disabled)")
    else:
        route_maps = [
            RouteMap(methods=methods, tags={tag}, mcp_type=MCPType.TOOL)
            for tag in allowed_tags
        ]
        logger.info("profiles tag set (%d): %s", len(allowed_tags), list(allowed_tags))
    route_maps.append(RouteMap(pattern=r".*", mcp_type=MCPType.EXCLUDE))

    mcp = FastMCP.from_openapi(
        openapi_spec=spec,
        client=client,
        name="portainer",
        route_maps=route_maps,
        validate_output=False,
    )
    if no_proxy:
        logger.info("proxy tools skipped (PORTAINER_NO_PROXY=1)")
    else:
        proxy.register(mcp, client, read_only=read_only)
    mcp.add_transform(shaping.SelectArgTransform())

    # Fail fast at startup rather than silently shipping tools without `select`.
    tools = asyncio.run(mcp.list_tools())
    missing = [t.name for t in tools if not shaping._has_select(t)]
    if missing:
        raise RuntimeError(
            f"SelectArgTransform did not reach {len(missing)} tool(s): "
            f"{missing[:5]}{'...' if len(missing) > 5 else ''}"
        )
    logger.info("`select` arg present on all %d tools", len(tools))

    max_chars = int(
        os.environ.get("PORTAINER_MAX_RESPONSE_CHARS")
        or shaping.DEFAULT_MAX_RESPONSE_CHARS
    )
    mcp.add_middleware(shaping.ResponseCapMiddleware(max_chars))
    logger.info("response cap: %d chars", max_chars)
    return mcp


def main() -> None:
    build_server().run()


if __name__ == "__main__":
    main()
