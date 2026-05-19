"""Tag profiles for the Portainer MCP server.

`PORTAINER_PROFILES` selects which profiles to enable (comma-separated);
their tag sets are union'd. `PORTAINER_TAGS_EXTRA` appends arbitrary tags
as an escape hatch for surfaces no profile covers yet.

The `ALL` sentinel bypasses the tag filter entirely so new upstream tags
don't require a profile edit. See `docs/profiles.md` for the per-profile
tag list and the orphan tags not covered by any profile.
"""

from __future__ import annotations

import logging
from collections.abc import Iterable

logger = logging.getLogger("portainer_mcp")

ALL = "ALL"

TAG_PROFILES: dict[str, tuple[str, ...]] = {
    "BASE": ("auth", "system", "status", "settings", "motd"),
    "DOCKER": ("docker", "endpoints", "stacks"),
    "KUBERNETES": ("kubernetes", "helm", "endpoints", "stacks"),
    "EDGE": (
        "edge",
        "edge_stacks",
        "edge_jobs",
        "edge_groups",
        "edge_update_schedules",
        "edge_configs",
    ),
    "ADMIN": (
        "users",
        "teams",
        "team_memberships",
        "roles",
        "ldap",
        "license",
        "backup",
        "registries",
        "endpoint_groups",
        "policies",
        "resource_controls",
        "tags",
    ),
}

DEFAULT_PROFILES = "BASE,DOCKER,KUBERNETES"


def _split(s: str) -> list[str]:
    return [p.strip() for p in s.split(",") if p.strip()]


def resolve(
    profiles_env: str,
    extras_env: str = "",
    known_tags: Iterable[str] | None = None,
) -> tuple[str, ...] | None:
    """Resolve env-var selections to the tag tuple for RouteMap construction.

    Returns `None` when `ALL` is selected — caller must skip tag filtering.
    Raises `ValueError` on an unknown profile name (fail fast at startup).
    Logs a warning for extras not in `known_tags` (warn-and-continue;
    they're harmless — they just won't match any operation).
    """
    selected = set(_split(profiles_env))
    extras = set(_split(extras_env))

    if ALL in selected:
        return None

    unknown = selected - TAG_PROFILES.keys()
    if unknown:
        available = sorted(TAG_PROFILES) + [ALL]
        raise ValueError(
            f"unknown PORTAINER_PROFILES: {sorted(unknown)}; available: {available}"
        )

    if known_tags is not None:
        for extra in sorted(extras - set(known_tags)):
            logger.warning(
                "PORTAINER_TAGS_EXTRA tag %r not in spec; will match nothing",
                extra,
            )

    return tuple(sorted({t for p in selected for t in TAG_PROFILES[p]} | extras))
