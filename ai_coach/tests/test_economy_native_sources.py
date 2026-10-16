from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import native_reconstruction as domain
from ai_coach.economy import native_sources as io
from ai_coach.economy.candidate_io import sha
from ai_coach.tests.test_economy_opportunities import source_fixture, write_source


def fact(value, source, tick):
    return {
        "value": value,
        "source": source,
        "status": "observed" if value is not None else "unavailable",
        "observation_tick": tick if value is not None else None,
        "availability_tick": tick if value is not None else None,
    }


def array(family, entries, tick):
    root, members = io.ARRAYS[family]
    return {
        "status": "observed",
        "source": root,
        "reason": "native_array_and_members_observed",
        "length": len(entries),
        "entries": [
            {
                name: fact(value, f"{root}.{index:04d}.{name}", tick)
                for name, value in zip(members, entry, strict=True)
            }
            for index, entry in enumerate(entries)
        ],
    }


def native_fixture(source=None):
    source = source or source_fixture()
    rows = []
    for index, (tick, money, spent, counts, entries) in enumerate(
        [
            (100, 800, 0, [], []),
            (150, 500, 300, [(45, 1)], [(45, 300, 0, False, 123)]),
            (170, 800, 0, [(45, 0)], []),
            (200, 150, 650, [(45, 0), (50, 1)], [(50, 650, 0, False, 4294967295)]),
        ],
        1,
    ):
        rows.append(
            {
                "observation_id": f"n{index:08d}",
                "instance": 1,
                "tick": tick,
                "actor_alias": "p0001",
                "side": "CT",
                "connected": True,
                "money": fact(money, "m_pInGameMoneyServices.m_iAccount", tick),
                "spent_native": fact(
                    spent, "m_pInGameMoneyServices.m_iCashSpentThisRound", tick
                ),
                "sellback": array("sellback", entries, tick),
                "purchases_round": array("purchases_round", counts, tick),
                "purchases_match": array("purchases_match", counts, tick),
            }
        )
    data = {
        "schema_id": "stratai.economy_native_sources@1",
        "contract_version": "1.0.0-candidate.1",
        **{k: source[k] for k in ("source_alias", "source_sha256", "build_identifier")},
        "source_candidate_manifest_sha256": "a" * 64,
        "sampling_policy": "every_freeze_frame_retain_changes_and_boundaries_no_intermediate_transaction_guarantee",
        "partition": "execution_evidence_only_not_initial_features",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "observations": {
            "version_observations": [
                {
                    "source": "CDemoFileHeader",
                    "tick": 0,
                    "fields": {
                        "build_num": fact(14000, "CDemoFileHeader.build_num", 0),
                        "patch_version": fact(
                            14000, "CDemoFileHeader.patch_version", 0
                        ),
                    },
                }
            ],
            "snapshots": rows,
            "generic_event_counts": {"item_pickup": 20},
            "snapshot_checks": 100,
        },
    }
    return data


def bindings(data):
    return {
        "source_manifest_sha256": "b" * 64,
        "source_data_sha256": "c" * 64,
        "source_candidate_manifest_sha256": data["source_candidate_manifest_sha256"],
    }


def build(data=None):
    source = source_fixture()
    data = data or native_fixture(source)
    io.validate_data(data, source)
    bound = bindings(data)
    result = domain.reconstruct(data, source, bound)
    return result, data, source, bound


@pytest.mark.parametrize("sentinel", [-1, 4294967295])
def test_previous_armor_native_metadata_is_not_a_physical_amount(sentinel):
    data = native_fixture()
    data["observations"]["snapshots"][1]["sellback"]["entries"][0]["m_nPrevArmor"][
        "value"
    ] = sentinel
    report, data, source, bound = build(data)
    domain.validate_report(report, data, source, bound)
    assert (
        report["records"][1]["sellback_entries_added"][0]["previous_armor"] == sentinel
    )
    assert report["records"][1]["refund_amount"] is None


