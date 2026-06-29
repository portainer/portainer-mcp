# Portainer MCP

Official MCP server for Portainer, generated from the Portainer OpenAPI spec via [FastMCP](https://github.com/PrefectHQ/fastmcp).

## Overview

This MCP server exposes the Portainer REST API as MCP tools: list and inspect environments, manage GitOps workflows, troubleshoot Docker and Kubernetes resources. It also supports proxying requests to the underlying Docker and K8s APIs of each environment.

Match the MCP server's minor version to your Portainer instance's minor — e.g. MCP server 2.43.x with Portainer 2.43.x. See [Version compatibility](#version-compatibility) for details.

## Getting started

The MCP server supports different deployment scenarios:
* execute it locally via `uvx`
* install it as a MCP bundle
* deploy it as a container

Use the `uvx` approach or the MCP bundle to explore the MCP capabilities locally and deploy it inside your infrastructure as a container for a team based deployment setup.

> [!NOTE]
> Before using the MCP, make sure to generate an API key in Portainer under **My Account → Access tokens** first as both paths need it.

### MCP bundle (one-click install)

The recommended way to test the MCP server locally. Your client must support [MCP bundles](https://github.com/modelcontextprotocol/mcpb):

1. Fetch the self-contained `.mcpb` bundle for your platform from the [latest release](https://github.com/portainer/portainer-mcp/releases/latest)
2. Double-click to install
3. Enter your Portainer URL and API key.

### Single user (stdio via `uvx`)

The other way to test the MCP server locally. Runs as a stdio process on your machine and connects directly to the Portainer instance.

> [!NOTE]
> `uv` must be installed and available on `PATH`.
> See [the uv install docs](https://docs.astral.sh/uv/getting-started/installation/).
>
> Set `PORTAINER_TLS_VERIFY=0` if your Portainer instance uses self-signed TLS certificates.

Register with Claude Code:

````bash
claude mcp add portainer \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -- uvx --from "mcp-portainer~=2.43.0" mcp-portainer
````

For other clients, see
[`docs/distribution/`](https://github.com/portainer/portainer-mcp/tree/main/docs/distribution).

### Team deployment (container)

The recommended way to have multiple users interacting with your Portainer instance via MCP. Deployed as a [`container`](https://hub.docker.com/r/portainer/portainer-mcp) inside your infrastructure, accessed by users from their workstations over HTTPS. A shared secret gates the MCP server and every client also forwards its own Portainer API key so that each user acts under their own Portainer identity.

> [!IMPORTANT]
> Both the gate secret and each user Portainer API key are sent across the wire. The container deployment requires you to declare a transport posture: bring your own TLS certificates, attest a TLS-terminating reverse proxy setup or explicitly opt-in to plaintext. 
> 
> Plaintext is a deliberate, dangerous choice — see the three options below.
> 
> It is **NOT** recommended to expose this MCP server on the public internet, host it inside your private infrastructure even behind a TLS proxy.

See more info below about the different deployment scenarios. For any of these scenarios:
* Set `PORTAINER_MCP_ALLOWED_HOSTS` to the hostname or IP address that users will use to reach the MCP — otherwise the DNS-rebinding allowlist 421-rejects the request.
* `PORTAINER_MCP_AUTH_TOKEN` is **required** in HTTP mode. It's the shared front-gate secret you distribute to your users; their MCP client sends it via the `Authorization` header. It only admits the request — what each user can *do* is governed by their own Portainer API key.


#### Option A - BYO certificates

> [!NOTE]
> The server will warn if using self-signed certificates. Using a private CA cert won't warn, but in both cases you will likely need to jump through some hoops to configure the MCP clients to accept it.

Deploy the container to use your own set of TLS certificates:

````bash
TOKEN=$(openssl rand -hex 32)
docker run -d --name portainer-mcp -p 17717:17717 \
	-v /etc/portainer-mcp/tls:/tls:ro \
	-e PORTAINER_URL=https://portainer.example.com \
	-e PORTAINER_MCP_AUTH_TOKEN="$TOKEN" \
	-e PORTAINER_MCP_ALLOWED_HOSTS=mcp.example.com:17717 \
	-e PORTAINER_MCP_TLS_CERT=/tls/cert.pem \
	-e PORTAINER_MCP_TLS_KEY=/tls/key.pem \
	portainer/portainer-mcp:2.43
````

Then connect your client:

````bash
claude mcp add portainer --transport http https://mcp.example.com:17717/mcp \
  --header "Authorization: Bearer <gate-token>" \
  --header "X-Portainer-API-Key: <ptr_user_key>"
````

#### Option B - TLS-terminated reverse proxy

> [!NOTE]
> Don't publish the container port when using a reverse proxy in front of the MCP container, only the proxy should be able to reach it.
>
> Use your proxy exact IP if stable for `PORTAINER_MCP_FORWARDED_ALLOW_IPS`.
>
> Make sure that your proxy forwards the original `Host` and the `X-Forwarded-Proto: https` headers.

BYO proxy and set up a TLS-terminated proxy in front of the container:

````bash
TOKEN=$(openssl rand -hex 32)
docker run -d --name portainer-mcp \
	-e PORTAINER_URL=https://portainer.example.com \
	-e PORTAINER_MCP_AUTH_TOKEN="$TOKEN" \
	-e PORTAINER_MCP_ALLOWED_HOSTS=mcp.example.com \
	-e PORTAINER_MCP_TRUST_PROXY_TLS=1 \
	-e PORTAINER_MCP_FORWARDED_ALLOW_IPS=172.18.0.0/16 \
	portainer/portainer-mcp:2.43
````

Then connect your client:

````bash
claude mcp add portainer --transport http https://mcp.example.com/mcp \
  --header "Authorization: Bearer <gate-token>" \
  --header "X-Portainer-API-Key: <ptr_user_key>"
````

#### Option C - Plaintext HTTP

> [!WARNING]
> It is **NOT** recommended to use this outside of a trusted private network deployment.

Use the `PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP=1` flag to start the server with HTTP only.

````bash
TOKEN=$(openssl rand -hex 32)
docker run -d --name portainer-mcp -p 17717:17717 \
	-e PORTAINER_URL=https://portainer.example.com \
	-e PORTAINER_MCP_AUTH_TOKEN="$TOKEN" \
	-e PORTAINER_MCP_ALLOWED_HOSTS=mcp.example.com:17717 \
	-e PORTAINER_MCP_DANGEROUSLY_ALLOW_PLAINTEXT_HTTP=1 \
	portainer/portainer-mcp:2.43
````

Then connect your client:

````bash
claude mcp add portainer --transport http http://mcp.example.com:17717/mcp \
  --header "Authorization: Bearer <gate-token>" \
  --header "X-Portainer-API-Key: <ptr_user_key>"
````

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
| `2.43.x`       | `2.43.x`            |
| `2.42.x`       | `2.42.x`            |
| `2.41.x`       | `2.41.x`            |

For more information about the versioning policy, see [`docs/versioning.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/versioning.md).

## Configuration

The MCP server exposes different capabilities such as:
* Enable different set of tools based on specific profile configuration
* Widen the API coverage by specifying extra tags to cover
* Expose only read-only capabilities
* Disable proxy capabilities
* Tuning the transport capabilities and configuring the TLS posture
* Logging configuration

For more information about the MCP server configuration, refer to [`docs/configuration.md`](https://github.com/portainer/portainer-mcp/blob/main/docs/configuration.md).
