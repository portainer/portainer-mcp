# Production readiness

Tracking the gaps to close before promoting this from `experiments/` to a
real MCP server (and a candidate replacement for the existing one).

## 1. Testing — done

`tests/` covers the pure-data surface (`pytest`, 31 tests, no live
Portainer required). CI runs `uv sync --frozen` + `uv run pytest` on
push to `master` and every PR (`.github/workflows/ci.yml`).

Covered:

- `spec/patch_spec.py` — `patch()` rule table (each `EXCLUDED_*` /
  `ENUM_STRIPS` entry) and the YAML `=`-tag constructor.
- `shaping.project()` and `ResponseCapMiddleware` — JMESPath error
  mapping, truncation hint, `structured_content` clearing.
- `proxy._validate_path` / `_validate_headers` / `_apply_select` —
  blocked headers, `..` / `?` / `#` rejection, non-JSON passthrough.

Deliberate gaps:

- `shaping._select_wrapper` — needs a live FastMCP runtime (`forward()`
  only works inside a tool-call context). The pure-data subset (non-JSON
  passthrough, JMESPath projection) is covered indirectly via
  `_apply_select`, which exercises the same branches.
- `shaping.inject_select_arg` — better addressed by the §4 startup
  canary (covers the real FastMCP version on every deploy) than a
  synthetic unit test.
- `proxy._call` / `docker_proxy` / `kubernetes_proxy` — the validators
  (the security boundary) are covered; the remaining HTTP plumbing
  belongs in an integration test, not a unit test.
- `patch_spec.main()` tab normalisation — self-canarying; a tab leaking
  through breaks `yaml.safe_load` immediately when the patcher runs.

## 2. Spec generation and patching — done

`spec/portainer-patched.yaml` is now committed as a build artifact (EE
2.41.1). End users skip the patcher entirely; contributors can iterate
without access to the private spec repo.

Spec bumps are maintainer-driven via `make specs VERSION=X.Y.Z`, which
shallow-clones `portainer/portainer-api-docs` (SSH default,
`UPSTREAM_REPO=` override) into `spec/upstream/` (gitignored) using
`--depth=1 --filter=blob:none` + `sparse-checkout --no-cone` so only
the requested YAML is downloaded (~1.7M working tree, not the full 66M
repo). On subsequent runs it `fetch --depth=1` + `reset --hard
FETCH_HEAD`, then re-applies the sparse pattern to support VERSION
changes.

`DEFAULT_INPUT` is removed from the patcher — the positional arg is now
required, so accidental invocations without a path fail loudly instead
of silently reading the original author's filesystem.

EE-only — CE is a subset and treated as best-effort. No CI drift guard:
the maintainer flow is the only sanctioned regeneration path, and a
drift check only pays off once the upstream repo is fetchable from CI.

## 3. Tool surface — done

Five named profiles (`BASE`, `DOCKER`, `KUBERNETES`, `EDGE`, `ADMIN`) plus
an `ALL` sentinel that bypasses the tag filter — selected via
`PORTAINER_PROFILES`, union'd, with `PORTAINER_TAGS_EXTRA` as the escape
hatch for orphan tags. `PORTAINER_READ_ONLY` and `PORTAINER_NO_PROXY` are
orthogonal modifiers (env-only, no CLI flags for v1). Default
`BASE,DOCKER,KUBERNETES` covers 10 tags and ~180 of the 387 spec
operations; the five-profile union covers 28 tags and ~306 operations;
`ALL` covers everything.

Unknown profile names fail loudly at startup. Unknown extras log a warning
and pass through harmlessly. See [`profiles.md`](profiles.md) for the
per-profile tag list, orphan tags (15 not in any profile), and examples.

`READ_ONLY` filters by HTTP method (strict: `GET`/`HEAD` only). A few
read-shaped POSTs are hidden as a side effect — deliberate, since an
operationId denylist would rot faster than the method rule.

## 4. FastMCP internals — done

Resolved. `inject_select_arg()` is gone; `shaping.SelectArgTransform`
(a `fastmcp.server.transforms.Transform` subclass) is registered via
`mcp.add_transform(...)` and wraps each tool through the public
`Tool.from_tool(transform_fn=...)` API. No private attributes accessed.

`build_server()` runs an `await mcp.list_tools()` smoke check at
startup and raises `RuntimeError` if any tool is missing `select` —
breakage surfaces at boot, not first call.

Pin tightened to `fastmcp>=3.3,<4` (the `OpenAPIProvider` import path
this code uses only exists on 3.x; the prior `>=2.8` was wrong).

## 5. Versioning — done

Policy mirrors `portainer-go-sdk/docs/versioning.md`: major+minor pins
to the Portainer API version the embedded spec was generated against;
the patch slot belongs to the MCP server. Current version: `2.41.0`
(targets Portainer 2.41.x). See [`versioning.md`](versioning.md) for the
full policy (edge cases, what does and does not bump the minor, consumer
pinning) and [`../CHANGELOG.md`](../CHANGELOG.md) for the running log.

README has a Compatibility table; `pyproject.toml` is at `2.41.0`.

## 6. Distribution

Tracked separately from versioning. No Dockerfile, no release workflow,
no PyPI publish yet. Install path is still "clone + `uv sync`".

To make this a viable replacement for whatever users currently
`docker run`:

- Ship a Dockerfile (multi-stage: `uv` build → slim runtime, spec
  baked in per §2).
- Tag releases, publish images (GHCR or Docker Hub), publish the
  package to PyPI so `uvx portainer-mcp` works. Tag scheme follows §5
  (e.g. `2.41.0`, image tags `2.41` / `2.41.0` / `latest`).
- Update README install snippets per client (Claude Desktop, Codex,
  ChatGPT desktop) — `TODO.local.md` already calls this out.
- Ship the in-repo `skills/` directory with the server via
  `fastmcp.server.providers.skills.SkillsDirectoryProvider`, so the
  hygiene skill travels with the distribution (PyPI / Docker image)
  rather than requiring a repo clone. Pair with a small `install-skills`
  entrypoint (Make target or CLI subcommand) that runs FastMCP's
  `sync_skills` against the local server to materialize resources into
  `~/.claude/skills/` — MCP clients fetch `skill://` resources on
  demand but don't auto-install them into the client's skill discovery
  path. The repo-local `.claude/skills/portainer-mcp-hygiene` symlink
  stays for the dev loop (edit-and-reload without a sync step).
