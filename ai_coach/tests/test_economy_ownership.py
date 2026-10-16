from __future__ import annotations

import copy

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import ownership as domain


def fixture():
    meta = {
        "source_alias": "demo-" + "a" * 20,
        "source_sha256": "a" * 64,
        "build_identifier": "fixture",
    }
    a, b, weapon = [f"native-entity-{n:06d}" for n in (1, 2, 3)]

    def ref(index, serial, alias):
        return {
            "handle": serial << 14 | index,
            "status": "resolved",
            "target_alias": alias,
        }

    def pawn(alias, actor, slots):
        return {
            "entity_alias": alias,
            "actor_alias": actor,
            "side": "T",
            "inventory_status": "observed",
            "inventory_length": len(slots),
            "slots": slots,
        }

    def state(seq, owner, first_slots, second_slots):
        return {
            "sequence": seq,
            "first_cut": seq,
            "pawns": [pawn(a, "p0001", first_slots), pawn(b, "p0002", second_slots)],
            "equipment": [{"entity_alias": weapon, "owner": owner}],
        }

    inventory = ref(5, 20000, weapon)
    states = [
        state(1, ref(1, 2, a), [inventory], []),
        state(2, ref(2, 3, b), [], [copy.deepcopy(inventory)]),
    ]
    candidate = {
        **meta,
        "rounds": [{"instance": 1, "start_tick": 0, "freeze_end_tick": 10}],
        "initial_frontiers": [
            {
                "instance": 1,
                "round_link": {"status": "verified"},
                "round_start_probe": {
                    "actors": [
                        {"actor_alias": "p0001", "side": "T"},
                        {"actor_alias": "p0002", "side": "T"},
                    ]
                },
            }
        ],
    }
    identity = {
        **meta,
        "observations": {
            "clocks": [
                {
                    "sequence": 1,
                    "frame": 1,
                    "demo_tick": 1,
                    "network_tick": 100,
                    "epoch": 1,
                },
                {
                    "sequence": 5,
                    "frame": 2,
                    "demo_tick": 2,
                    "network_tick": 101,
                    "epoch": 1,
                },
            ],
            "rows": [
                {
                    "sequence": n,
                    "frame": 1,
                    "kind": "entity_created",
                    "entity_alias": alias,
                    "actor_alias": None,
                    "value": {
                        "entity_index": index,
                        "entity_serial": serial,
                        "entity_kind": kind,
                    },
                }
                for n, alias, index, serial, kind in [
                    (2, a, 1, 2, "pawn"),
                    (3, b, 2, 3, "pawn"),
                    (4, weapon, 5, 20000, "equipment"),
                ]
            ],
        },
    }
    data = {
        **meta,
        "schema_id": "stratai.economy_ownership_observations@1",
        "contract_version": "1.0.0-candidate.1",
        "source_identity": {
            "path": "economy_native_identity.json",
            "bytes": 1,
            "sha256": "b" * 64,
        },
        "cut_contract": "fresh_FrameDone_after_queued_messages_and_frame_events_native_freeze_strict_instance_window",
        "partition": "execution_evidence_only_no_initial_features_or_player_knowledge",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "observations": {
            "cuts": [
                {
                    "sequence": 1,
                    "frame": 1,
                    "demo_tick": 1,
                    "instance": 1,
                    "identity_sequence": 4,
                    "clock_sequence": 1,
                    "state_sequence": 1,
                },
                {
                    "sequence": 2,
                    "frame": 2,
                    "demo_tick": 2,
                    "instance": 1,
                    "identity_sequence": 5,
                    "clock_sequence": 5,
                    "state_sequence": 2,
                },
            ],
            "states": states,
        },
    }
    return data, identity, candidate


