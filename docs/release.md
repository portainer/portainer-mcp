# Releasing

The release workflow at
[`../.github/workflows/release.yml`](../.github/workflows/release.yml) builds
the wheel and publishes to PyPI on every tag push matching `X.Y.Z`. Auth is
OIDC via PyPI Trusted Publishing — no API tokens or repo secrets.

## One-time setup

A Pending Publisher must exist on PyPI. Do once per distribution name:

- **pypi.org → Account → Publishing → Add pending publisher**
  - Repository: `portainer/portainer-mcp`
  - Workflow: `release.yml`
  - Environment: leave blank (any)

No GitHub secrets, no environments, no token rotation. If you ever rename the
distribution, register a new Pending Publisher under the new name.

## Cutting a release

1. Decide the version per [`versioning.md`](versioning.md). Spec bump → minor;
   MCP-only change → patch.
2. Bump `version` in [`../pyproject.toml`](../pyproject.toml).
3. `uv lock` to refresh [`../uv.lock`](../uv.lock).
4. Move the `[Unreleased]` block in [`../CHANGELOG.md`](../CHANGELOG.md) under
   a new `[X.Y.Z] — YYYY-MM-DD` heading; leave a fresh empty `[Unreleased]`
   block on top.
5. Commit: `Release X.Y.Z`.
6. Tag and push:
   ```bash
   git tag X.Y.Z
   git push origin X.Y.Z
   ```
7. The workflow verifies the tag matches `pyproject.version`, runs tests,
   builds, and publishes. Watch it under **Actions** on GitHub.
8. Once green, the new version is live at
   `https://pypi.org/project/mcp-portainer/X.Y.Z/`.

## Recovery

- **Tag/version mismatch:** workflow fails fast. Delete the tag locally and
  remotely (`git tag -d X.Y.Z && git push --delete origin X.Y.Z`), fix the
  version, retag.
- **PyPI rejects upload (version exists):** PyPI doesn't allow re-uploading
  the same version, even after a yank. Bump the patch and retag.
- **Trusted Publishing OIDC failure:** the Pending Publisher's repo /
  workflow values must match exactly (case-sensitive).
