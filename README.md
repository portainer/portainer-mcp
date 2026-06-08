# Portainer MCP

Official MCP server for Portainer, generated from the Portainer OpenAPI spec via [FastMCP](https://github.com/PrefectHQ/fastmcp).

## Overview

This MCP server exposes the Portainer REST API as MCP tools: list and inspect environments, manage GitOps workflows, troubleshoot Docker and Kubernetes resources. It also supports proxying requests to the underlying Docker and K8s APIs of each environment.

Match the MCP server's minor version to your Portainer instance's minor — e.g. MCP server 2.42.x with Portainer 2.42.x. See [Version compatibility](#version-compatibility) for details.

## Getting started

The MCP server can be executed locally via `uvx` or as a container.

Use the first approach to explore the MCP capabilities locally and deploy it inside your infrastructure as a container for a team based deployment setup.

> [!NOTE]
> Before using the MCP, make sure to generate an API key in Portainer under **My Account → Access tokens** first as both paths need it.

### Single user (stdio via `uvx`)

The recommended way to test the MCP server locally. Runs as a stdio process on your machine and connects directly to the Portainer instance.

> [!NOTE]
> `uv` must be installed and available on `PATH`.
> See [the uv install docs](https://docs.astral.sh/uv/getting-started/installation/).

Register with Claude Code:

```bash
claude mcp add portainer \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -- uvx --from "mcp-portainer~=2.42.0" mcp-portainer
```

> [!NOTE]
> Set `PORTAINER_TLS_VERIFY=0` if your Portainer instance uses self-signed TLS certificates.

For other clients, see
[`docs/distribution/`](https://github.com/portainer/portainer-mcp/tree/main/docs/distribution).
Contributions for other client instructions are welcome!

### Team deployment (container)

The recommended way to have multiple users interacting with your Portainer instance via MCP. Deployed as a [`container`](https://hub.docker.com/r/portainer/portainer-mcp) inside your infrastructure, accessed by users from their workstations over HTTP. A shared secret gates the MCP server and every client also forwards its own Portainer API key so that each user acts under their own Portainer identity.

> [!IMPORTANT]
> The container terminates HTTP, **NOT HTTPS**. Both the gate secret and each user's Portainer API key can be intercepted on any path between client and server without TLS termination.
> 
> It is **NOT** recommended to expose this MCP server on the public internet.
> 
> Even with a TLS-terminating reverse proxy in front, the recommendation is to host this MCP server inside your private infrastructure.

Run the MCP server as a container in your infrastructure:

```bash
TOKEN=$(openssl rand -hex 32)
docker run -d --name portainer-mcp -p 17717:17717 \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_MCP_AUTH_TOKEN="$TOKEN" \
  -e PORTAINER_MCP_ALLOWED_HOSTS=mcp.example.com:17717 \
  portainer/portainer-mcp:2.42
```

Set `PORTAINER_MCP_ALLOWED_HOSTS` to the hostname or IP address that users will use to reach the MCP — otherwise the DNS-rebinding allowlist 421-rejects the request.

`PORTAINER_MCP_AUTH_TOKEN` is **required** in HTTP mode. It's the shared front-gate secret you distribute to your users; their MCP client sends it via the `Authorization` header. It only admits the request — what each user can *do* is governed by their own Portainer API key.

Adding the MCP endpoint on Claude Code:
```bash
claude mcp add portainer --transport http http://mcp.example.com:17717/mcp \
  --header "Authorization: Bearer <gate-token>" \
  --header "X-Portainer-API-Key: <ptr_user_key>"
```

### Hygiene skill (recommended)

This repo ships a Claude Code skill ([`portainer-mcp-hygiene`](https://github.com/portainer/portainer-mcp/blob/main/skills/portainer-mcp-hygiene/SKILL.md)) that helps the model query the MCP efficiently and keep responses within context. Install user-wide, pinned to the same tag as the server:

```bash
mkdir -p ~/.claude/skills/portainer-mcp-hygiene && \
  curl -fsSL https://raw.githubusercontent.com/portainer/portainer-mcp/2.42.0/skills/portainer-mcp-hygiene/SKILL.md \
  -o ~/.claude/skills/portainer-mcp-hygiene/SKILL.md
```

It is recommended to re-run on each server upgrade so the skill stays in sync.

## Restricting and expanding the MCP server capabilities

The MCP server comes with the following capabilities enabled by default:
* Basic Portainer operation support (settings, version, environments...)
* Docker operation support
* Kubernetes operation support
* Docker and Kubernetes proxy support
* Redacting environment variables values (enabled by default)

For restricting or expanding this set of capabilities, see [`docs/profiles.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/profiles.md).

## Version compatibility

**Match the MCP server's minor to your Portainer minor.** The major+minor tracks the Portainer API version the embedded spec targets.

| Server version | Portainer (CE / EE) |
| -------------- | ------------------- |
| `2.42.x`       | `2.42.x`            |
| `2.41.x`       | `2.41.x`            |

For more information about the versioning policy, see [`docs/versioning.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/versioning.md).

## Configuration

The MCP server exposes different capabilities such as:
* Enable different set of tools based on specific profile configuration
* Widen the API coverage by specifying extra tags to cover
* Expose only read-only capabilities
* Disable proxy capabilities

For more information about the MCP server configuration, refer to [`docs/configuration.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/configuration.md).
