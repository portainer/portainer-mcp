# Architecture

This MCP server bridges Portainer's REST API to MCP clients. It is
bootstrapped from the Portainer OpenAPI spec at startup — almost no
hand-written tool surface — with a small filtering and response-shaping
layer applied uniformly to every tool.

## Pipeline

```
data/portainer-patched.yaml ──► FastMCP.from_openapi ──► tag filter ──┐
                                                                      │
                            hand-written proxy tools ─────────────────┤
                                                                      ▼
                                                  SelectArgTransform (per tool)
                                                                      │
                                                                      ▼
                                                  ResponseCapMiddleware (per call)
                                                                      │
                                                                      ▼
                                                                  MCP client
```

## Components

### Spec & tool generation — `server.py`

`build_server()` loads `src/portainer_mcp/data/portainer-patched.yaml` (a
locally-patched copy of Portainer's EE OpenAPI spec, bundled into the
wheel and read via `importlib.resources`) and hands it to
`FastMCP.from_openapi` with a shared `httpx.AsyncClient` carrying the
operator's `X-API-KEY`. Each operation becomes an MCP tool.

### Tag filter — `profiles.py`

The spec exposes ~380 operations across 40+ tags — too noisy for a model
to navigate. Profiles are named bundles of tags (e.g. `DOCKER`,
`KUBERNETES`). `PORTAINER_PROFILES` selects which to enable; the union
of their tags becomes a `RouteMap` per tag (FastMCP intersects multi-tag
`RouteMap`s, so we emit one per tag). `PORTAINER_READ_ONLY=1` further
restricts the surface to `GET`/`HEAD`.

### Proxy tools — `proxy.py`

Two hand-written tools — `docker_proxy` and `kubernetes_proxy` — forward
arbitrary paths under `/endpoints/{id}/docker/...` and
`/endpoints/{id}/kubernetes/...`. These paths aren't enumerated in the
OpenAPI spec (Portainer forwards them as a subpath to the underlying
daemon), so they can't be generated. Each tool validates the path (no
`..`, no `?#`) and blocks auth-bypass headers.

### HTTP auth — `auth.py`

Only relevant when `PORTAINER_MCP_TRANSPORT=http`. `build_server()` reads
`PORTAINER_MCP_AUTH_TOKEN`, validates it (min 32 chars, ASCII printable,
no whitespace, loud-fail on any defect), and wires a
`StaticBearerVerifier` — a `fastmcp.server.auth.TokenVerifier` subclass —
into the FastMCP constructor. Every HTTP request must carry
`Authorization: Bearer <token>`; the verifier uses `hmac.compare_digest`
for constant-time comparison and returns `None` on mismatch, at which
point FastMCP renders the 401 + `WWW-Authenticate` response itself. Stdio
transport short-circuits the auth path entirely (`_get_auth_context()` in
FastMCP). Every verify attempt — success or mismatch — emits a structured
audit record on the `portainer_mcp.audit` sub-logger carrying the
per-request context (see below); the attempted token bytes are never
logged.

### HTTP security — `http_security.py`

Only relevant when `PORTAINER_MCP_TRANSPORT=http`. FastMCP doesn't plumb
the MCP SDK's `TransportSecuritySettings` through to its streamable-HTTP
manager, so DNS-rebinding protection is silently off in the bundled
stack. `DNSRebindingMiddleware` is a Starlette ASGI middleware that wraps
the SDK's `TransportSecurityMiddleware` and reinstates the `Host`/
`Origin` allowlist check. `main()` passes it to `server.run(...,
middleware=[…])`; Starlette appends user middleware *after* the auth
backend, so bearer-auth runs first and the Host check runs inside the
auth chain. Operators control the `Host` allowlist via
`PORTAINER_MCP_ALLOWED_HOSTS` (default: localhost set); the `Origin`
allowlist is hardcoded to localhost since the only browser-hosted MCP
client in scope today is the local Inspector. `misconfig_warning()` runs
at startup and emits a WARNING when the bind host is non-loopback but
the host allowlist is still the default — turning the "I deployed it
and every request 421s" first-deploy moment into a self-diagnosing
error. The 421 response body is also rewritten to name
`PORTAINER_MCP_ALLOWED_HOSTS` so the operator sees the same pointer from
the client side.

