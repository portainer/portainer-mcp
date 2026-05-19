"""Unit tests for `src/portainer_mcp/proxy.py` validators.

Covers the pure path/header guards. The HTTP-call path (`_call`,
`docker_proxy`, `kubernetes_proxy`) needs a running httpx client and
isn't covered here.
"""

from __future__ import annotations

import pytest

from portainer_mcp.proxy import _apply_select, _validate_headers, _validate_path


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


def test_apply_select_passthrough_when_no_select():
    assert _apply_select('{"k": "v"}', None) == '{"k": "v"}'
    assert _apply_select("plain text", None) == "plain text"


def test_apply_select_projects_valid_json():
    text = '[{"Id": "a"}, {"Id": "b"}]'
    assert _apply_select(text, "[].Id") == '["a", "b"]'


def test_apply_select_passes_through_non_json():
    # Plain text / binary / Docker error pages reach here; the proxy must
    # not raise — the model sees the upstream body as-is.
    assert _apply_select("not json at all", "[].Id") == "not json at all"
