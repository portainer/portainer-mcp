"""Portainer MCP server, bootstrapped from the Portainer OpenAPI spec via FastMCP.

Requires PORTAINER_URL. Stdio also requires PORTAINER_API_KEY (the single
upstream credential); HTTP forwards each client's own key per request instead
and refuses to boot if PORTAINER_API_KEY is set. Tunables:

- PORTAINER_PROFILES (default: BASE,DOCKER,KUBERNETES) — named tag bundles.
- PORTAINER_TAGS_EXTRA — comma-separated tags to append, escape hatch for
  surfaces no profile covers.
- PORTAINER_READ_ONLY=1 — strict: registers GET/HEAD operations only.
- PORTAINER_NO_PROXY=1 — skip `docker_proxy` / `kubernetes_proxy` registration.
- PORTAINER_TLS_VERIFY=0 — skip TLS verification (self-signed certs).
- PORTAINER_MCP_GUIDANCE_TTL — idle seconds before the guidance toll booth
  re-delivers the operating guide to a caller (default 1800).
- PORTAINER_MCP_DISABLE_GUIDANCE_GATE=1 — disable in-band guide delivery
  entirely (get_guidance stays available on demand).
- PORTAINER_MCP_LOG_LEVEL — log level (default INFO; DEBUG, WARNING, ERROR, CRITICAL).
- PORTAINER_MCP_LOG_FORMAT — text (default) or json. json emits a single
  per-line JSON envelope and hoists fields from records whose message is
  itself a JSON object (audit log + request log).
- PORTAINER_MCP_TRANSPORT — stdio (default) or http. http binds an HTTP server
  for the dev workflow and the eventual remote container.
- PORTAINER_MCP_HTTP_HOST — bind host when transport=http (default 127.0.0.1).
- PORTAINER_MCP_HTTP_PORT — bind port when transport=http (default 17717).
- PORTAINER_MCP_AUTH_TOKEN — shared bearer gate secret. Required when
  transport=http; ignored for stdio. Over HTTP it admits the request; the
  caller's own Portainer key (X-Portainer-API-Key header) is then validated
  and forwarded upstream as X-API-KEY.
- PORTAINER_MCP_AUTH_CACHE_TTL — seconds to cache a validated per-user key
  (default 60, positive results only, 0 disables). HTTP only.
- PORTAINER_MCP_ALLOWED_HOSTS — comma-separated `Host` allowlist for
  DNS-rebinding protection. Defaults to the localhost set
  (127.0.0.1, localhost, [::1]); operator must extend for non-local
  deployments. The `Origin` allowlist is hardcoded to the localhost set
  (the only browser MCP client in scope is the local Inspector).
- PORTAINER_MCP_TLS_CERT / PORTAINER_MCP_TLS_KEY — PEM cert+key; uvicorn
  serves HTTPS directly (server-terminated TLS). Set together.
- PORTAINER_MCP_TRUST_PROXY_TLS=1 + PORTAINER_MCP_FORWARDED_ALLOW_IPS —
  attest a TLS-terminating proxy; trust its X-Forwarded-Proto from the
  given IPs/subnets.
- PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP=1 — serve plain HTTP
  (unencrypted) on a non-loopback bind. The one loud opt-out; otherwise a
  non-loopback bind refuses to boot without a TLS posture.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import sys
from importlib.resources import files
from pathlib import Path

import httpx
import yaml
from fastmcp import FastMCP
from fastmcp.server.middleware.logging import StructuredLoggingMiddleware
from fastmcp.server.providers.openapi import MCPType, RouteMap
from fastmcp.tools.tool import Tool
from fastmcp.utilities.openapi import HTTPRoute
from mcp.types import ToolAnnotations
from starlette.middleware import Middleware

from portainer_mcp import (
    auth,
    guidance,
    http_security,
    passthrough,
    profiles,
    proxy,
    redaction,
    request_context,
    shaping,
    tls,
)

SPEC_PATH = files("portainer_mcp") / "data" / "portainer-patched.yaml"

# Single source of truth is skills/portainer-mcp-hygiene/SKILL.md. The wheel
# (hatch force-include) and the frozen .mcpb binary (PyInstaller datas) ship a
# copy here; an editable dev checkout has neither, so _load_instructions falls
# back to the repo source.
HYGIENE_SKILL = files("portainer_mcp") / "data" / "SKILL.md"
_DEV_SKILL = (
    Path(__file__).resolve().parents[2]
    / "skills"
    / "portainer-mcp-hygiene"
    / "SKILL.md"
)

logger = logging.getLogger("portainer_mcp")

# Short pointer surfaced as MCP server instructions. The full guide can't ride
# here — clients truncate instructions at ~2KB — so this names get_guidance and
# carries only the floor the model needs even if it never calls the tool.
INSTRUCTIONS = (
    "Portainer MCP server. Its responses are large and several fields are easy "
    "to misread (e.g. an edge environment's health comes from its heartbeat, "
    "not its `Status` field), so working without guidance tends to produce "
    "wrong answers and burned context. Call the `get_guidance` tool once at the "
    "start of any Portainer task — before interpreting responses or planning "
    "multi-step changes. It pays for itself immediately: it covers projecting "
    "responses with `select`, where the heavy fields live, how to read results "
    "correctly, and how to deploy / scale / delete safely.\n\n"
    "Core practice regardless: always pass a JMESPath `select` to project large "
    "responses, and verify mutations out-of-band — Portainer write calls often "
    "return an empty body on success."
)


def _strip_frontmatter(text: str) -> str:
    # The SKILL.md YAML frontmatter (name/description/triggers) is skill-runtime
    # machinery, meaningless as server instructions — keep only the body.
    if text.startswith("---"):
        fence = text.find("\n---", 3)
        if fence != -1:
            eol = text.find("\n", fence + 4)
            if eol != -1:
                return text[eol + 1 :].lstrip()
    return text.lstrip()


def _load_guide() -> str | None:
    """The full hygiene guide, served on demand by the get_guidance tool."""
    if HYGIENE_SKILL.is_file():
        return _strip_frontmatter(HYGIENE_SKILL.read_text(encoding="utf-8"))
    if _DEV_SKILL.is_file():
        return _strip_frontmatter(_DEV_SKILL.read_text(encoding="utf-8"))
    return None


def _env_flag(name: str, *, default: bool) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw not in {"0", "false", "False"}


_READ_ONLY_METHODS = {"GET", "HEAD"}


def _annotate_read_only(route: HTTPRoute, component: Tool) -> None:
    # readOnlyHint=False (not None) on mutating tools also activates the MCP
    # spec's destructiveHint default, so write methods need no enumeration.
    if isinstance(component, Tool):
        component.annotations = ToolAnnotations(
            readOnlyHint=route.method.upper() in _READ_ONLY_METHODS
        )


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
    when one gate bearer is shared across many MCP clients. Under HTTP
    passthrough, also attribute the validated Portainer identity (read from
    the validation cache; never the key itself).
    """

    def __init__(self, *, cache: passthrough.ValidationCache | None = None, **kwargs):
        super().__init__(**kwargs)
        self._cache = cache

    def _enrich(self, message: dict, context) -> dict:
        # `source` is a Literal["client","server"] the base middleware stamps,
        # but every request-path context hard-codes "client" (server-initiated
        # messages don't flow through here) — a constant, so drop the noise.
        message.pop("source", None)
        message.update(request_context.snapshot())
        message.update(passthrough.identity_audit_fields(self._cache))
        # `method` is just "tools/call"; surface the actual tool name from the
        # call params so a request can be read without joining to the payload.
        if context.method == "tools/call":
            name = getattr(context.message, "name", None)
            if name:
                message["tool"] = name
        return message

    def _create_before_message(self, context):
        return self._enrich(super()._create_before_message(context), context)

    def _create_after_message(self, context, start_time):
        return self._enrich(super()._create_after_message(context, start_time), context)

    def _create_error_message(self, context, start_time, error):
        return self._enrich(
            super()._create_error_message(context, start_time, error), context
        )


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
    base = os.environ["PORTAINER_URL"].rstrip("/") + "/api"
    verify = _env_flag("PORTAINER_TLS_VERIFY", default=True)

    auth_provider = None
    cache: passthrough.ValidationCache | None = None
    if transport == "http":
        if os.environ.get("PORTAINER_API_KEY"):
            # HTTP is per-user passthrough: each client forwards its own key.
            # A shared PORTAINER_API_KEY here is a misconfiguration, not a
            # fallback — fail loudly rather than silently ignore it.
            raise SystemExit(
                "PORTAINER_API_KEY is only valid with PORTAINER_MCP_TRANSPORT=stdio; "
                "the HTTP transport forwards each client's own key from the "
                "X-Portainer-API-Key header — remove PORTAINER_API_KEY"
            )
        token = auth.require_token(os.environ.get(auth.ENV_VAR))
        ttl = passthrough.resolve_ttl()
        cache = passthrough.ValidationCache(ttl)
        # No baked X-API-KEY: the per-request hook injects the caller's own key.
        client = httpx.AsyncClient(
            base_url=base,
            verify=verify,
            timeout=30,
            event_hooks={"request": [passthrough.inject_api_key]},
        )
        auth_provider = auth.PassthroughVerifier(token, client, cache)
        logger.info(
            "HTTP auth: per-user passthrough (gate %s, validation cache ttl=%ds)",
            auth.fingerprint(token),
            ttl,
        )
    else:
        client = httpx.AsyncClient(
            base_url=base,
            headers={passthrough.UPSTREAM_KEY_HEADER: os.environ["PORTAINER_API_KEY"]},
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

    guide = _load_guide()
    if not guide:
        # The guide is load-bearing — the get_guidance tool and its gate both
        # depend on it. A missing guide means a broken build (the bundled copy
        # wasn't packaged) or code running outside both the wheel and the repo
        # tree. Loud-fail like the other startup invariants rather than ship a
        # server that silently can't guide the model.
        raise SystemExit(
            "hygiene guide not found (neither bundled data/SKILL.md nor the "
            "repo-source skills/portainer-mcp-hygiene/SKILL.md) — packaging defect"
        )

    mcp = FastMCP.from_openapi(
        openapi_spec=spec,
        client=client,
        name="portainer",
        route_maps=route_maps,
        mcp_component_fn=_annotate_read_only,
        validate_output=False,
        auth=auth_provider,
        instructions=INSTRUCTIONS,
    )
    if no_proxy:
        logger.info("proxy tools skipped (PORTAINER_NO_PROXY=1)")
    else:
        proxy.register(mcp, client, read_only=read_only)
    guidance.register(mcp, guide)
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

    mcp.add_middleware(
        _ContextualStructuredLogging(include_payload_length=True, cache=cache)
    )
    logger.info("structured request logging: enabled")

    # Inside logging (bounced attempts are recorded) but outside the response
    # cap: the bounce carries the full guide, which must never be truncated.
    if _env_flag(guidance.DISABLE_ENV_VAR, default=False):
        logger.info("guidance gate: DISABLED (%s)", guidance.DISABLE_ENV_VAR)
    else:
        ttl = guidance.resolve_ttl()
        mcp.add_middleware(guidance.GuidanceGateMiddleware(guide, ttl=ttl))
        logger.info(
            "guidance gate: enabled (guide delivered in-band, idle ttl=%ds)", ttl
        )

    max_chars = int(
        os.environ.get("PORTAINER_MAX_RESPONSE_CHARS")
        or shaping.DEFAULT_MAX_RESPONSE_CHARS
    )
    mcp.add_middleware(shaping.ResponseCapMiddleware(max_chars))
    logger.info("response cap: %d chars", max_chars)

    logger.info(
        "env value redaction: %s",
        "DISABLED (env values exposed)" if redaction.is_expose_enabled() else "enabled",
    )
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

    posture = tls.resolve_posture(host)
    for line in posture.warnings:
        logger.warning(line)
    if posture.insecure_transport:
        auth.mark_insecure_transport()
    if posture.enforce_https:
        # Installed via the verifier so it runs before the bearer-auth backend
        # (not the `middleware=` list below, which Starlette appends after auth).
        server.auth.add_pre_auth_middleware(Middleware(tls.TLSRequiredMiddleware))
    logger.info(
        "transport posture: %s",
        "PLAINTEXT (insecure)"
        if posture.insecure_transport
        else "https enforced"
        if posture.enforce_https
        else "loopback (dev)",
    )

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
        uvicorn_config={"log_config": None, **posture.uvicorn_kwargs},
    )


if __name__ == "__main__":
    main()
