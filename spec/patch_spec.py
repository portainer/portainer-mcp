"""Apply spec-defect mitigations to a Portainer OpenAPI spec.

Mirrors the workarounds catalogued in portainer-go-sdk's
`docs/spec-upstream-fixes.md`: drops operations whose path/parameter shape
is structurally broken, drops `/websocket/*` paths (protocol upgrades, not
REST), and strips malformed `enum` blocks that defeat naive generators.

Also normalises stray tab characters in scalar text (swaggo occasionally
emits a literal `\t` inside a description string, which PyYAML rejects).
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import yaml

EXCLUDED_OPERATION_IDS = {
    "SharedGitGet",
    "SharedGitUpdate",
    "SharedGitDelete",
    "UpdateKubernetesNamespaceDeprecated",
    "providerInfo",
    "provisionCluster",
}

EXCLUDED_PATH_PREFIXES = ("/websocket",)

ENUM_STRIPS = (
    ("os.FileMode",),
    ("policies.PolicyType",),
    ("v1.Duration", "properties", "time.Duration"),
)

DEFAULT_INPUT = Path("/workspace/portainer-api-docs/versions/ee/2.41.1.yaml")
DEFAULT_OUTPUT = Path(__file__).resolve().parent / "portainer-patched.yaml"

# `portaineree.ConditionOperator` lists `=` as a bare enum value. YAML 1.1
# parses `=` as the special "value" tag; treat it as a plain string instead.
yaml.SafeLoader.add_constructor(
    "tag:yaml.org,2002:value", lambda loader, node: node.value
)


def patch(spec: dict) -> dict:
    paths = spec.setdefault("paths", {})
    for path in list(paths):
        if any(path.startswith(p) for p in EXCLUDED_PATH_PREFIXES):
            paths.pop(path)
            continue
        for method in list(paths[path]):
            op = paths[path][method]
            if isinstance(op, dict) and op.get("operationId") in EXCLUDED_OPERATION_IDS:
                paths[path].pop(method)

    schemas = spec.get("components", {}).get("schemas", {})
    for trail in ENUM_STRIPS:
        node = schemas.get(trail[0])
        for key in trail[1:]:
            node = node.get(key) if isinstance(node, dict) else None
        if isinstance(node, dict):
            node.pop("enum", None)
    return spec


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, nargs="?", default=DEFAULT_INPUT)
    parser.add_argument("output", type=Path, nargs="?", default=DEFAULT_OUTPUT)
    args = parser.parse_args(argv)

    raw = args.input.read_text().replace("\t", " ")
    spec = yaml.safe_load(raw)
    patch(spec)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w") as f:
        yaml.safe_dump(spec, f, sort_keys=False)
    print(f"wrote {args.output}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
