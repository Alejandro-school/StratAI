from __future__ import annotations

import copy

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import transaction_trace as trace


def fixture():
    identity = {
        "source_alias": "demo-" + "a" * 20,
        "source_sha256": "a" * 64,
        "build_identifier": "fixture",
    }
    candidate = dict(
        **identity,
        participants=[{"participant_alias": "p0001"}],
        rounds=[
            {"instance": 2, "round_number": 1, "start_tick": 10, "freeze_end_tick": 20}
        ],
        initial_frontiers=[
            {
                "instance": 2,
                "round_start_probe": {
                    "actors": [{"actor_alias": "p0001", "side": "CT"}]
                },
            }
        ],
    )
    rows = [
        {
            "sequence": i + 1,
            "tick": tick,
            "frame": frame,
            "channel": channel,
            "source": trace.ACCOUNT,
            "actor_alias": "p0001",
            "side": "CT",
            "entity_alias": "trace-entity-000001",
            "instance": 2,
            "value": {"status": "observed", "value": value},
        }
        for i, (tick, frame, channel, value) in enumerate(
            [
                (10, 1, "property_baseline", 800),
                (12, 2, "property_update", 150),
                (12, 2, "property_update", 800),
                (12, 2, "property_update", 800),
            ]
        )
    ]
    rows.append(
        {
            "sequence": 5,
            "tick": 20,
            "frame": 3,
            "channel": "sdk_callback",
            "source": "ItemPickup",
            "actor_alias": "p0001",
            "side": "CT",
            "entity_alias": None,
            "instance": 2,
            "value": {"item": "P250"},
        }
    )
    data = dict(
        **identity,
        schema_id="stratai.economy_transaction_trace@1",
        contract_version="1.0.0-candidate.1",
        source_candidate={
            "path": "economy_candidate_manifest.json",
            "bytes": 1,
            "sha256": "b" * 64,
        },
        sdk_version="demoinfocs-golang/v5@v5.2.0",
        order_contract="observer_delivery_sequence_not_server_subtick",
        partition="execution_evidence_only_no_player_knowledge_or_initial_features",
        comparison_allowed=False,
        training_allowed=False,
        product_supported=False,
        observations={
            "rows": rows,
            "unmapped_native_events": 0,
            "native_descriptor_lists": 1,
        },
    )
    return data, candidate


def test_compensated_and_duplicate_updates_survive_without_fake_transactions():
    data, candidate = fixture()
    before = copy.deepcopy(candidate)
    result = trace.validate_data(data, candidate, require_observed=True)
    assert result["rows"] == 5
    assert result["same_frame_tick_distinct_account_groups"] == 1
    assert result["same_frame_tick_compensated_account_groups"] == 1
    assert list(result["linked_opportunities"].values()) == [3]
    assert result["verified_itemized_transactions"] == 0
    assert candidate == before


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("source_sha256",), "c" * 64),
        (("comparison_allowed",), True),
        (("training_allowed",), True),
        (("product_supported",), True),
        (("observations", "rows", 1, "sequence"), 1),
        (("observations", "rows", 1, "frame"), 0),
        (("observations", "rows", 1, "actor_alias"), "p9999"),
        (("observations", "rows", 1, "instance"), 1),
        (("observations", "rows", 1, "entity_alias"), None),
        (("observations", "rows", 1, "value", "status"), "absent"),
        (("observations", "rows", 1, "value", "value"), False),
        (("observations", "rows", 1, "source"), "guessed"),
    ],
)
def test_trace_rejects_corruption(path, value):
    data, candidate = fixture()
    current = data
    for key in path[:-1]:
        current = current[key]
    current[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        trace.validate_data(data, candidate)


def test_missing_identity_and_roster_disagreement_do_not_join():
    data, candidate = fixture()
    data["observations"]["rows"][1]["actor_alias"] = None
    data["observations"]["rows"][2]["side"] = "T"
    report = trace.validate_data(data, candidate)
    assert list(report["linked_opportunities"].values()) == [1]
    assert report["abstentions"]["opportunity_roster_disagreement"] == 1
    assert report["abstentions"]["actor_unavailable_at_observation"] == 1


def test_native_repeated_fields_false_and_missing_channels_remain_explicit():
    data, candidate = fixture()
    row = data["observations"]["rows"][-1]
    row.update(
        channel="native_event",
        source="item_pickup",
        value={
            "server_tick": None,
            "fields": [
                {
                    "ordinal": i,
                    "name": "silent",
                    "descriptor_type": 6,
                    "wire_type": 6,
                    "status": "observed",
                    "value": False,
                }
                for i in range(2)
            ],
        },
    )
    report = trace.validate_data(data, candidate)
    assert report["channel_counts"]["native_event:item_pickup"] == 1
    assert "sdk_callback:ItemPickup" not in report["channel_counts"]
    row["value"]["fields"][0].update(name="userid", value=123)
    with pytest.raises(RuntimeContractError):
        trace.validate_data(data, candidate)


def test_real_smoke_requires_affected_channels():
    data, candidate = fixture()
    data["observations"]["rows"] = []
    with pytest.raises(RuntimeContractError):
        trace.validate_data(data, candidate, require_observed=True)


def test_read_preflight_rejects_populated_and_non_temporary(tmp_path):
    (tmp_path / "existing").write_text("protected")
    with pytest.raises(RuntimeContractError):
        trace.candidate_io.validate_output_directory(tmp_path)
    with pytest.raises(RuntimeContractError):
        trace.candidate_io.validate_output_directory(trace.candidate_io.CONTRACTS)
