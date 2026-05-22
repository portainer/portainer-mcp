---
name: portainer-mcp-release
description: How to cut a portainer-mcp release that bumps the embedded Portainer OpenAPI spec to a new Portainer minor (e.g. 2.41.x → 2.42.x). Walks through finding the upstream patch target, regenerating the spec, recounting ops/tags per profile, refreshing the orphan-tag inventory, bumping version pins across the README and distribution docs, promoting CHANGELOG entries, committing, pushing, merging to main, and tagging — in that order. Trigger this skill whenever the user mentions cutting a release, bumping the Portainer version target, supporting a new Portainer minor, regenerating the spec, running `make specs` for a new minor, or any phrasing like "let's update for Portainer X.Y" — the spec regen is only the first step of a multi-file release and the model should consult this skill before touching anything.
---

# portainer-mcp release (spec bump)

This skill covers releases where the MCP minor moves with a new Portainer minor (e.g. `2.41.0 → 2.42.0`). Per [`docs/versioning.md`](../../../docs/versioning.md), the minor only moves when the embedded spec moves — patch-only releases (shaping fixes, dep bumps, profile tweaks against the same Portainer minor) follow a simpler flow that this skill does not cover. If unsure which kind you're cutting, the test is: did `src/portainer_mcp/data/portainer-patched.yaml` change? Spec bump → use this skill. Otherwise → patch release.

The release pipeline itself (PyPI publish via OIDC on tag push) is documented in [`docs/release.md`](../../../docs/release.md); this skill is the spec-bump-specific work that has to land *before* the tag.

## Why a skill for this

Spec bumps are infrequent (one per Portainer minor — quarterly-ish) and the work spans nine files. Most of those touches are mechanical version-pin updates, but two are easy to half-do:

- **Op/tag deltas in `docs/profiles.md`** — the orphan table and default-coverage numbers are derived from the spec. Upstream often adds/removes/renames tags between minors. Stale numbers here mean users get the wrong picture of what the default profiles cover.
- **CHANGELOG entry** — a one-liner that says "bumped spec" is much less useful than naming the deltas (operation count change, dropped tags). Skim of the changelog is how users decide whether the upgrade matters to them.

The script in `scripts/spec_deltas.py` automates the counting so neither slips.

## The release in one breath

1. Find the latest 2.X patch upstream.
2. Branch off `main` as `X.Y` if you aren't already.
3. `make specs VERSION=X.Y.Z`.
4. `uv run pytest` and boot the server end-to-end.
5. Run `spec_deltas.py`, capture the new numbers.
6. Update nine files (see Step 6 checklist).
7. Commit as `Release X.Y.0`, push branch.
8. Open PR, merge to `main` (merge or rebase — not squash).
9. From `main`, `git tag X.Y.0 && git push origin X.Y.0`.
10. Watch the release workflow finish, verify on PyPI.

## Step-by-step

### 1. Find the latest 2.X patch upstream

Upstream is `portainer/portainer-api-docs`, with per-version YAML at `versions/ee/X.Y.Z.yaml`. We target the **latest patch** of the minor we're moving to, not the `.0`.

Preferred — `gh` (no clone):

```bash
gh api /repos/portainer/portainer-api-docs/contents/versions/ee \
  --jq '.[].name' | grep -E '^2\.42\.[0-9]+\.yaml$' | sort -V
```

