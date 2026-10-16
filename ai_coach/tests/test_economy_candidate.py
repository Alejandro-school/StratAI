"""Economic causal boundaries, uncertainty, joins and artifact integrity."""

import copy
import hashlib
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.contracts.validate_economy_candidate import (
    ACTOR_SOURCES,
    DATA_FILE,
    MANIFEST_FILE,
    VERSION,
    initial_team_state,
    validate_candidate,
    validate_data,
)


def observed(value, source, tick=100):
    return {
        "value": value,
        "source": source,
        "status": "observed",
        "observation_tick": tick,
        "availability_tick": tick,
    }


def missing(source):
    return {
        "value": None,
        "source": source,
        "status": "unavailable",
        "observation_tick": None,
        "availability_tick": None,
    }


def event(seq, tick, kind="pickup", actor="p0001", entity="w00001", prior=None):
    attribution = (
        "callback_only" if kind != "pickup" else "purchase_or_pickup_unresolved"
    )
    if prior:
        attribution = "prior_allied_drop_same_entity"
    return {
        "event_id": f"e{seq:08d}",
        "callback_sequence": seq,
        "tick": tick,
        "kind": kind,
        "actor_alias": actor,
        "side": "CT",
        "weapon": "m4a1",
        "entity_alias": entity,
        "prior_drop_event_id": prior,
        "attribution": attribution,
    }


def candidate_fixture():
    checksum = "a" * 64
    actors, participants = [], []
    for i in range(1, 11):
        alias = f"p{i:04d}"
        participants.append(
            {
                "participant_alias": alias,
                "actor_key": hashlib.sha256(
                    (checksum + f"\nsteam:{i}").encode()
                ).hexdigest(),
            }
        )
        facts = {
            key: observed(False if key in {"helmet", "kit", "buy_zone"} else 0, source)
            for key, source in ACTOR_SOURCES.items()
        }
        facts["inventory"] = observed(
            [{"entity_alias": f"w{i:05d}", "weapon": "m4a1"}],
            ACTOR_SOURCES["inventory"],
        )
        actors.append(
            {
                "actor_alias": alias,
                "side": "CT" if i <= 5 else "T",
                "connected": True,
                "facts": facts,
                "prepurchase_status": "available",
                "prepurchase_reason": "observed_at_round_start_without_prior_activity",
            }
        )
    context = {
        key: missing("convar:" + key)
        for key in (
            "mp_buytime",
            "mp_freezetime",
            "mp_maxmoney",
            "mp_startmoney",
            "mp_maxrounds",
            "mp_overtime_enable",
        )
    }
    context.update(score_CT=observed(0, "m_iScore"), score_T=observed(0, "m_iScore"))
    snapshot = {
        "phase": "initial_available",
        "tick": 100,
        "callback_sequence": 2,
        "context": context,
        "actors": actors,
    }
    return {
        "schema_id": "stratai.observed_economy_candidate@1",
        "contract_version": VERSION,
        "source_alias": "demo-" + checksum[:20],
        "source_sha256": checksum,
        "map_id": "de_mirage",
        "tick_rate": 64,
        "build_identifier": "fixture",
        "visibility_contract": "self_and_same_side_only_per_actor_no_opponent_economy",
        "order_contract": "parser_callback_sequence_not_engine_subtick_order",
        "training_allowed": False,
        "product_supported": False,
        "rules_compatibility": "unverified_no_price_or_reward_inference",
        "configs": {
            name: {
                "version": "fixture",
                "sha256": "b" * 64,
                "use": "reference_only_not_applied",
            }
            for name in ("legacy_prices", "legacy_rules")
        },
        "participants": participants,
        "rounds": [
            {
                "instance": 1,
                "round_number": 1,
                "start_tick": 100,
                "freeze_end_tick": 200,
                "end_tick": 300,
                "initial_state": snapshot,
                "execution": {"freeze_snapshot": None, "events": [event(3, 150)]},
                "reconciliation": {
                    "round_end_snapshot": None,
                    "other_events": [],
                    "comparisons": [],
                },
            }
        ],
    }


