# Releasing

The release workflow at
[`../.github/workflows/release.yml`](../.github/workflows/release.yml) builds
the wheel and publishes to PyPI on every tag push matching `X.Y.Z`. Auth is
OIDC via PyPI Trusted Publishing — no API tokens or repo secrets.

## One-time setup

A Pending Publisher must exist on PyPI **and** TestPyPI. Do once per
distribution name:

- **pypi.org → Account → Publishing → Add pending publisher**
  - Repository: `portainer/portainer-mcp`
  - Workflow: `release.yml`
  - Environment: leave blank (any)
- **test.pypi.org → Account → Publishing → Add pending publisher**
  - Repository: `portainer/portainer-mcp`
  - Workflow: `release-test.yml`
  - Environment: leave blank (any)

No GitHub secrets, no environments, no token rotation. If you ever rename the
distribution, register new Pending Publishers under the new name.

### Docker Hub (image publish)

The container image at `docker.io/portainer/portainer-mcp` ships from
[`release-docker.yml`](../.github/workflows/release-docker.yml) on the same
`X.Y.Z` tag trigger. Docker Hub doesn't support OIDC the way PyPI does, so this
one uses a scoped access token in repo secrets:

- **GitHub → Settings → Secrets and variables → Actions** — add:
  - `DOCKERHUB_USERNAME` — the bot account that owns the repo.
  - `DOCKERHUB_TOKEN` — a Docker Hub access token scoped **push-only** to
    `portainer/portainer-mcp` (Docker Hub → Account Settings → Personal access
    tokens). Rotate on personnel change.

Image tag scheme: `:X.Y.Z` and `:X.Y` per release, multi-arch
`linux/amd64,linux/arm64`. No `:latest`. See
[`docker.md`](docker.md).

### GitHub Release (`.mcpb` bundles)

[`release-mcpb.yml`](../.github/workflows/release-mcpb.yml) builds the Claude
Desktop one-click bundles on the same `X.Y.Z` tag trigger and attaches them to
the **GitHub Release** for the tag (creating the Release if it doesn't exist
yet — PyPI and Docker Hub don't need one, so this workflow owns it).

It's a runner matrix, not multi-arch-in-one-job: PyInstaller freezes the
running interpreter and can't cross-compile, so each platform builds natively.
Current targets:

- `darwin-arm64` (macOS, `macos-14`) — Apple Silicon only for now.
- `win32-x64` (`windows-latest`)
- `linux-x64` (`ubuntu-latest`)

Each leg runs the same `tag == pyproject.version` gate and `pytest` as the PyPI
workflow, then calls [`packaging/mcpb/build.sh`](../packaging/mcpb/build.sh),
which freezes the binary (the `mcpb` dependency group pins PyInstaller), stamps
the tag into `manifest.json` (the committed file carries a `0.0.0`
placeholder — nothing to hand-bump at release), and packs
`portainer-mcp-X.Y.Z-<target>.mcpb`. No repo secrets: it uses the built-in
`GITHUB_TOKEN` with `contents: write`.

The workflow is deliberately off the PyPI/Docker critical path — a PyInstaller
failure on one platform won't block the wheel publish.

**The bundles are unsigned.** macOS Gatekeeper and Windows SmartScreen flag
downloaded unsigned binaries, so the install isn't yet one-click from a
download — the workaround is in
[`distribution/claude-desktop.md`](distribution/claude-desktop.md). Code
signing (Apple Developer ID notarization + Windows Authenticode) and a single
universal `.mcpb` are tracked follow-ups, not in this workflow.

To iterate locally, run `packaging/mcpb/build.sh <target> <version>` on a
machine of that platform (it can't cross-build).

## Dry run on TestPyPI

Before tagging a real release, do a dry run against TestPyPI to confirm the
build and OIDC publish path work end-to-end:

1. Bump `version` in [`../pyproject.toml`](../pyproject.toml) and commit.
2. **GitHub → Actions → Release (TestPyPI) → Run workflow** on the branch
   carrying the bump.
3. The workflow ([`release-test.yml`](../.github/workflows/release-test.yml))
   runs tests, builds, and publishes to TestPyPI.
4. Verify at `https://test.pypi.org/project/mcp-portainer/X.Y.Z/`.

TestPyPI doesn't allow re-uploading the same version (separate from PyPI's
copy of the rule). If the dry run finds a problem, fix it and bump to
`X.Y.Z.post1` for the next TestPyPI attempt — PyPI itself stays free to
receive plain `X.Y.Z`.

## Cutting a release

1. Decide the version per [`versioning.md`](versioning.md).
2. Bump `version` in [`../pyproject.toml`](../pyproject.toml).
3. `uv lock` to refresh [`../uv.lock`](../uv.lock).
4. Move the `[Unreleased]` block in [`../CHANGELOG.md`](../CHANGELOG.md) under
   a new `[X.Y.Z] — YYYY-MM-DD` heading; leave a fresh empty `[Unreleased]`
   block on top.
5. Bump the `Skill version:` footer in
   [`../skills/portainer-mcp-hygiene/SKILL.md`](../skills/portainer-mcp-hygiene/SKILL.md)
   to the new `X.Y.Z`. (The guide is bundled and served via `get_guidance`, so
   there are no manual-install snippets to re-pin.)
6. Commit: `Release X.Y.Z`.
7. Tag and push:
   ```bash
   git tag X.Y.Z
   git push origin X.Y.Z
   ```
8. The workflow verifies the tag matches `pyproject.version`, runs tests,
   builds, and publishes.
9. Once green, the new version is live at
   `https://pypi.org/project/mcp-portainer/X.Y.Z/`.

## Recovery

- **Tag/version mismatch:** workflow fails fast. Delete the tag locally and
  remotely (`git tag -d X.Y.Z && git push --delete origin X.Y.Z`), fix the
  version, retag.
- **PyPI rejects upload (version exists):** PyPI doesn't allow re-uploading
  the same version, even after a yank. Bump the patch and retag.
- **Trusted Publishing OIDC failure:** the Pending Publisher's repo /
  workflow values must match exactly (case-sensitive).
