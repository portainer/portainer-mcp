"""Apply spec-defect mitigations to a Portainer OpenAPI spec.

Drops operations that shouldn't reach the tool surface (deprecated
duplicates, edge-agent-only callbacks), drops `/websocket/*` paths (protocol
upgrades, not REST), strips malformed `enum` blocks that defeat naive
generators, and injects real properties into undocumented bare-object
request-body schemas that would otherwise collapse into a single opaque
tool parameter FastMCP mis-serializes.

Also normalises stray tab characters in scalar text — swaggo occasionally
emits a literal tab inside a description string. Current specs keep them
inside block scalars, where YAML tolerates them, so this is hygiene for the
generated tool descriptions plus insurance against one landing in an
indentation position, where it *would* be a parse error.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import yaml

# `UpdateKubernetesNamespaceDeprecated` (PUT /kubernetes/{id}/namespaces) is
# upstream's superseded twin of `UpdateKubernetesNamespace`
# (PUT /kubernetes/{id}/namespaces/{namespace}); it takes the target namespace
# from the body instead of the path. Dropping it keeps the namespace-update
# surface unambiguous for the model. Through 2.42 it was *also* structurally
# broken — it declared a required `namespace` path param absent from its own
# template — but upstream fixed that in 2.43, so this is now a deliberate
# policy exclusion, not a defect workaround. New defects get added here.
EXCLUDED_OPERATION_IDS = {"UpdateKubernetesNamespaceDeprecated"}

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

# Schemas whose `enum` lists every value twice (swaggo emits the varname list
# a second time). The duplicates parse fine, but FastMCP folds the schema into
# an `allOf` whose inner enum then contradicts the outer one — visible on
# `PolicyUpdate`'s `Type` property. Dropping the enum leaves a plain string.
# `images.Status` reaches only *output* schemas (`ServiceImageStatus`,
# `containerImageStatus`, `stackImagesStatus`), so it's cosmetic there — listed
# for consistency, since it's the same upstream defect.
ENUM_STRIPS = frozenset(
    {
        "policies.PolicyType",
        "github_com_portainer_portainer-ee_api_docker_images.Status",
    }
)

# `policies.policyCreatePayload` and `policies.policyConflictsPayload` — the
# request-body schemas for `PolicyCreate` (POST /policies) and
# `PolicyConflicts` (POST /policies/conflicts) — are undocumented upstream:
# swaggo emits a bare `{"type": "object"}` with no declared properties. With
# nothing to flatten, FastMCP falls back to a single opaque `body` tool
# parameter for the whole payload, but then serializes the call *wrapped*
# under that parameter's own name (`{"body": {...}}`) instead of using its
# value as the literal request body — Portainer never sees the real fields,
# so every call fails with a generic 400 ("Policy name is required and
# cannot be empty" / "Policy type must be one of the valid types") no
# matter what's supplied. `PolicyUpdate`'s payload schema *does* declare
# real properties, so it flattens (and works) correctly — this defect is
# specific to the two bare-object schemas. Field shapes confirmed by hand
# against a live 2.45.0 server: Create mirrors `policyUpdatePayload`'s
# fields; Conflicts uses the server's actual lowercase JSON tags
# (`policyId`, `type`, `environmentGroups`) — capitalised names would still
# decode (Go's `encoding/json` matches case-insensitively) but would
# advertise names the API doesn't use. `policyId` lets a conflict check
# exclude the policy being edited from its own results (the update-preview
# case); omitting it would silently limit the tool to create-preview only.
#
# `Type` is built by `_policy_type_property` as a plain `type: string` +
# `enum` — never `allOf: [$ref policies.PolicyType]`. That shape is exactly
# what `ENUM_STRIPS` exists to unwind (an inner enum on the referenced
# schema contradicting an outer one), and it would only *appear* to work
# here because `ENUM_STRIPS` happens to run first in `patch()` and empties
# `policies.PolicyType`'s enum before this runs — nothing pins that
# ordering. The enum values are read from `policies.PolicyType`'s own enum
# in `patch()` before `ENUM_STRIPS` empties it (deduplicated, since that's
# the very defect `ENUM_STRIPS` targets), not hand-copied, so they can't
# silently drift the way `policyUpdatePayload`'s own local enum already
# has — it's missing three policy types added in 2.43/2.44
# (`cleanup-docker`, `network-security-k8s`, `pod-security-standards-k8s`).
# That's upstream's pre-existing omission on a schema this project doesn't
# patch (`PolicyUpdate` isn't part of this defect), not fixed here.
def _policy_type_property(policy_types: list[str]) -> dict:
    prop: dict = {"type": "string"}
    if policy_types:
        prop["enum"] = policy_types
    return prop


def _policy_payload_fixes(policy_types: list[str]) -> dict:
    type_property = _policy_type_property(policy_types)
    data_property = {
        "type": "object",
        "additionalProperties": {},
        "description": (
            "Data contains the policy-type-specific configuration. Namespace "
            "fields on RBAC/Registry/Network Security policies are lowercased "
            "automatically and must be valid RFC 1123 DNS labels; values still "
            "invalid after lowercasing are rejected with a 400."
        ),
    }
    return {
        "policies.policyCreatePayload": {
            "properties": {
                "AllowOverride": {"type": "boolean", "example": False},
                "Data": data_property,
                "EnvironmentGroups": {"type": "array", "items": {"type": "integer"}},
                "Name": {"type": "string", "example": "Development Policy"},
                "Type": type_property,
            },
            "required": ["Name", "Type"],
        },
        "policies.policyConflictsPayload": {
            "properties": {
                "environmentGroups": {"type": "array", "items": {"type": "integer"}},
                "policyId": {
                    "type": "integer",
                    "description": (
                        "The policy being edited, if any — excluded from its own "
                        "conflicts so an update-preview doesn't flag itself."
                    ),
                },
                "type": type_property,
            },
            "required": ["type"],
        },
    }


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

    # Read the deduplicated policy-type list before ENUM_STRIPS empties
    # `policies.PolicyType`'s own (duplicated) enum below — see
    # `_policy_type_property` for why this must happen first.
    policy_types = sorted(
        set((schemas.get("policies.PolicyType") or {}).get("enum") or [])
    )

    for name in ENUM_STRIPS:
        node = schemas.get(name)
        if isinstance(node, dict):
            node.pop("enum", None)

    for name, fix in _policy_payload_fixes(policy_types).items():
        node = schemas.get(name)
        if isinstance(node, dict) and not node.get("properties"):
            node["properties"] = fix["properties"]
            node["required"] = fix["required"]
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
