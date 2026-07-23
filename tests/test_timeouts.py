"""Unit tests for `src/portainer_mcp/timeouts.py`.

Covers the env resolver and the timeout-hint middleware, including one
end-to-end run through FastMCP's real exception-wrapping path (a raw
httpx.ReadTimeout from a tool, as the proxy tools produce).
"""

from __future__ import annotations

import httpx
import pytest
from fastmcp import FastMCP
from fastmcp.exceptions import ToolError

from portainer_mcp import timeouts


# --- resolve() ---------------------------------------------------------------


def test_resolve_default(monkeypatch):
    monkeypatch.delenv(timeouts.ENV_VAR, raising=False)
    t = timeouts.resolve()
    assert t.read == timeouts.DEFAULT_SECONDS
    assert t.write == timeouts.DEFAULT_SECONDS
    assert t.pool == timeouts.DEFAULT_SECONDS
    assert t.connect == timeouts.CONNECT_SECONDS


def test_resolve_override_keeps_connect_cap(monkeypatch):
    monkeypatch.setenv(timeouts.ENV_VAR, "600")
    t = timeouts.resolve()
    assert t.read == 600
    assert t.connect == 10


def test_resolve_small_value_lowers_connect_too(monkeypatch):
    # An operator asking for fast failures shouldn't get a connect phase
    # that outlives the whole allowance.
    monkeypatch.setenv(timeouts.ENV_VAR, "5")
    t = timeouts.resolve()
    assert t.read == 5
    assert t.connect == 5


@pytest.mark.parametrize("raw", ["abc", "", "0", "-3", "inf", "nan"])
def test_resolve_rejects_invalid(monkeypatch, raw):
    monkeypatch.setenv(timeouts.ENV_VAR, raw)
    with pytest.raises(SystemExit, match=timeouts.ENV_VAR):
        timeouts.resolve()


# --- _post_send_timeout ------------------------------------------------------


def test_post_send_timeout_walks_openapi_cause_chain():
    # FastMCP's OpenAPI tools raise ToolError <- ValueError <- ReadTimeout.
    rt = httpx.ReadTimeout("read deadline")
    ve = ValueError("HTTP request timed out (ReadTimeout)")
    ve.__cause__ = rt
    te = ToolError("Error calling tool 'StackCreateDockerStandaloneString'")
    te.__cause__ = ve
    assert timeouts._post_send_timeout(te) is rt


def test_post_send_timeout_ignores_connect_and_pool():
    for exc_type in (httpx.ConnectTimeout, httpx.PoolTimeout):
        te = ToolError("Upstream request timed out, please retry")
        te.__cause__ = exc_type("x")
        assert timeouts._post_send_timeout(te) is None


def test_post_send_timeout_ignores_unrelated_errors():
    te = ToolError("Error calling tool 'x': boom")
    te.__cause__ = ValueError("boom")
    assert timeouts._post_send_timeout(te) is None


# --- TimeoutHintMiddleware end-to-end ----------------------------------------


def _server_raising(exc: Exception) -> FastMCP:
    mcp = FastMCP("test")

    @mcp.tool
    async def flaky() -> str:
        raise exc

    mcp.add_middleware(timeouts.TimeoutHintMiddleware(120))
    return mcp


async def test_middleware_rewrites_read_timeout():
    # A raw httpx.ReadTimeout is what the proxy tools surface; FastMCP first
    # wraps it in its own "please retry" ToolError, which the middleware
    # must catch via the cause chain and replace.
    mcp = _server_raising(httpx.ReadTimeout("read deadline"))
    with pytest.raises(ToolError) as exc:
        await mcp.call_tool("flaky", {})
    message = str(exc.value)
    assert "'flaky'" in message
    assert "may still have completed" in message
    assert "verify current state" in message
    assert timeouts.ENV_VAR in message
    assert "120s" in message


async def test_middleware_leaves_connect_timeout_alone():
    mcp = _server_raising(httpx.ConnectTimeout("no route"))
    with pytest.raises(ToolError) as exc:
        await mcp.call_tool("flaky", {})
    assert timeouts.ENV_VAR not in str(exc.value)


async def test_middleware_leaves_other_errors_alone():
    mcp = _server_raising(ValueError("boom"))
    with pytest.raises(ToolError) as exc:
        await mcp.call_tool("flaky", {})
    assert timeouts.ENV_VAR not in str(exc.value)
    assert "boom" in str(exc.value)


async def test_middleware_passes_successful_results_through():
    mcp = FastMCP("test")

    @mcp.tool
    async def ok() -> str:
        return "fine"

    mcp.add_middleware(timeouts.TimeoutHintMiddleware(120))
    result = await mcp.call_tool("ok", {})
    assert result.content[0].text == "fine"
