# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The versioning policy is described in [`docs/versioning.md`](docs/versioning.md)
— major+minor tracks the Portainer API version; the patch slot belongs to
the MCP server.

## [Unreleased]

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
