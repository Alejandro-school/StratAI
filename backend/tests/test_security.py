from pathlib import Path

import pytest

from backend.app.middleware.security import is_safe_path, sanitize_match_id


def test_sanitize_match_id_keeps_supported_characters() -> None:
    assert sanitize_match_id("match_2026-07-24") == "match_2026-07-24"


def test_sanitize_match_id_removes_path_traversal_characters() -> None:
    assert sanitize_match_id("../../match.json") == "matchjson"


def test_sanitize_match_id_rejects_empty_result() -> None:
    with pytest.raises(ValueError, match="Invalid match_id"):
        sanitize_match_id("../..")


def test_is_safe_path_accepts_children_and_rejects_siblings(tmp_path: Path) -> None:
    base = tmp_path / "matches"
    child = base / "demo.dem"
    sibling = tmp_path / "secrets.txt"

    assert is_safe_path(str(base), str(child))
    assert not is_safe_path(str(base), str(sibling))
