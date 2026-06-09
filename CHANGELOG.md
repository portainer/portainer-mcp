# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The versioning policy is described in [`docs/versioning.md`](docs/versioning.md)
— major+minor tracks the Portainer API version; the patch slot belongs to
the MCP server.

## [Unreleased]

## [2.42.5] — 2026-06-09

Targets Portainer 2.42.x.

### Changed

- **BREAKING (HTTP transport): the HTTP credential model is now per-user
  passthrough.** Previously the HTTP bearer token gated access while a single
  shared `PORTAINER_API_KEY` was the upstream Portainer credential for every
  caller. Now the bearer token is only a gate, and each caller must supply
  **their own** Portainer API key in the `X-Portainer-API-Key` header; it is
  validated against `/users/me` and injected upstream per-request. **Setting
  `PORTAINER_API_KEY` under `PORTAINER_MCP_TRANSPORT=http` now hard-fails at
  startup** — it is a stdio-only credential. Migration: drop `PORTAINER_API_KEY`
  from your HTTP deployment and have each client send `X-Portainer-API-Key`
  alongside the existing `Authorization: Bearer` gate token. Stdio transport is
  unchanged. Validation is cached positive-only (TTL
  `PORTAINER_MCP_AUTH_CACHE_TTL`, default 60s); audit records gain
  `no_user_key` / `invalid_user_key` outcomes and the `tool` name, and the
  per-user key is never logged. See [#66](https://github.com/portainer/portainer-mcp/pull/66).
- **Bumped `fastmcp` to `>=3.4.2` and dropped the direct `starlette` pin.** The
  CVE-2026-48710 floor is now carried transitively by the newer fastmcp, so the
  explicit starlette pin added in 2.42.3 is no longer needed.

### Added

- **TLS posture enforcement on non-loopback HTTP binds.** The server refuses to
  boot on a non-loopback bind unless the operator declares one of three shapes:
  a server-terminated cert (`PORTAINER_MCP_TLS_CERT` / `_TLS_KEY`), proxy TLS
  attestation (`PORTAINER_MCP_TRUST_PROXY_TLS=1` + `..._FORWARDED_ALLOW_IPS`),
  or the loud plaintext opt-out
  (`PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP=1`, which marks every audit
  record `insecure_transport: true`). Loopback binds are exempt for dev. A
  `TLSRequiredMiddleware` backstop runs *before* auth, so a plaintext request is
  rejected before any per-user key is validated or forwarded upstream.
  Self-signed certs WARN, never block. See [#67](https://github.com/portainer/portainer-mcp/pull/67).

### Documentation

- Clarified in the `portainer-mcp-hygiene` skill that an edge agent's health
  comes from its heartbeat, not its `Status` field. See
  [#70](https://github.com/portainer/portainer-mcp/pull/70).

## [2.42.4] — 2026-06-02

Targets Portainer 2.42.x.

### Changed

- **Container images are now multi-arch (`linux/amd64` + `linux/arm64`).**
  The Docker Hub release workflow adds a QEMU setup step and builds both
  platforms into a single manifest list under the existing `:X.Y.Z` / `:X.Y`
  tags.

## [2.42.3] — 2026-05-29

Targets Portainer 2.42.x.

### Security

- **Pinned `starlette>=1.0.1` to close CVE-2026-48710 (BadHost).**
  Starlette is transitive via `fastmcp`→`mcp` with no upstream floor at the
  fixed version; the pin is asserted directly so a re-lock can't drift back to
  a vulnerable release. Resolves to 1.2.0.

### Fixed

- **Proxy `query_params` / `headers` tolerate how different MCP clients
  serialize object arguments.** Some clients (notably Claude Desktop) send the
  whole object as a JSON string, which pydantic rejected before the tool ran —
  blocking every Docker/Kubernetes endpoint that needs a query string (logs,
  `all=true`, label/field selectors, stats). A `BeforeValidator` now parses a
  JSON-string argument back into an object and normalizes each value to its
  wire form, so native bools/numbers and nested `filters` objects work too. The
  tool schema is unchanged, so the model still sees one canonical contract.
- **Proxy tools surface upstream HTTP failures as errors.** `docker_proxy` /
  `kubernetes_proxy` previously returned a 4xx/5xx body as a normal result, so
  a failed call (e.g. a wrong `environment_id` 404) could be silently nulled
  out by a `select` projection and look like empty data. They now raise a tool
  error carrying the status and the (truncated) upstream body, so the model can
  tell a failed request from a missing field.

## [2.42.2] — 2026-05-28

Targets Portainer 2.42.x.

### Added

- **Env value redaction on every response.** Stack, container, and
  Kubernetes env values are rewritten to `[REDACTED]` before leaving the
  MCP tool boundary so secrets don't leak into the model's context just
  because a tool happened to include them. The redaction runs *before*
  JMESPath `select`, so a projection like `select="Env[0].value"` lands
  on the sentinel. The response carries a one-line summary naming the
  toggle. Set `PORTAINER_EXPOSE_ENV_VALUES=1` to disclose; the posture
  is logged at startup. Covers Portainer `Env`/`EnvVars` pairs, Docker
  `"KEY=VAL"` strings, and Kubernetes `env[].value`; K8s `valueFrom`
  references are preserved.
  See [#61](https://github.com/portainer/portainer-mcp/issues/61).
- **`readOnlyHint` tool annotation.** Every generated tool now carries the
  MCP `readOnlyHint` annotation so clients can relax approval prompts for
  non-mutating calls. Spec-derived tools derive it from the HTTP method
  (`GET`/`HEAD` are read-only, everything else a write — setting it `False`
  also activates the spec's `destructiveHint` default); `docker_proxy` /
  `kubernetes_proxy` track `PORTAINER_READ_ONLY`, honest because the proxy
  hard-rejects non-`GET` in read-only mode. The hint is a client-side UX
  signal, not enforcement — the read-only guarantee remains the `GET`/`HEAD`
  route filter and the proxy's method check.

### Changed

- Proxy responses (`docker_proxy`, `kubernetes_proxy`) are now re-serialised
  through `json.dumps` whenever they're JSON and the redaction posture is
  active (i.e. by default). Output is byte-identical for the model but no
  longer preserves upstream whitespace or key ordering. Non-JSON bodies
  (logs, stats, error pages) still pass through verbatim.

## [2.42.1] — 2026-05-26

Targets Portainer 2.42.x. First build to ship a container image alongside
the PyPI wheel, and the first release with a bearer-gated HTTP transport.

### Added

- **Container image** at `docker.io/portainer/portainer-mcp`, published
  on every `X.Y.Z` tag push from
  [`.github/workflows/release-docker.yml`](.github/workflows/release-docker.yml).
  Tagged `X.Y.Z` and `X.Y` per release; no `latest`. See
  [`docs/docker.md`](docs/docker.md).
- **HTTP bearer auth.** New `PORTAINER_MCP_AUTH_TOKEN` env, **required**
  when `PORTAINER_MCP_TRANSPORT=http` and ignored for stdio. Strict
  validation at startup (min 32 chars, ASCII printable, no whitespace —
  loud-fail on any defect); constant-time comparison via
  `hmac.compare_digest`; masked fingerprint in the startup log, full
  value never logged. Wired through FastMCP's `TokenVerifier` protocol —
  FastMCP renders the 401 + `WWW-Authenticate` response on failure.
- **DNS-rebinding allowlist** for the HTTP transport.
  `PORTAINER_MCP_ALLOWED_HOSTS` (default `127.0.0.1:*,localhost:*,[::1]:*`)
  validates the `Host` header on every request; mismatches return 421
  with a body that names the env var. The `Origin` allowlist is hardcoded
  to localhost — programmatic MCP clients omit `Origin` and pass through.
  A startup WARNING fires when the bind host is non-loopback while the
  allowlist is still the localhost defaults, so the
  "deployed-then-it-421s" case self-diagnoses.
- **Auth audit log.** Every HTTP auth attempt emits a structured record
  under the `portainer_mcp.audit` sub-logger with `outcome`, `client_ip`,
  `user_agent`, and the MCP `session_id` — joinable against the
  FastMCP-layer `request_start` / `request_success` records by
  `session_id`. The attempted token is never written.
- **Selectable log shape.** `PORTAINER_MCP_LOG_FORMAT=text|json`
  (default `text`; container image overrides to `json`). In `json` mode,
  records whose message is itself a JSON object are merged into the
  envelope, so audit and request records become first-class fields
  rather than nested strings.
- **Consolidated operator config reference** at
  [`docs/configuration.md`](docs/configuration.md), grouped by concern
  (transport, hardening, profiles, behaviour, logging) with the audit
  and traceability story documented end-to-end.

### Changed

- **`make dev` now requires `PORTAINER_MCP_AUTH_TOKEN`.** Local HTTP dev
  loop is no longer auth-less — add the token to `.env` and pass it via
  `claude mcp add … --header "Authorization: Bearer <token>"`.

## [2.42.0] — 2026-05-22

Targets Portainer 2.42.x.

### Added

- **Maintainer release skill** at
  [`.claude/skills/portainer-mcp-release/`](.claude/skills/portainer-mcp-release/SKILL.md).
  Project-local Claude Code skill that walks through spec-bump releases
  (Portainer minor → MCP minor): finding the upstream patch target, spec
  regeneration, op/tag delta recounting, orphan inventory refresh, pin
  bumps across distribution docs, CHANGELOG promotion, commit, push,
  merge, tag. Bundles a `spec_deltas.py` script that prints the new
  per-profile coverage and orphan table paste-ready for `docs/profiles.md`.

### Changed

- **Embedded spec bumped to Portainer EE 2.42.0** (was 2.41.1). Default
  `BASE,DOCKER,KUBERNETES` profile coverage grows from ~180 to ~197
  operations; the five-profile union grows from ~306 to ~342. Upstream
  removed the `intel` tag (6 operations); dropped from the orphan tag
  list in [`docs/profiles.md`](docs/profiles.md).
- Bump `actions/checkout` to v6.0.2 and `astral-sh/setup-uv` to v8.1.0 in
  the CI and release workflows. Clears the Node.js 20 deprecation warning
  ahead of the forced Node.js 24 default on 2026-06-02.

## [2.41.0] — 2026-05-19

Initial release. Targets Portainer 2.41.x. Distributed
on PyPI as `mcp-portainer`.

### Added

- **Tool surface from the Portainer OpenAPI spec** via
  `FastMCP.from_openapi`. The patched spec (EE 2.41.1) ships inside the
  wheel at `src/portainer_mcp/data/portainer-patched.yaml`, loaded via
  `importlib.resources`. End users do not run the patcher.
- **Profile-based tag allowlist** at
  [`src/portainer_mcp/profiles.py`](src/portainer_mcp/profiles.py): five
  named bundles (`BASE`, `DOCKER`, `KUBERNETES`, `EDGE`, `ADMIN`) plus an
  `ALL` sentinel, selected via `PORTAINER_PROFILES`. Unknown profile
  names fail loudly at startup; `PORTAINER_TAGS_EXTRA` is the escape
  hatch for orphan tags. Default `BASE,DOCKER,KUBERNETES` covers ~180 of
  387 spec operations. See [`docs/profiles.md`](docs/profiles.md).
- **Orthogonal modifiers**: `PORTAINER_READ_ONLY=1` filters to `GET` /
  `HEAD` only; `PORTAINER_NO_PROXY=1` skips proxy-tool registration.
- **Universal response shaping**: every tool — generated OpenAPI tools
  and hand-written proxies alike — accepts an optional JMESPath `select`
  argument applied server-side. Implemented via
  `shaping.SelectArgTransform` (a `fastmcp.server.transforms.Transform`
  subclass) registered with `mcp.add_transform(...)`. Startup canary
  (`await mcp.list_tools()`) raises if any tool is missing `select`.
- **Response truncation hint**: `ResponseCapMiddleware` caps responses
  at `PORTAINER_MAX_RESPONSE_CHARS` (default 50000, ~80% of Claude
  Code's MCP ceiling) and appends a `select`-teaching hint with a
  concrete example before the client's own cap fires.
- **Hand-written proxy tools** for endpoints the OpenAPI spec can't
  express cleanly: `docker_proxy` and `kubernetes_proxy`, with
  validators rejecting `..` / `?` / `#` in paths and a blocked-header
  list. JMESPath projection passes through non-JSON responses unchanged
  (logs, stats, exec).
- **HTTP transport mode** via `PORTAINER_MCP_TRANSPORT=http` plus
  `PORTAINER_MCP_HTTP_HOST` (default `127.0.0.1`) and
  `PORTAINER_MCP_HTTP_PORT` (default `8000`). Powers `make dev` — a
  long-running local server connected via
  `claude mcp add … --transport http http://127.0.0.1:8000/mcp` — and
  the eventual remote-container deployment.
- **Logging routed to stderr** per the MCP spec (stdio servers' logging
  surface). FastMCP banner and its version-check call to
  `pypi.org/pypi/fastmcp/json` are suppressed so deployed-server stderr
  stays ours.
- **PyPI release pipeline** at
  [`.github/workflows/release.yml`](.github/workflows/release.yml): tag
  push (`X.Y.Z`) builds the wheel, verifies the tag matches
  `pyproject.version`, runs tests, and publishes to PyPI via OIDC-based
  Trusted Publishing. No API tokens or repo secrets. Process docs:
  [`docs/release.md`](docs/release.md).
- **Maintainer spec-refresh pipeline**: `make specs VERSION=X.Y.Z`
  shallow-clones `portainer/portainer-api-docs` (SSH default,
  `UPSTREAM_REPO=` override) into `spec/upstream/` and runs
  `spec/patch_spec.py` against the requested EE YAML.
  [`spec/patch_spec.py`](spec/patch_spec.py) applies workarounds for
  known upstream spec defects (excluded operations, `/websocket/*`
  paths, malformed enums, YAML tab/`=`-tag defects).
- **Test suite + CI**: 41 tests under [`tests/`](tests/) covering the
  pure-data surface (spec patcher, shaping, proxy validators). CI runs
  `uv sync --frozen` + `uv run pytest` on push to `main` and every PR.
- **Hygiene skill** at
  [`skills/portainer-mcp-hygiene/`](skills/portainer-mcp-hygiene/) —
  guidance for MCP clients on when to project with `select`, where the
  heavy fields live, and how to handle non-JSON proxy responses.
- **FastMCP pin** at `>=3.3,<4` (the `OpenAPIProvider` import path used
  here only exists on 3.x).
- **Versioning policy** at [`docs/versioning.md`](docs/versioning.md):
  major+minor pins to Portainer's API version; patch slot is the MCP
  server's own.

### Known gaps

- **CE coverage** is best-effort. The embedded spec is EE; CE is a
  subset and operations missing from CE surface as 404s at call time.
- **Remote container deployment** (HTTP transport + auth) is not yet
  shipped. The transport switch and `make dev` workflow lay the
  groundwork; auth and a Dockerfile come after PyPI lands.
