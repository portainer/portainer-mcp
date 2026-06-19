# Claude Desktop

There are two ways to install: a **`.mcpb` bundle** (no Python/uv required, set
up through the Claude Desktop UI) or the **manual JSON config** (requires `uv`
on `PATH`).

## Option A — `.mcpb` bundle (one-click install)

1. Fetch the self-contained `.mcpb` bundle for your platform from the [latest release](https://github.com/portainer/portainer-mcp/releases/latest)
2. Double-click to install
3. Enter your Portainer URL and API key.

## Option B — manual JSON config

Claude Desktop > Settings > Developer > Edit Config and choose one of the path below, alternatively you can edit that file directly with your preferred text editor.

* macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
* Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "portainer": {
      "command": "uvx",
      "args": ["--from", "mcp-portainer~=2.42.0", "mcp-portainer"],
      "env": {
        "PORTAINER_URL": "https://portainer.example.com",
        "PORTAINER_API_KEY": "ptr_xxxxxxxxxxxxxxxx"
      }
    }
  }
}
```

`uv` must be on `PATH` — see
[the uv install docs](https://docs.astral.sh/uv/getting-started/installation/).

Restart Claude Desktop after every config change. Logs land in
`~/Library/Logs/Claude/mcp*.log` (macOS) or `%APPDATA%\Claude\logs\`
(Windows).
