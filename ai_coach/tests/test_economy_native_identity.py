from __future__ import annotations

import copy

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import native_identity as domain


def fixture():
    identity = {
        "source_alias": "demo-" + "a" * 20,
        "source_sha256": "a" * 64,
        "build_identifier": "fixture",
    }
    trace = {
        **identity,
        "observations": {
            "rows": [
                {
                    "sequence": 1,
                    "frame": 2,
                    "tick": 12,
                    "source": "item_pickup",
                    "channel": "native_event",
                    "actor_alias": "p0001",
                    "side": "CT",
                    "instance": 2,
                    "value": {"server_tick": 102},
                }
            ]
        },
    }
    candidate = {
        **identity,
        "rounds": [{"instance": 2, "start_tick": 10, "freeze_end_tick": 20}],
        "initial_frontiers": [{"instance": 2, "official_end_tick": 40}],
    }
    rows = []

    def add(seq, kind, source, entity, value, frame=1):
        rows.append(
            {
                "sequence": seq,
                "frame": frame,
                "demo_tick": 10 if frame == 1 else 12,
                "clock_sequence": 1 if frame == 1 else 6,
                "kind": kind,
                "source": source,
                "entity_alias": entity,
                "actor_alias": None,
                "side": None,
                "instance": 2,
                "value": value,
            }
        )

    pawn, equipment, reused = (f"native-entity-{i:06d}" for i in (1, 2, 3))
    add(
        2,
        "entity_created",
        "CCSPlayerPawn",
        pawn,
        {"entity_index": 1, "entity_serial": 2, "entity_kind": "pawn"},
    )
    add(
        3,
        "entity_created",
        "CWeaponAK47",
        equipment,
        {"entity_index": 5, "entity_serial": 20000, "entity_kind": "equipment"},
    )
    add(
        4,
        "reference_baseline",
        "m_hOwnerEntity",
        equipment,
        {
            "phase": "creation_frame_or_replay",
            "reference": {
                "handle": 2 << 14 | 1,
                "status": "resolved",
                "target_alias": pawn,
            },
        },
    )
    add(
        5,
        "reference_baseline",
        domain.INVENTORY + ".0000",
        pawn,
        {
            "phase": "creation_frame_or_replay",
            "reference": {
                "handle": 20000 << 14 | 5,
                "status": "resolved",
                "target_alias": equipment,
            },
        },
    )
    add(
        7,
        "trace_event",
        "item_pickup",
        None,
        {"trace_sequence": 1, "server_tick": 102},
        2,
    )
    rows[-1].update(actor_alias="p0001", side="CT")
    add(8, "entity_destroyed", "CWeaponAK47", equipment, None, 2)
    add(
        9,
        "reference_callback",
        domain.INVENTORY + ".0000",
        pawn,
        {
            "phase": "later_frame_callback",
            "reference": {
                "handle": 20000 << 14 | 5,
                "status": "entity_unavailable",
                "target_alias": None,
            },
        },
        2,
    )
    add(
        10,
        "entity_created",
        "CWeaponAK47",
        reused,
        {"entity_index": 5, "entity_serial": 20001, "entity_kind": "equipment"},
        2,
    )
    add(
        11,
        "reference_callback",
        domain.INVENTORY + ".0000",
        pawn,
        {
            "phase": "later_frame_callback",
            "reference": {
                "handle": 20000 << 14 | 5,
                "status": "serial_mismatch",
                "target_alias": None,
            },
        },
        2,
    )
    data = {
        **identity,
        "schema_id": "stratai.economy_native_identity@1",
        "contract_version": "1.0.0-candidate.1",
        "source_trace": {
            "path": "economy_transaction_trace.json",
            "bytes": 1,
            "sha256": "b" * 64,
        },
        "sdk_version": "demoinfocs-golang/v5@v5.2.0",
        "order_contract": "sdk_delivery_after_priority_sort_not_wire_or_server_subtick",
        "partition": "execution_evidence_only_no_initial_features_or_player_knowledge",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "observations": {
            "clocks": [
                {
                    "sequence": 1,
                    "frame": 1,
                    "demo_tick": 10,
                    "network_tick": 100,
                    "epoch": 1,
                    "epoch_reason": "first_sample",
                },
                {
                    "sequence": 6,
                    "frame": 2,
                    "demo_tick": 12,
                    "network_tick": 102,
                    "epoch": 1,
                    "epoch_reason": "continuous",
                },
            ],
            "rows": rows,
            "user_message_counts": {s: 0 for s in domain.USER_MESSAGES},
        },
    }
    return data, trace, candidate


