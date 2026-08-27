import json

from backend.app.utils.match_artifact_catalog import (
    load_match_manifest,
    resolve_match_artifact,
)


def test_resolves_catalog_path_before_legacy(tmp_path) -> None:
    canonical_path = tmp_path / "canonical" / "core" / "match.json"
    canonical_path.parent.mkdir(parents=True)
    canonical_path.write_text("{}", encoding="utf-8")
    (tmp_path / "metadata.json").write_text("{}", encoding="utf-8")
    (tmp_path / "manifest.json").write_text(
        json.dumps(
            {
                "artifacts": [
                    {
                        "artifact_type": "match",
                        "path": "canonical/core/match.json",
                    }
                ]
            }
        ),
        encoding="utf-8",
    )

    assert resolve_match_artifact(tmp_path, "match", "metadata.json") == canonical_path


def test_falls_back_to_legacy_path(tmp_path) -> None:
    legacy_path = tmp_path / "replay.json"
    legacy_path.write_text("{}", encoding="utf-8")

    assert resolve_match_artifact(tmp_path, "replay", "replay.json") == legacy_path


def test_rejects_catalog_path_outside_match_directory(tmp_path) -> None:
    outside_path = tmp_path.parent / "secret.json"
    outside_path.write_text("{}", encoding="utf-8")
    (tmp_path / "manifest.json").write_text(
        json.dumps(
            {
                "artifacts": [
                    {"artifact_type": "match", "path": "../secret.json"}
                ]
            }
        ),
        encoding="utf-8",
    )

    assert resolve_match_artifact(tmp_path, "match") is None


def test_invalid_manifest_is_treated_as_empty(tmp_path) -> None:
    (tmp_path / "manifest.json").write_text("not-json", encoding="utf-8")

    assert load_match_manifest(tmp_path) == {}
