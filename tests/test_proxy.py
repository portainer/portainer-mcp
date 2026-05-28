"""Unit tests for `src/portainer_mcp/proxy.py` validators.

Covers the pure path/header/param guards. One end-to-end test drives a
proxy tool through FastMCP over a mock transport to confirm the
`query_params` coercion survives argument validation.
"""

from __future__ import annotations

import json

import httpx
import pytest
from fastmcp import FastMCP

from portainer_mcp.proxy import (
    _apply_select,
    _coerce_param_map,
    _validate_headers,
    _validate_path,
    register,
)
from portainer_mcp.redaction import EXPOSE_ENV_VAR, SENTINEL


@pytest.fixture(autouse=True)
def _redact_by_default(monkeypatch):
    """Default to the redacted posture; individual tests opt in to expose."""
    monkeypatch.delenv(EXPOSE_ENV_VAR, raising=False)


# --- _validate_path ---------------------------------------------------------


def test_validate_path_accepts_normal_path():
    _validate_path("/containers/json")
    _validate_path("/api/v1/namespaces/default/pods")


@pytest.mark.parametrize(
    "path, message",
    [
        ("containers/json", "leading slash"),
        ("/foo?bar=1", "query_params"),
        ("/foo#frag", "query_params"),
        ("/../etc/passwd", r"must not contain '\.\.'"),
        ("/a/../b", r"must not contain '\.\.'"),
    ],
)
def test_validate_path_rejects(path: str, message: str):
    with pytest.raises(ValueError, match=message):
        _validate_path(path)


def test_validate_path_allows_double_dot_inside_segment():
    # `..` is only blocked as a standalone segment; substrings are fine.
    _validate_path("/foo..bar")


# --- _validate_headers ------------------------------------------------------


def test_validate_headers_accepts_none_and_empty():
    _validate_headers(None)
    _validate_headers({})


def test_validate_headers_accepts_normal_headers():
    _validate_headers({"Accept": "application/json", "X-Custom": "value"})


@pytest.mark.parametrize(
    "header",
    ["X-API-Key", "x-api-key", "Authorization", "AUTHORIZATION", "Cookie", "Host"],
)
def test_validate_headers_rejects_blocked(header: str):
    with pytest.raises(ValueError, match="not allowed"):
        _validate_headers({header: "v"})


# --- _apply_select ----------------------------------------------------------


def test_apply_select_passthrough_when_no_select_and_exposed(monkeypatch):
    monkeypatch.setenv(EXPOSE_ENV_VAR, "1")
    assert _apply_select('{"k": "v"}', None) == '{"k": "v"}'
    assert _apply_select("plain text", None) == "plain text"


def test_apply_select_projects_valid_json_with_no_env_to_redact():
    text = '[{"Id": "a"}, {"Id": "b"}]'
    assert _apply_select(text, "[].Id") == '["a", "b"]'


def test_apply_select_passes_through_non_json():
    # Plain text / binary / Docker error pages reach here; the proxy must
    # not raise — the model sees the upstream body as-is.
    assert _apply_select("not json at all", "[].Id") == "not json at all"


# --- redaction in proxy ----------------------------------------------------


def test_apply_select_redacts_env_without_select():
    text = json.dumps({"Config": {"Env": ["FOO=bar"]}})
    out = _apply_select(text, None)
    body, _, hint_text = out.partition("\n\n")
    assert json.loads(body) == {"Config": {"Env": [f"FOO={SENTINEL}"]}}
    assert "1 env value(s) redacted" in hint_text
    assert EXPOSE_ENV_VAR in hint_text


def test_apply_select_redacts_env_then_projects():
    # Projection runs *after* redaction — `select` landing on env values
    # lands on the sentinel, not the real secret.
    text = json.dumps({"Config": {"Env": ["FOO=bar"]}})
    out = _apply_select(text, "Config.Env")
    body, _, _ = out.partition("\n\n")
    assert json.loads(body) == [f"FOO={SENTINEL}"]
    assert "1 env value(s) redacted" in out


def test_apply_select_no_hint_when_no_env(monkeypatch):
    text = json.dumps({"Id": "abc"})
    out = _apply_select(text, None)
    # Redaction posture is on but nothing matched; the wrapper still parses
    # + re-serialises (compact form), but no hint should be appended.
    assert "redacted" not in out
    assert json.loads(out) == {"Id": "abc"}


def test_apply_select_exposes_when_toggle_set(monkeypatch):
    monkeypatch.setenv(EXPOSE_ENV_VAR, "1")
    text = json.dumps({"Config": {"Env": ["FOO=bar"]}})
    out = _apply_select(text, None)
    # Fast path: no select + exposed = pass-through verbatim.
    assert out == text


def test_apply_select_non_json_passes_through_under_redaction():
    # Even with redaction on, non-JSON bodies (logs, stats text) must pass
    # through unchanged.
    assert _apply_select("not json at all", None) == "not json at all"


# --- _coerce_param_map ------------------------------------------------------


def test_coerce_param_map_none_passes_through():
    assert _coerce_param_map(None) is None


def test_coerce_param_map_native_string_dict_unchanged():
    assert _coerce_param_map({"all": "true"}) == {"all": "true"}


def test_coerce_param_map_parses_json_string():
    # Claude Desktop serializes the whole object argument as a JSON string.
    assert _coerce_param_map('{"all": "true"}') == {"all": "true"}


def test_coerce_param_map_stringifies_scalar_values():
    # The model may send native bools/numbers; they belong in the query
    # string as their wire form.
    assert _coerce_param_map({"all": True, "limit": 5}) == {
        "all": "true",
        "limit": "5",
    }


def test_coerce_param_map_stringifies_nested_value():
    # Docker's `filters` query parameter expects a JSON-encoded string.
    assert _coerce_param_map({"filters": {"status": ["running"]}}) == {
        "filters": '{"status": ["running"]}'
    }


def test_coerce_param_map_drops_none_values():
    assert _coerce_param_map({"all": "true", "since": None}) == {"all": "true"}


def test_coerce_param_map_keeps_literal_string_null():
    # A real null is an unset optional; the literal string "null" is a value.
    assert _coerce_param_map({"x": "null"}) == {"x": "null"}


@pytest.mark.parametrize("value", ["{not json", '["a"]', "5", '"x"'])
def test_coerce_param_map_rejects_non_object(value: str):
    with pytest.raises(ValueError, match="expected a JSON object"):
        _coerce_param_map(value)


# --- end-to-end: BeforeValidator survives FastMCP arg validation ------------


async def test_proxy_coerces_query_params_end_to_end():
    # FastMCP validates tool arguments with pydantic; this confirms the
    # BeforeValidator runs there, so a stringified dict reaches Docker as
    # real query params rather than being rejected as a string.
    captured: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["params"] = dict(request.url.params)
        return httpx.Response(200, json={"ok": True})

    client = httpx.AsyncClient(
        base_url="http://test", transport=httpx.MockTransport(handler)
    )
    mcp = FastMCP("test")
    register(mcp, client, read_only=True)
    tool = await mcp.get_tool("docker_proxy")

    await tool.run(
        {
            "environment_id": 1,
            "path": "/containers/json",
            "query_params": '{"all": "true"}',
        }
    )
    assert captured["params"] == {"all": "true"}
