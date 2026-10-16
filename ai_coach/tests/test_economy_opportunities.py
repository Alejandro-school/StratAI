"""Economic windows, factual reconstruction, uncertainty and rehashed tampering."""

import copy
import hashlib
import json
import tempfile
from pathlib import Path

import pytest

from ai_coach.contracts import economy_initial_frontiers as initial
from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.contracts.validate_contract import (
    ContractValidationError,
    validate_contract,
)
from ai_coach.economy import candidate_io as io
from ai_coach.economy import opportunities as domain
from ai_coach.economy.validate_opportunities import validate_candidate, validate_data
from ai_coach.tests.test_economy_candidate import event, missing, observed
from ai_coach.tests.test_economy_initial_frontiers import (
    fixture,
    refresh_frame_statuses,
    write_fixture,
)


def source_fixture():
    data = fixture()
    round_ = data["rounds"][0]
    for snapshot in (
        round_["initial_state"],
        data["initial_frontiers"][0]["round_start_probe"],
    ):
        for key, value in {
            "mp_buytime": "20",
            "mp_freezetime": "20",
            "mp_startmoney": "800",
            "mp_maxrounds": "24",
            "mp_overtime_enable": "true",
        }.items():
            snapshot["context"][key] = observed(value, "convar:" + key)
        for actor in snapshot["actors"]:
            actor["facts"]["money"]["value"] = 800
            actor["facts"]["buy_zone"]["value"] = True
    end = copy.deepcopy(round_["initial_state"])
    end.update(phase="freeze_end", tick=200, callback_sequence=30)
    for fact in [
        *end["context"].values(),
        *[f for a in end["actors"] for f in a["facts"].values()],
    ]:
        if fact["status"] == "observed":
            fact["observation_tick"] = fact["availability_tick"] = 200
    for index, actor in enumerate(end["actors"], 1):
        actor.update(
            prepurchase_status="unavailable", prepurchase_reason="not_initial_snapshot"
        )
        actor["facts"]["money"]["value"] = 150
        actor["facts"]["spent_native"]["value"] = 650
        actor["facts"]["armor"]["value"] = 100
        actor["facts"]["inventory"]["value"].append(
            {"entity_alias": f"w{100 + index:05d}", "weapon": "Smoke Grenade"}
        )
    round_["execution"]["freeze_snapshot"] = end
    round_["execution"]["events"] = [event(20, 150, entity="w00101")]
    return data


def bindings():
    return io.load_bindings()


def build(source=None):
    source = source_fixture() if source is None else source
    protocol, protocol_hash, catalog, catalog_hash = bindings()
    data = domain.build_opportunities(
        source,
        protocol,
        "a" * 64,
        "b" * 64,
        protocol_hash,
        catalog["version"],
        catalog_hash,
    )
    return data, source, protocol


def write_source(root, source):
    path = write_fixture(root, source)
    for item in source["rounds"][0]["reconciliation"]["comparisons"]:
        if item["metric"] == "initial_minus_legacy_initial_money":
            actor = next(
                a
                for a in source["rounds"][0]["initial_state"]["actors"]
                if a["actor_alias"] == item["actor_alias"]
            )
            item["delta"] = actor["facts"]["money"]["value"] - 800
    data_path = root / "observed_economy_candidate.json"
    payload = json.dumps(source).encode()
    data_path.write_bytes(payload)
    manifest = json.loads(path.read_text())
    manifest["artifact"].update(
        bytes=len(payload), sha256=hashlib.sha256(payload).hexdigest()
    )
    manifest["summary"] = initial.validate_data(source)
    path.write_text(json.dumps(manifest))
    return path


def test_materialize_round_trip_with_independent_recomputation(tmp_path):
    source = write_source(tmp_path / "source", source_fixture())
    out = tmp_path / "out"
    out.mkdir()
    result = io.materialize(source, out)
    report = validate_candidate(
        out / domain.MANIFEST_FILE, source, require_observed=True
    )
    assert result["summary"] == report["summary"]
    assert report["summary"]["initial_teams_available"] == 2
    assert report["independently_recomputed_numeric_differences"] == 40
    assert report["independently_recomputed_inventory_transitions"] == 10
    assert report["summary"]["classified_actions"] == 0
    data = json.loads((out / domain.DATA_FILE).read_text())
    row = data["opportunities"][0]
    assert (
        row["observed_action"]["actor_changes"][0]["changes"]["money"]["delta"] == -650
    )
    assert row["rules"]["missing_convars"] == ["mp_maxmoney"]
    assert data["comparison_allowed"] is False


