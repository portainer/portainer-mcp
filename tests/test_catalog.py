"""Budget guard on the `tools/list` payload.

The tool catalog is the fixed context cost every client pays on every
request, before any Portainer data is fetched. It regresses silently — a
widened profile, a longer parameter description multiplied across 200+
generated tools, an upstream spec that grew — so the size is asserted here
rather than merely observed at startup.

This builds the real server against the bundled spec (no network: the httpx
client is constructed but never used), so the number tracks the shipped
default rather than a fixture.
"""

from __future__ import annotations

import asyncio

import pytest

from portainer_mcp import server

# Calibrated 2026-07-27 against the default profile set
# (BASE,DOCKER,KUBERNETES,GITOPS) at 205,036 chars over 205 tools, plus ~15%
# headroom. Sized to bite: the pre-dedup SELECT_DESCRIPTION put the catalog at
# 259,156, so reintroducing a paragraph-length per-tool description trips this.
# Raise it only with a note on what grew and why that growth is worth the
# context it costs every caller — shrinking the catalog is the intended
# response to a red test here.
MAX_CATALOG_CHARS = 235_000


@pytest.fixture
def default_env(monkeypatch):
    """Boot config for the shipped default profile set, isolated from the
    developer's shell.
    """
    # build_server() reconfigures global logging (propagate=False on the
    # portainer_mcp/fastmcp/httpx loggers), which would break `caplog` for
    # every test that runs after this module.
    monkeypatch.setattr(server, "_setup_logging", lambda: None)
    monkeypatch.setenv("PORTAINER_URL", "http://portainer.test")
    monkeypatch.setenv("PORTAINER_API_KEY", "k" * 32)
    for var in (
        "PORTAINER_MCP_TRANSPORT",
        "PORTAINER_PROFILES",
        "PORTAINER_TAGS_EXTRA",
        "PORTAINER_READ_ONLY",
        "PORTAINER_NO_PROXY",
    ):
        monkeypatch.delenv(var, raising=False)


def test_default_catalog_within_budget(default_env):
    mcp = server.build_server()
    tools = asyncio.run(mcp.list_tools())
    size = server.tool_catalog_chars(tools)

    assert size <= MAX_CATALOG_CHARS, (
        f"tool catalog is {size:,} chars across {len(tools)} tools, over the "
        f"{MAX_CATALOG_CHARS:,} char budget. This payload ships to the client "
        f"on every request — prefer shrinking it (fewer tools via profiles, "
        f"shorter parameter descriptions) over raising the budget."
    )
