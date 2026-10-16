"""CS2 causal access, local uncertainty and separation of factual evidence."""

from copy import deepcopy
import hashlib
import json

import pytest

from ai_coach.interpretation import (
    candidate_context as candidate,
    common_context as common,
)
from ai_coach.interpretation import (
    common_context_io as io,
    native_place_context as places,
)
from ai_coach.tests.test_candidate_context import fixture, kill_event, rehash, remove_c4


def value(*, death=False, old=False):
    packet, args = fixture(death=death)
    if old:
        args.update(instrument="stratai.review_web@5", extra=None, bomb=None)
    return candidate.build(packet, **args)


def fact(result, key, group, subject=None):
    return next(
        f
        for f in result["situation"][group]
        if f["key"] == key and f["subject"] == subject
    )


def semantic(result):
    # IDs/hash lineage describe the whole retained source, never predictive data.
    result = deepcopy(result)
    result.pop("common_context_id")
    result.pop("source")
    return result


def test_preserves_real_contract_resources_partial_counts_and_exact_reference():
    src = value()
    before = deepcopy(src)
    result = common.build(src)
    common.validate(result, src)
    assert result["history"]["sample_count"] == 5
    count = fact(result, "opponents_count", "players")
    assert count["status"] == "partial" and count["value"]["alive"] is None
    assert count["value"]["unknown"] == 4
    assert (
        common.evidence(result, src, count["evidence"][0])
        == src["packet"]["team_status"][-1]["opponents"]
    )
    actor = src["packet"]["context"]["samples"][-1]["actors"][0]
    assert (
        fact(result, "ammo_in_magazine", "resources", actor["actor_alias"])["value"]
        == 0
    )
    assert fact(result, "walking", "resources", actor["actor_alias"])["value"] is False
    assert (
        fact(result, "health", "resources", actor["actor_alias"])["status"]
        == "not_supplied"
    )
    assert src == before
    count["value"]["unknown"] = 0
    assert src == before


def test_future_mutation_does_not_change_earlier_factual_input():
    src = value()
    early = common.build(src, through_index=2)
    packet = deepcopy(src["packet"])
    packet["context"]["samples"][-1]["actors"][0]["position"]["value"]["x"] = 12345
    rehash(packet)
    changed = candidate.build(
        packet,
        case_id=src["case_id"],
        instrument=src["instrument"],
        **src["supplements"],
    )
    assert semantic(common.build(changed, through_index=2)) == semantic(early)
    with pytest.raises(ValueError, match="outside causal frontier"):
        common.sample(early, src, 3)
    with pytest.raises(ValueError, match="not cited"):
        common.evidence(
            early, src, {"document": "candidate", "pointer": "/snapshot/actors"}
        )
    assert common.sample(early, src, 2) == early["situation"]


def test_forged_frontier_or_changed_summary_rejected():
    src = value()
    result = common.build(src, through_index=1)
    result["through_index"] = 4
    with pytest.raises(ValueError, match="differ"):
        common.sample(result, src, 4)
    result = common.build(src)
    result["evolution"].clear()
    result["common_context_id"] = candidate.content_hash(
        {k: v for k, v in result.items() if k != "common_context_id"}
    )
    with pytest.raises(ValueError, match="differ"):
        common.validate(result, src)


def test_planted_unknown_round_and_conflict_keep_local_limits():
    packet, args = fixture()
    remove_c4(packet)
    packet["objective"][-1]["planted"] = True
    src = candidate.build(packet, **args)
    result = common.build(src)
    assert fact(result, "objective_state", "objective")["value"]["phase"] == "postplant"
    assert fact(result, "opponents_count", "players")["status"] == "partial"
    packet["objective"][-1]["planted"] = None
    for row in packet["round_context"]:
        row["eligibility"].update(
            kind="unknown",
            analysis_allowed=False,
            reason="round_kind_unconfirmed",
            basis_sequence=row["source_sequence"],
        )
    packet["analysis_scope"] = "pending_context"
    src = candidate.build(packet, **args)
    result = common.build(src)
    assert fact(result, "eligibility", "frame")["status"] == "unknown"
    assert result["interpretations"] == []
    assert any(f["key"] == "position" for f in result["situation"]["players"])


def test_missing_resource_limits_only_its_fact():
    src = value()
    packet = deepcopy(src["packet"])
    actor = packet["context"]["samples"][-1]["actors"][0]
    actor["physical"]["ammo_reserve"].update(
        status="unavailable", value=None, reason="native_property_unavailable"
    )
    rehash(packet)
    missing = candidate.build(
        packet,
        case_id=src["case_id"],
        instrument=src["instrument"],
        **src["supplements"],
    )
    before, after = common.build(src), common.build(missing)
    assert (
        fact(after, "ammo_reserve", "resources", actor["actor_alias"])["status"]
        == "unavailable"
    )
    assert before["situation"]["objective"] == after["situation"]["objective"]
    assert before["situation"]["players"] == after["situation"]["players"]


