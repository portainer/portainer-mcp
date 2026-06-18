"""Frozen entry point for the mcpb binary bundle.

PyInstaller needs a real script as the analysis root; the console-script
entry (`portainer_mcp.server:main`) isn't a file it can freeze directly.
"""

from portainer_mcp.server import main

if __name__ == "__main__":
    main()
