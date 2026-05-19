# portainer-mcp-fastmcp

Experimental MCP server for Portainer, generated from the Portainer OpenAPI
spec via [FastMCP](https://github.com/PrefectHQ/fastmcp)'s `from_openapi`
pipeline.

## Run

```bash
uv sync
PORTAINER_URL=https://portainer.example.com \
PORTAINER_API_KEY=<key> \
  uv run portainer-mcp
```

The patched spec (`spec/portainer-patched.yaml`) is committed — no
patcher run needed. Currently tracks Portainer EE 2.41.1.

## What's here

- `spec/patch_spec.py` — applies the spec-defect mitigations catalogued in
  `portainer-go-sdk/docs/spec-upstream-fixes.md` (excluded operations,
  `/websocket/*` paths, malformed enums) plus the two YAML-syntax defects
  in [`docs/spec-upstream-fixes.md`](docs/spec-upstream-fixes.md) that the
  Go SDK's toolchain doesn't hit. Writes `spec/portainer-patched.yaml`,
  which is committed; end users do not run the patcher (see
  [Refreshing the spec](#refreshing-the-spec-maintainers)).
- `src/portainer_mcp/server.py` — `FastMCP.from_openapi` wired to the
  patched spec, an `httpx.AsyncClient` carrying the Portainer API key, and
  a profile-based tag allowlist (default: `BASE,DOCKER,KUBERNETES`) that
  excludes the rest of the 380+ operation surface.
- `src/portainer_mcp/profiles.py` — named tag bundles selected via
  `PORTAINER_PROFILES`. See [`docs/profiles.md`](docs/profiles.md).

## Configuration

| Env var | Default | Effect |
|---|---|---|
| `PORTAINER_PROFILES` | `BASE,DOCKER,KUBERNETES` | Tag bundles to enable. `ALL` disables the filter. |
| `PORTAINER_TAGS_EXTRA` | _empty_ | Extra tags appended to the union (escape hatch). |
| `PORTAINER_READ_ONLY` | `0` | `1` restricts to `GET`/`HEAD` operations only. |
| `PORTAINER_NO_PROXY` | `0` | `1` skips `docker_proxy` / `kubernetes_proxy` registration. |
| `PORTAINER_TLS_VERIFY` | `1` | `0` skips TLS verification (self-signed certs). |
| `PORTAINER_MCP_LOG` | `logs/portainer-mcp.log` | Override the log file path. |
| `PORTAINER_MAX_RESPONSE_CHARS` | `75000` | Response truncation target (see below). |

See [`docs/profiles.md`](docs/profiles.md) for per-profile tag lists, orphan
tags not covered by any profile, and read-only semantics.

## Response shaping (universal)

Every tool — auto-generated OpenAPI tools and the hand-written proxy
tools alike — accepts an optional JMESPath `select` parameter applied
server-side before the response reaches the model. Use it to project
just the fields you need:

```
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
docker_proxy(path="/containers/json", select="[].{id:Id,name:Names[0],state:State}")
```

Responses are capped at approximately `75_000` chars by default
(deliberately conservative — dense Docker/K8s JSON packs at ~3
chars/token, targeting ~25k tokens with margin); override with
`PORTAINER_MAX_RESPONSE_CHARS=<int>`. Truncated responses carry a hint
asking the model to narrow `select`. The cap is a target, not an exact
ceiling: the appended hint adds ~130 chars, and char-vs-token mismatch
means the actual token count varies with content. See
[`docs/proxy-tools.md`](docs/proxy-tools.md) for the proxy tools'
specific design and the planned evolution if filtering alone proves
insufficient.

## Refreshing the spec (maintainers)

The unpatched Portainer OpenAPI spec lives in a private repo, so spec
bumps are maintainer-driven:

```bash
make specs VERSION=2.42.0
```

This clones (or fast-forwards) `portainer/portainer-api-docs` into
`spec/upstream/` and runs the patcher against
`versions/ee/$(VERSION).yaml`, overwriting `spec/portainer-patched.yaml`.
Commit the regenerated file and bump the tested-against version above.

EE spec only — CE is a subset and works on a best-effort basis.

## Troubleshooting

The server logs every httpx request/response (method, URL, status, first
2 KB of body) and FastMCP DEBUG output to `logs/portainer-mcp.log`. Tail it
in a second terminal while exercising tools from an MCP client:

```bash
tail -F logs/portainer-mcp.log
```

When a tool errors, the log shows the raw Portainer response immediately
before the FastMCP validation error pointing at the offending field —
enough to identify spec-vs-server mismatches without guessing.

Override the path with `PORTAINER_MCP_LOG=/some/other/path` (passed as
`-e PORTAINER_MCP_LOG=…` on `claude mcp add`). Stdio MCP servers are
long-lived, so restart the server (`claude mcp remove portainer && claude
mcp add portainer …`, or restart your MCP client) after editing
`server.py`.