def test_fresh_bilateral_holding_change_has_no_transaction_cause():
    data, identity, candidate = fixture()
    before = copy.deepcopy((identity, candidate))
    report = domain.build_report(
        data,
        identity,
        candidate,
        {"path": domain.DATA, "bytes": 1, "sha256": "c" * 64},
        require_observed=True,
    )
    assert report["summary"]["status_counts"]["consistent"] == 2
    assert report["summary"]["observed_holding_transitions"] == 1
    assert report["transitions"][0]["cause"] == "not_accredited"
    assert report["summary"]["verified_itemized_transactions"] == 0
    assert (identity, candidate) == before
    report["transitions"][0]["cause"] = "gift"
    with pytest.raises(RuntimeContractError):
        domain.validate_report(
            report,
            data,
            identity,
            candidate,
            {"path": domain.DATA, "bytes": 1, "sha256": "c" * 64},
        )


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("comparison_allowed",), True),
        (("training_allowed",), True),
        (("product_supported",), True),
        (("source_sha256",), "d" * 64),
        (("observations", "cuts", 0, "identity_sequence"), 5),
        (("observations", "cuts", 0, "clock_sequence"), 5),
        (("observations", "cuts", 1, "frame"), 1),
        (("observations", "cuts", 1, "instance"), 99),
        (("observations", "cuts", 0, "demo_tick"), 0),
        (("observations", "cuts", 1, "state_sequence"), 1),
        (("observations", "states", 1, "first_cut"), 3),
        (("observations", "states", 0, "pawns", 0, "inventory_length"), 0),
        (("observations", "states", 0, "pawns", 0, "inventory_status"), "absent"),
        (("observations", "states", 0, "pawns", 0, "actor_alias"), "p9999"),
        (("observations", "states", 0, "equipment", 0, "owner", "handle"), 1),
        (
            ("observations", "states", 0, "equipment", 0, "entity_alias"),
            "native-entity-000004",
        ),
    ],
)
def test_rejects_wrong_prefix_future_identity_slots_or_provenance(path, value):
    data, identity, candidate = fixture()
    target = data
    for key in path[:-1]:
        target = target[key]
    target[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        domain.reconcile(data, identity, candidate)


@pytest.mark.parametrize(
    "case",
    [
        "conflict",
        "duplicate_slot",
        "missing_inventory",
        "late_actor",
        "nonunique_actor",
        "clock_missing",
        "epoch_reset",
        "frame_gap",
    ],
)
def test_disagreement_absence_and_time_gaps_do_not_become_holder_changes(case):
    data, identity, candidate = fixture()
    states = data["observations"]["states"]
    if case == "conflict":
        states[1]["equipment"][0]["owner"] = copy.deepcopy(
            states[0]["equipment"][0]["owner"]
        )
    elif case == "duplicate_slot":
        states[1]["pawns"][1]["slots"] *= 2
        states[1]["pawns"][1]["inventory_length"] = 2
    elif case == "missing_inventory":
        states[1]["pawns"][0].update(inventory_status="absent", inventory_length=None)
    elif case == "late_actor":
        states[0]["pawns"][0]["actor_alias"] = None
    elif case == "nonunique_actor":
        states[1]["pawns"][0]["actor_alias"] = "p0002"
    elif case == "clock_missing":
        identity["observations"]["clocks"][1]["network_tick"] = None
    elif case == "epoch_reset":
        identity["observations"]["clocks"][1].update(network_tick=1, epoch=2)
    else:
        identity["observations"]["clocks"][1].update(frame=3, demo_tick=3)
        data["observations"]["cuts"][1].update(frame=3, demo_tick=3)
    result = domain.reconcile(data, identity, candidate)
    assert not result["transitions"]
    if case == "late_actor":
        assert result["states"][0]["records"][0]["actor_alias"] is None
    if case == "conflict":
        assert result["summary"]["status_counts"]["owner_inventory_conflict"] == 1


def test_index_replacement_without_destroy_does_not_join_two_lives():
    data, identity, candidate = fixture()
    entity = copy.deepcopy(identity["observations"]["rows"][-1])
    entity.update(sequence=6, frame=2, entity_alias="native-entity-000004")
    entity["value"]["entity_serial"] += 1
    identity["observations"]["rows"].append(entity)
    data["observations"]["cuts"][1]["identity_sequence"] = 6
    state = data["observations"]["states"][1]
    state["equipment"][0]["entity_alias"] = entity["entity_alias"]
    state["pawns"][1]["slots"][0].update(
        handle=20001 << 14 | 5, target_alias=entity["entity_alias"]
    )
    report = domain.reconcile(data, identity, candidate)
    assert report["summary"]["status_counts"]["consistent"] == 2
    assert not report["transitions"]


def test_output_guard_runs_before_missing_input_read(tmp_path):
    (tmp_path / "preserve").write_text("existing")
    with pytest.raises(RuntimeContractError, match="empty"):
        domain.materialize(tmp_path / "missing.json", tmp_path)


def test_observed_unassignment_is_not_a_voluntary_drop():
    data, identity, candidate = fixture()
    state = data["observations"]["states"][1]
    state["pawns"][1].update(inventory_length=0, slots=[])
    state["equipment"][0]["owner"] = {
        "handle": 4294967295,
        "status": "invalid_sentinel",
        "target_alias": None,
    }
    result = domain.reconcile(data, identity, candidate)
    transition = result["transitions"][0]
    assert (
        transition["classification"]
        == "holding_disappeared_between_adjacent_delivery_cuts"
    )
    assert transition["to_actor"] is None and transition["cause"] == "not_accredited"
    state["equipment"][0]["owner"] = {
        "handle": None,
        "status": "absent",
        "target_alias": None,
    }
    assert not domain.reconcile(data, identity, candidate)["transitions"]
