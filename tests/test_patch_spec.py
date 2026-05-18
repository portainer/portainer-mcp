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
    assert spec["paths"]["/git/{id}"] == {}


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
