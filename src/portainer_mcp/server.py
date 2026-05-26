"""Portainer MCP server, bootstrapped from the Portainer OpenAPI spec via FastMCP.

Requires PORTAINER_URL and PORTAINER_API_KEY. Tunables:

- PORTAINER_PROFILES (default: BASE,DOCKER,KUBERNETES) — named tag bundles.
- PORTAINER_TAGS_EXTRA — comma-separated tags to append, escape hatch for
  surfaces no profile covers.
- PORTAINER_READ_ONLY=1 — strict: registers GET/HEAD operations only.
- PORTAINER_NO_PROXY=1 — skip `docker_proxy` / `kubernetes_proxy` registration.
- PORTAINER_TLS_VERIFY=0 — skip TLS verification (self-signed certs).
- PORTAINER_MCP_LOG_LEVEL — log level (default INFO; DEBUG, WARNING, ERROR, CRITICAL).
- PORTAINER_MCP_LOG_FORMAT — text (default) or json. json emits a single
  per-line JSON envelope and hoists fields from records whose message is
  itself a JSON object (audit log + request log).
- PORTAINER_MCP_TRANSPORT — stdio (default) or http. http binds an HTTP server
  for the dev workflow and the eventual remote container.
- PORTAINER_MCP_HTTP_HOST — bind host when transport=http (default 127.0.0.1).
- PORTAINER_MCP_HTTP_PORT — bind port when transport=http (default 17717).
- PORTAINER_MCP_AUTH_TOKEN — shared bearer secret. Required when
  transport=http; ignored for stdio.
- PORTAINER_MCP_ALLOWED_HOSTS — comma-separated `Host` allowlist for
  DNS-rebinding protection. Defaults to the localhost set
  (127.0.0.1, localhost, [::1]); operator must extend for non-local
  deployments. The `Origin` allowlist is hardcoded to the localhost set
  (the only browser MCP client in scope is the local Inspector).
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import sys
from importlib.resources import files

import httpx
import yaml
from fastmcp import FastMCP
from fastmcp.server.middleware.logging import StructuredLoggingMiddleware
from fastmcp.server.providers.openapi import MCPType, RouteMap
from starlette.middleware import Middleware

from portainer_mcp import (
    auth,
    http_security,
    profiles,
    proxy,
    request_context,
    shaping,
)

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


def _resolve_log_format() -> str:
    raw = (os.environ.get("PORTAINER_MCP_LOG_FORMAT") or "text").lower()
    if raw not in {"text", "json"}:
        raise SystemExit(
            f"PORTAINER_MCP_LOG_FORMAT must be 'text' or 'json' (got {raw!r})"
        )
    return raw


def _resolve_transport() -> str:
    raw = (os.environ.get("PORTAINER_MCP_TRANSPORT") or "stdio").lower()
    if raw not in {"stdio", "http"}:
        raise SystemExit(
            f"PORTAINER_MCP_TRANSPORT must be 'stdio' or 'http' (got {raw!r})"
        )
    return raw


class _JsonFormatter(logging.Formatter):
    """Single-line JSON envelope. Records whose message parses as a JSON
    object get their fields merged into the envelope, so audit and request
    records become first-class fields rather than nested strings.
    """

    def format(self, record: logging.LogRecord) -> str:
        envelope: dict[str, object] = {
            "ts": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "logger": record.name,
        }
        msg = record.getMessage()
        # Only audit + structured-request records emit a leading `{`; this
        # short-circuit avoids running the JSON parser on every plain-text
        # boot/access line just to discover it isn't JSON.
        if msg.startswith("{"):
            try:
                parsed = json.loads(msg)
            except json.JSONDecodeError:
                envelope["msg"] = msg
            else:
                if isinstance(parsed, dict):
                    envelope.update(parsed)
                else:
                    envelope["msg"] = msg
        else:
            envelope["msg"] = msg
        if record.exc_info:
            envelope["exc_info"] = self.formatException(record.exc_info)
        return json.dumps(envelope)


class _ContextualStructuredLogging(StructuredLoggingMiddleware):
    """Add per-request context (client_ip, user_agent, session_id) to every
    record. `source: "client"` from the upstream middleware just means the
    request came over the transport — useless for distinguishing callers
    when one bearer is shared across many MCP clients.
    """

    def _create_before_message(self, context):
        message = super()._create_before_message(context)
        message.update(request_context.snapshot())
        return message

    def _create_after_message(self, context, start_time):
        message = super()._create_after_message(context, start_time)
        message.update(request_context.snapshot())
        return message

    def _create_error_message(self, context, start_time, error):
        message = super()._create_error_message(context, start_time, error)
        message.update(request_context.snapshot())
        return message


def _setup_logging() -> None:
    handler = logging.StreamHandler(sys.stderr)
    fmt = _resolve_log_format()
    if fmt == "json":
        handler.setFormatter(_JsonFormatter())
    else:
        handler.setFormatter(
            logging.Formatter("%(asctime)s %(levelname)s %(name)s %(message)s")
        )
    level = _resolve_log_level()
    # fastmcp attaches RichHandlers at import time; uvicorn would install
    # its own at serve() time, but `uvicorn_config={"log_config": None}`
    # in main() suppresses that. Strip any pre-existing handlers and
    # disable propagation so a single handler at this level owns the
    # output shape regardless of import order.
    for name in (
        "portainer_mcp",
        "fastmcp",
        "httpx",
        "uvicorn",
        "uvicorn.access",
        "uvicorn.error",
    ):
        log = logging.getLogger(name)
        for existing in list(log.handlers):
            log.removeHandler(existing)
        log.setLevel(level)
        log.addHandler(handler)
        log.propagate = False
    logger.info(
        "logging to stderr (level=%s format=%s)",
        logging.getLevelName(level),
        fmt,
    )


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

    mcp.add_middleware(_ContextualStructuredLogging(include_payload_length=True))
    logger.info("structured request logging: enabled")

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
    port = int(os.environ.get("PORTAINER_MCP_HTTP_PORT") or 17717)
    settings = http_security.build_settings(
        hosts=os.environ.get(http_security.ALLOWED_HOSTS_ENV),
    )
    logger.info(
        "DNS rebinding protection: hosts=%s origins=%s",
        settings.allowed_hosts,
        settings.allowed_origins,
    )
    warning = http_security.misconfig_warning(host, settings)
    if warning is not None:
        logger.warning(warning)
    server.run(
        transport="http",
        host=host,
        port=port,
        middleware=[
            Middleware(http_security.DNSRebindingMiddleware, settings=settings),
        ],
        show_banner=False,
        # Uvicorn calls logging.config.dictConfig at server start, which
        # would overwrite the handlers we attached to uvicorn.* loggers.
        # Skip it so a single formatter owns every record.
        uvicorn_config={"log_config": None},
    )


if __name__ == "__main__":
    main()
