from pathlib import Path

import pytest

from backend.app.matches.canonical_repository import find_canonical_match, iter_canonical_matches
from backend.tests.test_replay_access_security import write_replay_bundle


def test_match_path_cannot_escape_exports(tmp_path):
    exports = tmp_path / "exports"
    (exports / "match_x").mkdir(parents=True)
    write_replay_bundle(tmp_path / "private_bundle")
    assert find_canonical_match(exports, "x/../../private_bundle") is None
    assert find_canonical_match(exports, "x\\..\\..\\private_bundle") is None


@pytest.mark.parametrize("identifier", ["", "../x", "x/y", "x\\y", "x:y", "x%2fy", "x " , "x" * 129])
def test_invalid_match_identifiers_are_rejected_before_lookup(tmp_path, identifier, monkeypatch):
    def no_filesystem_access(_self):
        raise AssertionError("Malformed identifier reached filesystem lookup")

    monkeypatch.setattr(Path, "is_dir", no_filesystem_access)
    assert find_canonical_match(tmp_path, identifier) is None


def test_valid_id_and_prefixed_id_still_resolve(tmp_path):
    write_replay_bundle(tmp_path / "match_abc-123")
    assert find_canonical_match(tmp_path, "abc-123") is not None
    assert find_canonical_match(tmp_path, "match_abc-123") is not None


def test_resolved_link_outside_exports_is_rejected(tmp_path, monkeypatch):
    exports = tmp_path / "exports"
    link = exports / "match_link"
    link.mkdir(parents=True)
    outside = tmp_path / "outside"
    write_replay_bundle(outside)
    original = Path.resolve

    def resolve(path, *args, **kwargs):
        return outside if path == link else original(path, *args, **kwargs)

    monkeypatch.setattr(Path, "resolve", resolve)
    assert find_canonical_match(exports, "link") is None
    assert list(iter_canonical_matches(exports)) == []
