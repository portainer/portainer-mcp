# Profiles

The Portainer API spec exposes 400+ operations across 40+ tags. Auto-converting
all of them produces a tool list too noisy for MCP clients to navigate, so the
server runs with a tag allowlist. Profiles are named bundles of those tags.

## Quick reference

| Env var | Default | Effect |
|---|---|---|
| `PORTAINER_PROFILES` | `BASE,DOCKER,KUBERNETES` | Comma-separated profiles to enable. Tag sets are union'd. Empty or unset → default. |
| `PORTAINER_TAGS_EXTRA` | _empty_ | Comma-separated extra tags appended to the union. Literal tag names only — no wildcards or globs. Escape hatch for orphan tags below. |
| `PORTAINER_READ_ONLY` | `0` | `1` restricts to `GET`/`HEAD` operations only (strict — HTTP method is the read/write classifier). |
| `PORTAINER_NO_PROXY` | `0` | `1` skips `docker_proxy` / `kubernetes_proxy` registration. |

Unknown profile names fail loudly at startup. Unknown extras (tags) log a warning
and pass through harmlessly — they just won't match any operation.

## Profiles

| Profile | Tags | Use case |
|---|---|---|
| `BASE` | `auth`, `system`, `status`, `settings`, `motd` | Server identity, login, settings. Effectively required — most workflows assume these are present. |
| `DOCKER` | `docker`, `endpoints`, `stacks` | Docker workloads on Portainer-managed environments. |
| `KUBERNETES` | `kubernetes`, `helm`, `endpoints`, `stacks` | Kubernetes workloads, including Helm releases. Shares `endpoints`/`stacks` with `DOCKER` — union'd, no duplication. |
| `EDGE` | `edge`, `edge_stacks`, `edge_jobs`, `edge_groups`, `edge_update_schedules`, `edge_configs` | Portainer Edge fleet management. |
| `ADMIN` | `users`, `teams`, `team_memberships`, `roles`, `ldap`, `license`, `backup`, `registries`, `endpoint_groups`, `policies`, `resource_controls`, `tags` | Platform administration: identity, registries, backups, RBAC. |

### `ALL` option

`PORTAINER_PROFILES=ALL` bypasses the tag filter entirely — every operation
in the spec is registered as a tool. It is **not** a bundle of tags; using it
means future upstream tags appear automatically without any code change here.

Compose with `PORTAINER_READ_ONLY=1` for an inventory/audit persona that can
see everything but mutate nothing.

## Default coverage

`PORTAINER_PROFILES=BASE,DOCKER,KUBERNETES` (the default) covers 10 tags and
~197 spec operations. The five-profile union covers 28 tags and
~342 operations. `ALL` covers everything.

## Orphan tags

These tags don't live in any profile. Add them via `PORTAINER_TAGS_EXTRA`
when you need them, or switch to `ALL`:

| Count | Tag | Notes |
|---:|---|---|
| 15 | `observability` | Container/pod logs, metrics, stats. |
| 12 | `omni` | Talos Kubernetes cluster management. |
| 10 | `custom_templates` | User-defined app templates. |
| 9 | `gitops` | GitOps configuration surface. |
| 8 | `cloud_credentials` | Cloud provider credentials. |
| 6 | `webhooks` | Webhook management. |
| 4 | `kaas` | Kubernetes-as-a-Service provisioning. |
| 4 | `useractivity` | Audit log. |
| 3 | `support` | Support bundles / diagnostics. |
| 2 | `recommendations` | Recommendation engine. |
| 2 | `ssl` | Server TLS certificates. |
| 2 | `templates` | App template library. |
| 1 | `auto_updates` | Auto-update configuration. |
| 1 | `upload` | File upload endpoint. |

Total orphaned: 14 tags, 79 operations. Tags repeatedly requested via
`PORTAINER_TAGS_EXTRA` are the signal that a profile should grow or a new
profile should be added.

## Examples

```bash
# Default — Docker + Kubernetes workloads
PORTAINER_PROFILES=BASE,DOCKER,KUBERNETES uv run portainer-mcp

# Edge-only fleet operator
PORTAINER_PROFILES=BASE,EDGE uv run portainer-mcp

# Audit persona: everything, read-only
PORTAINER_PROFILES=ALL PORTAINER_READ_ONLY=1 uv run portainer-mcp

# Curated tools only, no Docker/K8s proxy escape hatch
PORTAINER_PROFILES=BASE,DOCKER PORTAINER_NO_PROXY=1 uv run portainer-mcp

# Default + observability for a troubleshooting workflow
PORTAINER_PROFILES=BASE,DOCKER,KUBERNETES PORTAINER_TAGS_EXTRA=observability \
  uv run portainer-mcp
```

## Read-only semantics

`PORTAINER_READ_ONLY=1` filters by HTTP method: only `GET` and `HEAD`
operations are registered as tools, and the proxy tools reject non-GET
requests at call time. A handful of Portainer endpoints use `POST` for
read-shaped operations (e.g. some snapshot listings); read-only mode hides
those too. This is deliberate — the simple method-based rule is predictable
and won't rot, where an operationId denylist would.
