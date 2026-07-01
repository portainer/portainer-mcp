"""Unit tests for `portainer_mcp.profiles.resolve`.

Covers the env-string parsing, profile union, ALL sentinel, fail-fast on
unknown profiles, and warn-and-continue on unknown extras.
"""

from __future__ import annotations

import logging

import pytest

from portainer_mcp import profiles


def test_default_profiles_union_dedups_shared_tags():
    # BASE ∪ DOCKER ∪ KUBERNETES ∪ GITOPS. `endpoints`, `stacks`, and `gitops`
    # each appear in more than one profile — this proves the union dedups them.
    # Exact tag content is intentionally not asserted: that lives in
    # `TAG_PROFILES` and pinning the literal here would just duplicate the data.
    tags = profiles.resolve(profiles.DEFAULT_PROFILES)
    assert tags == tuple(sorted(set(tags)))
    assert tags.count("endpoints") == 1
    assert tags.count("stacks") == 1
    assert tags.count("gitops") == 1
    assert {"auth", "docker", "kubernetes", "gitops"} <= set(tags)


def test_gitops_travels_with_stacks_and_standalone():
    # Since Portainer 2.43 a GitOps source is required to deploy a git-backed
    # stack, so `gitops` must accompany `stacks` in any stack-capable profile,
    # and also stand alone for a source-management-only persona.
    assert "gitops" in profiles.resolve("DOCKER")
    assert "gitops" in profiles.resolve("KUBERNETES")
    assert profiles.resolve("GITOPS") == ("gitops",)


def test_all_sentinel_returns_none():
    assert profiles.resolve(profiles.ALL) is None


def test_all_short_circuits_even_with_other_names():
    # ALL combined with anything else still disables the filter — no point
    # validating the other names when the filter's off.
    assert profiles.resolve("ALL,DOCKER,BOGUS") is None


def test_unknown_profile_raises():
    with pytest.raises(ValueError, match="unknown PORTAINER_PROFILES"):
        profiles.resolve("BASE,NOPE")


def test_unknown_profile_error_lists_available_names():
    # The error message is the user's first clue about valid names — assert
    # the full set surfaces, not just one example.
    with pytest.raises(ValueError) as exc:
        profiles.resolve("NOPE")
    msg = str(exc.value)
    for name in (*profiles.TAG_PROFILES, profiles.ALL):
        assert name in msg


def test_extras_are_appended_and_deduped():
    tags = profiles.resolve("BASE", "observability,custom_templates,auth")
    assert "observability" in tags
    assert "custom_templates" in tags
    # `auth` was already in BASE — union semantics, not duplication.
    assert tags.count("auth") == 1


def test_extras_unknown_in_spec_warn_but_pass_through(caplog):
    with caplog.at_level(logging.WARNING, logger="portainer_mcp"):
        tags = profiles.resolve("BASE", "definitely_not_a_tag", known_tags={"auth"})
    assert "definitely_not_a_tag" in tags
    assert any("definitely_not_a_tag" in r.message for r in caplog.records)


def test_extras_in_spec_do_not_warn(caplog):
    with caplog.at_level(logging.WARNING, logger="portainer_mcp"):
        profiles.resolve("BASE", "observability", known_tags={"auth", "observability"})
    assert caplog.records == []


def test_no_warn_when_known_tags_not_provided(caplog):
    # When the caller has no spec to validate against (e.g. tests, dry runs),
    # extras pass through silently.
    with caplog.at_level(logging.WARNING, logger="portainer_mcp"):
        profiles.resolve("BASE", "anything_goes")
    assert caplog.records == []


def test_whitespace_and_blank_segments_are_tolerated():
    assert profiles.resolve(" BASE , , DOCKER ") == profiles.resolve("BASE,DOCKER")
