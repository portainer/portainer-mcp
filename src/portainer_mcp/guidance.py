"""A meta-tool that serves the server's operating guidance on demand.

The full hygiene guide is far larger than the MCP `instructions` field can
carry (clients truncate instructions at ~2KB), so the short instructions point
here and the model fetches the full doc through a tool call when it needs it —
progressive disclosure rebuilt inside MCP.
"""

from __future__ import annotations

import logging
from typing import Annotated

from fastmcp import FastMCP
from mcp.types import ToolAnnotations
from pydantic import Field

from portainer_mcp.shaping import SELECT_DESCRIPTION

logger = logging.getLogger("portainer_mcp")


def register(mcp: FastMCP, guide: str) -> None:
    """Register the `get_guidance` tool returning the hygiene guide verbatim."""

    @mcp.tool(
        name="get_guidance",
        annotations=ToolAnnotations(readOnlyHint=True),
        description=(
            "Operating guide for this Portainer MCP server: projecting responses "
            "with `select`, where the heavy fields live, results that are easy to "
            "misread (e.g. an edge environment's health comes from its heartbeat, "
            "not its `Status` field; typed K8s tools use different field names than "
            "the raw proxy), and how to deploy / scale / delete safely. Call it "
            "once at the start of any Portainer task, before interpreting responses "
            "or planning multi-step changes — it materially improves correctness "
            "and saves context."
        ),
    )
    async def get_guidance(
        # `select` is declared so the tool satisfies the universal-select
        # invariant and SelectArgTransform skips it (it would otherwise re-encode
        # the markdown body). The guide isn't a projectable JSON payload, so the
        # parameter is a no-op here.
        select: Annotated[str | None, Field(description=SELECT_DESCRIPTION)] = None,
    ) -> str:
        return guide

    logger.info("guidance tool registered (%d chars)", len(guide))
