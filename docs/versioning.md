# Versioning policy

**Major+minor tracks the Portainer API; patch is the MCP server's to spend.**

- Tag format: `<portainer-major>.<portainer-minor>.<mcp-patch>`.
- Compatibility statement: **"MCP 2.41.x ↔ Portainer 2.41.x"** — minor
  granularity.
- Spec cadence: regenerate `spec/portainer-patched.yaml` against the
  **latest patch** of the targeted Portainer minor (e.g. `make specs
  VERSION=2.41.3` when 2.41.3 is current).
- MCP-internal iterations (shaping fixes, new profiles, transform changes,
  dep bumps, doc-only changes) burn the patch slot — e.g. `2.41.1` is an
  MCP-only release still targeting Portainer 2.41.x.

## Edge cases

1. **Upstream breaks in a patch (rare).** Ship an MCP patch and annotate
   the compat table at sub-minor granularity in a README footnote — e.g.
   "MCP 2.41.0–2.41.2 ↔ Portainer 2.41.0–2.41.4; 2.41.3+ ↔ 2.41.5+". One
   footnote per incident, no general scheme.
2. **New endpoint appears in a Portainer patch.** Wait for the next
   minor. The profile-based tool surface absorbs new operations only when
   we regenerate; partial-patch chasing isn't worth the churn.
3. **Which patch to generate against.** Always the latest patch of the
   targeted minor.
4. **MCP-only releases between Portainer minors.** Fully supported — tag
   the next patch (`2.41.N+1`). Use this slot for shaping, profile, and
   proxy changes that don't depend on a spec bump.

## Consumer ergonomics

```toml
# pyproject.toml — pin to the Portainer 2.41 minor, pick up MCP patches
"portainer-mcp ~= 2.41.0"
```

The PEP 440 compatible-release operator (`~=`) selects the latest
`2.41.x` automatically — consumers who want "track this Portainer minor"
pin loosely without naming a patch.

For Docker images, the equivalent is a floating minor tag (e.g.
`ghcr.io/portainer/mcp:2.41`); concrete distribution details are tracked
in [`production-readiness.md`](production-readiness.md) §5.

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
