"""Unit tests for the wiring bits in `src/portainer_mcp/server.py` that are
cheap to exercise without booting a full FastMCP server.

The `tool`-name extraction is unit-tested against the real
`CallToolRequestParams` shape; the live middleware dispatch (which feeds that
context) is covered by FastMCP itself.
"""

from __future__ import annotations

from mcp.types import CallToolRequestParams, InitializeRequestParams

from portainer_mcp.server import _ContextualStructuredLogging


class _Ctx:
    def __init__(self, method, message):
        self.method = method
        self.message = message


def _mw() -> _ContextualStructuredLogging:
    # cache=None → identity enrichment is a no-op (the path stdio takes).
    return _ContextualStructuredLogging(include_payload_length=True, cache=None)


def test_enrich_adds_tool_name_for_tools_call():
    ctx = _Ctx("tools/call", CallToolRequestParams(name="docker_proxy", arguments={}))
    assert _mw()._enrich({}, ctx)["tool"] == "docker_proxy"


def test_enrich_omits_tool_for_non_tools_call():
    ctx = _Ctx(
        "initialize",
        InitializeRequestParams(
            protocolVersion="2025-03-26",
            capabilities={},
            clientInfo={"name": "x", "version": "1"},
        ),
    )
    assert "tool" not in _mw()._enrich({}, ctx)


def test_enrich_drops_constant_source_field():
    ctx = _Ctx("tools/call", CallToolRequestParams(name="EndpointList", arguments={}))
    # The base middleware stamps source="client" on every record; it's a
    # constant in our request path, so the enriched record omits it.
    assert "source" not in _mw()._enrich({"source": "client"}, ctx)
