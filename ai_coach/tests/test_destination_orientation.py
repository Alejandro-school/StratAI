"""Provisional ordinal ranking respects factual limits and carrier invariance."""

from copy import deepcopy

import pytest
from jsonschema import ValidationError

from ai_coach.interpretation import candidate_context
from ai_coach.interpretation import destination_orientation as orientation
from ai_coach.tests.test_candidate_context import fixture, rehash, remove_c4


def inputs(*, complete=True, case_id="E03"):
    packet, kwargs = fixture()
    if complete:
        dead_packet, _ = fixture(death=True)
        dead = dead_packet["context"]["samples"][-1]["actors"][1]
        inventory = dead_packet["inventory"][-1][dead["actor_alias"]]
        for sample, inv, counts in zip(
            packet["context"]["samples"],
            packet["inventory"],
            packet["team_status"],
            strict=True,
        ):
            for number in range(3, 6):
                actor = deepcopy(dead)
                actor["actor_alias"] = "support-actor-" + str(number) * 20
                actor["identity"]["pawn_handle"] = 16384 + number
                actor["identity"]["pawn_life"] = f"support-pawn-{number:06}"
                for ref in actor["references"]:
                    ref.update(
                        actor_alias=actor["actor_alias"],
                        source_sequence=sample["cut"]["sequence"],
                        availability_frame=sample["cut"]["frame"],
                        availability_tick=sample["cut"]["tick"],
                    )
                sample["actors"].append(actor)
                inv[actor["actor_alias"]] = {
                    **deepcopy(inventory),
                    "source_sequence": sample["cut"]["sequence"],
                }
            counts["allies"].update(
                status="observed",
                alive=2,
                observed_alive=2,
                observed_dead=3,
                unknown=0,
                reason=None,
            )
    kwargs["case_id"] = case_id
    for name in ("kills", "extra", "bomb"):
        kwargs[name]["case_id"] = case_id
    rehash(packet)
    return packet, kwargs


def seal_native(value):
    value["native_place_context_id"] = candidate_context.content_hash(
        {k: v for k, v in value.items() if k != "native_place_context_id"}
    )
    return value


def native(candidate, names=("BombsiteA", "Middle")):
    cut = candidate["snapshot"]["cut"]
    observations = []
    for index, actor in enumerate(candidate["snapshot"]["actors"]):
        identity = actor["identity"]
        live = actor["alive"]["value"] is True
        value = names[index] if live else None
        observations.append(
            {
                "actor_alias": actor["actor_alias"],
                "label": actor["label"],
                "pawn_life": identity["pawn_life"],
                "pawn_handle": identity["pawn_handle"],
                "side": identity["side"],
                "alive": actor["alive"]["value"],
                "connected": actor["connected"],
                "position": deepcopy(actor["position"]["value"]),
                "property": "m_szLastPlaceName",
                "value_status": "observed" if live else "unavailable",
                "value": value,
                "native_type": "string" if live else None,
                "last_delivery": (
                    {
                        **{
                            k: cut[k]
                            for k in (
                                "frame",
                                "tick",
                                "epoch",
                                "instance",
                                "round_number",
                            )
                        },
                        "value_status": "observed",
                        "value": value,
                    }
                    if live
                    else None
                ),
            }
        )
    return seal_native(
        {
            "schema_id": "stratai.native_place_context@1",
            "candidate_context_id": candidate["candidate_context_id"],
            "source_demo_sha256": candidate["packet"]["context"]["source_sha256"],
            "cut": deepcopy(cut),
            "status": "observed",
            "reason": None,
            "observations": observations,
            "provenance": {
                "manifest_sha256": "a" * 64,
                "snapshot_sha256": "b" * 64,
                "targets_sha256": "c" * 64,
                "candidate_context_sha256": "d" * 64,
                "target_id": "constructed-control",
                "map_name": "de_mirage",
                "build": 10847,
                "patch": 14176,
                "source_semantics": "current_native_property_not_geometric_zone",
            },
        }
    )


def result(packet=None, kwargs=None, *, names=("BombsiteA", "Middle")):
    if packet is None:
        packet, kwargs = inputs()
    value = candidate_context.build(packet, **kwargs)
    return orientation.infer(value, native(value, names))


def decision(value):
    return {
        k: value[k]
        for k in (
            "status",
            "preferred_site",
            "preferred_qualification",
            "alternatives",
            "reasons",
        )
    }


