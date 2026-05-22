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

The server is distributed on PyPI as `mcp-portainer`. MCP clients launch it as
a subprocess via [`uvx`](https://docs.astral.sh/uv/), so `uv` must be on
`PATH` — see [the uv install docs](https://docs.astral.sh/uv/getting-started/installation/).

Generate an API key in Portainer under **My Account → Access tokens**, then
register the server with Claude Code:

```bash
claude mcp add portainer \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -- uvx --from "mcp-portainer~=2.42.0" mcp-portainer
```

`~=2.42.0` picks up MCP-only patch fixes against the same Portainer minor —
see [Version compatibility](#version-compatibility) for the policy.

**Recommended: install the hygiene skill.** This repo ships a Claude Code
skill ([`portainer-mcp-hygiene`](https://github.com/portainer/portainer-mcp/blob/main/skills/portainer-mcp-hygiene/SKILL.md))
that helps the model query the MCP efficiently and keep responses within
context. Install user-wide, pinned to the same tag as the server:

```bash
mkdir -p ~/.claude/skills/portainer-mcp-hygiene && \
  curl -fsSL https://raw.githubusercontent.com/portainer/portainer-mcp/2.42.0/skills/portainer-mcp-hygiene/SKILL.md \
  -o ~/.claude/skills/portainer-mcp-hygiene/SKILL.md
```

Re-run on each server upgrade so the skill stays in sync.

For other clients, see [`docs/distribution/`](https://github.com/portainer/portainer-mcp/tree/main/docs/distribution). See
[Configuration](#configuration) for optional knobs.

Contribution are welcome for other client instructions !

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

Advanced profile setup — per-profile tag lists, orphan tags, read-only
semantics — see [`docs/profiles.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/profiles.md).