def test_window_covers_freeze_not_generic_two_seconds_or_nominal_duration():
    data, source, protocol = build()
    assert validate_data(data, source, protocol)["team_round_records_verified"] == 2
    row = data["opportunities"][0]
    assert row["execution_window"]["duration_ticks"] == 100
    assert row["execution_window"]["end_exclusive_tick"] == 200
    source["rounds"][0]["freeze_end_tick"] = 5000
    source["rounds"][0]["execution"]["events"].append(event(25, 4500, kind="refund"))
    changed, _, _ = build(source)
    assert (
        changed["opportunities"][0]["observed_action"]["callbacks"][-1]["tick"] == 4500
    )
    assert changed["opportunities"][0]["observed_action"]["checkpoint"] is None
    assert changed["opportunities"][0]["execution_window"]["duration_ticks"] == 4900


def test_boundary_callbacks_and_postfreeze_never_enter_execution():
    source = source_fixture()
    round_ = source["rounds"][0]
    round_["reconciliation"]["other_events"] = [
        event(7, 100),
        event(29, 200, kind="refund"),
        event(40, 201),
    ]
    refresh_frame_statuses(source)
    data, _, protocol = build(source)
    validate_data(data, source, protocol)
    row = data["opportunities"][0]
    assert [e["tick"] for e in row["observed_action"]["callbacks"]] == [150]
    assert [e["tick"] for e in row["observed_action"]["boundary_callbacks"]] == [
        100,
        200,
    ]
    assert row["initial_gate"]["status"] == "unavailable"
    assert domain.summary(data)["team_round_records"] == 2


def test_future_outcomes_and_opponent_economy_do_not_change_own_features_or_execution():
    data, source, _ = build()
    before = copy.deepcopy(data["opportunities"][0])
    round_ = source["rounds"][0]
    round_["reconciliation"]["comparisons"] = [{"future": 999999}]
    round_["reconciliation"]["other_events"] = [event(40, 201, kind="refund")]
    round_["reconciliation"]["round_end_snapshot"] = {"future": "changed"}
    for snapshot in (
        round_["initial_state"],
        source["initial_frontiers"][0]["round_start_probe"],
        round_["execution"]["freeze_snapshot"],
    ):
        for actor in snapshot["actors"]:
            if actor["side"] == "T":
                actor["facts"]["money"]["value"] = 9999
    changed, _, _ = build(source)
    assert changed["opportunities"][0] == before
    projected = domain.initial_features(before)
    before["observed_action"] = {"future": "arbitrary"}
    before["execution_window"] = {"future": "arbitrary"}
    before["rules"] = {"future": "arbitrary"}
    assert domain.initial_features(before) == projected
    projected["actors"][0]["facts"]["money"]["value"] = 99999
    assert (
        domain.initial_features(before)["actors"][0]["facts"]["money"]["value"] == 800
    )
    assert "execution_window" not in projected


@pytest.mark.parametrize(
    "case",
    [
        "unlinked",
        "missing_team",
        "missing_freeze",
        "late_checkpoint",
        "missing_final_fact",
        "disconnected",
        "side_changed",
        "entity_reused",
    ],
)
def test_unavailable_states_remain_explicit(case):
    source = source_fixture()
    frontier, round_ = source["initial_frontiers"][0], source["rounds"][0]
    if case == "unlinked":
        frontier["round_link"]["status"] = "excluded"
    if case == "missing_team":
        for snapshot in (frontier["round_start_probe"], round_["initial_state"]):
            snapshot["actors"][0]["facts"]["inventory"] = missing(
                "m_pWeaponServices.m_hMyWeapons"
            )
    if case == "missing_freeze":
        round_["freeze_end_tick"] = None
    if case == "late_checkpoint":
        round_["execution"]["freeze_snapshot"]["tick"] = 201
    final = round_["execution"]["freeze_snapshot"]["actors"][0]
    if case == "missing_final_fact":
        final["facts"]["money"] = missing("m_pInGameMoneyServices.m_iAccount")
    if case == "disconnected":
        final["connected"] = False
        final["prepurchase_reason"] = "participant_unavailable"
        final["facts"] = {k: missing(v["source"]) for k, v in final["facts"].items()}
    if case == "side_changed":
        final["side"] = "T"
    if case == "entity_reused":
        final["facts"]["inventory"]["value"][0]["weapon"] = "AWP"
    data, _, protocol = build(source)
    validate_data(data, source, protocol)
    row = data["opportunities"][0]
    if case in {"unlinked", "missing_team"}:
        with pytest.raises(RuntimeContractError):
            domain.initial_features(row)
        assert row["observed_action"]["actor_changes"] == []
    elif case == "entity_reused":
        assert (
            row["observed_action"]["actor_changes"][0]["inventory_transition"]["status"]
            == "unavailable"
        )
    else:
        fact = row["observed_action"]["actor_changes"][0]["changes"]["money"]
        assert (
            fact["status"] == "unavailable"
            and fact["after"] is None
            and fact["delta"] is None
        )


