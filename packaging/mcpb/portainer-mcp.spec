# PyInstaller spec for the portainer-mcp mcpb binary bundle.
# Build: uv run --with pyinstaller pyinstaller packaging/mcpb/portainer-mcp.spec
#
# Produces a single self-contained executable so the .mcpb needs no Python /
# uv / Node on the user's machine (server.type="binary").

import os

from PyInstaller.utils.hooks import (
    collect_data_files,
    collect_dynamic_libs,
    collect_submodules,
    copy_metadata,
)

# The bundled OpenAPI spec is loaded via importlib.resources.files("portainer_mcp")
# — it must be collected as package data or the frozen app raises
# FileNotFoundError at startup.
datas = collect_data_files("portainer_mcp")

# The hygiene skill ships as MCP server instructions. In an editable checkout
# it isn't under the package (hatch force-include only fires on a wheel build),
# so add it explicitly from its source-of-truth location into the same data dir
# the server reads (portainer_mcp/data/SKILL.md).
datas += [(os.path.join(SPECPATH, "..", "..", "skills", "portainer-mcp-hygiene",
           "SKILL.md"), "portainer_mcp/data")]
binaries = []
hiddenimports = []

# FastMCP + the MCP SDK + their ASGI stack use dynamic imports PyInstaller's
# static analysis misses. mcp.cli is filtered out: it imports the optional
# `typer` CLI extra and, when absent, calls sys.exit(1) at import time — which
# raises SystemExit and kills the submodule walk (on_error only traps
# ImportError). The server never touches mcp.cli, so dropping it is safe.
def _keep(name):
    return not name.startswith("mcp.cli")


for pkg in ("fastmcp", "mcp", "uvicorn", "starlette"):
    datas += collect_data_files(pkg)
    binaries += collect_dynamic_libs(pkg)
    hiddenimports += collect_submodules(pkg, filter=_keep, on_error="ignore")

# fastmcp/__init__.py reads its own version via importlib.metadata.version()
# (trying "fastmcp-slim" then "fastmcp"), which needs the .dist-info metadata
# bundled. recursive=True also pulls dependency metadata that other libs read
# the same way. Without this the frozen app dies at import with
# PackageNotFoundError.
for dist in ("fastmcp", "fastmcp-slim"):
    datas += copy_metadata(dist, recursive=True)

a = Analysis(
    # SPECPATH-anchored so the build works from any CWD (e.g. repo root in CI).
    [os.path.join(SPECPATH, "entry.py")],
    pathex=[],
    binaries=binaries,
    datas=datas,
    hiddenimports=hiddenimports,
    hookspath=[],
    runtime_hooks=[],
    excludes=[],
    noarchive=False,
)

pyz = PYZ(a.pure)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name="portainer-mcp",
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=False,
    console=True,
)
