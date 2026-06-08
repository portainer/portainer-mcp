# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

MCP server for Portainer, distributed on PyPI as `mcp-portainer`. The tool
surface is generated from Portainer's EE OpenAPI spec at startup via
`FastMCP.from_openapi`, with a small filter + response-shaping layer applied
uniformly. Two hand-written escape-hatch tools (`docker_proxy`,
`kubernetes_proxy`) forward arbitrary paths the spec doesn't enumerate.

Python ≥ 3.11. `uv` is the package manager — there is no `pip`/`poetry`
workflow. Source layout: `src/portainer_mcp/`.

## Commands

```bash
uv sync                              # install deps from uv.lock
uv run pytest                        # run the full test suite
uv run pytest tests/test_proxy.py    # one file
uv run pytest -k select_unwraps      # one test by name
make dev                             # local HTTP server via uv + .env (port 17717)
make specs VERSION=2.41.1            # refresh src/portainer_mcp/data/portainer-patched.yaml
```

`make dev` requires `.env` (copy from `.env.example`). It runs the server
over HTTP at `127.0.0.1:17717` so you can iterate without restarting an MCP
client — the client (added with `claude mcp add portainer-dev --transport
http http://127.0.0.1:17717/mcp`) reconnects automatically after a ctrl-c +
`make dev`.

Lint/format: none configured. CI runs only `uv sync --frozen && uv run
pytest` (see `.github/workflows/ci.yml`).

## Architecture

Read [`docs/architecture.md`](docs/architecture.md) for the full picture.
Key things to internalise before changing code:

- **`server.py:build_server()` is the wiring point.** It loads the bundled
  spec, builds the httpx client (carrying `X-API-KEY`), constructs
  `RouteMap`s from the resolved profile tags, instantiates FastMCP, then
  registers proxy tools, adds `SelectArgTransform`, and finally adds
  `ResponseCapMiddleware`. Order matters — the transform must run before
  the middleware so every tool exposes `select`.
