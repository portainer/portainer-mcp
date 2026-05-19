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

Architecture overview: [`docs/architecture.md`](docs/architecture.md).

## Getting started

The server is distributed on PyPI as `mcp-portainer`. MCP clients launch it as
a subprocess via [`uvx`](https://docs.astral.sh/uv/) (the `uv` equivalent of
`npx`), which downloads and runs the package in an ephemeral environment — no
clone, no install step.

Set `PORTAINER_URL` and `PORTAINER_API_KEY` (generate the key in Portainer
under **My Account → Access tokens**). See [Configuration](#configuration) for
the optional knobs.

### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "portainer": {
      "command": "uvx",
      "args": ["mcp-portainer@2.41.0"],
      "env": {
        "PORTAINER_URL": "https://portainer.example.com",
        "PORTAINER_API_KEY": "ptr_xxxxxxxxxxxxxxxx"
      }
    }
  }
}
```

Restart Claude Desktop. Logs land in `~/Library/Logs/Claude/mcp*.log` (macOS)
or `%APPDATA%\Claude\logs\` (Windows).

### Claude Code

```bash
claude mcp add portainer \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -- uvx mcp-portainer@2.41.0
```

### Other MCP clients

The JSON snippet above works for any MCP client that supports stdio servers
(Cursor, Continue, Cline, etc.) — the `mcpServers` shape is the same. For
clients with a different config format (TOML, YAML), use the same
`command`/`args`/`env` semantics.

### Pin policy

The example pins exactly to `2.41.0`. To pick up MCP-only patch fixes against
the same Portainer minor automatically, use `uv`'s `--from` form with a PEP
508 specifier:

```json
"args": ["--from", "mcp-portainer~=2.41.0", "mcp-portainer"]
```

See [Version compatibility](#version-compatibility) for the policy.

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
| `PORTAINER_MCP_LOG_LEVEL` | `INFO` | One of `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`. Logs go to stderr. |
| `PORTAINER_MCP_TRANSPORT` | `stdio` | `stdio` (default) or `http`. `http` binds a local server for dev / remote deployment. |
| `PORTAINER_MCP_HTTP_HOST` | `127.0.0.1` | Bind host when `PORTAINER_MCP_TRANSPORT=http`. |
| `PORTAINER_MCP_HTTP_PORT` | `8000` | Bind port when `PORTAINER_MCP_TRANSPORT=http`. |

Advanced profile setup — per-profile tag lists, orphan tags, read-only
semantics — see [`docs/profiles.md`](docs/profiles.md).
