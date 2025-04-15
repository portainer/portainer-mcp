# Portainer MCP

Ever wished you could just ask Portainer what's going on?

![portainer-mcp-demo](https://downloads.portainer.io/mcp-demo3.gif)

## Overview

Portainer MCP is a work in progress implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) for Portainer environments. This project aims to provide a standardized way to connect Portainer's container management capabilities with AI models and other services.

MCP (Model Context Protocol) is an open protocol that standardizes how applications provide context to LLMs (Large Language Models). Similar to how USB-C provides a standardized way to connect devices to peripherals, MCP provides a standardized way to connect AI models to different data sources and tools.

This implementation focuses on exposing Portainer environment data through the MCP protocol, allowing AI assistants and other tools to interact with your containerized infrastructure in a secure and standardized way.

*Note: This project is currently under development.*

It is currently designed to work with a Portainer administrator API token.

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

| MCP Version  | Supported Portainer Version |
|--------------|----------------------------|
| 0.1.0        | 2.28.1                     |

# Supported Capabilities

The following table lists the currently (latest version) supported operations through MCP tools:

| Resource | Operation | Description |
|----------|-----------|-------------|
| **Environments** |
| | ListEnvironments | List all available environments |
| | UpdateEnvironmentTags | Update tags associated with an environment |
| | UpdateEnvironmentUserAccesses | Update user access policies for an environment |
| | UpdateEnvironmentTeamAccesses | Update team access policies for an environment |
| **Environment Groups (Edge Groups)** |
| | ListEnvironmentGroups | List all available environment groups |
| | CreateEnvironmentGroup | Create a new environment group |
| | UpdateEnvironmentGroupName | Update the name of an environment group |
| | UpdateEnvironmentGroupEnvironments | Update environments associated with a group |
| | UpdateEnvironmentGroupTags | Update tags associated with a group |
| **Access Groups (Endpoint Groups)** |
| | ListAccessGroups | List all available access groups |
| | CreateAccessGroup | Create a new access group |
| | UpdateAccessGroupName | Update the name of an access group |
| | UpdateAccessGroupUserAccesses | Update user accesses for an access group |
| | UpdateAccessGroupTeamAccesses | Update team accesses for an access group |
| | AddEnvironmentToAccessGroup | Add an environment to an access group |
| | RemoveEnvironmentFromAccessGroup | Remove an environment from an access group |
| **Stacks (Edge Stacks)** |
| | ListStacks | List all available stacks |
| | GetStackFile | Get the compose file for a specific stack |
| | CreateStack | Create a new Docker stack |
| | UpdateStack | Update an existing Docker stack |
| **Tags** |
| | ListEnvironmentTags | List all available environment tags |
| | CreateEnvironmentTag | Create a new environment tag |
| **Teams** |
| | ListTeams | List all available teams |
| | CreateTeam | Create a new team |
| | UpdateTeamName | Update the name of a team |
| | UpdateTeamMembers | Update the members of a team |
| **Users** |
| | ListUsers | List all available users |
| | UpdateUser | Update an existing user |
| | GetSettings | Get the settings of the Portainer instance |
| **Docker** |
| | DockerProxy | Proxy ANY Docker API requests |
