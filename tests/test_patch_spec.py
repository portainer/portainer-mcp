"""Unit tests for `spec/patch_spec.py`.

One test per defect-mitigation rule, plus the YAML `=` constructor.
Inputs are hand-rolled minimal specs — no live Portainer YAML required.
"""

from __future__ import annotations

import yaml

from patch_spec import patch


def _spec(paths: dict | None = None, schemas: dict | None = None) -> dict:
    return {
        "paths": paths or {},
        "components": {"schemas": schemas or {}},
    }


# --- EXCLUDED_OPERATION_IDS -------------------------------------------------


def test_excluded_operation_ids_are_removed():
    spec = _spec(
        paths={
            "/cloud/{id}": {
                "get": {"operationId": "providerInfo"},
                "put": {"operationId": "provisionCluster"},
                "delete": {"operationId": "UpdateKubernetesNamespaceDeprecated"},
            },
        }
    )
    patch(spec)
    # Path is removed entirely once all its methods drop out.
    assert "/cloud/{id}" not in spec["paths"]


def test_partial_method_drop_keeps_path():
    spec = _spec(
        paths={
            "/cloud/{id}": {
                "get": {"operationId": "provisionCluster"},
                "post": {"operationId": "KeepMe"},
            },
        }
    )
    patch(spec)
    assert spec["paths"]["/cloud/{id}"] == {"post": {"operationId": "KeepMe"}}


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
