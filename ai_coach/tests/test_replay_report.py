"""Presentation boundaries, causal isolation and non-destructive output."""

import json
from copy import deepcopy

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.interpretation import replay_report as report
from ai_coach.interpretation import temporal
from ai_coach.tests.test_support_anchors import ENEMY
from ai_coach.tests.test_temporal_description import inputs


def selected():
    source, sources = inputs()
    return report.select_records(source, sources, side="CT", sequences=[1, 3])


def test_observed_zero_false_missing_and_c4_are_distinct():
    rows = selected()
    text = report.render(rows)
    actor, ally = rows[-1]["actors"]
    assert report.fact_text(actor["physical"]["ammo_in_magazine"]) == "0"
    assert report.fact_text(actor["physical"]["reloading"]) == "No"
    assert report.fact_text(actor["has_c4"]) == "Sí"
    assert report.fact_text(ally["has_c4"]) == "No"
    assert "No disponible (no_previous_sample)" in text
    assert ENEMY not in text
    assert "no demuestra qué vio, oyó o sabía" in text
    assert "no esté seleccionada" in text
    assert all(row["record_id"] in text for row in rows)


def test_not_applicable_and_hostile_source_text():
    assert report.fact_text(
        {"status": "not_applicable", "value": None, "reason": "non_ammunition_weapon"}
    ).startswith("No aplicable")
    assert report.fact_text({"status": "observed", "value": "[x](url)|<b>\n`"}) == (
        "&#91;x&#93;(url)&#124;&lt;b&gt; &#96;"
    )


@pytest.mark.parametrize(
    "sequences", [[], [0], [3, 1], [1, 1], [True], [1.0], list(range(1, 102))]
)
def test_reject_invalid_selection(sequences):
    source, sources = inputs()
    with pytest.raises(ValueError):
        report.select_records(source, sources, side="CT", sequences=sequences)


@pytest.mark.parametrize("mutation", ["hash", "side", "order", "match", "provenance"])
def test_reject_corrupt_or_mixed_records(mutation):
    rows = selected()
    if mutation == "hash":
        rows[-1]["actors"][0]["has_c4"]["value"] = False
    elif mutation == "side":
        rows[-1]["perspective"]["side"] = "T"
    elif mutation == "order":
        rows.reverse()
    elif mutation == "match":
        rows[-1]["source_sha256"] = "f" * 64
    elif mutation == "provenance":
        rows[-1]["actors"][0]["references"][0]["availability_tick"] += 1
    if mutation != "hash":
        rows[-1]["record_id"] = temporal.digest(
            {k: v for k, v in rows[-1].items() if k != "record_id"}
        )
    with pytest.raises(ValueError):
        report.render(rows)


def test_future_enemy_answers_and_outcomes_cannot_change_report():
    source, sources = inputs()
    baseline = deepcopy((source, sources))
    expected = report.render(
        report.select_records(source, sources, side="CT", sequences=[1, 3])
    )
    assert (source, sources) == baseline
    source["observations"]["frames"][3:] = [None]
    for frame in source["observations"]["frames"][:3]:
        frame["players"][-1]["position"] = {"x": float("nan")}
    for name in ("physical_source", "objective_source"):
        sources[name]["observations"]["frames"][3:] = [None]
    for name in ("cuts", "states"):
        sources["c4_source"]["observations"][name][3:] = [None]
    source["expert_answers"] = {"E01": "attack"}
    source["future_outcomes"] = {"winner": "CT"}
    assert (
        report.render(
            report.select_records(source, sources, side="CT", sequences=[1, 3])
        )
        == expected
    )


def test_output_guard_runs_before_source_read(tmp_path, monkeypatch):
    (tmp_path / "retained.txt").write_text("keep", encoding="utf-8")

    def forbidden(*args):
        pytest.fail("source loaded before output rejection")

    monkeypatch.setattr(report.ledger_io, "load_sources", forbidden)
    with pytest.raises(RuntimeContractError):
        report.materialize(tmp_path / "missing", tmp_path, side="CT", sequences=[1])
    assert (tmp_path / "retained.txt").read_text() == "keep"


def test_written_evidence_keeps_original_records_and_hashes(tmp_path, monkeypatch):
    source, sources = inputs()
    monkeypatch.setattr(
        report.ledger_io, "load_sources", lambda _: (source, sources, {}, {})
    )
    manifest = report.materialize(tmp_path, tmp_path, side="CT", sequences=[1, 3])
    actual = [
        json.loads(line)
        for line in (tmp_path / report.EVIDENCE).read_text().splitlines()
    ]
    assert actual == selected()
    assert manifest["records"] == 2 and manifest["actors"] == 4
    for item in manifest["artifacts"]:
        assert item == report.physical.file_reference(tmp_path / item["path"])
    assert (tmp_path / report.REPORT).read_text(encoding="utf-8") == report.render(
        actual
    )
