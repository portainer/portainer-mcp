# Versioning policy

**Major+minor tracks the Portainer API; patch is the MCP server's to spend.**

- Tag format: `<portainer-major>.<portainer-minor>.<mcp-patch>`.
- Compatibility statement: **"MCP 2.43.x ↔ Portainer 2.43.x"** — minor
  granularity.
- Spec cadence: regenerate `src/portainer_mcp/data/portainer-patched.yaml` against the
  **latest patch** of the targeted Portainer minor (e.g. `make specs
  VERSION=2.43.0` when 2.43.0 is current).
- MCP-internal iterations (shaping fixes, new profiles, transform changes,
  dep bumps, doc-only changes) burn the patch slot — e.g. `2.43.1` is an
  MCP-only release still targeting Portainer 2.43.x.

## What does *not* bump the minor

- Adding a profile, renaming a profile, or shuffling tags between
  profiles — patch.
- Adding a `select` projection helper, changing the truncation hint,
  adjusting the default cap — patch.
- Pinning a stricter `fastmcp` range, dropping a Python version — patch
  (server-side; consumers picking up via `~=` keep working).
- Adding a new hand-written proxy tool (e.g. an additional `*_proxy`) —
  patch, since it widens capability without changing the Portainer
  target.

The minor only moves when the embedded spec moves.
