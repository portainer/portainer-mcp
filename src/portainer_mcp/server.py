"""Portainer MCP server, bootstrapped from the Portainer OpenAPI spec via FastMCP.

Requires PORTAINER_URL and PORTAINER_API_KEY. Tunables:

- PORTAINER_PROFILES (default: BASE,DOCKER,KUBERNETES) — named tag bundles.
- PORTAINER_TAGS_EXTRA — comma-separated tags to append, escape hatch for
  surfaces no profile covers.
- PORTAINER_READ_ONLY=1 — strict: registers GET/HEAD operations only.
- PORTAINER_NO_PROXY=1 — skip `docker_proxy` / `kubernetes_proxy` registration.
- PORTAINER_TLS_VERIFY=0 — skip TLS verification (self-signed certs).
- PORTAINER_MCP_LOG_LEVEL — log level (default INFO; DEBUG, WARNING, ERROR, CRITICAL).
- PORTAINER_MCP_TRANSPORT — stdio (default) or http. http binds an HTTP server
  for the dev workflow and the eventual remote container.
- PORTAINER_MCP_HTTP_HOST — bind host when transport=http (default 127.0.0.1).
- PORTAINER_MCP_HTTP_PORT — bind port when transport=http (default 8000).
- PORTAINER_MCP_AUTH_TOKEN — shared bearer secret. Required when
  transport=http; ignored for stdio.
"""

from __future__ import annotations

import asyncio
import logging
import os
import sys
from importlib.resources import files

import httpx
import yaml
from fastmcp import FastMCP
from fastmcp.server.providers.openapi import MCPType, RouteMap

from portainer_mcp import auth, profiles, proxy, shaping

SPEC_PATH = files("portainer_mcp") / "data" / "portainer-patched.yaml"

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


def _resolve_log_level() -> int:
    raw = (os.environ.get("PORTAINER_MCP_LOG_LEVEL") or "INFO").upper()
    return logging.getLevelNamesMapping()[raw]


def _resolve_transport() -> str:
    raw = (os.environ.get("PORTAINER_MCP_TRANSPORT") or "stdio").lower()
    if raw not in {"stdio", "http"}:
        raise SystemExit(
            f"PORTAINER_MCP_TRANSPORT must be 'stdio' or 'http' (got {raw!r})"
        )
    return raw


def _setup_logging() -> None:
    handler = logging.StreamHandler(sys.stderr)
    handler.setFormatter(
        logging.Formatter("%(asctime)s %(levelname)s %(name)s %(message)s")
    )
    level = _resolve_log_level()
    for name in ("portainer_mcp", "fastmcp", "httpx"):
        log = logging.getLogger(name)
        log.setLevel(level)
        log.addHandler(handler)
    logger.info("logging to stderr (level=%s)", logging.getLevelName(level))


def build_server() -> FastMCP:
    _setup_logging()

    transport = _resolve_transport()
    auth_provider = None
    if transport == "http":
        token = auth.require_token(os.environ.get(auth.ENV_VAR))
        auth_provider = auth.StaticBearerVerifier(token)
        logger.info("HTTP auth: enabled (token %s)", auth.fingerprint(token))

    base = os.environ["PORTAINER_URL"].rstrip("/") + "/api"
    verify = _env_flag("PORTAINER_TLS_VERIFY", default=True)
    client = httpx.AsyncClient(
        base_url=base,
        headers={"X-API-KEY": os.environ["PORTAINER_API_KEY"]},
        verify=verify,
        timeout=30,
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
        auth=auth_provider,
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
    server = build_server()
    transport = _resolve_transport()
    if transport == "stdio":
        server.run(show_banner=False)
        return
    host = os.environ.get("PORTAINER_MCP_HTTP_HOST") or "127.0.0.1"
    port = int(os.environ.get("PORTAINER_MCP_HTTP_PORT") or 8000)
    server.run(transport="http", host=host, port=port, show_banner=False)


if __name__ == "__main__":
    main()