def test_zero_spend_and_refunds_never_become_eco_or_complete_purchase():
    source = source_fixture()
    for actor in source["rounds"][0]["execution"]["freeze_snapshot"]["actors"]:
        actor["facts"]["money"]["value"] = 800
        actor["facts"]["spent_native"]["value"] = 0
    source["rounds"][0]["execution"]["events"].append(event(21, 151, kind="refund"))
    data, _, protocol = build(source)
    validate_data(data, source, protocol)
    action = data["opportunities"][0]["observed_action"]
    assert action["action_id"] is None and action["basket"] is None
    assert action["status"] == "partial"
    assert action["actor_changes"][0]["changes"]["spent_native"]["delta"] == 0


def test_observed_convars_and_legacy_hashes_do_not_certify_prices_or_rewards():
    source = source_fixture()
    for snapshot in (
        source["rounds"][0]["initial_state"],
        source["initial_frontiers"][0]["round_start_probe"],
    ):
        snapshot["context"]["mp_maxmoney"] = observed("16000", "convar:mp_maxmoney")
    data, _, protocol = build(source)
    validate_data(data, source, protocol)
    rules = data["opportunities"][0]["rules"]
    assert rules["server_configuration"] == "observed"
    assert rules["missing_convars"] == []
    assert rules["comparison_allowed"] is False
    assert "reward_rules_and_prior_bonus_unverified" in rules["blockers"]


@pytest.mark.parametrize(
    "case",
    [
        "delta",
        "availability",
        "inventory",
        "initial_money",
        "opponent",
        "window",
        "boundary",
        "action",
        "price_compatibility",
        "summary_denominator",
        "winner",
    ],
)
def test_semantic_or_schema_mutations_are_rejected(case):
    data, source, protocol = build()
    row = data["opportunities"][0]
    action = row["observed_action"]
    if case == "delta":
        action["actor_changes"][0]["changes"]["money"]["delta"] = -600
    if case == "availability":
        action["actor_changes"][0]["changes"]["money"]["availability_tick"] = 100
    if case == "inventory":
        action["actor_changes"][0]["inventory_transition"]["added"] = []
    if case == "initial_money":
        row["initial_state"]["actors"][0]["facts"]["money"]["value"] = 16000
    if case == "opponent":
        row["initial_state"]["actors"].append(
            source["rounds"][0]["initial_state"]["actors"][5]
        )
    if case == "window":
        row["execution_window"]["end_exclusive_tick"] = 228
    if case == "boundary":
        action["boundary_callbacks"] = action["callbacks"]
        action["callbacks"] = []
    if case == "action":
        action["action_id"] = "ACT-ECO"
    if case == "price_compatibility":
        row["rules"]["comparison_allowed"] = True
    if case == "summary_denominator":
        data["opportunities"] = data["opportunities"][:1]
    if case == "winner":
        row["winner_side"] = "CT"
    with pytest.raises(RuntimeContractError):
        validate_data(data, source, protocol)


def test_rehashed_arithmetic_is_rejected_against_original_source(tmp_path):
    source = write_source(tmp_path / "source", source_fixture())
    output = tmp_path / "out"
    output.mkdir()
    io.materialize(source, output)
    data_path = output / domain.DATA_FILE
    data = json.loads(data_path.read_text())
    data["opportunities"][0]["observed_action"]["actor_changes"][0]["changes"]["money"][
        "delta"
    ] = -500
    encoded = json.dumps(data).encode()
    data_path.write_bytes(encoded)
    path = output / domain.MANIFEST_FILE
    manifest = json.loads(path.read_text())
    manifest["artifact"].update(bytes=len(encoded), sha256=io.sha(encoded))
    path.write_text(json.dumps(manifest))
    with pytest.raises(RuntimeContractError, match="source arithmetic"):
        validate_candidate(path, source)


@pytest.mark.parametrize("kind", ["dirty", "temp_root", "outside_temp", "nonexistent"])
def test_output_is_checked_before_source_processing(tmp_path, monkeypatch, kind):
    def unexpected(*args):
        raise AssertionError("source must not be processed")

    monkeypatch.setattr(io, "load_source", unexpected)
    output = tmp_path / "output"
    if kind == "dirty":
        output.mkdir()
        (output / "keep").write_text("preserve")
    elif kind == "temp_root":
        output = Path(tempfile.gettempdir())
    elif kind == "outside_temp":
        output = Path(__file__).resolve().parents[1]
    with pytest.raises(RuntimeContractError):
        io.materialize(Path("unused"), output)
    if kind == "dirty":
        assert (output / "keep").read_text() == "preserve"


def test_catalog_cannot_fall_back_to_generic_window():
    _, _, catalog, _ = bindings()
    next(d for d in catalog["decisions"] if d["id"] == "DEC-ECONOMY-001")[
        "action_label_window"
    ] = "0.25-2s after T0"
    with pytest.raises(ContractValidationError, match="freeze execution"):
        validate_contract(catalog)


def test_protocol_cannot_authorize_comparison():
    protocol, _, _, _ = bindings()
    protocol["comparison_allowed"] = True
    with pytest.raises(RuntimeContractError):
        io.check_schema(protocol, "economy_opportunity_protocol.schema.json")
