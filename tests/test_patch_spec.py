"""Unit tests for `spec/patch_spec.py`.

One test per defect-mitigation rule, plus the YAML `=` constructor.
Inputs are hand-rolled minimal specs — no live Portainer YAML required.
"""

from __future__ import annotations

import copy

import yaml

from patch_spec import ENDPOINT_ID_DESCRIPTION, patch


def _spec(paths: dict | None = None, schemas: dict | None = None) -> dict:
    return {
        "paths": paths or {},
        "components": {"schemas": schemas or {}},
    }


# --- EXCLUDED_OPERATION_IDS -------------------------------------------------


def test_excluded_operation_ids_are_removed():
    spec = _spec(
        paths={
            "/kubernetes/{id}/namespaces": {
                "put": {"operationId": "UpdateKubernetesNamespaceDeprecated"},
            },
        }
    )
    patch(spec)
    # Path is removed entirely once all its methods drop out.
    assert "/kubernetes/{id}/namespaces" not in spec["paths"]


def test_partial_method_drop_keeps_path():
    # The live `UpdateKubernetesNamespace` shares no path with its deprecated
    # twin, but sibling methods on the deprecated one's path must survive.
    spec = _spec(
        paths={
            "/kubernetes/{id}/namespaces": {
                "put": {"operationId": "UpdateKubernetesNamespaceDeprecated"},
                "get": {"operationId": "GetKubernetesNamespaces"},
            },
        }
    )
    patch(spec)
    assert spec["paths"]["/kubernetes/{id}/namespaces"] == {
        "get": {"operationId": "GetKubernetesNamespaces"},
    }


def test_unrelated_operations_are_kept():
    spec = _spec(
        paths={
            "/keep": {
                "get": {"operationId": "KeepMe"},
                "post": {"operationId": "AlsoKeep"},
            }
        }
    )
    patch(spec)
    assert set(spec["paths"]["/keep"]) == {"get", "post"}


# --- EXCLUDED_PATH_PREFIXES -------------------------------------------------


def test_websocket_paths_are_dropped():
    spec = _spec(
        paths={
            "/websocket/exec": {"get": {"operationId": "ExecWS"}},
            "/websocket/attach": {"get": {"operationId": "AttachWS"}},
            "/endpoints": {"get": {"operationId": "EndpointList"}},
        }
    )
    patch(spec)
    assert set(spec["paths"]) == {"/endpoints"}


# --- edge-agent-only callbacks (tag-based) ----------------------------------


def test_edge_agent_callbacks_dropped():
    # Every op carrying the `edge_agent` tag is an agent-only callback that
    # 403s for an MCP caller, so the patcher drops it regardless of
    # operationId. `EndpointEdgeStackInspect` also carries `edge_stacks` (it
    # would otherwise surface in the EDGE profile) and must still go.
    spec = _spec(
        paths={
            "/endpoints/{id}/edge/stacks/{stackId}": {
                "get": {
                    "operationId": "EndpointEdgeStackInspect",
                    "tags": ["edge_agent", "edge_stacks"],
                },
            },
            "/endpoints/{id}/edge/async": {
                "post": {"operationId": "endpointEdgeAsync", "tags": ["edge_agent"]},
            },
            # No operationId — still dropped by the tag.
            "/endpoints/{id}/edge/charts": {
                "get": {"summary": "Get edge charts", "tags": ["edge_agent"]},
            },
        }
    )
    patch(spec)
    assert spec["paths"] == {}


def test_edge_admin_tools_are_kept():
    # `EdgeStackList` / `EdgeStackInspect` are admin-facing edge tools that
    # do *not* require the agent header. They carry `edge_stacks` but not
    # `edge_agent`, so they must survive.
    spec = _spec(
        paths={
            "/edge_stacks": {
                "get": {"operationId": "EdgeStackList", "tags": ["edge_stacks"]},
            },
            "/edge_stacks/{id}": {
                "get": {"operationId": "EdgeStackInspect", "tags": ["edge_stacks"]},
            },
        }
    )
    patch(spec)
    assert set(spec["paths"]) == {"/edge_stacks", "/edge_stacks/{id}"}


