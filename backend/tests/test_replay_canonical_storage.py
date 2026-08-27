import asyncio
import gzip
import json

from backend.app.routes import dashboard
from backend.app.routes.dashboard import (
    _load_canonical_replay_index,
    _load_canonical_replay_round,
)


def test_loads_segmented_canonical_replay(tmp_path) -> None:
    replay_dir = tmp_path / "canonical" / "presentation" / "replay"
    replay_dir.mkdir(parents=True)
    round_path = replay_dir / "round_001.json.gz"
    with gzip.open(round_path, "wt", encoding="utf-8") as file:
        json.dump(
            {
                "schema_id": "stratai.replay_round@2",
                "match_id": "123",
                "round": {"round": 1, "frames": [{"tick": 100}], "events": []},
            },
            file,
        )
    index_path = replay_dir / "index.json"
    index_path.write_text(
        json.dumps(
            {
                "schema_id": "stratai.replay_index@2",
                "match_id": "123",
                "metadata": {"tick_rate": 64},
                "rounds": [
                    {
                        "round_number": 1,
                        "path": "presentation/replay/round_001.json.gz",
                    }
                ],
            }
        ),
        encoding="utf-8",
    )
    (tmp_path / "manifest.json").write_text(
        json.dumps(
            {
                "artifacts": [
                    {
                        "artifact_type": "replay_index",
                        "path": "canonical/presentation/replay/index.json",
                    }
                ]
            }
        ),
        encoding="utf-8",
    )

    assert _load_canonical_replay_index(str(tmp_path))["metadata"]["tick_rate"] == 64
    assert _load_canonical_replay_round(str(tmp_path), 1)["frames"] == [{"tick": 100}]


def test_rejects_replay_round_path_outside_canonical_directory(tmp_path) -> None:
    replay_dir = tmp_path / "canonical" / "presentation" / "replay"
    replay_dir.mkdir(parents=True)
    index_path = replay_dir / "index.json"
    index_path.write_text(
        json.dumps(
            {
                "rounds": [
                    {"round_number": 1, "path": "../../../outside.json.gz"}
                ]
            }
        ),
        encoding="utf-8",
    )
    (tmp_path / "manifest.json").write_text(
        json.dumps(
            {
                "artifacts": [
                    {
                        "artifact_type": "replay_index",
                        "path": "canonical/presentation/replay/index.json",
                    }
                ]
            }
        ),
        encoding="utf-8",
    )

    assert _load_canonical_replay_round(str(tmp_path), 1) is None


def test_metadata_adapter_preserves_legacy_uppercase_winner(monkeypatch) -> None:
    class MatchStub:
        def replay_index(self):
            return {
                "metadata": {"tick_rate": 64},
                "rounds": [{"round_number": 1, "winner_side": "ct"}],
            }

    monkeypatch.setattr(dashboard, "find_canonical_match", lambda *_args: MatchStub())

    response = asyncio.run(dashboard.get_match_replay_metadata("123"))

    assert response["rounds_summary"][0]["winner"] == "CT"
