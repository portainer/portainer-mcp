# Configuration

This files document every knob that the `portainer-mcp` exposes, grouped by concern. All values are read from the process environment at startup. Defaults are sized for two documented deployment shapes: **local stdio** (local process, no auth) and **shared-secret HTTP** (multi-user, bearer-gated).

> [!NOTE]
> Truthy parsing: any value other than `0`, `false`, or `False` is treated as truthy.

## Required

| Var | Notes |
|---|---|
| `PORTAINER_URL` | Base URL of the Portainer instance, e.g. `https://portainer.example.com`. |
| `PORTAINER_API_KEY` | Portainer-issued API key, carried as `X-API-KEY` on every upstream call. |

The MCP server will not start if either is missing.

## Transport

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_TRANSPORT` | `stdio` | `stdio` or `http`. The container image overrides to `http`. |
| `PORTAINER_MCP_HTTP_HOST` | `127.0.0.1` | Bind address when `transport=http`. Container image overrides to `0.0.0.0` so it's reachable from outside the container. |
| `PORTAINER_MCP_HTTP_PORT` | `17717` | Bind port when `transport=http`. |
| `PORTAINER_MCP_AUTH_TOKEN` | _required for http_ | Shared bearer secret. ≥32 ASCII-printable characters, no whitespace. Generate with `openssl rand -hex 32`. Ignored under stdio. |

The stdio transport ignores everything in this section except `PORTAINER_MCP_TRANSPORT` itself.

## Hardening (HTTP transport only)

Two controls layered on top of the bearer secret:

- **DNS-rebinding allowlist** — `Host` and `Origin` headers validated against an allowlist on every request. Wildcard ports supported
  (`mcp.example.com:*`). Mismatches return 421 (Host) or 403 (Origin).
- **Audit log** — every auth attempt emits a JSON record under the `portainer_mcp.audit` sub-logger.

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_ALLOWED_HOSTS` | `127.0.0.1:*,localhost:*,[::1]:*` | Comma-separated `Host` allowlist. **Extend whenever the container is reached via a non-local hostname** — including any reverse proxy that preserves the original `Host` header. |

> [!NOTE]
> The `Origin` allowlist is not configurable and ships pinned to the localhost to provide secure defaults.
> MCP clients such as Claude Code, Claude Desktop, etc... uses programmatic access that omits the `Origin` header
> so these won't be impacted.

## Profiles

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_PROFILES` | `BASE,DOCKER,KUBERNETES` | Comma-separated named tag bundles. `ALL` disables the tag filter and exposes every operation. Unknown names will prevent the server from starting. Full list in [profiles.md](profiles.md). |
| `PORTAINER_TAGS_EXTRA` | _empty_ | Escape hatch: comma-separated raw tags appended to the resolved set. Unknown tags log a warning and pass through (they just don't match anything). |

## Behaviour

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_READ_ONLY` | `0` | When truthy, registers `GET`/`HEAD` operations only — useful for "look but don't touch" agents. Also restricts the proxy tools. |
| `PORTAINER_NO_PROXY` | `0` | When truthy, skips registration of the `docker_proxy` and `kubernetes_proxy` escape-hatch tools. |
| `PORTAINER_TLS_VERIFY` | `1` | When falsy, skips TLS verification on the upstream Portainer client. |
| `PORTAINER_MAX_RESPONSE_CHARS` | `50000` | Tool-response cap. Sized to fire before Claude Code's MCP output cap so the truncation hint (which names `select` with examples) reaches the model. |
| `PORTAINER_EXPOSE_ENV_VALUES` | `0` | When truthy, env values in stack / container / Kubernetes responses are returned as-is. Default redacts them to `[REDACTED]` and appends a one-line summary naming this variable. Redaction runs *before* `select`, so a JMESPath projection lands on the sentinel rather than the real value. |

## Logging

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_LOG_LEVEL` | `INFO` | Standard Python levels: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`. |
| `PORTAINER_MCP_LOG_FORMAT` | `text` | `text` or `json`. `text` is the human-readable shape. `json` emits one JSON envelope per line and hoists fields from records whose message is itself a JSON object — see below. The container image overrides this to `json`. |

