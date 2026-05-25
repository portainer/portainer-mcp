# Portainer MCP

Official MCP server for Portainer, generated from the Portainer OpenAPI spec via
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
both paths need it. Every env var and tunable knob is documented in
[`docs/configuration.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/configuration.md)
— the examples below only show what's required to get each path running.

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

For shared deployments — one MCP endpoint running on a VM in your
infrastructure, accessed by team members from their workstations over
HTTP, gated by a shared bearer secret.
[`docker.io/portainer/portainer-mcp`](https://hub.docker.com/r/portainer/portainer-mcp)
ships multi-arch (`linux/amd64`, `linux/arm64`).

> **Do not expose port 8000 to the public internet without TLS in front.**
> The image serves plain HTTP with a shared bearer; the secret is
> interceptable on any path between client and server without TLS
> termination.

On the VM that will host the MCP endpoint:

```bash
docker run -d --name portainer-mcp -p 8000:8000 \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -e PORTAINER_MCP_AUTH_TOKEN="$(openssl rand -hex 32)" \
  -e PORTAINER_MCP_ALLOWED_HOSTS=mcp.example.com:8000 \
  portainer/portainer-mcp:2.42
```

Set `PORTAINER_MCP_ALLOWED_HOSTS` to the hostname your team will use to
reach the VM — otherwise the DNS-rebinding allowlist 421-rejects the
request. The startup log emits a WARNING flagging the mismatch and the
421 body names the env var, so the fix is visible from either side.

`PORTAINER_MCP_AUTH_TOKEN` is **required** in HTTP mode; startup fails
loudly without it. Distribute the same token to every team member; their
MCP client sends it as `Authorization: Bearer <token>`:

```bash
claude mcp add portainer --transport http http://mcp.example.com:8000/mcp \
  --header "Authorization: Bearer <token>"
```

Full setup, tag scheme, and reverse-proxy guidance:
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

Full env-var reference, grouped by concern (transport, hardening,
profiles, behaviour, logging):
[`docs/configuration.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/configuration.md).
Advanced profile setup — per-profile tag lists, orphan tags, read-only
semantics — see [`docs/profiles.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/profiles.md).
