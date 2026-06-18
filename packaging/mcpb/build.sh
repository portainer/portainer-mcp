#!/usr/bin/env bash
# Build a single-platform .mcpb bundle (server.type="binary").
#
#   packaging/mcpb/build.sh <target> <version>
#     target  — <os>-<arch> label, e.g. darwin-arm64 | win32-x64 | linux-x64.
#               Anything containing "win32" gets the .exe treatment.
#     version — stamped into the bundled manifest (the committed manifest.json
#               carries a 0.0.0 placeholder; this is the single source of truth).
#
# Output: dist/mcpb/portainer-mcp-<version>-<target>.mcpb
#
# Run with `shell: bash` on every OS in CI — git-bash covers windows-latest.
# PyInstaller can't cross-compile, so <target> must match the host this runs on.
set -euo pipefail

TARGET="${1:?usage: build.sh <target> <version>}"
VERSION="${2:?usage: build.sh <target> <version>}"

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

OUT="dist/mcpb"
BIN="$OUT/bin"
STAGE="$OUT/stage-$TARGET"
PKG="$OUT/portainer-mcp-$VERSION-$TARGET.mcpb"

rm -rf "$BIN" "$STAGE"
mkdir -p "$BIN" "$STAGE/server"

case "$TARGET" in
  *win32*) EXE=".exe" ;;
  *)       EXE="" ;;
esac

# 1. Freeze the standalone executable. The mcpb group pins pyinstaller.
uv run --group mcpb pyinstaller packaging/mcpb/portainer-mcp.spec \
  --distpath "$BIN" --workpath "$OUT/build" -y

# 2. Stage the binary + a version-stamped, platform-corrected manifest.
cp "$BIN/portainer-mcp$EXE" "$STAGE/server/portainer-mcp$EXE"

STAGE="$STAGE" VERSION="$VERSION" EXE="$EXE" uv run python - <<'PY'
import json
import os
import pathlib

manifest = json.loads(pathlib.Path("packaging/mcpb/manifest.json").read_text())
manifest["version"] = os.environ["VERSION"]

# Claude Desktop runs mcp_config.command verbatim; point it at the real
# filename so Windows doesn't rely on implicit .exe resolution.
exe = os.environ["EXE"]
if exe:
    manifest["server"]["entry_point"] += exe
    manifest["server"]["mcp_config"]["command"] += exe

out = pathlib.Path(os.environ["STAGE"]) / "manifest.json"
out.write_text(json.dumps(manifest, indent=2) + "\n")
print(f"stamped manifest: version={manifest['version']} "
      f"command={manifest['server']['mcp_config']['command']}")
PY

# 3. Pack. Pinned for reproducibility — bump deliberately.
npx -y @anthropic-ai/mcpb@2.1.2 pack "$STAGE" "$PKG"

echo "built $PKG"
