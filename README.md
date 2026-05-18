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
