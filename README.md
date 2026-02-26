# Portainer MCP
[![Go Report Card](https://goreportcard.com/badge/github.com/portainer/portainer-mcp)](https://goreportcard.com/report/github.com/portainer/portainer-mcp)
![coverage](https://raw.githubusercontent.com/portainer/portainer-mcp/badges/.badges/main/coverage.svg)

Ever wished you could just ask Portainer what's going on?

Now you can! Portainer MCP connects your AI assistant directly to your Portainer environments. Manage Portainer resources such as users and environments, or dive deeper by executing any Docker or Kubernetes command directly through the AI.

![portainer-mcp-demo](https://downloads.portainer.io/mcp-demo5.gif)

## Overview

Portainer MCP is a work in progress implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) for Portainer environments. This project aims to provide a standardized way to connect Portainer's container management capabilities with AI models and other services.

MCP (Model Context Protocol) is an open protocol that standardizes how applications provide context to LLMs (Large Language Models). Similar to how USB-C provides a standardized way to connect devices to peripherals, MCP provides a standardized way to connect AI models to different data sources and tools.

This implementation focuses on exposing Portainer environment data through the MCP protocol, allowing AI assistants and other tools to interact with your containerized infrastructure in a secure and standardized way.

> [!NOTE]
> This tool is designed to work with specific Portainer versions. If your Portainer version doesn't match the supported version, you can use the `--disable-version-check` flag to attempt connection anyway. See [Portainer Version Support](#portainer-version-support) for compatible versions and [Disable Version Check](#disable-version-check) for bypass instructions.

See the [Supported Capabilities](#supported-capabilities) sections for more details on compatibility and available features.

*Note: This project is currently under development.*

It is currently designed to work with a Portainer administrator API token.

## Installation

You can download pre-built binaries for Linux (amd64, arm64) and macOS (arm64) from the [**Latest Release Page**](https://github.com/portainer/portainer-mcp/releases/latest). Find the appropriate archive for your operating system and architecture under the "Assets" section.

**Download the archive:**
You can usually download this directly from the release page. Alternatively, you can use `curl`. Here's an example for macOS (ARM64) version `v0.2.0`:

```bash
# Example for macOS (ARM64) - adjust version and architecture as needed
curl -Lo portainer-mcp-v0.2.0-darwin-arm64.tar.gz https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-darwin-arm64.tar.gz
```

(Linux AMD64 binaries are also available on the release page.)

**(Optional but recommended) Verify the checksum:**
First, download the corresponding `.md5` checksum file from the release page.
Example for macOS (ARM64) `v0.2.0`:

```bash
# Download the checksum file (adjust version/arch)
curl -Lo portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5 https://github.com/portainer/portainer-mcp/releases/download/v0.2.0/portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5
# Now verify (output should match the content of the .md5 file)
if [ "$(md5 -q portainer-mcp-v0.2.0-darwin-arm64.tar.gz)" = "$(cat portainer-mcp-v0.2.0-darwin-arm64.tar.gz.md5)" ]; then echo "OK"; else echo "FAILED"; fi
```

(For Linux, you can use `md5sum -c <checksum_file_name>.md5`)
If the verification command outputs "OK", the file is intact.

**Extract the archive:**

```bash
# Adjust the filename based on the downloaded version/OS/architecture
tar -xzf portainer-mcp-v0.2.0-darwin-arm64.tar.gz
```

This will extract the `portainer-mcp` executable.

**Move the executable:**
Move the executable to a location in your `$PATH` (e.g., `/usr/local/bin`) or note its location for the configuration step below.

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
                "[TOKEN]",
                "-tools",
                "/tmp/tools.yaml"
            ]
        }
    }
}
```

Replace `[IP]`, `[PORT]` and `[TOKEN]` with the IP, port and API access token associated with your Portainer instance.

> [!NOTE]
> By default, the tool looks for "tools.yaml" in the same directory as the binary. If the file does not exist, it will be created there with the default tool definitions. You may need to modify this path as described above, particularly when using AI assistants like Claude that have restricted write permissions to the working directory.

## Disable Version Check

By default, the application validates that your Portainer server version matches the supported version and will fail to start if there's a mismatch. If you have a Portainer server version that doesn't have a corresponding Portainer MCP version available, you can disable this version check to attempt connection anyway.

To disable the version check, add the `-disable-version-check` flag to your command arguments:

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
                "-disable-version-check"
            ]
        }
    }
}
```

> [!WARNING]
> Disabling the version check may result in unexpected behavior or API incompatibilities if your Portainer server version differs significantly from the supported version. The tool may work partially or not at all with unsupported versions.

When using this flag:
- The application will skip Portainer server version validation at startup
- Some features may not work correctly due to API differences between versions
- Newer Portainer versions may have API changes that cause errors
- Older Portainer versions may be missing APIs that the tool expects

This flag is useful when:
- You're running a newer Portainer version that doesn't have MCP support yet
- You're running an older Portainer version and want to try the tool anyway

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

The default tools file is available for reference at `internal/tooldef/tools.yaml` in the source code. You can modify the descriptions of the tools and their parameters to alter how AI models interpret and decide to use them. You can even decide to remove some tools if you don't wish to use them.

> [!WARNING]
> Do not change the tool names or parameter definitions (other than descriptions), as this will prevent the tools from being properly registered and functioning correctly.

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
- The Docker and Kubernetes proxy tools are available but restricted to GET requests only

# Portainer Version Support

This tool is pinned to support a specific version of Portainer. The application will validate the Portainer server version at startup and fail if it doesn't match the required version.

| Portainer MCP Version  | Supported Portainer Version |
|--------------|----------------------------|
| 0.1.0 | 2.28.1 |
| 0.2.0 | 2.28.1 |
| 0.3.0 | 2.28.1 |
| 0.4.0 | 2.29.2 |
| 0.4.1 | 2.29.2 |
| 0.5.0 | 2.30.0 |
| 0.6.0 | 2.31.2 |
| 0.7.0 | 2.31.2 |

> [!NOTE]
> If you need to connect to an unsupported Portainer version, you can use the `-disable-version-check` flag to bypass version validation. See the [Disable Version Check](#disable-version-check) section for more details and important warnings about using this feature.

# Supported Capabilities

The following table lists the currently (latest version) supported operations through MCP tools:

> [!NOTE]
> **Edge Stacks vs Local Stacks**: The original Portainer MCP only supports Edge Stacks (distributed via Edge Groups). Local Stack tools add support for regular standalone Docker Compose stacks deployed directly on environments — the most common stack type in non-Edge setups. Local Stack tools use raw HTTP requests to the Portainer REST API since the official SDK (`client-api-go`) does not expose regular stack endpoints.

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
| | DockerProxy | Proxy ANY Docker API requests (GET only in read-only mode) | 0.2.0 |
| **Local Stacks (Standalone Docker Compose)** | | | |
| | ListLocalStacks | List all local (non-edge) stacks deployed on environments | fork |
| | GetLocalStackFile | Get the docker-compose file content for a local stack | fork |
| | CreateLocalStack | Create a new local standalone Docker Compose stack | fork |
| | UpdateLocalStack | Update an existing local stack with new compose file | fork |
| | StartLocalStack | Start a stopped local stack | fork |
| | StopLocalStack | Stop a running local stack | fork |
| | DeleteLocalStack | Delete a local stack permanently | fork |
| **Kubernetes** | | | |
| | KubernetesProxy | Proxy ANY Kubernetes API requests (GET only in read-only mode) | 0.3.0 |
| | getKubernetesResourceStripped | Proxy GET Kubernetes API requests and automatically strip verbose metadata fields | 0.6.0 |

# Development

## Code Statistics

The repository includes a helper script `cloc.sh` to calculate lines of code and other metrics for the Go source files using the `cloc` tool. You might need to install `cloc` first (e.g., `sudo apt install cloc` or `brew install cloc`).

Run the script from the repository root to see the default summary output:

```bash
./cloc.sh
```

Refer to the comment header within the `cloc.sh` script for details on available flags to retrieve specific metrics.

## Token Counting

To get an estimate of how many tokens your current tool definitions consume in prompts, you can use the provided Go program and shell script to query the Anthropic API's token counting endpoint.

**1. Generate the Tools JSON:**

First, use the `token-count` Go program to convert your YAML tool definitions into the JSON format required by the Anthropic API. Run this from the repository root:

```bash
# Replace internal/tooldef/tools.yaml with your YAML file if different
# Replace .tmp/tools.json with your desired output path
go run ./cmd/token-count -input internal/tooldef/tools.yaml -output .tmp/tools.json
```

This command reads the tool definitions from the specified input YAML file and writes a JSON array of tools (containing `name`, `description`, and `input_schema`) to the specified output file.

**2. Query the Anthropic API:**

Next, use the `token.sh` script to send these tool definitions along with a sample message to the Anthropic API. You will need an Anthropic API key for this step.

```bash
# Ensure you have jq installed
# Replace sk-ant-xxxxxxxx with your actual Anthropic API key
# Replace .tmp/tools.json with the path to the file generated in step 1
./token.sh -k sk-ant-xxxxxxxx -i .tmp/tools.json
```

The script will output the JSON response from the Anthropic API, which includes the estimated token count for the provided tools and sample message under the `usage.input_tokens` field.

This process helps in understanding the token cost associated with the toolset provided to the language model.
