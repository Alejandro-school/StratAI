import importlib.util
import json
import sys
from json import dumps
from pathlib import Path

SCRIPT_PATH = Path(__file__).with_name("reprocess_parallel.py")
SPEC = importlib.util.spec_from_file_location("reprocess_parallel", SCRIPT_PATH)
reprocess_parallel = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = reprocess_parallel
SPEC.loader.exec_module(reprocess_parallel)


def test_extract_match_id() -> None:
    assert reprocess_parallel.extract_match_id(Path("match_123.dem")) == "123"
    assert reprocess_parallel.extract_match_id(Path("demo_123.dem")) is None


def test_existing_metadata_uses_prefixed_export_directory(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    metadata_path = exports_dir / "match_123" / "metadata.json"
    metadata_path.parent.mkdir(parents=True)
    metadata_path.write_text(json.dumps({"date": "2026-07-24T12:00:00Z"}), encoding="utf-8")
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    assert reprocess_parallel.get_existing_metadata("123") == {"date": "2026-07-24T12:00:00Z"}


def test_affected_steam_ids_are_read_from_reprocessed_exports(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    players_path = exports_dir / "match_123" / "players_summary.json"
    players_path.parent.mkdir(parents=True)
    players_path.write_text(
        json.dumps({"players": [{"steam_id": "76561198000000000"}, {"steam_id": "invalid"}]}),
        encoding="utf-8",
    )
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    assert reprocess_parallel.get_affected_steam_ids(["123"]) == {"76561198000000000"}


def test_process_demo_preserves_existing_metadata(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    export_dir = exports_dir / "match_123"
    export_dir.mkdir(parents=True)
    (export_dir / "metadata.json").write_text(
        json.dumps({"date": "2026-07-24T12:00:00Z", "duration_seconds": 1800}),
        encoding="utf-8",
    )
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")
    captured_request = {}

    class Response:
        def raise_for_status(self) -> None:
            return None

    def post(_url, json, timeout):
        captured_request.update(json)
        (export_dir / "metadata.json").write_text(
            dumps({"date": json["match_date"]}), encoding="utf-8"
        )
        (export_dir / "players_summary.json").write_text('{"players": []}', encoding="utf-8")
        return Response()

    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    monkeypatch.setattr(reprocess_parallel.requests, "post", post)

    assert reprocess_parallel.process_demo(demo_path, timeout=10).status == "processed"
    assert captured_request["match_date"] == "2026-07-24T12:00:00Z"
    assert captured_request["match_duration"] == 1800