- **One `RouteMap` per tag.** FastMCP intersects multi-tag `RouteMap(tags=…)`
  (it's all-of, not any-of), so we emit one `RouteMap` per allowed tag and
  union the matches. Don't collapse them into a single multi-tag map.
- **`select` is universal.** `SelectArgTransform` (`shaping.py`) wraps
  every tool with an optional JMESPath `select` parameter, including the
  two hand-written proxy tools (their existing `select` arg makes
  `_has_select` skip re-wrapping them). After registration, `build_server`
  asserts every tool exposes `select` and raises at startup if any are
  missing — keep that invariant.
- **Response cap sits below Claude Code's MCP output cap.** Default
  `PORTAINER_MAX_RESPONSE_CHARS=50_000` is sized so our truncation hint
  (which names `select` with examples) reaches the model before Claude
  Code's own ~62k-char cap triggers its generic "saved to file" handling.
  When truncation fires, `structured_content` is also cleared so the model
  can't read around the cap.
- **JMESPath unwrap for non-dict responses.** FastMCP wraps list/scalar
  OpenAPI responses as `{"result": …}` to fit MCP's structured-content
  schema. `_select_wrapper` unwraps that single-key envelope before
  projecting, so callers write `[].Id` rather than `result[].Id`.
- **Env values redacted before projection.** `redaction.redact_envs()`
  walks the parsed response in `_select_wrapper` and in the proxy tools
  *before* JMESPath `select` runs — so `select="Env[0].value"` lands on
  the `[REDACTED]` sentinel rather than the real value. The walker is
  field-name driven (`env` / `envvars`, case-insensitive) and handles
  Shapes A/F/G (list of `{name, value}` dicts) and Shape C (Docker
  `"KEY=VAL"` strings); K8s `valueFrom` references are preserved.
  Disabled with `PORTAINER_EXPOSE_ENV_VALUES=1`; logged at startup so
  the posture is greppable. When redaction fires, the response carries a
  one-line summary TextContent naming the env var.
- **HTTP transport requires a bearer token.** `auth.py` defines
  `StaticBearerVerifier` (a `fastmcp.server.auth.TokenVerifier` subclass
  using `hmac.compare_digest`); `build_server()` wires a verifier into
  `FastMCP.from_openapi(..., auth=…)` only when transport=http. Stdio
  ignores `PORTAINER_MCP_AUTH_TOKEN`. Strict validation at startup
  (min 32 chars, ASCII printable, no whitespace) — loud-fail like the
  unknown-profile check. Don't relax this for "convenience"; the strict
  rule eliminates the make-dev-no-token footgun.
- **HTTP is per-user passthrough, not a shared upstream key.** Over HTTP
  the verifier is `auth.PassthroughVerifier` (subclass of
  `StaticBearerVerifier`), and `PORTAINER_API_KEY` is *not* loaded — it's
  the stdio-only credential, and `build_server()` hard-fails if it's set
  under http (a misconfiguration, not a silent fallback). Two layered
  checks run inside one `verify_token` so a failure 401s before any tool
  dispatch: (1) the gate token in `Authorization` is constant-time
  compared by the parent; (2) the caller's own key in the separate
  `X-Portainer-API-Key` header is validated against `/users/me`
  (`passthrough.validate`, positive-only `ValidationCache` keyed by the
  SHA-256 of the key, TTL `PORTAINER_MCP_AUTH_CACHE_TTL` default 60).
  The validated key is injected upstream as `X-API-KEY` by the
  `passthrough.inject_api_key` httpx request hook, which reads *only* the
  in-flight request (so one caller can't borrow another's key) and **fails
  closed** — it raises rather than ever sending a keyless upstream call.
  The two headers carry distinct credentials (gate vs per-user key), so
  the verified token and the forwarded token are never the same value;
  the httpx client under http therefore carries no baked `X-API-KEY`
  (stdio still does). Audit outcomes gain `no_user_key` /
  `invalid_user_key`. `ok` fires only on a *validation* (a cache miss that
  hits `/users/me`), attributed with `portainer_user_id/username` — so it
  marks a validation event (~one per key per TTL window), not every
  admitted request; cache hits admit silently (`validate()` returns
  `(identity, validated_now)` so the verifier knows which). The failure
  outcomes are uncached and fire per request. The per-user key itself is
  never logged (regression-tested). The structured request log adds the
  `tool` name on a `tools/call` (the bare `method` is only ever
  `tools/call`). The cache TTL is a
  perf/DoS knob, not the authz boundary (Portainer rejects a revoked key
  on every real call); never negative-cache (it would lock out a fresh
  key).
- **Two HTTP hardening layers stack on top of the bearer.** Wired in
  `build_server()` + `main()`: a contextualised `StructuredLoggingMiddleware`
  applies to every transport; `http_security.DNSRebindingMiddleware` is
  passed to `server.run(..., middleware=[…])` only for http. Starlette
  appends user middleware *after* the auth backend, so DNS-rebinding
  fires inside the auth chain — bearer-auth runs first, then the Host
  check. Practical impact is small (the audit record may include
  rebinding-probe attempts that present a valid token; failed-auth
  attempts hit 401 before any Host check), but don't assume the Host
  reject precedes bearer-auth when reading audit logs.
  `StaticBearerVerifier.verify_token` emits a structured audit record on
  every attempt under the `portainer_mcp.audit` sub-logger — never include
  the attempted token in those records. In-process rate limiting was
  intentionally dropped: at numbers that didn't impede legitimate clients
  it didn't bound blast radius either, and a reverse proxy is the right
  place for that control.
- **Per-request context is read from the live HTTP request.**
  `request_context.snapshot()` returns `client_ip`, `user_agent`, and
  the MCP `Mcp-Session-Id` from `fastmcp.server.dependencies.get_http_request()`.
  Both the audit log (in `verify_token`) and the FastMCP-layer structured
  request log (`_ContextualStructuredLogging`) call it. Custom outer
  ContextVars don't work here: MCP's streamable-HTTP session manager
  dispatches each JSON-RPC message into a long-lived task whose context
  was captured at session-creation time, so subsequent requests would
  log the stale `initialize`-time values. `get_http_request()` reads
  through MCP SDK's per-message `request_ctx` instead, which is current.
  FastMCP's own `RequestContextMiddleware` is inserted at position 0 of
  the middleware stack (`fastmcp.server.http.create_base_app`), so it
  runs outside the bearer-auth middleware and `get_http_request()` is
  already populated by the time `verify_token` executes — no custom
  prepend needed. If a future FastMCP refactor moves that insertion or
  the auth backend grows to read the request before fastmcp's
  middleware runs, the audit log will silently lose its context fields;
  re-add a small ASGI middleware via `StaticBearerVerifier.get_middleware()`
  if that happens. With a single shared bearer the audit deliberately
  omits `token_fp` (it would be a constant); `session_id` is what
  actually joins an audit row to its request rows.
- **DNS-rebinding rejections carry the env var name back to the operator.**
  `_enrich` rewrites the SDK's bare 421 body to include
  `PORTAINER_MCP_ALLOWED_HOSTS`; `misconfig_warning` logs a startup
  WARNING when the bind host is non-loopback while the allowlist is
  still the localhost defaults. The two together turn the "I deployed
  it and it 421s" first-deploy moment into a self-diagnosing error —
  keep the env-var name in both signals when refactoring. The `Origin`
  allowlist is hardcoded (no env var): programmatic MCP clients omit
  `Origin` and pass through, the local Inspector is covered by the
  localhost defaults, and the MCP spec MUSTs the check itself, not the
  configurability. Don't re-add an `ALLOWED_ORIGINS` env var unless a
  real browser-hosted client use case shows up.
- **Log shape is selectable.** `PORTAINER_MCP_LOG_FORMAT=text|json`
  (default `text`, container image overrides to `json`). The `json`
  formatter merges records whose `msg` is itself a JSON object into the
  envelope, so audit and request records become first-class fields. Keep
  this property when adding new structured loggers — emit
  `json.dumps({...})` as the message and the formatter does the right
  thing in both modes.

## Spec generation

The bundled spec lives at `src/portainer_mcp/data/portainer-patched.yaml`
and is loaded via `importlib.resources` (so it's read from the wheel in
production, not relative paths). To regenerate:

1. `make specs VERSION=<portainer-version>` — clones/refreshes
   `spec/upstream/` (sparse, single-version), then runs `spec/patch_spec.py`.
2. `patch_spec.py` drops structurally broken operations (see
   `EXCLUDED_OPERATION_IDS`), strips `/websocket/*` paths, normalises a
   few malformed `enum` blocks, and rewrites stray tabs. Extend those
   constants when the upstream spec ships new defects — don't hand-edit
   `portainer-patched.yaml`.

## Versioning

Tag format `<portainer-major>.<portainer-minor>.<mcp-patch>` — major+minor
mirrors the Portainer API target; patch is the MCP server's. **The minor
only moves when the embedded spec moves.** Refactors, profile additions,
new proxy tools, shaping changes — all patch. See
[`docs/versioning.md`](docs/versioning.md) and [`docs/release.md`](docs/release.md)
(release is OIDC-driven via PyPI Trusted Publishing on tag push).

## Profiles

Spec exposes ~380 operations across 40+ tags; profiles in `profiles.py`
bundle them. `PORTAINER_PROFILES` (default `BASE,DOCKER,KUBERNETES`)
selects which to enable; `PORTAINER_TAGS_EXTRA` appends raw tags as an
escape hatch. `PORTAINER_PROFILES=ALL` disables the tag filter entirely.
Unknown profile names fail at startup; unknown extras log a warning and
pass through (they just don't match anything). Full per-profile tag list
and orphan-tag inventory in [`docs/profiles.md`](docs/profiles.md).

## Tests

`pytest` with `asyncio_mode = "auto"` (see `pyproject.toml`). Tests live
in `tests/` and import the spec patcher via `tests/conftest.py` which
prepends `spec/` to `sys.path` (it's a script dir, not a package).

## Conventions

- This repo follows a YAGNI / minimal-surface style: no speculative
  scaffolding, no literal-guard tests, no refactor-for-testability without
  independent merit. Trust internal code, validate at boundaries.
- Comments are sparse and exist to explain *why* (hidden constraints,
  surprising behaviour, workarounds for spec defects). Don't add WHAT
  comments — identifiers carry that.
- Env-var flags are parsed via `_env_flag` in `server.py`; falsy values
  are `0`, `false`, `False`. Operator-facing knob reference lives in
  [`docs/configuration.md`](docs/configuration.md); keep it in sync when
  adding or renaming env vars.
