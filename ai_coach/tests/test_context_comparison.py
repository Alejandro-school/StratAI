"""CS3 causal isolation, equal inputs and honest separation of evaluation."""

from copy import deepcopy
import json

import pytest

from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation import context_comparison_io as io
from ai_coach.tests.test_candidate_context import fixture, kill_event, rehash, remove_c4
from ai_coach.tests.test_common_context import value


def test_equal_inputs_no_fake_semantic_run_or_tactical_scores():
    session = comparison.Session(value())
    result = comparison.compare_local(session)
    assert result["A"]["receipt"] == result["B"]["receipt"]
    assert result["C"]["executed"] is False
    assert result["A"]["reading"]["interpretation"] == []
    assert result["B"]["verification"]["tactical_quality"] == "not_evaluated"
    assert session.consultation().initial()["bank"]["examples"]
    assert any("X01" in c["principles"] for c in result["B"]["reading"]["uncertainty"])
    cut = next(f for f in session.consultation().initial()["situation"] if f["key"] == "cut")
    assert cut["value"]["sample_index"] == 4 and cut["value"]["seconds_in_window"] > 0


def test_future_mutation_and_case_ids_are_not_predictive_input():
    src = value()
    first = comparison.Session(src, through_index=2).consultation().initial()
    packet = deepcopy(src["packet"])
    packet["context"]["samples"][-1]["actors"][0]["position"]["value"]["x"] = 12345
    rehash(packet)
    supplements = deepcopy(src["supplements"])
    for row in supplements.values():
        if row:
            row["case_id"] = "E99"
    changed = candidate.build(packet, case_id="E99", instrument=src["instrument"], **supplements)
    assert comparison.Session(changed, through_index=2).consultation().initial() == first
    text = json.dumps(first)
    assert "E03" not in text and "E99" not in text
    assert src["packet"]["context"]["source_sha256"] not in text
    for actor in src["packet"]["context"]["samples"][0]["actors"]:
        assert actor["actor_alias"] not in text
        assert actor["identity"]["pawn_life"] not in text


def test_query_frontier_budget_and_unconsulted_citations():
    access = comparison.Session(value(), through_index=2).consultation()
    with pytest.raises(ValueError, match="frontier"):
        access.sample(3)
    with pytest.raises(ValueError, match="frontier"):
        access.sample(True)
    reading = {k: [] for k in comparison.SECTIONS}
    reading["situation"] = [comparison.claim("observed", "Dato", [{"id": "F0:0"}])]
    with pytest.raises(ValueError, match="citation"):
        access.verify(reading)
    access.sample(0)
    assert access.verify(reading)["semantic_truth"] == "not_evaluated"
    for _ in range(comparison.QUERY_LIMIT - 1):
        access.sample(0)
    with pytest.raises(ValueError, match="budget"):
        access.sample(0)


def test_valid_citation_does_not_prove_free_text_truth():
    access = comparison.Session(value()).consultation()
    reading = {k: [] for k in comparison.SECTIONS}
    reading["interpretation"] = [comparison.claim("inferred", "Texto falso que aún necesita revisión semántica", access.initial()["situation"][:1])]
    result = access.verify(reading)
    assert result["citation_integrity"] and result["semantic_truth"] == "not_evaluated"


def test_zero_false_partial_counts_and_postplant_keep_limits():
    packet, args = fixture()
    remove_c4(packet)
    packet["objective"][-1]["planted"] = True
    session = comparison.Session(candidate.build(packet, **args))
    facts = session.consultation().initial()["situation"]
    assert any(f["key"] == "ammo_in_magazine" and f["value"] == 0 for f in facts)
    assert any(f["key"] == "walking" and f["value"] is False for f in facts)
    assert any(f["key"] == "opponents_count" and f["value"]["alive"] is None for f in facts)
    reading = comparison.compare_local(session)["B"]["reading"]
    cited = {p for claims in reading.values() for c in claims for p in c["principles"]}
    assert {"P03", "P05"} <= cited
    assert reading["interpretation"] == []


def test_unknown_eligibility_preserved_without_knife_classification():
    packet, args = fixture()
    for row in packet["round_context"]:
        row["eligibility"].update(kind="unknown", analysis_allowed=False,
            reason="round_kind_unconfirmed", basis_sequence=row["source_sequence"])
    packet["analysis_scope"] = "pending_context"
    result = comparison.compare_local(comparison.Session(candidate.build(packet, **args)))
    assert any("P04" in c["principles"] for c in result["B"]["reading"]["uncertainty"])
    assert result["B"]["reading"]["interpretation"] == []


def test_events_keep_age_without_raw_identity_or_contact_claims():
    packet, args = fixture(death=True)
    args["kills"]["events"] = [kill_event(packet)]
    src = candidate.build(packet, **args)
    view = comparison.Session(src).consultation().initial()
    event = view["events"][0]
    assert event["key"] == "kill" and event["seconds_since_event"] == 0
    assert "ct-life" not in json.dumps(view) and "kill-" not in json.dumps(view)
    assert event["value"]["attacker"]["participant"].startswith("O")
    assert comparison.Session(src, through_index=3).consultation().initial()["events"] == []


def test_utility_object_identity_is_local_without_losing_counts_or_continuity():
    packet, args = fixture()
    actor = packet["context"]["samples"][0]["actors"][0]["actor_alias"]
    item = {"kind": "flash", "definition": 43, "count": 1, "minimum_count": 1,
            "quantity_source": "native_properties", "entity_life": "raw-utility-object-001"}
    for row in packet["inventory"]:
        row[actor]["items"] = [deepcopy(item)]
    session = comparison.Session(candidate.build(packet, **args))
    access = session.consultation()
    start, end = access.sample(0), access.initial()["situation"]
    inv = [next(f for f in facts if f["key"] == "inventory" and f["subject"] == "J1") for facts in (start, end)]
    assert inv[0]["value"]["items"] == inv[1]["value"]["items"]
    assert inv[0]["value"]["items"][0]["local_item_life"] == "I1"
    assert inv[0]["value"]["items"][0]["count"] == 1
    assert item["entity_life"] not in json.dumps(access.initial())


def test_ct_cannot_certify_multi_round_patterns():
    session = comparison.Session(value())
    # Test retrieval only, without fabricating a CT candidate or tactical label.
    session._view["side"] = "CT"
    result = comparison.compare_local(session)["B"]
    assert any("P07" in c["principles"] for c in result["reading"]["uncertainty"])
    assert result["reading"]["interpretation"] == []
    assert session.consultation().initial()["prior_rounds_status"] == "not_supplied"


def test_guarded_io_rejects_overwrite_before_source_read_and_tampering(tmp_path):
    source = tmp_path / "candidate.json"
    source.write_text(json.dumps(value()), encoding="utf-8")
    output = tmp_path / "comparison"
    output.mkdir()
    result = io.materialize(output, candidate_path=source, candidate_sha256=io.sha(source))
    assert result["calls_external_executed"] == 0
    io.validate(output)
    with pytest.raises(ValueError, match="empty"):
        io.materialize(output, candidate_path="missing", candidate_sha256="0" * 64)
    data = json.loads((output / io.DATA).read_bytes())
    data["C"]["executed"] = True
    (output / io.DATA).write_text(json.dumps(data), encoding="utf-8")
    with pytest.raises(ValueError, match="bytes"):
        io.validate(output)
