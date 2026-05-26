# Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

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

## Recommended: install the hygiene skill

This repo ships a skill
([`portainer-mcp-hygiene`](https://github.com/portainer/portainer-mcp/blob/main/skills/portainer-mcp-hygiene/SKILL.md))
that helps the model query the MCP efficiently and keep responses within
context. Install user-wide, pinned to the same tag as the server:

```bash
mkdir -p ~/.claude/skills/portainer-mcp-hygiene && \
  curl -fsSL https://raw.githubusercontent.com/portainer/portainer-mcp/2.42.0/skills/portainer-mcp-hygiene/SKILL.md \
  -o ~/.claude/skills/portainer-mcp-hygiene/SKILL.md
```

Re-run on each server upgrade so the skill stays in sync.