def write_candidate(root, data):
    def write(path, payload):
        target = root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        encoded = json.dumps(payload, separators=(",", ":")).encode()
        target.write_bytes(encoded)
        return {
            "path": path,
            "bytes": len(encoded),
            "sha256": hashlib.sha256(encoded).hexdigest(),
        }

    players = [
        {
            "round_number": 1,
            "player_id": "steam:1",
            "money": {
                "round_start_observed": {
                    "amount": 800,
                    "status": "observed",
                    "source": "demoinfocs.player_money_round_start",
                },
                "round_end_observed": {
                    "amount": 300,
                    "status": "observed",
                    "source": "demoinfocs.player_money_round_end",
                },
            },
        }
    ]
    data["rounds"][0]["reconciliation"]["comparisons"] = [
        {
            "actor_alias": "p0001",
            "metric": "end_frame_minus_legacy_round_end_money",
            "delta": None,
            "status": "unavailable",
        },
        {
            "actor_alias": "p0001",
            "metric": "initial_minus_legacy_initial_money",
            "delta": -800,
            "status": "diagnostic_different_observation_frontiers",
        },
    ]
    canonical = f"match_{data['source_alias']}/canonical/"
    aggregate = write(
        canonical + "derived/economy_players.json",
        {"match_id": data["source_alias"], "players": players},
    )
    aggregate["path"] = "derived/economy_players.json"
    aggregate["record_count"] = 1
    base = write(
        canonical + "manifest.json",
        {
            "schema_id": "stratai.canonical_manifest@3",
            "export_format_version": "3.8.0",
            "match_id": data["source_alias"],
            "demo_checksum_sha256": data["source_sha256"],
            "artifacts": [aggregate],
        },
    )
    artifact = write(DATA_FILE, data)
    manifest = {
        key: data[key]
        for key in (
            "contract_version",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "training_allowed",
            "product_supported",
        )
    }
    manifest.update(
        schema_id="stratai.economy_candidate_manifest@1",
        base_export_version="3.8.0",
        artifact=artifact,
        base_manifest=base,
        summary=validate_data(data),
    )
    write(MANIFEST_FILE, manifest)
    return root / MANIFEST_FILE


def test_economic_zero_false_missing_and_team_visibility():
    data = candidate_fixture()
    summary = validate_data(data)
    assert summary["money_zero"] == summary["prepurchase_available"] == 10
    state = initial_team_state(data, 1, "p0001", 100)
    assert len(state["actors"]) == 5
    assert {a["side"] for a in state["actors"]} == {"CT"}
    assert state["context"]["mp_buytime"]["value"] is None
    assert state["actors"][0]["facts"]["kit"]["value"] is False
    state["actors"][0]["facts"]["money"]["value"] = 999
    assert (
        data["rounds"][0]["initial_state"]["actors"][0]["facts"]["money"]["value"] == 0
    )


def test_initial_state_invariant_to_later_execution_reconciliation_opponents_and_next_round():
    data = candidate_fixture()
    before = initial_team_state(data, 1, "p0001", 100)
    later = copy.deepcopy(data["rounds"][0])
    later["instance"], later["round_number"], later["start_tick"] = 2, 2, 500
    data["rounds"].append(later)
    data["rounds"][0]["reconciliation"] = {
        "future_result": "changed",
        "later_purchase": 9999,
    }
    data["rounds"][0]["execution"] = {"future_refund": 9999}
    data["rounds"][0]["initial_state"]["actors"][5]["facts"]["money"]["value"] = 9999
    assert initial_team_state(data, 1, "p0001", 100) == before


