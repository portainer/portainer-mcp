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
            "/git/{id}": {
                "get": {"operationId": "SharedGitGet"},
                "put": {"operationId": "SharedGitUpdate"},
                "delete": {"operationId": "SharedGitDelete"},
            },
        }
    )
    patch(spec)
    # Path is removed entirely once all its methods drop out.
    assert "/git/{id}" not in spec["paths"]


def test_partial_method_drop_keeps_path():
    spec = _spec(
        paths={
            "/git/{id}": {
                "get": {"operationId": "SharedGitGet"},
                "post": {"operationId": "KeepMe"},
            },
        }
    )
    patch(spec)
    assert spec["paths"]["/git/{id}"] == {"post": {"operationId": "KeepMe"}}


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


# --- EXCLUDED_PATH_METHODS (edge-agent-only callbacks) ----------------------


def test_edge_agent_callbacks_dropped_even_without_operation_id():
    # Two of the four agent-callback ops have no operationId in the spec —
    # FastMCP names them from `summary`. The path-method filter catches them.
    spec = _spec(
        paths={
            "/endpoints/{id}/edge/stacks/{stackId}": {
                "get": {"summary": "Inspect an Edge Stack for an Environment(Endpoint)"},
            },
            "/endpoints/{id}/edge/status": {
                "get": {"operationId": "EndpointEdgeStatusInspect"},
            },
            "/endpoints/{id}/edge/alerts": {
                "post": {"operationId": "EndpointEdgeAlertsReceive"},
            },
            "/endpoints/{id}/edge/jobs/{jobID}/logs": {
                "post": {"summary": "Update the logs collected from an Edge Job"},
            },
        }
    )
    patch(spec)
    # All four paths had a single method each; after the drop the path
    # itself is removed by the empty-path cleanup.
    assert spec["paths"] == {}


def test_edge_admin_tools_are_kept():
    # `EdgeStackList` / `EdgeStackInspect` are admin-facing edge tools that
    # do *not* require the agent header. They share the `edge` / `edge_stacks`
    # tags with the callbacks but live on different paths — they must survive.
    spec = _spec(
        paths={
            "/edge_stacks": {"get": {"operationId": "EdgeStackList"}},
            "/edge_stacks/{id}": {"get": {"operationId": "EdgeStackInspect"}},
        }
    )
    patch(spec)
    assert set(spec["paths"]) == {"/edge_stacks", "/edge_stacks/{id}"}


def test_path_with_remaining_methods_is_kept():
    # If a future spec adds a non-agent method to one of the edge paths,
    # the path must survive — only the targeted method is dropped.
    spec = _spec(
        paths={
            "/endpoints/{id}/edge/status": {
                "get": {"operationId": "EndpointEdgeStatusInspect"},
                "put": {"operationId": "HypotheticalAdminWrite"},
            },
        }
    )
    patch(spec)
    assert spec["paths"]["/endpoints/{id}/edge/status"] == {
        "put": {"operationId": "HypotheticalAdminWrite"},
    }


# --- ENUM_STRIPS ------------------------------------------------------------


def test_top_level_enum_strip_os_filemode():
    spec = _spec(schemas={"os.FileMode": {"type": "string", "enum": ["a", "b"]}})
    patch(spec)
    assert "enum" not in spec["components"]["schemas"]["os.FileMode"]
    assert spec["components"]["schemas"]["os.FileMode"]["type"] == "string"


def test_top_level_enum_strip_policy_type():
    spec = _spec(schemas={"policies.PolicyType": {"enum": [1, 2, 3]}})
    patch(spec)
    assert spec["components"]["schemas"]["policies.PolicyType"] == {}


def test_nested_enum_strip_time_duration():
    spec = _spec(
        schemas={
            "v1.Duration": {
                "properties": {
                    "time.Duration": {"type": "string", "enum": ["1s", "1m"]},
                }
            }
        }
    )
    patch(spec)
    node = spec["components"]["schemas"]["v1.Duration"]["properties"]["time.Duration"]
    assert "enum" not in node
    assert node["type"] == "string"


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