Fallback — sparse clone (if `gh` isn't authed):

```bash
git clone --depth=1 --filter=blob:none --no-checkout \
  git@github.com:portainer/portainer-api-docs.git /tmp/portainer-api-docs
git -C /tmp/portainer-api-docs ls-tree -r HEAD versions/ee/ \
  | grep -oE 'versions/ee/2\.42\.[0-9]+\.yaml' | sort -V
```

Pick the highest patch. If only `.0` exists, that's your target.

### 2. Branch off `main`

If there isn't already an `X.Y` branch, create one:

```bash
git checkout main && git pull
git checkout -b X.Y    # e.g. 2.42
```

### 3. Regenerate the spec

```bash
make specs VERSION=X.Y.Z
```

This sparse-clones `portainer-api-docs` into `spec/upstream/` and runs `spec/patch_spec.py` against the targeted YAML, writing the result to `src/portainer_mcp/data/portainer-patched.yaml`.

**If the patcher fails or `uv run pytest` shows the spec is broken**, the upstream YAML has shipped new defects — extend `EXCLUDED_OPERATION_IDS`, `EXCLUDED_PATH_PREFIXES`, or `ENUM_STRIPS` in [`spec/patch_spec.py`](../../../spec/patch_spec.py) rather than hand-editing `portainer-patched.yaml`. The patched file is regenerated every release; manual edits will be lost.

### 4. Sanity check: tests + boot

```bash
uv run pytest
```

Then boot the server end-to-end. This exercises `FastMCP.from_openapi` against the real spec and the startup `select` invariant — which is the only check that catches "spec parses but FastMCP can't build tools from it":

```bash
PORTAINER_URL=http://localhost PORTAINER_API_KEY=dummy uv run python -c \
  'from portainer_mcp.server import build_server; build_server()'
```

You want to see `select` arg present on all N tools` in the log output. If `build_server()` raises, the spec has structural issues that `pytest` couldn't catch — likely needing another `patch_spec.py` exclusion.

### 5. Compute the deltas

```bash
uv run python .claude/skills/portainer-mcp-release/scripts/spec_deltas.py
```

Output is in two parts: per-profile coverage numbers (use these in `docs/profiles.md`'s "Default coverage" paragraph) and the orphan-tag markdown table (paste-ready). Compare against the previous release's `docs/profiles.md` to spot:

- **Dropped tags**: any orphan from last release that's missing now means upstream removed that tag — drop the row, drop the mention anywhere else.
- **New tags**: any orphan that wasn't in last release means upstream added one — decide if it belongs in a profile or stays orphan. New orphans need a one-line description in the table; lift it from a quick scan of operations carrying that tag in the spec.
- **Op count shifts**: per-tag deltas tell you which areas got expanded upstream. Worth a line in the CHANGELOG if a profile grew meaningfully.

### 6. Update files

Nine files to touch. Listed in roughly the order it's natural to edit them:

| File | Change |
|---|---|
| `src/portainer_mcp/data/portainer-patched.yaml` | Already regenerated in step 3 — no manual edit. |
| `pyproject.toml` | `version = "X.Y.0"`. |
| `uv.lock` | `uv lock` — refreshes the self-entry to match `pyproject`. |
| `docs/profiles.md` | New per-profile and union numbers in the "Default coverage" paragraph; new orphan table (paste from the script); update the "350+/400+ operations across 40+ tags" lead if it crossed a round number. |
| `README.md` | `~=X.Y.0` pin in the `claude mcp add` snippet; raw URL tag in the skill curl; new row in the compat matrix (keep prior rows — the matrix is cumulative). |
| `docs/distribution/claude-desktop.md` | Same pin + skill URL bump as README. Check for additional client docs as `docs/distribution/` grows. |
| `Makefile` | Update the example `VERSION=` in the `specs` comment/help text. |
| `docs/versioning.md` | If the doc uses the old minor in prose examples (e.g. "2.41.x ↔ 2.41.x"), retarget to the new minor. |
| `CHANGELOG.md` | Move the `[Unreleased]` block under a new `[X.Y.0] — YYYY-MM-DD` heading; leave a fresh empty `[Unreleased]` on top. The spec bump should be the lead Changed entry, naming old → new spec version and the headline deltas (op count change, any dropped/added tags). |

A clean CHANGELOG spec-bump entry, modeled on prior releases:

```markdown
## [X.Y.0] — YYYY-MM-DD

Targets Portainer X.Y.x.

### Changed

- **Embedded spec bumped to Portainer EE X.Y.Z** (was A.B.C). Default
  `BASE,DOCKER,KUBERNETES` profile coverage grows from ~P1 to ~P2
  operations; the five-profile union grows from ~U1 to ~U2. Upstream
  removed the `<tag>` tag (N operations); dropped from the orphan tag
  list in [`docs/profiles.md`](docs/profiles.md).
```

Adapt to what actually happened — if no tags were dropped, drop that sentence; if a profile grew significantly, call that out specifically.

### 7. Commit and push

```bash
git add CHANGELOG.md Makefile README.md docs/distribution/ docs/profiles.md \
        docs/versioning.md pyproject.toml src/portainer_mcp/data/portainer-patched.yaml uv.lock
git commit -m "Release X.Y.0

Bump embedded Portainer EE spec to X.Y.Z. <One-liner naming the headline
deltas: op-count change, dropped tags, anything notable.>"
git push -u origin X.Y
```

Stage files explicitly (not `git add -A`) — the spec regeneration may leave artifacts in `spec/upstream/` that should not be committed.

### 8. Open the PR, merge to `main`

The PR description can be terse — the CHANGELOG entry is the source of truth for what shipped. The reviewer is checking: did the spec regenerate cleanly, do the recounted numbers look reasonable, did all pin references get updated.

**Prefer merge or rebase, not squash.** A squash collapses the spec-bump commit and any subsequent fix-up commits into one, which loses the clean "this commit is the spec bump" anchor that future spec diffs benefit from.

### 9. Tag on `main`

```bash
git checkout main && git pull
git tag X.Y.0
git push origin X.Y.0
```

**Tag on `main`, not on the feature branch.** The release workflow fires on tag push regardless of branch, but the tag should anchor on canonical history so it stays reachable from `main` and `git describe` works. Tagging on a branch that later gets squash-merged orphans the tag.

### 10. Watch the release workflow

```bash
gh run watch
# or: gh run list --workflow=release.yml --limit 3
```

The workflow verifies tag == `pyproject.version`, runs tests, builds the wheel, publishes to PyPI via OIDC Trusted Publishing. On success, the release is live at `https://pypi.org/project/mcp-portainer/X.Y.0/`.

## Gotchas

- **Tag/version mismatch fails fast.** If you forgot to bump `pyproject.toml` or tagged the wrong commit, the workflow refuses to publish. Delete the tag locally and remotely (`git tag -d X.Y.0 && git push --delete origin X.Y.0`), fix the version, retag. See [`docs/release.md`](../../../docs/release.md) for the full recovery flow.
- **PyPI doesn't allow re-uploading a version.** If a published version is broken, bump the patch — there's no "fix and reupload" path. (Yanking is possible but `X.Y.0` still consumes that version forever.)
- **TestPyPI dry-run is optional but cheap.** [`docs/release.md`](../../../docs/release.md) describes the `release-test.yml` workflow you can fire manually before tagging the real release. Useful if you've touched anything that affects the wheel build (e.g. `hatch` config, package data inclusion).
- **`spec/upstream/` is checked into `.gitignore`** — don't worry about it being a dirty working tree after `make specs`, but verify it's not staged.
- **Don't hand-edit `portainer-patched.yaml`.** It's regenerated every release; edits are lost. All spec workarounds live in [`spec/patch_spec.py`](../../../spec/patch_spec.py).