def write_native(root):
    source = source_fixture()
    source_path = write_source(root, source)
    data = native_fixture(source)
    data["source_candidate_manifest_sha256"] = sha(source_path.read_bytes())
    data_raw = json.dumps(data).encode()
    (root / io.DATA).write_bytes(data_raw)
    manifest = {
        "schema_id": "stratai.economy_native_sources_manifest@1",
        **{
            k: data[k]
            for k in (
                "contract_version",
                "source_alias",
                "source_sha256",
                "build_identifier",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        "source_candidate": {
            "path": source_path.name,
            "bytes": source_path.stat().st_size,
            "sha256": sha(source_path.read_bytes()),
        },
        "artifact": {"path": io.DATA, "bytes": len(data_raw), "sha256": sha(data_raw)},
        "summary": io.summary(data),
    }
    path = root / io.MANIFEST
    path.write_text(json.dumps(manifest))
    return path


def test_roundtrip_recomputes_counter_changes_and_entry_membership(tmp_path):
    path = write_native(tmp_path / "source")
    out = tmp_path / "out"
    out.mkdir()
    result = domain.materialize(path, out, require_observed=True)
    assert result["independently_recomputed_counter_differences"] == 3
    assert result["independently_recomputed_sellback_intervals"] == 3
    assert result["summary"]["positive_counter_differences"] == 2
    assert result["summary"]["negative_counter_differences"] == 1
    assert result["summary"]["classified_actions"] == 0
    assert result["rule_assessment"]["comparison_allowed"] is False


def test_counter_decrease_and_sellback_removal_do_not_prove_refund():
    report, data, source, bound = build()
    refund = report["records"][2]
    assert refund["purchase_counter_differences"][0]["delta"] == -1
    assert refund["sellback_entries_removed"][0]["recorded_cost"] == 300
    assert refund["refund_amount"] is refund["basket"] is None
    assert refund["decision_eligible"] is False
    assert domain.validate_report(report, data, source, bound)["ok"]


@pytest.mark.parametrize(
    "change",
    [
        "arithmetic",
        "refund",
        "price_scope",
        "binding",
        "future_time",
        "omitted_row",
        "oracle",
        "entry_cost",
    ],
)
def test_forged_reconstruction_is_rejected(change):
    report, data, source, bound = build()
    if change == "arithmetic":
        report["records"][1]["purchase_counter_differences"][0]["delta"] = 2
    elif change == "refund":
        report["records"][2]["refund_amount"] = 300
    elif change == "price_scope":
        report["rule_assessment"]["comparison_allowed"] = True
    elif change == "binding":
        report["source_data_sha256"] = "d" * 64
    elif change == "future_time":
        report["records"][1]["interval_start_tick"] = 90
    elif change == "omitted_row":
        report["records"].pop()
    elif change == "oracle":
        report["records"][1]["enemy_money"] = 16000
    elif change == "entry_cost":
        report["records"][1]["sellback_entries_added"][0]["recorded_cost"] = 200
    with pytest.raises(RuntimeContractError):
        domain.validate_report(report, data, source, bound)


@pytest.mark.parametrize(
    "change",
    [
        "late_fact",
        "length",
        "missing_as_empty",
        "forged_status",
        "native_source",
        "actor",
        "postfreeze",
        "initial_cash",
        "duplicate_id",
    ],
)
def test_native_source_contract_and_v2_boundary_binding(change):
    data = native_fixture()
    row = data["observations"]["snapshots"][1]
    if change == "late_fact":
        row["money"]["availability_tick"] = 151
    elif change == "length":
        row["sellback"]["length"] = 0
    elif change == "missing_as_empty":
        row["sellback"]["length"] = None
    elif change == "forged_status":
        row["sellback"]["entries"][0]["m_nCost"] = fact(
            None, row["sellback"]["entries"][0]["m_nCost"]["source"], 150
        )
    elif change == "native_source":
        row["sellback"]["entries"][0]["m_nCost"]["source"] = "legacy_price"
    elif change == "actor":
        row["actor_alias"] = "p9999"
    elif change == "postfreeze":
        data["observations"]["snapshots"][-1]["tick"] = 201
    elif change == "initial_cash":
        data["observations"]["snapshots"][0]["money"]["value"] = 900
    elif change == "duplicate_id":
        row["observation_id"] = "n00000001"
    with pytest.raises(RuntimeContractError):
        io.validate_data(data, source_fixture())


@pytest.mark.parametrize(
    "change,reason",
    [
        ("disconnect", "participant_disconnected"),
        ("side", "participant_side_changed_or_unknown"),
        ("duplicate_item", "invalid_or_duplicate_item_definition"),
        ("partial", "native_array_or_members_unavailable"),
    ],
)
def test_unavailable_intervals_are_kept_without_invented_actions(change, reason):
    data = native_fixture()
    row = data["observations"]["snapshots"][1]
    if change == "disconnect":
        row["connected"] = False
    elif change == "side":
        row["side"] = "T"
    elif change == "duplicate_item":
        row["purchases_round"] = array("purchases_round", [(45, 1), (45, 2)], 150)
    elif change == "partial":
        row["purchases_round"]["status"] = "partial"
        row["purchases_round"]["reason"] = "native_member_unavailable"
        old = row["purchases_round"]["entries"][0]["m_nCount"]
        row["purchases_round"]["entries"][0]["m_nCount"] = fact(
            None, old["source"], 150
        )
    report, data, source, bound = build(data)
    assert report["records"][1]["purchase_counter_reason"] == reason
    assert report["records"][1]["purchase_counter_differences"] == []
    assert domain.validate_report(report, data, source, bound)["ok"]


def test_future_values_cannot_rewrite_earlier_execution():
    report, data, source, bound = build()
    changed = copy.deepcopy(data)
    changed["observations"]["snapshots"][-1]["purchases_round"] = array(
        "purchases_round", [(45, 22)], 200
    )
    after = domain.reconstruct(changed, source, bound)
    assert after["records"][:3] == report["records"][:3]
    assert report["records"][-1]["scope"] == "boundary_or_unlinked"


def test_conflicting_and_missing_game_versions_never_match_rules():
    data = native_fixture()
    duplicate = copy.deepcopy(data["observations"]["version_observations"][0])
    duplicate["fields"]["build_num"]["value"] = 15000
    data["observations"]["version_observations"].append(duplicate)
    assessment = domain.rules(data)
    assert assessment["game_identity"]["build_num"]["status"] == "conflicting"
    assert assessment["game_identity"]["network_protocol"]["status"] == "unavailable"
    assert assessment["comparison_allowed"] is False


def test_rehashed_native_boundary_forgery_fails(tmp_path):
    path = write_native(tmp_path / "source")
    data_path = path.parent / io.DATA
    data = json.loads(data_path.read_text())
    data["observations"]["snapshots"][0]["money"]["value"] = 900
    payload = json.dumps(data).encode()
    data_path.write_bytes(payload)
    manifest = json.loads(path.read_text())
    manifest["artifact"].update(bytes=len(payload), sha256=sha(payload))
    path.write_text(json.dumps(manifest))
    with pytest.raises(RuntimeContractError, match="boundary balance"):
        io.load_candidate(path)


def test_output_preflight_precedes_input_access(tmp_path, monkeypatch):
    out = tmp_path / "occupied"
    out.mkdir()
    (out / "keep").write_text("preserve")
    monkeypatch.setattr(
        io,
        "load_candidate",
        lambda *a, **k: pytest.fail("input loaded before preflight"),
    )
    with pytest.raises(RuntimeContractError):
        domain.materialize(tmp_path / "missing", out)
    assert (out / "keep").read_text() == "preserve"
