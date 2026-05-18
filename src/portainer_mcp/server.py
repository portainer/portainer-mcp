"""Portainer MCP server, bootstrapped from the Portainer OpenAPI spec via FastMCP.

Requires PORTAINER_URL and PORTAINER_API_KEY in the environment. Set
PORTAINER_TLS_VERIFY=0 to skip TLS verification (self-signed certs). Set
PORTAINER_MCP_LOG to override the log file path
(default: ./logs/portainer-mcp.log). Set PORTAINER_READ_ONLY=1 to expose
only GET operations as tools.
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

from portainer_mcp import proxy, shaping

_ROOT = Path(__file__).resolve().parents[2]
SPEC_PATH = _ROOT / "spec" / "portainer-patched.yaml"
LOG_PATH = _ROOT / "logs" / "portainer-mcp.log"

logger = logging.getLogger("portainer_mcp")

# Start narrow — auto-converting all 380+ Portainer operations produces a tool
# list too noisy for clients to navigate. Widen as workflows prove out.
# Note: RouteMap.tags is all-of (subset check), so use one RouteMap per tag.
ALLOWED_TAGS: tuple[str, ...] = ("endpoints", "stacks", "auth")


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
    verify = os.environ.get("PORTAINER_TLS_VERIFY", "1") not in {"0", "false", "False"}
    client = httpx.AsyncClient(
        base_url=base,
        headers={"X-API-KEY": os.environ["PORTAINER_API_KEY"]},
        verify=verify,
        timeout=30,
        event_hooks={"response": [_log_response]},
    )
    with SPEC_PATH.open() as f:
        spec = yaml.safe_load(f)

    read_only = os.environ.get("PORTAINER_READ_ONLY", "0") not in {"0", "false", "False"}
    methods = ["GET"] if read_only else "*"
    if read_only:
        logger.info("read-only mode: exposing GET operations only")

    route_maps = [
        RouteMap(methods=methods, tags={tag}, mcp_type=MCPType.TOOL)
        for tag in ALLOWED_TAGS
    ]
    route_maps.append(RouteMap(pattern=r".*", mcp_type=MCPType.EXCLUDE))

    mcp = FastMCP.from_openapi(
        openapi_spec=spec,
        client=client,
        name="portainer",
        route_maps=route_maps,
        validate_output=False,
    )
    proxy.register(mcp, client, read_only=read_only)
    mcp.add_transform(shaping.SelectArgTransform())

    # Fail fast if the Transform stopped reaching the tool list (FastMCP
    # internals shifted, registration order regressed, etc.) — better to
    # blow up at startup than to silently ship un-projected tools.
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
