"""Print Portainer MCP spec totals and per-profile coverage.

Run AFTER `make specs VERSION=X.Y.Z` to get the numbers you'll paste into
`docs/profiles.md` for the release. Compare against the prior numbers
(check `git diff docs/profiles.md` from the previous release commit) to
see which counts moved and which tags appeared or disappeared upstream.

Usage:
    uv run python .claude/skills/portainer-mcp-release/scripts/spec_deltas.py
"""

from __future__ import annotations

from collections import Counter
from importlib.resources import files

import yaml

from portainer_mcp.profiles import TAG_PROFILES


def main() -> None:
    spec = yaml.safe_load(
        files("portainer_mcp.data").joinpath("portainer-patched.yaml").read_text()
    )

    tag_ops: Counter[str] = Counter()
    all_tags: set[str] = set()
    total_ops = 0
    for methods in spec.get("paths", {}).values():
        for op in methods.values():
            if not isinstance(op, dict):
                continue
            total_ops += 1
            for t in op.get("tags") or []:
                tag_ops[t] += 1
                all_tags.add(t)

    profile_tags = {name: set(tags) for name, tags in TAG_PROFILES.items()}
    default_union = (
        profile_tags["BASE"] | profile_tags["DOCKER"] | profile_tags["KUBERNETES"]
    )
    all_union: set[str] = set().union(*profile_tags.values())
    orphans = all_tags - all_union

    print(f"Spec totals: {total_ops} ops, {len(all_tags)} tags\n")

    print("Per-profile coverage:")
    for name, tags in profile_tags.items():
        ops = sum(tag_ops[t] for t in tags)
        print(f"  {name:11s} {len(tags):2d} tags  {ops:3d} ops")
    print(
        f"  {'DEFAULT':11s} {len(default_union):2d} tags  "
        f"{sum(tag_ops[t] for t in default_union):3d} ops  "
        f"(BASE+DOCKER+KUBERNETES)"
    )
    print(
        f"  {'UNION':11s} {len(all_union):2d} tags  "
        f"{sum(tag_ops[t] for t in all_union):3d} ops  (all 5 profiles)\n"
    )

    print(
        f"Orphan tags ({len(orphans)} tags, "
        f"{sum(tag_ops[t] for t in orphans)} ops) — paste into docs/profiles.md:\n"
    )
    print("| Count | Tag | Notes |")
    print("|---:|---|---|")
    for t in sorted(orphans, key=lambda x: (-tag_ops[x], x)):
        print(f"| {tag_ops[t]} | `{t}` | _(carry note from previous docs/profiles.md, or describe)_ |")


if __name__ == "__main__":
    main()
