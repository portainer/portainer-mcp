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
make dev                             # local HTTP server via uv + .env (port 8000)
make specs VERSION=2.41.1            # refresh src/portainer_mcp/data/portainer-patched.yaml
```

`make dev` requires `.env` (copy from `.env.example`). It runs the server
over HTTP at `127.0.0.1:8000` so you can iterate without restarting an MCP
client — the client (added with `claude mcp add portainer-dev --transport
http http://127.0.0.1:8000/mcp`) reconnects automatically after a ctrl-c +
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
- **HTTP transport requires a bearer token.** `auth.py` defines
  `StaticBearerVerifier` (a `fastmcp.server.auth.TokenVerifier` subclass
  using `hmac.compare_digest`); `build_server()` wires it into
  `FastMCP.from_openapi(..., auth=…)` only when transport=http. Stdio
  ignores `PORTAINER_MCP_AUTH_TOKEN`. Strict validation at startup
  (min 32 chars, ASCII printable, no whitespace) — loud-fail like the
  unknown-profile check. Don't relax this for "convenience"; the strict
  rule eliminates the make-dev-no-token footgun.

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
  are `0`, `false`, `False`.