@pytest.mark.parametrize("site,other", [("A", "B"), ("B", "A")])
def test_symmetric_nominal_site_hypothesis_keeps_alternative_possible(site, other):
    value = result(names=("Bombsite" + site, "Middle"))
    assert value["status"] == "hypothesis"
    assert value["preferred_site"] == site
    assert value["preferred_qualification"] == "more_likely"
    assert value["alternatives"] == [
        {"site": other, "qualification": "possible_not_excluded"}
    ]
    assert value["prepare_entry_assessed"] is False
    assert value["training_allowed"] is False
    assert value["product_supported"] is False
    assert orientation.validate(value) == value


def test_c4_swap_does_not_change_destination_or_explanation():
    packet, kwargs = inputs()
    before = result(packet, kwargs)
    for sample, objective in zip(
        packet["context"]["samples"], packet["objective"], strict=True
    ):
        first, second = sample["actors"][:2]
        first["has_c4"]["value"] = False
        second["has_c4"]["value"] = True
        objective["carrier_aliases"] = [second["actor_alias"]]
    rehash(packet)
    after = result(packet, kwargs)
    assert decision(before) == decision(after)
    assert before["inference"] == after["inference"]


def test_case_id_label_order_and_elapsed_time_do_not_choose_site():
    packet, kwargs = inputs(case_id="E36")
    for row in packet["round_context"]:
        row["round_elapsed_seconds"]["value"] += 90
    later_clock = result(packet, kwargs)
    baseline = result()
    assert decision(later_clock) == decision(baseline)
    reversed_positions = result(names=("Middle", "BombsiteA"))
    assert decision(reversed_positions) == decision(baseline)
    assert (
        reversed_positions["inference"]["site_ally"]
        != baseline["inference"]["site_ally"]
    )


def test_no_universal_delivery_age_expiry_or_preparation_time_threshold():
    packet, kwargs = inputs()
    candidate = candidate_context.build(packet, **kwargs)
    data = native(candidate)
    before = orientation.infer(candidate, data)
    for row in data["observations"][:2]:
        row["last_delivery"]["frame"] = 1
        row["last_delivery"]["tick"] = 1
    seal_native(data)
    after = orientation.infer(candidate, data)
    assert decision(before) == decision(after)
    assert after["observed_facts"]["native_places"][0]["last_delivery"]["frame"] == 1


@pytest.mark.parametrize(
    "names",
    [
        ("TSpawn", "TSpawn"),
        ("Middle", "Middle"),
        ("BombsiteA", "BombsiteB"),
        ("BombsiteA", "BombsiteA"),
        ("A", "Middle"),
        ("BombsiteA", "Palace"),
    ],
)
def test_other_arrangements_do_not_gain_a_destination(names):
    value = result(names=names)
    assert value["status"] == "abstain"
    assert value["preferred_site"] is None
    assert value["reasons"] == ["configuration_not_covered_more_observation_required"]
    assert value["prepare_entry_assessed"] is False


def test_partial_counts_do_not_make_two_observed_allies_the_only_survivors():
    packet, kwargs = inputs(complete=False)
    value = result(packet, kwargs)
    assert value["status"] == "abstain"
    assert value["reasons"] == ["ally_survivor_count_incomplete"]


def test_full_initial_team_does_not_emit_from_distribution_or_elapsed_time():
    packet, kwargs = inputs()
    for sample, inv, counts, round_row in zip(
        packet["context"]["samples"],
        packet["inventory"],
        packet["team_status"],
        packet["round_context"],
        strict=True,
    ):
        for index in range(2, 5):
            dead = sample["actors"][index]
            actor = deepcopy(sample["actors"][1])
            actor["actor_alias"] = dead["actor_alias"]
            actor["identity"] = dead["identity"]
            for ref in actor["references"]:
                ref["actor_alias"] = actor["actor_alias"]
            sample["actors"][index] = actor
            inv[actor["actor_alias"]] = deepcopy(
                inv[sample["actors"][1]["actor_alias"]]
            )
        counts["allies"].update(alive=5, observed_alive=5, observed_dead=0)
        round_row["round_elapsed_seconds"]["value"] = 4
    rehash(packet)
    value = result(packet, kwargs, names=("TSpawn",) * 5)
    assert value["status"] == "abstain"
    assert value["reasons"] == ["configuration_not_covered_more_observation_required"]
    for row in packet["round_context"]:
        row["round_elapsed_seconds"]["value"] = 40
    later = result(packet, kwargs, names=("TSpawn",) * 5)
    assert decision(later) == decision(value)


def test_opponent_counts_are_not_a_destination_feature():
    packet, kwargs = inputs()
    before = result(packet, kwargs)
    for counts in packet["team_status"]:
        counts["opponents"].update(
            status="observed",
            alive=5,
            observed_alive=5,
            observed_dead=0,
            unknown=0,
            reason=None,
        )
    after = result(packet, kwargs)
    assert decision(after) == decision(before)
    assert after["inference"] == before["inference"]


