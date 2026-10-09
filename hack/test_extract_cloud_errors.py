#!/usr/bin/env python3

# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

"""Tests for hack/extract-cloud-errors.py pure functions.

Run with: python -m pytest hack/test_extract_cloud_errors.py
"""

import importlib.util
import sys
from pathlib import Path

import pytest

# Load the module despite its hyphenated filename.
# Temporarily collapse sys.argv so REPO_ROOT falls back to the
# Path(__file__).parent.parent default (the repo root) rather than
# treating pytest's argv[1] as a repo path.
_saved_argv = sys.argv[:]
sys.argv = sys.argv[:1]
try:
    _spec = importlib.util.spec_from_file_location(
        "extract_cloud_errors",
        Path(__file__).parent / "extract-cloud-errors.py",
    )
    _mod = importlib.util.module_from_spec(_spec)
    _spec.loader.exec_module(_mod)
finally:
    sys.argv = _saved_argv

_is_admin = _mod._is_admin
_render_message = _mod._render_message
extract_args = _mod.extract_args


# ── _is_admin ────────────────────────────────────────────────────────────────

@pytest.mark.parametrize("sources,expected", [
    # /admin/ subdirectory in other packages
    (["`pkg/util/admin/foo.go:1`"], True),
    # admin_ prefix in pkg/frontend/
    (["`pkg/frontend/admin_openshiftcluster_etcdrecovery.go:25`"], True),
    # adminactions/ subdirectory
    (["`pkg/frontend/adminactions/openshiftcluster.go:30`"], True),
    # adminreplies.go is NOT admin-only (ReplyStream is used by non-admin handlers)
    (["`pkg/frontend/adminreplies.go:41`"], False),
    # fixetcd.go has no admin marker even though it is called from admin endpoints
    (["`pkg/frontend/fixetcd.go:80`"], False),
    # mixed sources: one admin path makes the whole row admin
    (["`pkg/frontend/fixetcd.go:80`", "`pkg/frontend/admin_foo.go:1`"], True),
    # regular user-facing file
    (["`pkg/frontend/openshiftcluster_get.go:36`"], False),
])
def test_is_admin(sources, expected):
    assert _is_admin(sources) == expected


# ── _render_message ──────────────────────────────────────────────────────────

@pytest.mark.parametrize("message,target,expected", [
    # plain message, no target
    ("Resource not found.", "", "Resource not found."),
    # plain message with non-empty target -> (target: `...`) annotation
    ("Resource not found.", "properties.foo", "Resource not found. (target: `properties.foo`)"),
    # Sprintf prefix stripped; format verb replaced with <target>
    ("Sprintf:The resource '%s' could not be found.", "name",
     "The resource '`<name>`' could not be found."),
    # target already wrapped in <>: embedded bare (no double wrapping)
    ("Sprintf:Invalid %s.", "<path>", "Invalid `<path>`."),
    # Sprintf message with no format verbs and non-empty target -> annotation
    ("Sprintf:Resource not found.", "foo", "Resource not found. (target: `foo`)"),
    # bare <...> tokens in message get backtick-wrapped
    ("The value <path> is invalid.", "", "The value `<path>` is invalid."),
    # Sprintf prefix stripped from target
    ("value is bad.", "Sprintf:properties.foo", "value is bad. (target: `properties.foo`)"),
    # message with no format verb and empty target: no annotation
    ("Cluster already exists.", "", "Cluster already exists."),
])
def test_render_message(message, target, expected):
    assert _render_message(message, target) == expected


# ── extract_args ─────────────────────────────────────────────────────────────

def _lines(code: str) -> list[str]:
    return code.splitlines()


@pytest.mark.parametrize("code,start_line,start_col,expected_args", [
    # single-line call
    ('foo(400, "InvalidParam", "", "Bad input.")', 0, 0,
     ["400", '"InvalidParam"', '""', '"Bad input."']),
    # string arg containing a comma: must not split
    ('foo(400, "A,B", "", "msg")', 0, 0,
     ["400", '"A,B"', '""', '"msg"']),
    # string arg containing parens: must not split
    ('foo(400, "A(B)", "", "msg")', 0, 0,
     ["400", '"A(B)"', '""', '"msg"']),
    # backtick raw-string arg containing a comma: must not split
    ("foo(400, `a,b`, \"\", \"msg\")", 0, 0,
     ["400", "`a,b`", '""', '"msg"']),
    # backtick raw-string arg containing a backslash: must NOT consume next char
    ("foo(400, `a\\b`, \"\", \"msg\")", 0, 0,
     ["400", "`a\\b`", '""', '"msg"']),
    # nested brackets in arg (depth tracking)
    ('foo(400, bar["key"], "", "msg")', 0, 0,
     ["400", 'bar["key"]', '""', '"msg"']),
])
def test_extract_args(code, start_line, start_col, expected_args):
    result = extract_args(_lines(code), start_line, start_col)
    assert result is not None
    args, _positions = result
    assert args == expected_args
