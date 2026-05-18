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

## 2. Spec generation and patching

`spec/patch_spec.py:37` defaults the input path to
`/workspace/portainer-api-docs/versions/ee/2.41.1.yaml` — works only on
the original author's machine, and README's `uv run python
spec/patch_spec.py` fails for everyone else.

Options, in increasing order of effort:

1. Commit `spec/portainer-patched.yaml` as a build artifact so end users
   skip the patcher entirely. Requires a maintainer step to refresh it
   on spec bumps.
2. Vendor the raw spec under `spec/` and point `DEFAULT_INPUT` there.
3. Drop the default, require the positional arg, document it in README.

Whatever the choice, the spec version (currently EE 2.41.1) needs an
explicit upgrade story — what gets re-tested when Portainer ships a new
spec, who runs the patcher, where the patched output lives.

Anthony: I'm leaning towards committing the portainer-patched.yaml as a build artifact. Benefits: as the original spec (unpatched) is currently coming from a private repo this would allow easier contributions. Also, the version bumps would likely be done by someone at Portainer team so i'm expecting something like make specs VERSION=2.42.0 to be available that would clone/fetch from https://github.com/portainer/portainer-api-docs and generate the patched specs.

## 3. Tool surface

`ALLOWED_TAGS = ("endpoints", "stacks", "auth")` at `server.py:32` is
frozen in source. Anyone wanting `kubernetes`, `registries`, `users`,
etc. has to fork.

`TODO.local.md` already sketches the right end state: cumulative
profiles (BASE + READ_ONLY/ADMIN/EDGE/DOCKER/KUBERNETES/NO_PROXY/
OPERATOR/TROUBLESHOOT). Until that lands, a `PORTAINER_TAGS` env var
(comma-separated, falls back to the current tuple) is a one-line
unblock for the replacement path.

ALAPENNA: we'll definitely be exploring the PROFILE approach for this.

## 4. FastMCP internals

`shaping.py:159, 165` reads and pops `provider._tools` — a private
attribute. A FastMCP minor release can break `inject_select_arg()`
silently.

Mitigations:

- Pin FastMCP narrowly in `pyproject.toml` (`fastmcp>=2.8,<2.X`).
- Add an import-time smoke check: after `from_openapi`, assert
  `inject_select_arg()` wrapped at least one tool. Catches the breakage
  on startup rather than at first call.
- File an upstream issue / PR for a public registry-walk API. Once
  available, drop the private access.

## 5. Versioning and distribution

`pyproject.toml` is at `0.0.1`, no Dockerfile, no release workflow, no
PyPI publish. Install path is "clone + `uv sync` + run a patcher
pointed at a spec you don't have."

To make this a viable replacement for whatever users currently
`docker run`:

- Ship a Dockerfile (multi-stage: `uv` build → slim runtime, spec
  baked in per §2).
- Tag releases, publish images (GHCR or Docker Hub), publish the
  package to PyPI so `uvx portainer-mcp` works.
- Document version compatibility: which Portainer versions this server
  is tested against, and the policy for spec drift.
- Update README install snippets per client (Claude Desktop, Codex,
  ChatGPT desktop) — `TODO.local.md` already calls this out.