In `json` mode, every line is a single JSON object: app logs carry a `msg` string; audit + request records have their fields merged into the envelope (no nested-string dance). Example, mixing audit, request, and plain startup lines:

```json
{"ts": "2026-05-25T12:00:00+0000", "level": "INFO", "logger": "portainer_mcp", "msg": "HTTP auth: enabled (token abcd…wxyz)"}
{"ts": "2026-05-25T12:00:01+0000", "level": "INFO", "logger": "portainer_mcp.audit", "event": "auth", "outcome": "ok", "client_ip": "203.0.113.7", "user_agent": "Claude-Code/1.2.3", "session_id": "abf3…"}
{"ts": "2026-05-25T12:00:01+0000", "level": "INFO", "logger": "fastmcp.middleware.structured_logging", "event": "request_success", "method": "tools/call", "source": "client", "duration_ms": 42.3, "client_ip": "203.0.113.7", "user_agent": "Claude-Code/1.2.3", "session_id": "abf3…"}
```

Audit and structured-request records both carry the same per-request context fields: `client_ip` (peer address), `user_agent` (HTTP header,
distinguishes Claude Code / Inspector / custom scripts), and `session_id` (the MCP `Mcp-Session-Id` assigned at `initialize` —
absent on the `initialize` request itself, present on every subsequent request in the session). With a single shared bearer the audit deliberately omits `token_fp` since it would be a constant; failed attempts likewise carry no token content.

### Audit & traceability

The audit log fires **once per HTTP request**, not once per JSON-RPC method call. The MCP Streamable HTTP transport turns a single user-visible action (e.g. one `tools/call`) into several HTTP requests:

- 1 POST carrying the JSON-RPC `tools/call` message → 1 audit row +
  1 `request_start` + 1 `request_success`
- 1 GET to open the SSE stream the response is delivered on → 1 audit
  row, no `request_*` record
- 1+ POST 202s for client-side notifications and responses → 1 audit
  row each, no `request_*` records

So one tool invocation typically produces ~4 audit rows but only one `request_start` / `request_success` pair. The "extra" audits aren't duplicates — they're real bearer checks on transport-level traffic that doesn't carry a JSON-RPC method, so the FastMCP middleware has nothing to log.

Practical consequences:

- **Count tool-call volume from `request_success`, not audit rows.**  Audit counts overstate tool traffic by ~4–6×.
- **Tune `mismatch` alerts, not `ok` rate alerts.** A single chatty agent can produce 100+ `ok` audits/minute on its own; `mismatch` is rare under normal traffic and is the signal worth paging on.
- **For "who called this tool at time T," start from the `request_*` row.** Take its `session_id`, then grep audit rows by `session_id` to see every HTTP request that session made — useful for spotting probes of unexpected paths or a client that authenticated and then went quiet.

**`session_id` is the join key.** Every audit row and every matching `request_start` / `request_success` from the same MCP session carry the same `session_id`. To trace a specific tool call back to its caller, find the `request_*` rows for the method, take the `session_id`, and grep for matching audit rows to see every authenticated HTTP request that session made — including the SSE stream, notifications, and the initial handshake.

Operator queries (`jq` against `docker logs` in JSON mode):

```bash
# Failed-auth attempts in the last hour — alert on bursts from one IP
docker logs --since 1h portainer-mcp | jq -c \
  'select(.logger == "portainer_mcp.audit" and .outcome == "mismatch")'

# Every record (audit + request) for a given session
docker logs portainer-mcp | jq -c \
  'select(.session_id == "abf3c2…")'

# Spot unfamiliar User-Agents that successfully authenticated
docker logs portainer-mcp | jq -r \
  'select(.logger == "portainer_mcp.audit" and .outcome == "ok") | .user_agent' \
  | sort -u

# Slowest tool calls (top 10) with their caller
docker logs portainer-mcp | jq -c \
  'select(.event == "request_success" and .method == "tools/call")
   | {ts, duration_ms, session_id, user_agent}' \
  | sort -t: -k3 -n -r | head -10
```
