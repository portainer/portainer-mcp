# portainer-mcp-fastmcp

MCP server for Portainer, generated from the Portainer OpenAPI spec via
[FastMCP](https://github.com/PrefectHQ/fastmcp).

## Overview

Exposes Portainer's REST API as MCP tools — list environments, manage
Docker containers and stacks, query Kubernetes resources, run Helm
releases. Two escape-hatch tools (`docker_proxy`, `kubernetes_proxy`)
forward arbitrary paths to the underlying Docker/K8s APIs for endpoints
the spec doesn't enumerate.

**Status: in development.** Tool names, env vars, and defaults can
change between releases. Pin loosely (see
[Version compatibility](#version-compatibility)) to pick up MCP-only
fixes without surprise.

Architecture overview: [`docs/architecture.md`](docs/architecture.md).

## Getting started

TBD.

## Version compatibility

**Match your server's minor to your Portainer minor.** The
major+minor tracks the Portainer API version the embedded spec targets.

| Server version | Portainer (CE / EE) |
| -------------- | ------------------- |
| `2.41.x`       | `2.41.x`            |

- EE spec only — CE is a subset and works on a best-effort basis.
- Full policy: [`docs/versioning.md`](docs/versioning.md).

## Configuration

All knobs are environment variables. Only `PORTAINER_URL` and
`PORTAINER_API_KEY` are required.

| Env var | Default | Effect |
|---|---|---|
| `PORTAINER_URL` | — | **Required.** Portainer base URL. |
| `PORTAINER_API_KEY` | — | **Required.** Portainer API key. |
| `PORTAINER_PROFILES` | `BASE,DOCKER,KUBERNETES` | Tag bundles to enable. `ALL` disables the filter. |
| `PORTAINER_TAGS_EXTRA` | _empty_ | Extra tags appended to the profile union (escape hatch). |
| `PORTAINER_READ_ONLY` | `0` | `1` restricts to `GET`/`HEAD` operations. |
| `PORTAINER_NO_PROXY` | `0` | `1` skips `docker_proxy` / `kubernetes_proxy`. |
| `PORTAINER_TLS_VERIFY` | `1` | `0` skips TLS verification (Portainer instance using self-signed certs). |
| `PORTAINER_MAX_RESPONSE_CHARS` | `50000` | Response truncation cap. Size to ~80% of your MCP client's output ceiling. |
| `PORTAINER_MCP_LOG` | `logs/portainer-mcp.log` | Log file path. |
| `PORTAINER_MCP_LOG_LEVEL` | `INFO` | One of `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`. |

Advanced profile setup — per-profile tag lists, orphan tags, read-only
semantics — see [`docs/profiles.md`](docs/profiles.md).