def test_historical_events_age_at_requested_cut_not_source_final_cut():
    packet, args = fixture(death=True)
    args["kills"]["events"] = [kill_event(packet)]
    src = candidate.build(packet, **args)
    assert common.build(src, through_index=3)["events"] == []
    events = common.build(src)["events"]
    assert len(events) == 1 and events[0]["seconds_since_event"] == 0
    assert events[0]["perspective"] == "retrospective_event_not_current_contact"
    assert events[0]["value"]["attacker"]["side"] == "CT"


def test_instrument_five_and_native_supplement_are_separate():
    src = value(old=True)
    result = common.build(src, native=places.unavailable(src))
    assert result["source"]["instrument"] == "stratai.review_web@5"
    assert result["later_supplements"] == []
    assert result["history"]["native_status"] == "unavailable"
    assert src["supplements"]["bomb"] is None
    with pytest.raises(ValueError, match="final cut"):
        common.build(src, through_index=0, native=places.unavailable(src))


def test_ct_structure_does_not_copy_t_tactics_or_ground_bomb_visibility():
    packet, args = fixture()
    remove_c4(packet)
    packet["context"]["perspective"]["side"] = "CT"
    for s in packet["context"]["samples"]:
        for a in s["actors"]:
            a["identity"]["side"] = "CT"
    for counts in packet["team_status"]:
        counts["allies"]["side"], counts["opponents"]["side"] = "CT", "T"
    for name in ("kills", "extra", "bomb"):
        args[name]["side"] = "CT"
    rehash(packet)
    src = candidate.build(packet, **args)
    result = common.build(src)
    assert fact(result, "perspective", "frame")["value"]["side"] == "CT"
    assert result["interpretations"] == result["recommendations"] == []
    assert (
        fact(result, "objective_state", "objective")["value"]["ground"]["position"]
        is None
    )


def test_conflicting_planted_and_carrier_facts_do_not_choose_phase():
    packet, args = fixture()
    packet["objective"][-1]["planted"] = True
    src = candidate.build(packet, **args)
    objective = fact(common.build(src), "objective_state", "objective")
    assert objective["status"] == "conflicting"
    assert objective["value"]["phase"] == "unknown"
    assert objective["derivation"] == "candidate_context._objective@1"


def test_positive_native_values_remain_later_supplements(tmp_path):
    from ai_coach.tests.test_native_place_context import export_fixture, load_args

    src, args, _, _ = export_fixture(tmp_path / "native")
    supplement = places.load(src, **load_args(args))
    result = common.build(src, native=supplement)
    common.validate(result, src, native=supplement)
    assert result["later_supplements"]
    assert all(f["original_instrument"] is False for f in result["later_supplements"])
    assert common.build(src)["situation"] == result["situation"]
    ref = result["later_supplements"][0]["evidence"][0]
    assert (
        common.evidence(result, src, ref, native=supplement)
        == supplement["observations"][0]
    )


@pytest.mark.parametrize("index", [-1, 5, True, 1.5])
def test_invalid_index_is_not_coerced(index):
    with pytest.raises(ValueError, match="exact index"):
        common.build(value(), through_index=index)


def write_source(path):
    raw = json.dumps(value(), sort_keys=True).encode()
    path.write_bytes(raw)
    return hashlib.sha256(raw).hexdigest()


def test_io_roundtrip_and_tampering_not_hidden_by_new_checksum(tmp_path):
    source, output = tmp_path / "source.json", tmp_path / "out"
    digest = write_source(source)
    output.mkdir()
    saved = io.materialize(output, candidate_path=source, candidate_sha256=digest)
    assert io.validate(output) == saved
    path = output / io.DATA
    result = json.loads(path.read_bytes())
    result["uncertainty"].clear()
    raw = io._bytes(result)
    path.write_bytes(raw)
    manifest = json.loads((output / io.MANIFEST).read_bytes())
    manifest["artifacts"][io.DATA] = {
        "sha256": hashlib.sha256(raw).hexdigest(),
        "bytes": len(raw),
    }
    (output / io.MANIFEST).write_bytes(io._bytes(manifest))
    with pytest.raises(ValueError, match="facts or hash"):
        io.validate(output)


def test_io_rejects_unsafe_output_before_source_reads(tmp_path):
    with pytest.raises(ValueError, match="dedicated directory"):
        io.materialize(
            tmp_path.parent.parent, candidate_path="absent", candidate_sha256="invalid"
        )
    (tmp_path / "occupied").write_text("keep")
    with pytest.raises(ValueError, match="empty"):
        io.materialize(tmp_path, candidate_path="absent", candidate_sha256="invalid")


def test_changed_input_bytes_are_rejected(tmp_path):
    source, output = tmp_path / "source.json", tmp_path / "out"
    digest = write_source(source)
    output.mkdir()
    io.materialize(output, candidate_path=source, candidate_sha256=digest)
    source.write_text("{}")
    with pytest.raises(ValueError, match="byte hash"):
        io.validate(output)
