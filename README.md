# Portainer MCP

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

Architecture overview: [`docs/architecture.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/architecture.md).

## Getting started

Pick the install path that matches your scenario. Both run the same server
from the same release — the difference is how clients reach it.

| Scenario | Transport | Install path |
|---|---|---|
| Single developer trying it / dev loop | stdio (subprocess) | [PyPI via `uvx`](#single-user-stdio-via-uvx) |
| Team deployment, shared MCP endpoint | HTTP + bearer token | [Container](#team-deployment-container) |

Generate an API key in Portainer under **My Account → Access tokens** first;
both paths need it.

### Single user (stdio via `uvx`)

For trying the MCP locally, single-user evaluation, dev loop. The MCP client
launches the server as a subprocess via [`uvx`](https://docs.astral.sh/uv/),
so `uv` must be on `PATH` — see
[the uv install docs](https://docs.astral.sh/uv/getting-started/installation/).

Register with Claude Code:

```bash
claude mcp add portainer \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -- uvx --from "mcp-portainer~=2.42.0" mcp-portainer
```

`~=2.42.0` picks up MCP-only patch fixes against the same Portainer minor —
see [Version compatibility](#version-compatibility) for the policy.

For Claude Desktop and other clients, see
[`docs/distribution/`](https://github.com/portainer/portainer-mcp/tree/main/docs/distribution).
Contributions for other client instructions are welcome!

### Team deployment (container)

For shared deployments — one MCP endpoint per Portainer tenant, accessed by
multiple users over HTTP, gated by a shared bearer secret.
[`docker.io/portainer/portainer-mcp`](https://hub.docker.com/r/portainer/portainer-mcp)
ships multi-arch (`linux/amd64`, `linux/arm64`).

> **Do not expose port 8000 to the public internet without TLS in front.**
> The image serves plain HTTP with a shared bearer; the secret is
> interceptable on any path between client and server without TLS
> termination.

```bash
docker run -d --name portainer-mcp -p 8000:8000 \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -e PORTAINER_MCP_AUTH_TOKEN="$(openssl rand -hex 32)" \
  portainer/portainer-mcp:2.42
```

`PORTAINER_MCP_AUTH_TOKEN` is **required** in HTTP mode — startup fails loudly
without it. Distribute the same token to every team member; their MCP client
sends it as `Authorization: Bearer <token>`. Full setup, tag scheme, and
reverse-proxy guidance:
[`docs/docker.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/docker.md).

### Hygiene skill (recommended for both paths)

This repo ships a Claude Code skill
([`portainer-mcp-hygiene`](https://github.com/portainer/portainer-mcp/blob/main/skills/portainer-mcp-hygiene/SKILL.md))
that helps the model query the MCP efficiently and keep responses within
context. Install user-wide, pinned to the same tag as the server:

```bash
mkdir -p ~/.claude/skills/portainer-mcp-hygiene && \
  curl -fsSL https://raw.githubusercontent.com/portainer/portainer-mcp/2.42.0/skills/portainer-mcp-hygiene/SKILL.md \
  -o ~/.claude/skills/portainer-mcp-hygiene/SKILL.md
```

Re-run on each server upgrade so the skill stays in sync.

## Version compatibility

**Match your server's minor to your Portainer minor.** The
major+minor tracks the Portainer API version the embedded spec targets.

| Server version | Portainer (CE / EE) |
| -------------- | ------------------- |
| `2.42.x`       | `2.42.x`            |
| `2.41.x`       | `2.41.x`            |

- Full policy: [`docs/versioning.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/versioning.md).

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
| `PORTAINER_MCP_LOG_LEVEL` | `INFO` | One of `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`. Logs go to stderr. |
| `PORTAINER_MCP_TRANSPORT` | `stdio` | `stdio` (default) or `http`. `http` binds a local server for dev / remote deployment. |
| `PORTAINER_MCP_HTTP_HOST` | `127.0.0.1` | Bind host when `PORTAINER_MCP_TRANSPORT=http`. |
| `PORTAINER_MCP_HTTP_PORT` | `8000` | Bind port when `PORTAINER_MCP_TRANSPORT=http`. |
| `PORTAINER_MCP_AUTH_TOKEN` | — | **Required** when `PORTAINER_MCP_TRANSPORT=http`; ignored for stdio. Shared bearer secret; clients must send `Authorization: Bearer <token>`. Min 32 chars, no whitespace — generate with `openssl rand -hex 32`. |

Advanced profile setup — per-profile tag lists, orphan tags, read-only
semantics — see [`docs/profiles.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/profiles.md).
