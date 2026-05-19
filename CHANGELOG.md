# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The versioning policy is described in [`docs/versioning.md`](docs/versioning.md)
— major+minor tracks the Portainer API version; the patch slot belongs to
the MCP server.

## [Unreleased]

Initial release. Targets Portainer 2.41.x (EE; CE best-effort).

### Added

- **Tool surface from the Portainer OpenAPI spec** via
  `FastMCP.from_openapi`, fed by the committed
  [`spec/portainer-patched.yaml`](spec/portainer-patched.yaml) (EE
  2.41.1). End users do not run the patcher.
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
  subclass) registered with `mcp.add_transform(...)`, using the public
  `Tool.from_tool(transform_fn=...)` API. Startup canary
  (`await mcp.list_tools()`) raises if any tool is missing `select`.
- **Response truncation hint**: `ResponseCapMiddleware` caps responses
  at `PORTAINER_MAX_RESPONSE_CHARS` (default 50000, ~80% of Claude
  Code's MCP ceiling) and appends a `select`-teaching hint with a
  concrete example before the client's own cap fires.
- **Hand-written proxy tools** for endpoints the OpenAPI spec can't
  express cleanly: `docker_proxy` and `kubernetes_proxy`, with
  validators rejecting `..` / `?` / `#` in paths and a blocked-header
  list. JMESPath projection passes through non-JSON responses unchanged
  (logs, stats, exec). See [`docs/proxy-tools.md`](docs/proxy-tools.md).
- **Maintainer spec-refresh pipeline**: `make specs VERSION=X.Y.Z`
  shallow-clones `portainer/portainer-api-docs` (SSH default,
  `UPSTREAM_REPO=` override) into `spec/upstream/` and runs
  `spec/patch_spec.py` against the requested EE YAML.
  [`spec/patch_spec.py`](spec/patch_spec.py) applies the spec-defect
  mitigations catalogued in
  [`docs/spec-upstream-fixes.md`](docs/spec-upstream-fixes.md)
  (excluded operations, `/websocket/*` paths, malformed enums, YAML
  tab/`=`-tag defects).
- **Test suite + CI**: 31 tests under [`tests/`](tests/) covering the
  pure-data surface (spec patcher, shaping, proxy validators). CI runs
  `uv sync --frozen` + `uv run pytest` on push to `master` and every PR.
- **Hygiene skill** at
  [`skills/portainer-mcp-hygiene/`](skills/portainer-mcp-hygiene/) —
  guidance for MCP clients on when to project with `select`, where the
  heavy fields live, and how to handle non-JSON proxy responses.
- **FastMCP pin** tightened to `>=3.3,<4` (the `OpenAPIProvider` import
  path used here only exists on 3.x).
- **Versioning policy** at [`docs/versioning.md`](docs/versioning.md):
  major+minor pins to Portainer's API version; patch slot is the MCP
  server's own.

### Known gaps

- **Distribution** is unfinished — no Dockerfile, no PyPI release, no
  release workflow. Install path remains "clone + `uv sync`". Tracked
  as the only open item in
  [`docs/production-readiness.md`](docs/production-readiness.md) §5.
- **CE coverage** is best-effort. The embedded spec is EE; CE is a
  subset and operations missing from CE surface as 404s at call time.
