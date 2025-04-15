# Portainer MCP
[![Go Report Card](https://goreportcard.com/badge/github.com/portainer/portainer-mcp)](https://goreportcard.com/report/github.com/portainer/portainer-mcp)
[![Go Coverage](https://github.com/portainer/portainer-mcp/wiki/coverage.svg)](https://raw.githack.com/wiki/portainer/portainer-mcp/coverage.html)

Ever wished you could just ask Portainer what's going on?

![portainer-mcp-demo](https://downloads.portainer.io/mcp-demo3.gif)

## Overview

Portainer MCP is a work in progress implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) for Portainer environments. This project aims to provide a standardized way to connect Portainer's container management capabilities with AI models and other services.

MCP (Model Context Protocol) is an open protocol that standardizes how applications provide context to LLMs (Large Language Models). Similar to how USB-C provides a standardized way to connect devices to peripherals, MCP provides a standardized way to connect AI models to different data sources and tools.

This implementation focuses on exposing Portainer environment data through the MCP protocol, allowing AI assistants and other tools to interact with your containerized infrastructure in a secure and standardized way.

See the [Portainer Version Support](#portainer-version-support) and [Supported Capabilities](#supported-capabilities) sections for more details on compatibility and available features.

*Note: This project is currently under development.*

It is currently designed to work with a Portainer administrator API token.

## Installation

You can download pre-built binaries for Linux (amd64) and macOS (arm64) from the [**Latest Release Page**](https://github.com/portainer/portainer-mcp/releases/latest). Find the appropriate archive for your operating system and architecture under the "Assets" section.

1.  **Download the archive.** You can usually download this directly from the release page.

    Alternatively, you can use `curl`. Here are examples for downloading the archive for version `v0.2.0`:

    *   **Linux (AMD64):**
        ```bash
        curl -Lo portainer-mcp-v0.2.0-linux-amd64.tar.gz https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-linux-amd64.tar.gz
        ```
    *   **macOS (ARM64):**
        ```bash
        curl -Lo portainer-mcp-v0.2.0-darwin-arm64.tar.gz https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-darwin-arm64.tar.gz
        ```

2.  **(Optional but recommended) Verify the checksum.** First, download the corresponding `.md5` checksum file. Use `curl` with the appropriate URL, replacing `<VERSION>`, `<OS>`, and `<ARCH>`.

    *   **Example for `v0.2.0` on Linux (AMD64):**
        ```bash
        curl -Lo portainer-mcp-v0.2.0-linux-amd64.tar.gz.md5 https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-linux-amd64.tar.gz.md5
        # Now verify
        md5sum -c portainer-mcp-v0.2.0-linux-amd64.tar.gz.md5
        ```
    *   **Example for `v0.2.0` on macOS (ARM64):**
        ```bash
        curl -Lo portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5 https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5
        # Now verify (output should match the content of the .md5 file)
        if [ "$(md5 -q portainer-mcp-v0.2.0-darwin-arm64.tar.gz)" = "$(cat portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5)" ]; then echo "OK"; else echo "FAILED"; fi
        ```
        If the verification command outputs "OK", the file is intact.

3.  **Extract the archive:**
    ```bash
    tar -xzf portainer-mcp-v<VERSION>-<OS>-<ARCH>.tar.gz
    ```
    Replace `<VERSION>`, `<OS>`, and `<ARCH>` with the appropriate values (e.g., `v0.2.0-linux-amd64`). This will extract the `portainer-mcp` executable.

4.  **Move the executable** to a location in your `$PATH` (e.g., `/usr/local/bin`) or note its location for the configuration step below.

# Usage

With Claude Desktop, configure it like so:

```
{
    "mcpServers": {
        "portainer": {
            "command": "/path/to/portainer-mcp",
            "args": [
                "-server",
                "[IP]:[PORT]",
                "-token",
                "[TOKEN]"
                "-tools",
                "/tmp/tools.yaml"
            ]
        }
    }
}
```

> [!NOTE]
> By default, the tool looks for "tools.yaml" in the same directory as the binary. You may need to modify this path as described above, particularly when using AI assistants like Claude that have restricted write permissions to the working directory.

Replace `[IP]`, `[PORT]` and `[TOKEN]` with the IP, port and API access token associated with your Portainer instance.

## Tool Customization

By default, the tool definitions are embedded in the binary. The application will create a tools file at the default location if one doesn't already exist.

You can customize the tool definitions by specifying a custom tools file path using the `-tools` flag:

```
{
    "mcpServers": {
        "portainer": {
            "command": "/path/to/portainer-mcp",
            "args": [
                "-server",
                "[IP]:[PORT]",
                "-token",
                "[TOKEN]",
                "-tools",
                "/path/to/custom/tools.yaml"
            ]
        }
    }
}
```

The default tools file is available for reference at `internal/tooldef/tools.yaml` in the source code. You can modify the descriptions of the tools and their parameters to alter how AI models interpret and decide to use them. **Important:** Do not change the tool names or parameter definitions (other than descriptions), as this will prevent the tools from being properly registered and functioning correctly.

## Read-Only Mode

For security-conscious users, the application can be run in read-only mode. This mode ensures that only read operations are available, completely preventing any modifications to your Portainer resources.

To enable read-only mode, add the `-read-only` flag to your command arguments:

```
{
    "mcpServers": {
        "portainer": {
            "command": "/path/to/portainer-mcp",
            "args": [
                "-server",
                "[IP]:[PORT]",
                "-token",
                "[TOKEN]",
                "-read-only"
            ]
        }
    }
}
```

When using read-only mode:
- Only read tools (list, get) will be available to the AI model
- All write tools (create, update, delete) are not loaded
- The Docker proxy requests tool is not loaded

# Portainer Version Support

This tool is pinned to support a specific version of Portainer. The application will validate the Portainer server version at startup and fail if it doesn't match the required version.

| Portainer MCP Version  | Supported Portainer Version |
|--------------|----------------------------|
| 0.1.0 | 2.28.1 |
| 0.2.0 | 2.28.1 |

# Supported Capabilities

The following table lists the currently (latest version) supported operations through MCP tools:

| Resource | Operation | Description | Supported In Version |
|----------|-----------|-------------|----------------------|
| **Environments** | | | |
| | ListEnvironments | List all available environments | 0.1.0 |
| | UpdateEnvironmentTags | Update tags associated with an environment | 0.1.0 |
| | UpdateEnvironmentUserAccesses | Update user access policies for an environment | 0.1.0 |
| | UpdateEnvironmentTeamAccesses | Update team access policies for an environment | 0.1.0 |
| **Environment Groups (Edge Groups)** | | | |
| | ListEnvironmentGroups | List all available environment groups | 0.1.0 |
| | CreateEnvironmentGroup | Create a new environment group | 0.1.0 |
| | UpdateEnvironmentGroupName | Update the name of an environment group | 0.1.0 |
| | UpdateEnvironmentGroupEnvironments | Update environments associated with a group | 0.1.0 |
| | UpdateEnvironmentGroupTags | Update tags associated with a group | 0.1.0 |
| **Access Groups (Endpoint Groups)** | | | |
| | ListAccessGroups | List all available access groups | 0.1.0 |
| | CreateAccessGroup | Create a new access group | 0.1.0 |
| | UpdateAccessGroupName | Update the name of an access group | 0.1.0 |
| | UpdateAccessGroupUserAccesses | Update user accesses for an access group | 0.1.0 |
| | UpdateAccessGroupTeamAccesses | Update team accesses for an access group | 0.1.0 |
| | AddEnvironmentToAccessGroup | Add an environment to an access group | 0.1.0 |
| | RemoveEnvironmentFromAccessGroup | Remove an environment from an access group | 0.1.0 |
| **Stacks (Edge Stacks)** | | | |
| | ListStacks | List all available stacks | 0.1.0 |
| | GetStackFile | Get the compose file for a specific stack | 0.1.0 |
| | CreateStack | Create a new Docker stack | 0.1.0 |
| | UpdateStack | Update an existing Docker stack | 0.1.0 |
| **Tags** | | | |
| | ListEnvironmentTags | List all available environment tags | 0.1.0 |
| | CreateEnvironmentTag | Create a new environment tag | 0.1.0 |
| **Teams** | | | |
| | ListTeams | List all available teams | 0.1.0 |
| | CreateTeam | Create a new team | 0.1.0 |
| | UpdateTeamName | Update the name of a team | 0.1.0 |
| | UpdateTeamMembers | Update the members of a team | 0.1.0 |
| **Users** | | | |
| | ListUsers | List all available users | 0.1.0 |
| | UpdateUser | Update an existing user | 0.1.0 |
| | GetSettings | Get the settings of the Portainer instance | 0.1.0 |
| **Docker** | | | |
| | DockerProxy | Proxy ANY Docker API requests | 0.2.0 |
