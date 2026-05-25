"""Unit tests for `src/portainer_mcp/request_context.py`.

`snapshot()` returns `client_ip` / `user_agent` / `session_id` drawn
from the live HTTP request (via FastMCP's `get_http_request()`),
omitting any field that isn't set. With no active request it returns
`{}`.
"""

from __future__ import annotations

from fastmcp.server.http import set_http_request
from starlette.requests import Request

from portainer_mcp import request_context


def _request(
    *,
    client: tuple[str, int] | None = ("203.0.113.7", 51234),
    headers: list[tuple[bytes, bytes]] | None = None,
) -> Request:
    return Request(
        {
            "type": "http",
            "method": "POST",
            "path": "/mcp",
            "raw_path": b"/mcp",
            "query_string": b"",
            "client": client,
            "headers": headers or [],
        }
    )


def test_snapshot_empty_when_no_request():
    # Outside any ASGI request, get_http_request() raises and we degrade
    # to an empty dict so callers can blindly `**snapshot()` into a log.
    assert request_context.snapshot() == {}


def test_snapshot_reads_live_request_fields():
    request = _request(
        headers=[
            (b"user-agent", b"Claude-Code/1.2.3"),
            (b"mcp-session-id", b"sess-xyz"),
        ]
    )
    with set_http_request(request):
        assert request_context.snapshot() == {
            "client_ip": "203.0.113.7",
            "user_agent": "Claude-Code/1.2.3",
            "session_id": "sess-xyz",
        }


def test_snapshot_omits_unset_fields():
    # Missing User-Agent / Mcp-Session-Id should result in keys absent
    # from the dict, not `null` values.
    with set_http_request(_request()):
        assert request_context.snapshot() == {"client_ip": "203.0.113.7"}


def test_snapshot_omits_client_ip_when_client_absent():
    with set_http_request(_request(client=None)):
        assert request_context.snapshot() == {}
