# Claude Desktop

There are two ways to install: a **`.mcpb` bundle** (no Python/uv required, set
up through the Claude Desktop UI) or the **manual JSON config** (requires `uv`
on `PATH`).

## Option A — `.mcpb` bundle

Each release attaches a self-contained `.mcpb` per platform to its
[GitHub Release](https://github.com/portainer/portainer-mcp/releases). Download
the one for your platform and double-click it (or Claude Desktop > Settings >
Extensions > install), then fill in your Portainer URL and API key — the key is
stored in your OS keychain.

- `portainer-mcp-X.Y.Z-darwin-arm64.mcpb` — macOS, Apple Silicon
- `portainer-mcp-X.Y.Z-win32-x64.mcpb` — Windows, x64
- `portainer-mcp-X.Y.Z-linux-x64.mcpb` — Linux, x64

> **The bundles are not yet code-signed**, so the OS flags the downloaded
> binary on first run:
> - **macOS:** "portainer-mcp can't be opened because the developer cannot be
>   verified." Clear the quarantine flag once, then install:
>   ```bash
>   xattr -d com.apple.quarantine ~/Downloads/portainer-mcp-*.mcpb
>   ```
> - **Windows:** SmartScreen shows "Windows protected your PC" — click
>   **More info → Run anyway**.

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