### Per-request context — `request_context.py`

`snapshot()` returns `client_ip`, `user_agent`, and the MCP
`Mcp-Session-Id` for the in-flight HTTP request via
`fastmcp.server.dependencies.get_http_request()`. Both the audit log (in
`auth.verify_token`) and the FastMCP request log call it. Reading
directly from the live request avoids a subtle bug: MCP's
streamable-HTTP session manager dispatches every JSON-RPC message into a
long-lived task whose ContextVars were captured at session-creation
time, so an outer-`ContextVar` approach would log the stale
`initialize`-time values on every subsequent request. FastMCP's own
`RequestContextMiddleware` sits at position 0 of the middleware stack
(outside the auth backend), so `get_http_request()` is already populated
by the time `verify_token` executes.

### Logging — in `server.py`

`PORTAINER_MCP_LOG_FORMAT=text|json` (default `text`; the container image
overrides to `json`). The JSON formatter emits a single per-line envelope
and merges records whose `msg` parses as a JSON object into that
envelope, so audit and request records become first-class fields rather
than nested strings. Two structured emitters feed it:

- **Auth audit** — `StaticBearerVerifier.verify_token` emits one record
  per attempt on `portainer_mcp.audit`, carrying outcome + per-request
  context (`client_ip`, `user_agent`, `session_id`).
- **Request log** — `_ContextualStructuredLogging`, a thin subclass of
  FastMCP's `StructuredLoggingMiddleware`, adds the same per-request
  context to the before/after/error records the middleware already
  emits. The upstream `source: "client"` field is dropped from
  consideration as a caller distinguisher — one shared bearer means
  every request shows the same value.

`_setup_logging()` also strips pre-existing handlers from `fastmcp`,
`httpx`, and `uvicorn.*` loggers and passes `uvicorn_config={"log_config":
None}` to `server.run()`, so a single formatter owns every record
regardless of which library installed handlers first.

### Response shaping — `shaping.py`

Two cooperating layers, applied to every tool the server exposes:

1. **`SelectArgTransform`** injects an optional `select` parameter on
   every tool. The caller passes a JMESPath expression; the server
   projects the response before returning it. This is how the model
   trims noisy Portainer payloads (snapshots, K8s `managedFields`, etc.)
   server-side instead of dragging them into context.

2. **`ResponseCapMiddleware`** is the final safety valve. If a tool
   result exceeds `PORTAINER_MAX_RESPONSE_CHARS` (default 50 000), the
   middleware truncates and appends a hint that names `select` with a
   concrete example. The cap is sized to fire *before* Claude Code's
   own MCP output cap (~62k chars for dense JSON), so our hint reaches
   the model instead of Claude Code's generic "saved to file" message.

`select` narrows first (cheaper bodies); the cap catches whatever still
slips through.

## Why this shape

- **OpenAPI-driven, not hand-coded.** Portainer ships a spec; let
  FastMCP generate the tools so spec bumps are mostly a regen.
- **Filter at the spec layer.** Profiles reduce the visible surface
  without touching individual tools.
- **Universal shaping.** `select` and the cap apply to every tool —
  generated and hand-written alike — so the model learns one pattern
  that works everywhere.

## Pointers

- Profiles & tag bundles: [`profiles.md`](profiles.md)
- Versioning policy: [`versioning.md`](versioning.md)
- User-facing knobs & env vars: [`../README.md`](../README.md)
- Spec patcher: [`spec/patch_spec.py`](../spec/patch_spec.py)