def test_path_with_remaining_methods_is_kept():
    # If a path mixes an agent callback with a non-agent method, only the
    # tagged method is dropped — the path survives.
    spec = _spec(
        paths={
            "/endpoints/{id}/edge/status": {
                "get": {
                    "operationId": "EndpointEdgeStatusInspect",
                    "tags": ["edge_agent"],
                },
                "put": {"operationId": "HypotheticalAdminWrite", "tags": ["endpoints"]},
            },
        }
    )
    patch(spec)
    assert spec["paths"]["/endpoints/{id}/edge/status"] == {
        "put": {"operationId": "HypotheticalAdminWrite", "tags": ["endpoints"]},
    }


# --- ENUM_STRIPS ------------------------------------------------------------


def test_top_level_enum_strip_policy_type():
    spec = _spec(schemas={"policies.PolicyType": {"enum": [1, 2, 3]}})
    patch(spec)
    assert spec["components"]["schemas"]["policies.PolicyType"] == {}


def test_enum_strip_missing_schema_is_noop():
    # Future spec versions may drop the schema entirely — patcher must not crash.
    spec = _spec(schemas={})
    patch(spec)
    assert spec["components"]["schemas"] == {}


# --- endpointId required on the stack git/migrate operations ----------------


_LEGACY_ENDPOINT_ID_TEXT = (
    "Stacks created before version 1.18.0 might not have an associated "
    "environment(endpoint) identifier."
)


def _stack_op(operation_id: str, required: bool | None = None) -> dict:
    param = {
        "name": "endpointId",
        "in": "query",
        "schema": {"type": "integer"},
        "description": _LEGACY_ENDPOINT_ID_TEXT,
    }
    if required is not None:
        param["required"] = required
    return {
        "operationId": operation_id,
        "parameters": [
            {"name": "id", "in": "path", "required": True, "schema": {"type": "integer"}},
            param,
        ],
    }


def _endpoint_id(op: dict) -> dict:
    return next(p for p in op["parameters"] if p["name"] == "endpointId")


def test_endpoint_id_becomes_required_on_affected_operations():
    ops = {
        ("/stacks/{id}/git/redeploy", "put"): "StackGitRedeploy",
        ("/stacks/{id}/git", "post"): "StackUpdateGit",
        ("/stacks/{id}/migrate", "post"): "StackMigrate",
    }
    spec = _spec(paths={p: {m: _stack_op(oid)} for (p, m), oid in ops.items()})
    patch(spec)
    for (path, method) in ops:
        param = _endpoint_id(spec["paths"][path][method])
        assert param["required"] is True
        assert param["description"] == ENDPOINT_ID_DESCRIPTION
        # The path param is untouched.
        assert spec["paths"][path][method]["parameters"][0]["name"] == "id"


def test_endpoint_id_untouched_on_other_operations():
    spec = _spec(
        paths={
            # Already required upstream — nothing to fix.
            "/stacks/{id}": {"delete": _stack_op("StackDelete", required=True)},
            # Genuinely optional elsewhere; the fix is per-operation, not per-name.
            "/stacks/{id}/file": {"get": _stack_op("StackFileInspect")},
        }
    )
    before = copy.deepcopy(spec)
    patch(spec)
    assert spec == before


# --- policy request-body property injection ---------------------------------


_POLICY_TYPES_WITH_DUPES = [
    # Mirrors the real defect: every value listed twice.
    "rbac-k8s", "change-confirmation", "cleanup-docker",
    "rbac-k8s", "change-confirmation", "cleanup-docker",
]
_POLICY_TYPES_DEDUPED = ["change-confirmation", "cleanup-docker", "rbac-k8s"]