def test_lifetime_serial_native_clock_and_no_future_features():
    data, trace, candidate = fixture()
    before = copy.deepcopy((trace, candidate))
    report = domain.validate_data(data, trace, candidate, require_observed=True)
    assert report["native_event_clock_comparison"] == {"equal_in_same_frame": 1}
    assert report["reference_counts"]["equipment:serial_mismatch"] == 1
    assert report["reference_counts"]["equipment:entity_unavailable"] == 1
    assert report["entity_lifetimes"] == 3
    assert report["itemized_transactions_verified"] == 0
    assert (trace, candidate) == before


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("training_allowed",), True),
        (("comparison_allowed",), True),
        (("product_supported",), True),
        (("source_sha256",), "c" * 64),
        (("observations", "clocks", 1, "epoch"), 2),
        (("observations", "clocks", 1, "network_tick"), 99),
        (("observations", "rows", 0, "clock_sequence"), 6),
        (("observations", "rows", 1, "entity_alias"), "native-entity-000001"),
        (("observations", "rows", 2, "sequence"), 3),
        (("observations", "rows", 2, "entity_alias"), "native-entity-999999"),
        (("observations", "rows", 2, "value", "phase"), "later_frame_callback"),
        (("observations", "rows", 3, "value", "reference", "handle"), 5),
        (
            ("observations", "rows", 3, "value", "reference", "target_alias"),
            "native-entity-000003",
        ),
        (("observations", "rows", 4, "value", "server_tick"), 101),
        (("observations", "rows", 4, "value", "trace_sequence"), 99),
        (("observations", "rows", 4, "actor_alias"), "p9999"),
        (("observations", "rows", 4, "instance"), 999),
    ],
)
def test_rejects_false_identity_epoch_or_provenance(path, value):
    data, trace, candidate = fixture()
    obj = data
    for key in path[:-1]:
        obj = obj[key]
    obj[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, trace, candidate)


def test_clock_reset_and_missing_tick_are_boundaries_not_guessed_offsets():
    data, trace, candidate = fixture()
    data["observations"]["clocks"][1].update(
        network_tick=99, epoch=2, epoch_reason="network_tick_regression"
    )
    report = domain.validate_data(data, trace, candidate)
    assert report["native_event_clock_comparison"] == {"different_in_same_frame": 1}
    data["observations"]["clocks"][1].update(
        network_tick=None, epoch_reason="missing_tick_boundary"
    )
    report = domain.validate_data(data, trace, candidate)
    assert report["native_event_clock_comparison"] == {"network_tick_absent": 1}


def test_message_item_identity_does_not_become_entity_or_actor():
    data, trace, candidate = fixture()
    row = copy.deepcopy(data["observations"]["rows"][-1])
    row.update(
        sequence=12,
        kind="user_message",
        source="CCSUsrMsg_ItemDrop",
        entity_alias=None,
        value={
            "item_id_alias": "native-item-000001",
            "death": False,
            "actor_status": "not_in_message",
            "entity_binding": "not_accredited",
        },
    )
    data["observations"]["rows"].append(row)
    data["observations"]["user_message_counts"]["CCSUsrMsg_ItemDrop"] = 1
    assert (
        domain.validate_data(data, trace, candidate)["user_message_counts"][
            "CCSUsrMsg_ItemDrop"
        ]
        == 1
    )
    row["actor_alias"] = "p0001"
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, trace, candidate)


def test_existing_round_boundary_is_verified_by_instance():
    data, trace, candidate = fixture()
    row = copy.deepcopy(data["observations"]["rows"][-1])
    row.update(
        sequence=12,
        kind="round_boundary",
        source="RoundFreezetimeEnd",
        entity_alias=None,
        value=None,
    )
    data["observations"]["rows"].append(row)
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, trace, candidate)
    candidate["rounds"][0]["freeze_end_tick"] = 12
    domain.validate_data(data, trace, candidate)


def test_output_preflight_rejects_nonempty_and_repository(tmp_path):
    (tmp_path / "preserve").write_text("existing")
    for path in (tmp_path, domain.candidate_io.CONTRACTS):
        with pytest.raises(RuntimeContractError):
            domain.candidate_io.validate_output_directory(path)