def test_dead_ally_does_not_supply_a_destination():
    packet, kwargs = inputs()
    for sample, inv, counts in zip(
        packet["context"]["samples"],
        packet["inventory"],
        packet["team_status"],
        strict=True,
    ):
        dead = deepcopy(sample["actors"][2])
        dead.update(actor_alias=sample["actors"][1]["actor_alias"])
        dead["identity"] = deepcopy(sample["actors"][1]["identity"])
        for ref in dead["references"]:
            ref["actor_alias"] = dead["actor_alias"]
        sample["actors"][1] = dead
        inv[dead["actor_alias"]] = deepcopy(inv[sample["actors"][2]["actor_alias"]])
        counts["allies"].update(alive=1, observed_alive=1, observed_dead=4)
    rehash(packet)
    value = result(packet, kwargs)
    assert value["status"] == "abstain"
    assert value["preferred_site"] is None


def test_unknown_eligibility_remains_unknown_not_knife():
    packet, kwargs = inputs()
    for row in packet["round_context"]:
        row["eligibility"].update(
            kind="unknown", analysis_allowed=False, reason="round_kind_unconfirmed"
        )
    packet["analysis_scope"] = "pending_context"
    value = result(packet, kwargs)
    assert value["status"] == "abstain"
    assert value["reasons"] == ["round_eligibility_unknown"]
    assert value["observed_facts"]["eligibility_kind"] == "unknown"


def test_confirmed_postplant_is_out_of_scope():
    packet, kwargs = inputs()
    remove_c4(packet)
    packet["objective"][-1]["planted"] = True
    value = result(packet, kwargs)
    assert value["status"] == "out_of_scope"
    assert value["reasons"] == ["postplant_outside_preplant_orientation"]
    assert value["preferred_site"] is None


def test_unknown_objective_and_missing_native_source_abstain_separately():
    packet, kwargs = inputs()
    candidate = candidate_context.build(packet, **kwargs)
    missing = orientation.infer(candidate)
    assert missing["reasons"] == ["native_source_not_materialized"]
    remove_c4(packet)
    unknown = result(packet, kwargs)
    assert unknown["reasons"] == ["objective_phase_unknown_or_conflicting"]


@pytest.mark.parametrize("mutation", ["life", "cut", "position", "future", "binding"])
def test_misaligned_or_future_native_facts_rejected_even_after_rehash(mutation):
    packet, kwargs = inputs()
    candidate = candidate_context.build(packet, **kwargs)
    data = native(candidate)
    if mutation == "life":
        data["observations"][0]["pawn_life"] = "different-life"
    elif mutation == "cut":
        data["cut"]["tick"] += 1
    elif mutation == "position":
        data["observations"][0]["position"]["x"] += 1
    elif mutation == "future":
        data["observations"][0]["last_delivery"]["tick"] += 1
    else:
        data["candidate_context_id"] = "f" * 64
    seal_native(data)
    with pytest.raises((ValueError, ValidationError)):
        orientation.infer(candidate, data)


def test_native_absence_is_not_a_blank_site_or_geometric_guess():
    packet, kwargs = inputs()
    candidate = candidate_context.build(packet, **kwargs)
    data = native(candidate)
    data["observations"][0].update(
        value_status="absent_property", value=None, native_type=None, last_delivery=None
    )
    seal_native(data)
    value = orientation.infer(candidate, data)
    assert value["status"] == "abstain"
    assert value["reasons"] == ["current_native_place_value_unavailable"]


def test_hidden_or_future_payload_is_not_an_input_and_inputs_are_immutable():
    packet, kwargs = inputs()
    candidate = candidate_context.build(packet, **kwargs)
    data = native(candidate)
    before = deepcopy((candidate, data))
    orientation.infer(candidate, data)
    assert (candidate, data) == before
    for field in ("human_response", "frozen_prediction", "hidden_opponents", "future"):
        forged = deepcopy(candidate)
        forged[field] = {"site": "B"}
        forged["candidate_context_id"] = candidate_context.content_hash(
            {k: v for k, v in forged.items() if k != "candidate_context_id"}
        )
        with pytest.raises(ValidationError):
            orientation.infer(forged, data)


@pytest.mark.parametrize(
    "field,altered",
    [
        ("preferred_site", "B"),
        ("alternatives", []),
        ("prepare_entry_assessed", True),
    ],
)
def test_changed_output_cannot_claim_other_site_or_preparation_even_rehashed(
    field, altered
):
    value = result()
    value[field] = altered
    value["orientation_id"] = candidate_context.content_hash(
        {k: v for k, v in value.items() if k != "orientation_id"}
    )
    with pytest.raises((ValueError, ValidationError)):
        orientation.validate(value)
