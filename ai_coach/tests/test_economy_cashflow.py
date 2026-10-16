from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import cashflow as domain
from ai_coach.tests.test_economy_entry_identity import fixture as entry_fixture
from ai_coach.tests.test_economy_entry_identity import scalar


def fixture():
    entries, own, identity, candidate = entry_fixture()
    own["observations"]["states"][1] = copy.deepcopy(own["observations"]["states"][0])
    own["observations"]["states"][1].update(sequence=2, first_cut=2)
    entries["observations"]["states"][0]["pawns"][0].update(length=0, entries=[])
    entries["observations"]["states"][1]["pawns"][0]["entries"][0]["fields"][
        "m_nCost"
    ] = scalar(300)
    meta = {
        k: entries[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    controllers = [
        {
            "alias": f"trace-entity-{i:06d}",
            "index": 100 + i,
            "serial": 20000 + i,
            "created_frame": 0,
            "destroyed_frame": None,
        }
        for i in (1, 2)
    ]
    rows = []
    deliveries = []

    def row(controller, key, value, frame, channel):
        p = controller - 1
        seq = len(rows) + 1
        rows.append(
            {
                "sequence": seq,
                "frame": frame,
                "tick": frame,
                "channel": channel,
                "source": domain.PROPERTIES[key],
                "entity_alias": controllers[p]["alias"],
                "actor_alias": f"p{controller:04d}",
                "side": "T",
                "instance": 1,
                "value": {"status": "observed", "value": value},
            }
        )
        deliveries.append(
            {
                "trace_sequence": seq,
                "pawn_handle": scalar(((2 + p) << 14) | (1 + p), "uint64"),
            }
        )

    for p in (1, 2):
        row(p, "account", 1000, 0, "property_baseline")
        row(p, "spent", 0, 0, "property_baseline")
    row(1, "account", 700, 2, "property_update")
    row(1, "spent", 300, 2, "property_update")
    trace = {**meta, "observations": {"rows": rows}}
    states = []
    for seq in (1, 2):
        pawns = []
        for i, p in enumerate(own["observations"]["states"][seq - 1]["pawns"]):
            pawns.append(
                {
                    "pawn_alias": p["entity_alias"],
                    "controller_alias": controllers[i]["alias"],
                    "pawn_handle": scalar(((2 + i) << 14) | (1 + i), "uint64"),
                    "account": scalar(700 if seq == 2 and i == 0 else 1000),
                    "spent": scalar(300 if seq == 2 and i == 0 else 0),
                }
            )
        states.append(
            {
                "sequence": seq,
                "first_cut": seq,
                "entry_state_sequence": seq,
                "pawns": pawns,
            }
        )
    data = {
        k: entries[k]
        for k in (
            "contract_version",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "partition",
            "comparison_allowed",
            "training_allowed",
            "product_supported",
        )
    }
    data.update(
        schema_id="stratai.economy_cashflow_observations@1",
        source_entries={
            "path": domain.entry_identity.DATA,
            "bytes": 1,
            "sha256": "b" * 64,
        },
        source_trace={
            "path": domain.transaction_trace.DATA,
            "bytes": 1,
            "sha256": "c" * 64,
        },
        cut_contract="fresh_accounts_in_entry_ownership_FrameDone_trace_next_frame_equals_cut_frame_plus_one",
        observations={
            "controllers": controllers,
            "deliveries": deliveries,
            "states": states,
            "cuts": [
                {
                    "ownership_cut": 1,
                    "state_sequence": 1,
                    "trace_sequence": 4,
                    "trace_next_frame": 2,
                },
                {
                    "ownership_cut": 2,
                    "state_sequence": 2,
                    "trace_sequence": 6,
                    "trace_next_frame": 3,
                },
            ],
        },
    )
    return data, entries, own, identity, candidate, trace


def report(args):
    return domain.build_report(
        *args,
        {"path": domain.DATA, "bytes": 1, "sha256": "d" * 64},
        require_observed=True,
    )


def first(args):
    return report(args)["intervals"][0]


def test_observed_cashflow_equality_is_only_arithmetic_consistency():
    args = fixture()
    before = copy.deepcopy(args)
    result = report(args)
    row = result["intervals"][0]
    assert row["status"] == "amounts_match_single_entry"
    assert (
        row["account_delta_plus_entry_cost_net"]
        == row["spent_delta_minus_entry_cost_net"]
        == 0
    )
    assert row["channels"]["account"]["negative_total"] == 300
    assert row["entries"]["added"][0]["recorded_cost"] == 300
    assert result["summary"]["omitted_unchanged_snapshot_pairs"] == 1
    assert result["summary"]["verified_itemized_transactions"] == 0
    assert args == before
    row["cause"] = "purchase"
    with pytest.raises(RuntimeContractError):
        domain.validate_report(
            result, *args, {"path": domain.DATA, "bytes": 1, "sha256": "d" * 64}
        )


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("comparison_allowed",), True),
        (("training_allowed",), True),
        (("product_supported",), True),
        (("source_sha256",), "e" * 64),
        (("observations", "cuts", 0, "trace_sequence"), 5),
        (("observations", "cuts", 0, "trace_next_frame"), 3),
        (("observations", "cuts", 1, "state_sequence"), 1),
        (("observations", "states", 0, "first_cut"), 2),
        (("observations", "states", 0, "entry_state_sequence"), 2),
        (
            ("observations", "states", 0, "pawns", 0, "pawn_handle", "value"),
            (1026 << 14) | 1,
        ),
        (("observations", "controllers", 0, "created_frame"), 2),
        (("observations", "controllers", 0, "destroyed_frame"), 1),
        (("observations", "deliveries", 4, "trace_sequence"), 6),
    ],
)
def test_rejects_future_prefix_wrong_cut_lifetime_and_presence(path, value):
    args = fixture()
    target = args[0]
    for key in path[:-1]:
        target = target[key]
    target[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        report(args)


def test_missing_snapshot_never_acquires_zero_or_future_anchor():
    args = fixture()
    args[0]["observations"]["states"][0]["pawns"][0]["account"] = scalar(None)
    r = first(args)
    assert r["status"] == "not_comparable"
    assert r["channels"]["account"]["from_value"] is None
    assert r["channels"]["account"]["deliveries"][0]["delta"] is None
    assert r["channels"]["account"]["negative_total"] is None


@pytest.mark.parametrize("mode", ["actor", "side", "instance", "handle"])
def test_delivery_binding_not_backfilled_from_endpoints(mode):
    args = fixture()
    if mode == "handle":
        args[0]["observations"]["deliveries"][4]["pawn_handle"] = scalar((3 << 14) | 2)
    else:
        args[-1]["observations"]["rows"][4][
            {"actor": "actor_alias", "side": "side", "instance": "instance"}[mode]
        ] = None
    r = first(args)
    assert r["status"] == "not_comparable"
    assert "delivery_binding_unavailable" in r["channels"]["account"]["reasons"]
    assert r["channels"]["account"]["deliveries"][0]["delta"] is None


def test_same_tick_different_frames_are_not_collapsed():
    args = fixture()
    args[2]["observations"]["cuts"][1]["demo_tick"] = 1
    args[3]["observations"]["clocks"][1]["demo_tick"] = 1
    for r in args[-1]["observations"]["rows"][4:]:
        r["tick"] = 1
    assert first(args)["status"] == "amounts_match_single_entry"


def test_compensated_deliveries_survive_zero_net():
    args = fixture()
    args[1]["observations"]["states"][0]["pawns"][0] = copy.deepcopy(
        args[1]["observations"]["states"][1]["pawns"][0]
    )
    args[0]["observations"]["states"][1]["pawns"][0].update(
        account=scalar(1000), spent=scalar(0)
    )
    row = args[-1]["observations"]["rows"][5]
    row.update(
        source=domain.PROPERTIES["account"], value={"status": "observed", "value": 1000}
    )
    r = first(args)
    assert r["status"] == "deliveries_without_net_change"
    assert [d["delta"] for d in r["channels"]["account"]["deliveries"]] == [-300, 300]
    assert r["channels"]["account"]["delivery_net"] == 0
    assert (
        r["channels"]["account"]["positive_total"]
        == r["channels"]["account"]["negative_total"]
        == 300
    )


def test_endpoint_change_without_update_is_a_residual_not_a_purchase():
    args = fixture()
    args[0]["observations"]["states"][1]["pawns"][0]["account"] = scalar(650)
    r = first(args)
    assert r["status"] == "not_comparable"
    assert r["channels"]["account"]["snapshot_residual"] == -50
    assert r["channels"]["account"]["status"] == "untracked_snapshot_change"


def test_recorded_cost_mismatch_remains_unassigned():
    args = fixture()
    args[1]["observations"]["states"][1]["pawns"][0]["entries"][0]["fields"][
        "m_nCost"
    ] = scalar(250)
    r = first(args)
    assert r["status"] == "residuals_present"
    assert r["account_delta_plus_entry_cost_net"] == -50
    assert r["spent_delta_minus_entry_cost_net"] == 50


def test_decreasing_spent_counter_is_not_automatically_a_refund():
    args = fixture()
    args[0]["observations"]["states"][0]["pawns"][0]["spent"] = scalar(600)
    r = first(args)
    assert r["status"] == "counter_decreased"
    assert r["channels"]["spent"]["negative_total"] == 300
    assert r["refund_amount"] is None


def test_reordered_duplicate_entries_use_multiset_not_ordinal_identity():
    args = fixture()
    e = copy.deepcopy(args[1]["observations"]["states"][1]["pawns"][0]["entries"][0])
    for state in args[1]["observations"]["states"]:
        other = copy.deepcopy(e)
        other["index"] = 1
        other["fields"]["m_nCost"] = scalar(350)
        state["pawns"][0].update(entries=[copy.deepcopy(e), other], length=2)
    after = args[1]["observations"]["states"][1]["pawns"][0]["entries"]
    after.reverse()
    for i, e in enumerate(after):
        e["index"] = i
    r = first(args)
    assert r["entries"]["reordered_only"]
    assert not r["entries"]["added"] and not r["entries"]["removed"]
    assert r["status"] == "residuals_present"


def test_identical_duplicate_multiplicity_is_preserved():
    args = fixture()
    p = args[1]["observations"]["states"][1]["pawns"][0]
    e = copy.deepcopy(p["entries"][0])
    e["index"] = 1
    p["entries"].append(e)
    p["length"] = 2
    r = first(args)
    assert len(r["entries"]["added"]) == 2 and r["entries"]["added_cost"] == 600
    assert r["entries"]["duplicate_multiplicity"]
    assert r["entries"]["added"][0]["link_status"] == "identity_ambiguous"


def test_partial_list_keeps_cashflow_but_blocks_entry_arithmetic():
    args = fixture()
    p = args[1]["observations"]["states"][1]["pawns"][0]
    p["status"] = "partial"
    p["entries"][0]["fields"]["m_nPrevArmor"] = scalar(None)
    r = first(args)
    assert r["entries"]["status"] == "unavailable"
    assert r["channels"]["account"]["snapshot_delta"] == -300
    assert r["account_delta_plus_entry_cost_net"] is None


def test_zero_cost_is_present_and_sentinel_does_not_gain_object_identity():
    args = fixture()
    e = args[1]["observations"]["states"][1]["pawns"][0]["entries"][0]
    e["fields"]["m_nCost"] = scalar(0)
    e["fields"]["m_hItem"] = scalar(16777215)
    e["fields"]["m_nPrevArmor"] = scalar(4294967295, "uint32")
    e["item"] = {"status": "invalid_sentinel", "handle": 16777215, "target_alias": None}
    e["entity_definition"] = scalar(None)
    args[0]["observations"]["states"][1]["pawns"][0].update(
        account=scalar(1000), spent=scalar(0)
    )
    args[-1]["observations"]["rows"][4]["value"]["value"] = 1000
    args[-1]["observations"]["rows"][5]["value"]["value"] = 0
    r = first(args)
    assert r["status"] == "amounts_match_without_unique_object"
    assert r["entries"]["added"][0]["entity_alias"] is None
    assert r["entries"]["cost_net"] == 0


def test_late_actor_and_temporal_boundary_abstain_without_backfill():
    args = fixture()
    args[2]["observations"]["states"][0]["pawns"][0]["actor_alias"] = None
    assert "actor_binding_unavailable" in first(args)["reasons"]
    args = fixture()
    args[3]["observations"]["clocks"][1]["epoch"] = 2
    assert "incompatible_delivery_cuts" in first(args)["reasons"]


def test_creation_replay_is_not_a_monetary_change():
    rows = [
        {
            "sequence": 1,
            "frame": 0,
            "channel": "property_update",
            "actor_alias": "p0001",
            "side": "T",
            "instance": 1,
            "value": {"status": "observed", "value": 700},
        }
    ]
    r = domain.channel_flow(
        scalar(1000),
        scalar(700),
        rows,
        {1: {"pawn_handle": scalar(32769)}},
        {"created_frame": 0},
        32769,
        "p0001",
        "T",
        1,
    )
    assert r["status"] == "baseline_or_creation_delivery"
    assert r["deliveries"][0]["delta"] is None and r["negative_total"] is None


def test_baseline_anchors_only_subsequent_updates_and_does_not_repair_interval():
    rows = [
        {
            "sequence": i,
            "frame": i,
            "channel": "property_baseline" if i == 1 else "property_update",
            "actor_alias": "p0001",
            "side": "T",
            "instance": 1,
            "value": {"status": "observed", "value": v},
        }
        for i, v in [(1, 1000), (2, 700)]
    ]
    r = domain.channel_flow(
        scalar(None),
        scalar(700),
        rows,
        {i: {"pawn_handle": scalar(32769)} for i in (1, 2)},
        {"created_frame": 1},
        32769,
        "p0001",
        "T",
        1,
    )
    assert [d["delta"] for d in r["deliveries"]] == [None, -300]
    assert r["negative_total"] is None


def test_load_rejects_wrong_entry_hash(tmp_path, monkeypatch):
    args = fixture()
    (tmp_path / domain.DATA).write_text(json.dumps(args[0]))
    monkeypatch.setattr(
        domain.entry_identity,
        "load_source",
        lambda path: (
            *args[1:5],
            {"path": domain.entry_identity.DATA, "bytes": 2, "sha256": "e" * 64},
        ),
    )
    with pytest.raises(RuntimeContractError, match="entry hash"):
        domain.load_source(tmp_path / domain.DATA)


def test_output_preflight_before_reading_source(tmp_path, monkeypatch):
    (tmp_path / "sentinel").write_text("preserve")
    monkeypatch.setattr(
        domain,
        "load_source",
        lambda path: pytest.fail("source read before output guard"),
    )
    with pytest.raises((RuntimeContractError, ValueError)):
        domain.materialize(tmp_path / "missing", tmp_path)


def test_controller_index_reuse_blocks_old_life_even_with_same_serial():
    args = fixture()
    old = args[0]["observations"]["controllers"][0]
    new = {**old, "alias": "trace-entity-000003", "created_frame": 2}
    args[0]["observations"]["controllers"].append(new)
    row = copy.deepcopy(args[-1]["observations"]["rows"][4])
    row.update(sequence=7, channel="property_baseline", entity_alias=new["alias"])
    args[-1]["observations"]["rows"].append(row)
    args[0]["observations"]["deliveries"].append(
        {"trace_sequence": 7, "pawn_handle": scalar(32769)}
    )
    args[0]["observations"]["cuts"][1]["trace_sequence"] = 7
    with pytest.raises(RuntimeContractError, match="controller lifetime"):
        report(args)


def test_controller_change_retains_deliveries_but_does_not_join_money():
    args = fixture()
    old = args[0]["observations"]["controllers"][0]
    new = {
        **old,
        "alias": "trace-entity-000003",
        "serial": old["serial"] + 1024,
        "created_frame": 2,
    }
    args[0]["observations"]["controllers"].append(new)
    args[0]["observations"]["states"][1]["pawns"][0]["controller_alias"] = new["alias"]
    for row in args[-1]["observations"]["rows"][4:]:
        row.update(channel="property_baseline", entity_alias=new["alias"])
    r = first(args)
    assert "controller_lifetime_changed" in r["reasons"]
    assert r["trace_sequences"] == [5, 6] and r["channels"]["account"] is None


def test_missing_trace_delivery_is_rejected_instead_of_assuming_no_movement():
    args = fixture()
    args[0]["observations"]["deliveries"].pop()
    with pytest.raises(RuntimeContractError, match="delivery coverage"):
        report(args)


def test_unobserved_actor_with_unchanged_snapshot_is_counted_without_claiming_no_change():
    args = fixture()
    for s in args[2]["observations"]["states"]:
        s["pawns"][1]["actor_alias"] = None
    r = report(args)
    assert r["summary"]["examined_pawn_pairs"] == 2
    assert r["summary"]["omitted_unchanged_snapshot_pairs"] == 1
    assert r["summary"]["status_counts"]["no_change"] == 0


def test_repeated_equal_updates_remain_individually_visible():
    args = fixture()
    row = copy.deepcopy(args[-1]["observations"]["rows"][4])
    row["sequence"] = 7
    args[-1]["observations"]["rows"].append(row)
    args[0]["observations"]["deliveries"].append(
        {"trace_sequence": 7, "pawn_handle": scalar(32769)}
    )
    args[0]["observations"]["cuts"][1]["trace_sequence"] = 7
    r = first(args)
    assert [d["delta"] for d in r["channels"]["account"]["deliveries"]] == [-300, 0]
    assert r["status"] == "amounts_match_single_entry"


def test_report_residual_and_delivery_order_cannot_be_forged():
    args = fixture()
    r = report(args)
    r["intervals"][0]["account_delta_plus_entry_cost_net"] = 10
    with pytest.raises(RuntimeContractError):
        domain.validate_report(
            r, *args, {"path": domain.DATA, "bytes": 1, "sha256": "d" * 64}
        )


def test_metadata_fields_remain_part_of_multiset_identity():
    args = fixture()
    args[1]["observations"]["states"][0]["pawns"][0] = copy.deepcopy(
        args[1]["observations"]["states"][1]["pawns"][0]
    )
    args[1]["observations"]["states"][1]["pawns"][0]["entries"][0]["fields"][
        "m_nPrevArmor"
    ] = scalar(0)
    r = first(args)
    assert len(r["entries"]["added"]) == len(r["entries"]["removed"]) == 1
    assert (
        r["entries"]["added"][0]["fields_sha256"]
        != r["entries"]["removed"][0]["fields_sha256"]
    )
    assert r["entries"]["cost_net"] == 0
