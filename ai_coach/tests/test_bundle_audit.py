from __future__ import annotations

import gzip
import json
from pathlib import Path

import pytest

from ai_coach.audits.bundle_audit import AuditError, audit_bundles


def _write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value), encoding="utf-8")


def _write_jsonl(path: Path, values: list[dict[str, object]], *, zipped: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    content = "".join(json.dumps(value) + "\n" for value in values)
    if zipped:
        path.write_bytes(gzip.compress(content.encode("utf-8"), mtime=0))
    else:
        path.write_text(content, encoding="utf-8")


def _bundle(root: Path, name: str = "match-1") -> Path:
    bundle = root / name
    canonical = bundle / "canonical"
    _write_json(bundle / "manifest.json", {"format_version": "3.8.0", "parser_version": "v16"})
    _write_json(
        canonical / "manifest.json",
        {
            "format_version": "3.8.0",
            "parser_version": "v16",
            "artifacts": [{"artifact_type": "tactical_observations", "record_count": 2}],
        },
    )
    _write_json(canonical / "core/match.json", {"map_name": "de_mirage", "tick_rate_hz": 64})
    _write_json(canonical / "core/participants.json", {"players": []})
    _write_json(canonical / "core/rounds.json", {"rounds": []})
    _write_json(
        canonical / "states/tactical/sampling.json",
        {
            "sampling": {"target_hz": 16},
            "physical_row_count": 2,
            "oracle_row_count": 0,
            "gap_count": 0,
        },
    )
    _write_jsonl(
        canonical / "states/tactical/observed.jsonl.gz",
        [
            {
                "status": "observed",
                "observed_state_ref": "match:state:steam:private-id",
                "state": {"health": 100, "ammo": None},
            },
            {
                "status": "observed",
                "observed_state_ref": "match:state:steam:other-private-id",
                "state": {"health": 75, "ammo": 12},
            },
        ],
        zipped=True,
    )
    for path in (
        "events/combat_events.jsonl",
        "events/utility_events.jsonl",
        "events/objective_events.jsonl",
        "states/tactical/oracle.jsonl",
        "states/tactical/gaps.jsonl",
        "causal/decisions.jsonl",
        "causal/decision_features.jsonl",
        "causal/oracle_context.jsonl",
        "causal/outcomes.jsonl",
        "causal/quality_masks.jsonl",
    ):
        _write_jsonl(canonical / path, [])
    for path, key in (
        ("derived/engagements.json", "engagements"),
        ("derived/trades.json", "trade_candidates"),
        ("derived/economy_players.json", "players"),
        ("derived/economy_rounds.json", "rounds"),
        ("derived/clutch_events.json", "clutch_events"),
        ("derived/player_stats.json", "players"),
    ):
        payload = {key: []}
        if path.endswith("trades.json"):
            payload["trade_completions"] = []
        _write_json(canonical / path, payload)
    _write_json(canonical / "diagnostics/quality_report.json", {"report": {}})
    _write_json(canonical / "presentation/replay/index.json", {"rounds": []})
    return bundle


def test_audit_streams_gzip_and_reports_nulls(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    _bundle(root)

    report = audit_bundles(root, expected_bundles=1)
    fields = {item["path"]: item for item in report["fields"]}

    assert report["input"]["bundle_count"] == 1
    assert report["artifacts"]["tactical_observations"]["records"] == 2
    assert fields["tactical_observations.state.health"]["minimum"] == 75
    assert fields["tactical_observations.state.health"]["maximum"] == 100
    assert fields["tactical_observations.state.ammo"]["null_pct_of_present"] == 50.0
    assert fields["tactical_observations.observed_state_ref"]["examples"] == []
    assert fields["tactical_observations.observed_state_ref"]["top_values"] == []


def test_audit_fails_clearly_on_corrupt_jsonl(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    bundle = _bundle(root)
    (bundle / "canonical/events/combat_events.jsonl").write_text("{broken\n", encoding="utf-8")

    with pytest.raises(AuditError, match="combat_events.jsonl, línea 1"):
        audit_bundles(root, expected_bundles=1)


def test_audit_fails_clearly_on_corrupt_gzip(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    bundle = _bundle(root)
    (bundle / "canonical/states/tactical/observed.jsonl.gz").write_bytes(b"not-gzip")

    with pytest.raises(AuditError, match="stream corrupto o ilegible"):
        audit_bundles(root, expected_bundles=1)


def test_audit_refuses_unexpected_bundle_count(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    _bundle(root)

    with pytest.raises(AuditError, match="se esperaban 44 bundles"):
        audit_bundles(root, expected_bundles=44)


def test_audit_is_deterministic(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    _bundle(root)

    first = audit_bundles(root, expected_bundles=1)
    second = audit_bundles(root, expected_bundles=1)

    assert first == second
