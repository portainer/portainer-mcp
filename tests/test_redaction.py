"""Unit tests for `src/portainer_mcp/redaction.py`.

Covers the pure recursive walker — every supported env shape, nested
locations, case-insensitive key match, K8s `valueFrom` preservation,
and the negative shapes the rule must leave alone.
"""

from __future__ import annotations

import pytest

from portainer_mcp.redaction import (
    EXPOSE_ENV_VAR,
    SENTINEL,
    hint,
    is_expose_enabled,
    redact_envs,
)


# --- redact_envs ------------------------------------------------------------


def test_shape_a_list_of_name_value_dicts():
    data = {"Env": [{"name": "DB_URL", "value": "postgres://secret"}]}
    out, count = redact_envs(data)
    assert out == {"Env": [{"name": "DB_URL", "value": SENTINEL}]}
    assert count == 1


def test_shape_c_docker_native_key_val_strings():
    data = {"Env": ["FOO=bar", "BAZ=qux", "MALFORMED"]}
    out, count = redact_envs(data)
    assert out == {"Env": [f"FOO={SENTINEL}", f"BAZ={SENTINEL}", "MALFORMED"]}
    assert count == 2


def test_shape_f_k8s_value_and_value_from():
    data = {
        "env": [
            {"name": "X", "value": "y"},
            {"name": "Z", "valueFrom": {"secretKeyRef": {"name": "s", "key": "k"}}},
        ]
    }
    out, count = redact_envs(data)
    assert out["env"][0]["value"] == SENTINEL
    assert out["env"][1] == {
        "name": "Z",
        "valueFrom": {"secretKeyRef": {"name": "s", "key": "k"}},
    }
    assert count == 1


def test_shape_g_envvars_capitalised_value():
    data = {"EnvVars": [{"Name": "K", "Value": "v"}]}
    out, count = redact_envs(data)
    assert out == {"EnvVars": [{"Name": "K", "Value": SENTINEL}]}
    assert count == 1


def test_case_insensitive_key_match():
    data = {"ENV": [{"name": "K", "value": "v"}]}
    out, count = redact_envs(data)
    assert out["ENV"][0]["value"] == SENTINEL
    assert count == 1


def test_nested_swarm_taskspec_env():
    data = {
        "Spec": {
            "TaskTemplate": {
                "ContainerSpec": {
                    "Env": ["DB=secret", "API_KEY=secret"],
                }
            }
        }
    }
    out, count = redact_envs(data)
    env = out["Spec"]["TaskTemplate"]["ContainerSpec"]["Env"]
    assert env == [f"DB={SENTINEL}", f"API_KEY={SENTINEL}"]
    assert count == 2


def test_list_root_with_multiple_objects():
    data = [
        {"Env": [{"name": "A", "value": "1"}]},
        {"Env": ["B=2"]},
    ]
    out, count = redact_envs(data)
    assert out[0]["Env"][0]["value"] == SENTINEL
    assert out[1]["Env"] == [f"B={SENTINEL}"]
    assert count == 2


def test_negative_template_variables_unchanged():
    # CustomTemplate's `Variables[].defaultValue` is intentionally out of
    # scope — template-author metadata, surfaced in the UI.
    data = {"Variables": [{"name": "X", "defaultValue": "y"}]}
    out, count = redact_envs(data)
    assert out == {"Variables": [{"name": "X", "defaultValue": "y"}]}
    assert count == 0


def test_negative_field_named_environment_id_unchanged():
    data = {"environment_id": 5, "EnvironmentName": "prod"}
    out, count = redact_envs(data)
    assert out == {"environment_id": 5, "EnvironmentName": "prod"}
    assert count == 0


def test_no_env_present_count_zero():
    data = {"Id": 1, "Name": "alpha", "children": [{"k": "v"}]}
    out, count = redact_envs(data)
    assert out == {"Id": 1, "Name": "alpha", "children": [{"k": "v"}]}
    assert count == 0


def test_in_place_mutation():
    data = {"Env": [{"name": "A", "value": "b"}]}
    out, _ = redact_envs(data)
    assert out is data
    assert data["Env"][0]["value"] == SENTINEL


def test_scalar_root_count_zero():
    out, count = redact_envs("just a string")
    assert out == "just a string"
    assert count == 0


def test_malformed_env_string_passes_through():
    # Strings inside an env list that don't match KEY=VAL pass through.
    data = {"Env": ["NOEQUALS", "123=numeric-prefix-ok"]}
    out, count = redact_envs(data)
    # "123=…" has a digit-leading key — doesn't match `[A-Za-z_]…`, so
    # both pass through unchanged.
    assert out == {"Env": ["NOEQUALS", "123=numeric-prefix-ok"]}
    assert count == 0


# --- is_expose_enabled -----------------------------------------------------


def test_expose_default_off(monkeypatch):
    monkeypatch.delenv(EXPOSE_ENV_VAR, raising=False)
    assert is_expose_enabled() is False


@pytest.mark.parametrize("falsey", ["0", "false", "False"])
def test_expose_falsey_values(monkeypatch, falsey: str):
    monkeypatch.setenv(EXPOSE_ENV_VAR, falsey)
    assert is_expose_enabled() is False


@pytest.mark.parametrize("truthy", ["1", "true", "yes", "anything"])
def test_expose_truthy_values(monkeypatch, truthy: str):
    monkeypatch.setenv(EXPOSE_ENV_VAR, truthy)
    assert is_expose_enabled() is True


# --- hint ------------------------------------------------------------------


def test_hint_mentions_env_var_and_count():
    h = hint(3)
    assert "3 env value(s) redacted" in h
    assert EXPOSE_ENV_VAR in h
