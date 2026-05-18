# portainer-mcp-fastmcp

Experimental MCP server for Portainer, generated from the Portainer OpenAPI
spec via [FastMCP](https://github.com/PrefectHQ/fastmcp)'s `from_openapi`
pipeline.

## Run

```bash
uv sync
uv run python spec/patch_spec.py
PORTAINER_URL=https://portainer.example.com \
PORTAINER_API_KEY=<key> \
  uv run portainer-mcp
```

## What's here

- `spec/patch_spec.py` — applies the spec-defect mitigations catalogued in
  `portainer-go-sdk/docs/spec-upstream-fixes.md` (excluded operations,
  `/websocket/*` paths, malformed enums) plus the two YAML-syntax defects
  in [`docs/spec-upstream-fixes.md`](docs/spec-upstream-fixes.md) that the
  Go SDK's toolchain doesn't hit. Writes `spec/portainer-patched.yaml`.
- `src/portainer_mcp/server.py` — `FastMCP.from_openapi` wired to the
  patched spec, an `httpx.AsyncClient` carrying the Portainer API key, and
  a tag allowlist (`endpoints`, `stacks`, `auth`) that excludes the rest of
  the 380+ operation surface. Widen `ALLOWED_TAGS` to expose more.

## Read-only mode

Set `PORTAINER_READ_ONLY=1` to restrict the exposed tools to GET
operations within the allowed tags. Non-GET routes are excluded from
registration — the MCP client sees a smaller tool list rather than tools
that fail at call time. Useful for monitoring/auditing workflows or when
giving an AI agent access to a production Portainer instance.

Note: HTTP method is used as the read/write classifier. A handful of
Portainer endpoints use POST for read-shaped operations (e.g. snapshot
listings); read-only mode hides those too.

## Docker / Kubernetes proxy tools

`docker_proxy` and `kubernetes_proxy` forward arbitrary requests to the
Docker and Kubernetes APIs of a Portainer-managed environment. They each
take an optional JMESPath `select` expression applied server-side before
the response reaches the model — e.g.
`[].{id:Id,name:Names[0],state:State}` against `/containers/json`.
Responses are capped at `75_000` chars by default (deliberately
conservative — dense Docker/K8s JSON packs at ~3 chars/token, targeting
~25k tokens with margin); override with `PORTAINER_PROXY_MAX_CHARS=<int>`. See
[`docs/proxy-tools.md`](docs/proxy-tools.md) for the design rationale,
metrics to watch during testing, and the planned evolution if
filtering alone proves insufficient.

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
