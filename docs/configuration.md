# Configuration

This files document every knob that the `portainer-mcp` exposes, grouped by concern. All values are read from the process environment at startup. Defaults are sized for two documented deployment shapes: **local stdio** (local process, no auth, single baked key) and **HTTP per-user passthrough** (multi-user, shared bearer gate + each client's own Portainer key).

> [!NOTE]
> Truthy parsing: any value other than `0`, `false`, or `False` is treated as truthy.

## Required

| Var | Notes |
|---|---|
| `PORTAINER_URL` | Base URL of the Portainer instance, e.g. `https://portainer.example.com`. Required for every transport. |
| `PORTAINER_API_KEY` | Portainer-issued API key, carried as `X-API-KEY` on every upstream call. **stdio only** — required under stdio. Under HTTP each client forwards its own key instead (see [HTTP per-user passthrough](#http-per-user-passthrough)), so setting it under HTTP is a misconfiguration and the server refuses to boot. |

The server will not start if `PORTAINER_URL` is missing, if `PORTAINER_API_KEY` is missing under stdio, or if it is *set* under HTTP.

## Transport

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_TRANSPORT` | `stdio` | `stdio` or `http`. The container image overrides to `http`. |
| `PORTAINER_MCP_HTTP_HOST` | `127.0.0.1` | Bind address when `transport=http`. Container image overrides to `0.0.0.0` so it's reachable from outside the container. |
| `PORTAINER_MCP_HTTP_PORT` | `17717` | Bind port when `transport=http`. |
| `PORTAINER_MCP_AUTH_TOKEN` | _required for http_ | Shared bearer **gate** secret. ≥32 ASCII-printable characters, no whitespace. Generate with `openssl rand -hex 32`. Ignored under stdio. Admits the request; the caller's own Portainer key is then validated and forwarded (see below). |
| `PORTAINER_MCP_AUTH_CACHE_TTL` | `60` | Seconds to cache a validated per-user key. Positive results only; `0` disables caching (validate every request). HTTP only. |

The stdio transport ignores everything in this section except `PORTAINER_MCP_TRANSPORT` itself.

### HTTP per-user passthrough

Over HTTP, the server does **not** carry a single shared upstream identity. Each client sends two headers:

| Header | Carries | Role |
|---|---|---|
| `Authorization: Bearer <gate-token>` | The shared `PORTAINER_MCP_AUTH_TOKEN` | Front gate — constant-time compared; admits the request. Same for every client of this deployment. |
| `X-Portainer-API-Key: <ptr_…>` | The caller's **own** Portainer API key | Validated against `/users/me` on first use (cached), then forwarded upstream as `X-API-KEY`. Per-user. |

```bash
claude mcp add portainer --transport http http://mcp.example.com:17717/mcp \
  --header "Authorization: Bearer <gate-token>" \
  --header "X-Portainer-API-Key: <ptr_user_key>"
```

A missing/invalid gate token **or** missing/invalid per-user key returns 401 — there is no fallback to a shared key. This gives each user Portainer's own RBAC and per-user attribution in both Portainer's audit log and the MCP audit log.


## Hardening (HTTP transport only)

Two controls layered on top of the bearer secret:

- **DNS-rebinding allowlist** — `Host` and `Origin` headers validated against an allowlist on every request. Wildcard ports supported
  (`mcp.example.com:*`). Mismatches return 421 (Host) or 403 (Origin).
- **Audit log** — every auth attempt emits a JSON record under the `portainer_mcp.audit` sub-logger.

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_ALLOWED_HOSTS` | `127.0.0.1:*,localhost:*,[::1]:*` | Comma-separated `Host` allowlist. **Extend whenever the server is reached via a non-local hostname**, including a reverse proxy. Set it to exactly the `host:port` clients target — see note below. |

> [!NOTE]
> Match the value to the `Host` header clients actually send:
> - standard-443 reverse proxy → bare hostname (`mcp.example.com`) — clients omit the default port
> - custom port (direct, or a non-443 proxy) → pin it (`mcp.example.com:17717`, `mcp.example.com:8443`)
> - port genuinely varies → `:*` wildcard
>
> A `base:*` entry matches **only** Hosts that include a port, so it will *not* match a bare `mcp.example.com` — don't reach for the wildcard by default. Pinning is tighter, but the DNS-rebinding boundary is the **hostname**; the port is secondary.

> [!NOTE]
> The `Origin` allowlist is not configurable and ships pinned to the localhost to provide secure defaults.
> MCP clients such as Claude Code, Claude Desktop, etc... uses programmatic access that omits the `Origin` header
> so these won't be impacted.

## TLS posture (HTTP transport only)

Over HTTP the server carries two secrets on the wire — the gate token and each
caller's own Portainer API key (which never expires, so a captured one is
usable until manually revoked). To avoid serving those in clear text by
accident, **a non-loopback bind refuses to boot unless one transport posture is
declared.** Loopback binds (`make dev`, `127.0.0.1`) are exempt. A broken
declaration (cert without key, an unreadable cert, `PORTAINER_MCP_TRUST_PROXY_TLS`
without `PORTAINER_MCP_FORWARDED_ALLOW_IPS`) hard-fails at startup — it never
silently downgrades.

| Var | Default | Notes |
|---|---|---|
| `PORTAINER_MCP_TLS_CERT` | _unset_ | PEM certificate path. Must be set together with `PORTAINER_MCP_TLS_KEY`. Server-terminated TLS — no plaintext hop exists. The server **warns** if the cert is self-signed; most MCP clients reject self-signed certs by default, so install the cert's CA on each client. |
| `PORTAINER_MCP_TLS_KEY` | _unset_ | PEM private-key path. Must be set together with `PORTAINER_MCP_TLS_CERT`. |
| `PORTAINER_MCP_TRUST_PROXY_TLS` | `0` | Attest that a TLS-terminating reverse proxy sits in front. When truthy, the server trusts the proxy's `X-Forwarded-Proto: https`. Requires `PORTAINER_MCP_FORWARDED_ALLOW_IPS`. It's an explicit acknowledgment — `PORTAINER_MCP_FORWARDED_ALLOW_IPS` alone (a functional knob) does **not** satisfy the TLS gate, since it could be set for a plaintext proxy. |
| `PORTAINER_MCP_FORWARDED_ALLOW_IPS` | _unset_ | Comma-separated IPs/subnets whose `X-Forwarded-*` headers the server trusts. **Prefer the proxy's exact IP** when it's stable; widen to its **subnet** only when the proxy's address is dynamic (Docker/Kubernetes reschedule containers onto new IPs, so you can only know the network range) — and keep that subnet free of untrusted workloads, since anything on it can then spoof the scheme. A container **name won't work** (this matches the numeric source IP, not DNS): give the proxy a static container IP, or trust the user-defined network's subnet. Use `*` only when nothing but the proxy can reach the container. If the container is directly reachable (e.g. a published port) while trusting any attacker-reachable range, an attacker can spoof `X-Forwarded-Proto: https` over plaintext and defeat the TLS check — so a proxy deployment should not publish the container's port (see the Tier-2 example, which omits `-p`). Also repairs audit attribution (real client IP instead of the proxy's). |
| `PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP` | `0` | **Danger.** The only way to serve plain HTTP on a non-loopback bind. The gate token and every Portainer key cross the wire unencrypted — acceptable only on a trusted private network you fully control. Emits a `WARNING` on every start and marks the audit log with `insecure_transport: true`. Exists because self-signed certs don't work in real MCP clients, leaving small no-CA/no-IdP deployments no other option. |

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
{"ts": "2026-05-25T12:00:00+0000", "level": "INFO", "logger": "portainer_mcp", "msg": "HTTP auth: per-user passthrough (gate abcd…wxyz, validation cache ttl=60s)"}
{"ts": "2026-05-25T12:00:01+0000", "level": "INFO", "logger": "portainer_mcp.audit", "event": "auth", "outcome": "ok", "client_ip": "203.0.113.7", "user_agent": "Claude-Code/1.2.3", "portainer_user_id": 1, "portainer_username": "admin"}
{"ts": "2026-05-25T12:00:01+0000", "level": "INFO", "logger": "fastmcp.middleware.structured_logging", "event": "request_success", "method": "tools/call", "tool": "EndpointList", "duration_ms": 42.3, "client_ip": "203.0.113.7", "user_agent": "Claude-Code/1.2.3", "session_id": "abf3…", "portainer_user_id": 1, "portainer_username": "admin"}
```

Audit and structured-request records both carry the same per-request context fields: `client_ip` (peer address), `user_agent` (HTTP header,
distinguishes Claude Code / Inspector / custom scripts), and `session_id` (the MCP `Mcp-Session-Id` assigned at `initialize` —
absent on the `initialize` request itself, present on every subsequent request in the session). With a single shared gate bearer the audit deliberately omits `token_fp` since it would be a constant; failed attempts likewise carry no token content, and the per-user `X-Portainer-API-Key` is never logged.

The audit `outcome` is one of `ok`, `mismatch` (wrong gate token), `no_user_key` (gate passed but no `X-Portainer-API-Key`), or `invalid_user_key` (key rejected by `/users/me` or Portainer unreachable). A successful `ok` — and every matching structured-request record — is attributed with the validated Portainer identity: `portainer_user_id`, `portainer_username`. This is the per-user attribution the passthrough model buys; the key that resolves to that identity is never recorded. Structured-request records for a `tools/call` also carry the `tool` name (the `method` field alone is just `tools/call`).

### Audit & traceability

The audit log records **auth events, not every request.** Successful `ok` fires only on a *validation* — i.e. a cache miss that round-trips `/users/me` — so a key produces roughly **one `ok` per `PORTAINER_MCP_AUTH_CACHE_TTL` window** (plus one on each new session's first request), not one per HTTP request. Cache hits admit silently. Failures (`mismatch`, `no_user_key`, `invalid_user_key`) are never cached, so they fire on **every** failing request — which is what you want for alerting.

The MCP Streamable HTTP transport turns a single user-visible action (e.g. one `tools/call`) into several HTTP requests — the JSON-RPC POST, a GET to open the SSE response stream, and notification/response 202s. Each is a separate auth check, but with validation caching only the first (the miss) emits an `ok`; the rest are silent hits. Per-tool-call activity instead lives in the **structured request log** (`request_start` / `request_success`), which fires once per JSON-RPC method call and carries `tool`, identity, and `session_id`.

Practical consequences:

- **Count tool-call volume from `request_success`, not audit `ok` rows.** An `ok` is a validation event (~one per key per TTL window), not a request counter.
- **Read `ok` as "key K validated as user U at time T".** It's the per-user attribution + the "this credential was exercised" signal, not a request trail.
- **Alert on the failure outcomes.** `mismatch` (wrong/probing gate token) and a burst of `invalid_user_key` from one IP are the signals worth paging on; both fire per-request, uncached.
- **For "who called this tool at time T," start from the `request_*` row** — it has `tool`, `portainer_username`, and `session_id` directly.

**`session_id` is the join key** across `request_start` / `request_success` records (and any failure audits) within a session. Note a validation `ok` often fires on the `initialize` request, before `Mcp-Session-Id` is assigned, so it may carry no `session_id` — trace activity through the request log, which always has it once the session is established.

Operator queries (`jq` against `docker logs` in JSON mode):

```bash
# Failed-auth attempts in the last hour — alert on bursts from one IP
docker logs --since 1h portainer-mcp | jq -c \
  'select(.logger == "portainer_mcp.audit" and (.outcome | IN("mismatch","no_user_key","invalid_user_key")))'

# Every record (audit + request) for a given session
docker logs portainer-mcp | jq -c \
  'select(.session_id == "abf3c2…")'

# Which users validated, and from where
docker logs portainer-mcp | jq -r \
  'select(.logger == "portainer_mcp.audit" and .outcome == "ok")
   | "\(.portainer_username) \(.client_ip) \(.user_agent)"' \
  | sort -u

# Slowest tool calls (top 10) with their caller
docker logs portainer-mcp | jq -c \
  'select(.event == "request_success" and .method == "tools/call")
   | {ts, duration_ms, session_id, user_agent}' \
  | sort -t: -k3 -n -r | head -10
```
