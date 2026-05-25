"""Unit tests for the JSON log formatter in `src/portainer_mcp/server.py`.

The formatter's contract: emit one JSON envelope per record; if the
message itself parses as a JSON object, hoist its fields into the
envelope rather than leaving them as a nested string. That's what lets
audit and request records become first-class fields in `json` mode.
"""

from __future__ import annotations

import json
import logging

import pytest

from portainer_mcp.server import (
    _JsonFormatter,
    _resolve_log_format,
)


def _record(msg: str, *, level: int = logging.INFO, name: str = "portainer_mcp") -> logging.LogRecord:
    return logging.LogRecord(
        name=name,
        level=level,
        pathname=__file__,
        lineno=0,
        msg=msg,
        args=None,
        exc_info=None,
    )


# --- _resolve_log_format -----------------------------------------------------


def test_resolve_log_format_defaults_to_text(monkeypatch):
    monkeypatch.delenv("PORTAINER_MCP_LOG_FORMAT", raising=False)
    assert _resolve_log_format() == "text"


@pytest.mark.parametrize("raw", ["json", "JSON", "Json"])
def test_resolve_log_format_accepts_json(monkeypatch, raw):
    monkeypatch.setenv("PORTAINER_MCP_LOG_FORMAT", raw)
    assert _resolve_log_format() == "json"


def test_resolve_log_format_rejects_unknown(monkeypatch):
    monkeypatch.setenv("PORTAINER_MCP_LOG_FORMAT", "yaml")
    with pytest.raises(SystemExit, match="must be 'text' or 'json'"):
        _resolve_log_format()


# --- _JsonFormatter ----------------------------------------------------------


def test_envelope_carries_ts_level_logger():
    payload = json.loads(_JsonFormatter().format(_record("hello")))
    assert payload["level"] == "INFO"
    assert payload["logger"] == "portainer_mcp"
    assert "ts" in payload


def test_plain_string_lands_in_msg_field():
    payload = json.loads(_JsonFormatter().format(_record("HTTP auth: enabled")))
    assert payload["msg"] == "HTTP auth: enabled"


def test_json_object_message_is_hoisted_into_envelope():
    # The shape the audit logger emits.
    raw = json.dumps({"event": "auth", "outcome": "ok", "token_fp": "abcd…wxyz"})
    payload = json.loads(
        _JsonFormatter().format(_record(raw, name="portainer_mcp.audit"))
    )
    assert payload["event"] == "auth"
    assert payload["outcome"] == "ok"
    assert payload["token_fp"] == "abcd…wxyz"
    assert payload["logger"] == "portainer_mcp.audit"
    assert "msg" not in payload


def test_json_array_message_falls_back_to_msg():
    # Only objects merge — arrays would clobber the envelope shape.
    raw = json.dumps([1, 2, 3])
    payload = json.loads(_JsonFormatter().format(_record(raw)))
    assert payload["msg"] == raw


def test_malformed_json_message_falls_back_to_msg():
    raw = "{not really json"
    payload = json.loads(_JsonFormatter().format(_record(raw)))
    assert payload["msg"] == raw


def test_exc_info_included_when_present():
    try:
        raise RuntimeError("boom")
    except RuntimeError:
        import sys
        exc_info = sys.exc_info()
    record = logging.LogRecord(
        name="portainer_mcp",
        level=logging.ERROR,
        pathname=__file__,
        lineno=0,
        msg="upstream failed",
        args=None,
        exc_info=exc_info,
    )
    payload = json.loads(_JsonFormatter().format(record))
    assert payload["msg"] == "upstream failed"
    assert "RuntimeError: boom" in payload["exc_info"]


def test_output_is_single_line():
    # SIEM/log shippers slice on newlines; a record with embedded
    # newlines in the message must not break the one-record-per-line rule.
    payload = _JsonFormatter().format(_record("line one\nline two"))
    assert "\n" not in payload
    parsed = json.loads(payload)
    assert parsed["msg"] == "line one\nline two"