def test_bare_object_payload_gets_properties_injected():
    spec = _spec(
        schemas={
            "policies.PolicyType": {"enum": _POLICY_TYPES_WITH_DUPES},
            "policies.policyCreatePayload": {"type": "object"},
            "policies.policyConflictsPayload": {"type": "object"},
        }
    )
    patch(spec)
    schemas = spec["components"]["schemas"]

    create = schemas["policies.policyCreatePayload"]
    assert set(create["properties"]) == {
        "AllowOverride", "Data", "EnvironmentGroups", "Name", "Type",
    }
    assert create["required"] == ["Name", "Type"]
    assert create["properties"]["Type"] == {
        "type": "string", "enum": _POLICY_TYPES_DEDUPED,
    }

    conflicts = schemas["policies.policyConflictsPayload"]
    # Lowercase, matching the server's actual JSON tags — not the create
    # payload's PascalCase.
    assert set(conflicts["properties"]) == {"environmentGroups", "policyId", "type"}
    assert conflicts["required"] == ["type"]
    assert conflicts["properties"]["type"] == {
        "type": "string", "enum": _POLICY_TYPES_DEDUPED,
    }


def test_policy_type_enum_is_read_before_enum_strips_empties_it():
    # `ENUM_STRIPS` empties `policies.PolicyType`'s own enum in the same
    # `patch()` call — the payload fixes must read it first, not end up
    # with an empty/absent enum because of ordering.
    spec = _spec(
        schemas={
            "policies.PolicyType": {"enum": _POLICY_TYPES_WITH_DUPES},
            "policies.policyCreatePayload": {"type": "object"},
        }
    )
    patch(spec)
    schemas = spec["components"]["schemas"]
    assert schemas["policies.PolicyType"] == {}  # ENUM_STRIPS still applies
    assert schemas["policies.policyCreatePayload"]["properties"]["Type"]["enum"] == (
        _POLICY_TYPES_DEDUPED
    )


def test_policy_type_property_has_no_ref_or_allof():
    # The `allOf: [$ref policies.PolicyType]` shape is what ENUM_STRIPS
    # exists to unwind — the injected `Type`/`type` must be a plain
    # `type: string` + `enum`, never coupled back to that schema.
    spec = _spec(
        schemas={
            "policies.PolicyType": {"enum": _POLICY_TYPES_WITH_DUPES},
            "policies.policyCreatePayload": {"type": "object"},
            "policies.policyConflictsPayload": {"type": "object"},
        }
    )
    patch(spec)
    schemas = spec["components"]["schemas"]
    for schema_name, field in [
        ("policies.policyCreatePayload", "Type"),
        ("policies.policyConflictsPayload", "type"),
    ]:
        prop = schemas[schema_name]["properties"][field]
        assert "$ref" not in prop and "allOf" not in prop


def test_policy_type_enum_missing_falls_back_to_plain_string():
    # If `policies.PolicyType` is ever missing or has no enum, don't inject
    # an empty (impossible-to-satisfy) enum — fall back to an unconstrained
    # string, same as ENUM_STRIPS's own fallback.
    spec = _spec(schemas={"policies.policyCreatePayload": {"type": "object"}})
    patch(spec)
    prop = spec["components"]["schemas"]["policies.policyCreatePayload"]["properties"]["Type"]
    assert prop == {"type": "string"}


def test_schema_property_fix_missing_schema_is_noop():
    # Future spec versions may drop the schema entirely — patcher must not crash.
    spec = _spec(schemas={})
    patch(spec)
    assert spec["components"]["schemas"] == {}


def test_schema_property_fix_does_not_override_real_properties():
    # If upstream ever documents these payloads for real, don't clobber it.
    spec = _spec(
        schemas={
            "policies.policyCreatePayload": {
                "type": "object",
                "properties": {"Upstream": {"type": "string"}},
            },
        }
    )
    patch(spec)
    assert set(spec["components"]["schemas"]["policies.policyCreatePayload"]["properties"]) == {"Upstream"}


# --- yaml `=` constructor ---------------------------------------------------


def test_yaml_equals_tag_loads_as_string():
    # `portaineree.ConditionOperator` ships `=` as a bare enum value.
    # The module-level constructor in patch_spec must coerce it to a string,
    # not the YAML 1.1 "value" sentinel.
    loaded = yaml.safe_load("op: =\n")
    assert loaded == {"op": "="}


# --- patch() returns the same dict it mutated ------------------------------


def test_patch_returns_same_dict():
    spec = _spec()
    assert patch(spec) is spec
