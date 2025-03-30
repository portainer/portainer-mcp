# Portainer MCP

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
            ]
        }
    }
}
```

Replace `[IP]`, `[PORT]` and `[TOKEN]` with the IP, port and API access token associated with your Portainer instance.

# Supported capabilities

The following table lists the currently supported operations through MCP tools:

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
| **Stacks** |
| | ListStacks | List all available stacks |
| | GetStackFile | Get the compose file for a specific stack |
| | CreateStack | Create a new stack |
| | UpdateStack | Update an existing stack |
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
| **Settings** |
| | GetSettings | Get the settings of the Portainer instance |