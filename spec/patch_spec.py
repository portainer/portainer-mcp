"""Apply spec-defect mitigations to a Portainer OpenAPI spec.

Drops operations whose path/parameter shape is structurally broken,
drops `/websocket/*` paths (protocol upgrades, not REST), and strips
malformed `enum` blocks that defeat naive generators.

Also normalises stray tab characters in scalar text (swaggo occasionally
emits a literal `\t` inside a description string, which PyYAML rejects).
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import yaml

EXCLUDED_OPERATION_IDS = {
    "UpdateKubernetesNamespaceDeprecated",
    "providerInfo",
    "provisionCluster",
}

# Edge-agent-only callbacks under `/endpoints/{id}/edge/*` (heartbeat/status,
# async poll, alert + chart status, edge stack/job sync). They 403 for any
# caller that isn't an Edge agent presenting `X-PortainerAgent-EdgeID`, so they
# can never succeed for an MCP caller. 2.43 gave them a dedicated `edge_agent`
# tag — before that they were mis-tagged `endpoints` and leaked into the DOCKER
# profile, and we matched four of them by (method, path). Now we drop every op
# carrying the tag: it catches all of them (not an arbitrary subset), and one
# (`EndpointEdgeStackInspect`) also carries `edge_stacks`, so it would otherwise
# surface in the EDGE profile.
EXCLUDED_TAGS = frozenset({"edge_agent"})

EXCLUDED_PATH_PREFIXES = ("/websocket",)

ENUM_STRIPS = (("policies.PolicyType",),)

DEFAULT_OUTPUT = (
    Path(__file__).resolve().parents[1]
    / "src"
    / "portainer_mcp"
    / "data"
    / "portainer-patched.yaml"
)

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
            if not isinstance(op, dict):
                continue
            if op.get("operationId") in EXCLUDED_OPERATION_IDS or (
                EXCLUDED_TAGS.intersection(op.get("tags") or ())
            ):
                paths[path].pop(method)
        if not paths[path]:
            paths.pop(path)

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
    parser.add_argument("input", type=Path)
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