@pytest.mark.parametrize(
    "variant",
    [
        "future",
        "zero_missing",
        "unknown_actor",
        "duplicate_actor",
        "early_buy",
        "late_frame",
        "spent",
        "disconnect",
        "boundary",
        "purchase",
        "inventory_duplicate",
        "provenance",
        "oracle",
    ],
)
def test_rejects_false_causal_or_identity_claims(variant):
    data = candidate_fixture()
    round_ = data["rounds"][0]
    snap = round_["initial_state"]
    actor = snap["actors"][0]
    if variant == "future":
        actor["facts"]["money"]["availability_tick"] = 101
    elif variant == "zero_missing":
        actor["facts"]["money"]["status"] = "unavailable"
    elif variant == "unknown_actor":
        round_["execution"]["events"][0]["actor_alias"] = "p9999"
    elif variant == "duplicate_actor":
        snap["actors"].append(copy.deepcopy(actor))
    elif variant == "early_buy":
        round_["reconciliation"]["other_events"].append(event(1, 100))
    elif variant == "late_frame":
        snap["tick"] = 101
    elif variant == "spent":
        actor["facts"]["spent_native"]["value"] = 200
    elif variant == "disconnect":
        actor["connected"] = False
    elif variant == "boundary":
        round_["execution"]["events"][0]["tick"] = 200
    elif variant == "purchase":
        round_["execution"]["events"][0]["attribution"] = "purchase"
    elif variant == "inventory_duplicate":
        actor["facts"]["inventory"]["value"] *= 2
    elif variant == "provenance":
        actor["facts"]["money"]["source"] = "future_aggregate"
    elif variant == "oracle":
        snap["context"]["opponent_money"] = observed(800, "oracle")
    with pytest.raises(RuntimeContractError):
        validate_data(data)


def test_partial_team_abstains_and_missing_is_not_empty_inventory():
    data = candidate_fixture()
    actor = data["rounds"][0]["initial_state"]["actors"][0]
    actor["facts"]["inventory"] = missing(ACTOR_SOURCES["inventory"])
    actor.update(
        prepurchase_status="unavailable",
        prepurchase_reason="required_initial_fact_unavailable",
    )
    assert validate_data(data)["prepurchase_available"] == 9
    with pytest.raises(RuntimeContractError):
        initial_team_state(data, 1, "p0001", 100)
    actor["facts"]["inventory"] = observed([], ACTOR_SOURCES["inventory"])
    actor.update(
        prepurchase_status="available",
        prepurchase_reason="observed_at_round_start_without_prior_activity",
    )
    assert validate_data(data)["prepurchase_available"] == 10


def test_drop_requires_same_entity_prior_ally_and_is_consumed():
    data = candidate_fixture()
    events = [event(3, 120, "drop"), event(4, 130, actor="p0002", prior="e00000003")]
    data["rounds"][0]["execution"]["events"] = events
    assert validate_data(data)["prior_allied_drop_links"] == 1
    events.append(event(5, 140, actor="p0003", prior="e00000003"))
    with pytest.raises(RuntimeContractError, match="reused"):
        validate_data(data)


def test_independent_reconciliation_and_integrity(tmp_path):
    path = write_candidate(tmp_path, candidate_fixture())
    report = validate_candidate(path, expected_sha256="a" * 64, require_observed=True)
    assert report["independently_recomputed_comparisons"] == 2
    with pytest.raises(RuntimeContractError, match="wrong economic source"):
        validate_candidate(path, expected_sha256="b" * 64)
    artifact = tmp_path / DATA_FILE
    artifact.write_bytes(artifact.read_bytes() + b" ")
    with pytest.raises(RuntimeContractError, match="integrity"):
        validate_candidate(path)


def test_rejects_rehashed_false_diagnostic(tmp_path):
    path = write_candidate(tmp_path, candidate_fixture())
    data = json.loads((tmp_path / DATA_FILE).read_text())
    data["rounds"][0]["reconciliation"]["comparisons"][1]["delta"] = 0
    encoded = json.dumps(data).encode()
    (tmp_path / DATA_FILE).write_bytes(encoded)
    manifest = json.loads(path.read_text())
    manifest["artifact"].update(
        bytes=len(encoded), sha256=hashlib.sha256(encoded).hexdigest()
    )
    path.write_text(json.dumps(manifest))
    with pytest.raises(RuntimeContractError, match="independent computation"):
        validate_candidate(path)


def test_rejects_path_escape_and_linked_parent(tmp_path):
    root = tmp_path / "real"
    path = write_candidate(root, candidate_fixture())
    link = tmp_path / "linked"
    try:
        link.symlink_to(root, target_is_directory=True)
    except OSError:
        pass  # Producer tests also exercise Windows junctions.
    else:
        with pytest.raises(RuntimeContractError, match="link/reparse"):
            validate_candidate(link / MANIFEST_FILE)
    manifest = json.loads(path.read_text())
    manifest["artifact"]["path"] = "../observed_economy_candidate.json"
    path.write_text(json.dumps(manifest))
    with pytest.raises(RuntimeContractError):
        validate_candidate(path)
