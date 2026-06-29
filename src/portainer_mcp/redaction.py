"""Redact env values from tool responses.

Walks parsed JSON in place and replaces env values with a sentinel. The
walker is field-name driven (`env` / `envvars`, case-insensitive) and
dispatches on value shape — list of {name, value} dicts or list of
KEY=VAL strings. K8s `valueFrom` references are preserved (they're
references, not secrets).

Called *before* JMESPath projection so callers can't bypass redaction
with `select="Env"` etc. Returns (data, count) so the caller can emit
a single summary message naming PORTAINER_EXPOSE_ENV_VALUES.
"""

from __future__ import annotations

import os
import re
from typing import Any

SENTINEL = "[REDACTED]"
EXPOSE_ENV_VAR = "PORTAINER_EXPOSE_ENV_VALUES"
_ENV_KEYS = frozenset({"env", "envvars"})
_KEY_VAL_RE = re.compile(r"^([A-Za-z_][A-Za-z0-9_]*)=")


def is_expose_enabled() -> bool:
    raw = os.environ.get(EXPOSE_ENV_VAR)
    return raw is not None and raw not in {"0", "false", "False"}


def redact_envs(data: Any) -> tuple[Any, int]:
    """Mutate `data` in place, redacting env values; return (data, count)
    for one-line caller use."""
    return data, _walk(data)


def hint(count: int) -> str:
    return (
        f"[{count} env value(s) redacted; "
        f"set {EXPOSE_ENV_VAR}=1 on the MCP server to disclose]"
    )


def count_in(text: str) -> int:
    """Count redaction sentinels left in a serialized response body.

    Redaction runs on the *full* upstream object before `select` projects it,
    so `redact_envs`'s own count includes values a projection may then drop.
    Sizing the hint off the final body instead keeps the reported number equal
    to what the caller actually receives — no "182 redacted" on a response that
    contains no env fields at all.
    """
    return text.count(SENTINEL)


def _walk(node: Any) -> int:
    if isinstance(node, dict):
        count = 0
        for key, value in node.items():
            if isinstance(key, str) and key.lower() in _ENV_KEYS and isinstance(value, list):
                count += _redact_list(value)
            else:
                count += _walk(value)
        return count
    if isinstance(node, list):
        return sum(_walk(item) for item in node)
    return 0


def _redact_list(items: list) -> int:
    count = 0
    for i, item in enumerate(items):
        if isinstance(item, dict):
            # Shapes A, F, G — {name, value} / {Name, Value}. Skip valueFrom
            # (K8s reference, not a secret) and `default` (Template variable
            # placeholder, surfaced verbatim in the Portainer UI — not a
            # runtime secret). Only one casing of value is present per item;
            # the loop covers both without a branch.
            for key in ("value", "Value"):
                if key in item:
                    item[key] = SENTINEL
                    count += 1
                    break
        elif isinstance(item, str):
            # Shape C — Docker-native "KEY=VAL" string list.
            match = _KEY_VAL_RE.match(item)
            if match:
                items[i] = f"{match.group(0)}{SENTINEL}"
                count += 1
    return count
