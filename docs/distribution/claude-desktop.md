# Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "portainer": {
      "command": "uvx",
      "args": ["--from", "mcp-portainer~=2.41.0", "mcp-portainer"],
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
