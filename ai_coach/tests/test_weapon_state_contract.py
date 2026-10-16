from __future__ import annotations

import copy
import json
import math
from pathlib import Path

import pytest

from ai_coach.contracts.runtime_contracts import (
    OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
    OBSERVED_WEAPON_STATE_SCHEMA_ID,
    RuntimeContractError,
    validate_observed_weapon_state,
    validate_observed_weapon_state_ledger,
)

SCHEMA_PATH = (
    Path(__file__).parents[1] / "contracts" / "observed_weapon_state.schema.json"
)


def _observed_fact(
    value: object,
    source: str,
    *,
    tick: int = 100,
    reconciliation: str = "not_applicable",
) -> dict[str, object]:
    return {
        "status": "observed",
        "value": value,
        "availability_tick": tick,
        "provenance": {
            "source": source,
            "source_tick": tick,
            "reconciliation": reconciliation,
        },
    }


def _unavailable_fact(
    reconciliation: str = "unavailable",
) -> dict[str, object]:
    return {
        "status": "unavailable",
        "value": None,
        "availability_tick": None,
        "provenance": {
            "source": "unavailable",
            "source_tick": None,
            "reconciliation": reconciliation,
        },
    }


def _row(
    *,
    weapon: str = "AK-47",
    magazine: int = 30,
    reserve: int = 90,
    reloading: bool = False,
    tick: int = 100,
    player_id: str = "participant-001",
) -> dict[str, object]:
    return {
        "schema_id": OBSERVED_WEAPON_STATE_SCHEMA_ID,
        "contract_version": OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
        "join_keys": {
            "match_id": "match-local",
            "round_id": "match-local:round:001",
            "player_id": player_id,
            "observation_id": (
                f"match-local:round:001:weapon-state:{tick}:{player_id}"
            ),
        },
        "round_number": 1,
        "tick": tick,
        "t0_tick": tick,
        "active_weapon": _observed_fact(
            weapon, "replay_active_weapon_snapshot", tick=tick
        ),
        "ammo_in_magazine": _observed_fact(
            magazine, "demoinfocs_active_weapon_ammo", tick=tick
        ),
        "ammo_reserve": _observed_fact(
            reserve, "demoinfocs_active_weapon_ammo", tick=tick
        ),
        "is_reloading": _observed_fact(
            reloading, "demoinfocs_causal_reload_tracker", tick=tick
        ),
        "abstention_reasons": [],
    }


def _ledger(*rows: dict[str, object]) -> dict[str, object]:
    return {
        "schema_id": OBSERVED_WEAPON_STATE_SCHEMA_ID,
        "contract_version": OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
        "sampling": {
            "target_hz": 16,
            "strategy": "round_anchored_right_closed_no_carry_forward",
        },
        "rows": list(rows),
    }


def test_schema_is_versioned_strict_and_separate_from_canonical_380() -> None:
    schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))
    assert schema["$id"] == OBSERVED_WEAPON_STATE_SCHEMA_ID
    assert schema["contract_version"] == OBSERVED_WEAPON_STATE_CONTRACT_VERSION
    assert schema["additionalProperties"] is False
    assert schema["properties"]["sampling"]["$ref"] == "#/$defs/sampling"
    assert "3.8.0" in schema["description"]


def test_positive_ammo_reserve_and_reload_true_are_observed() -> None:
    row = _row(magazine=21, reserve=63, reloading=True)
    validate_observed_weapon_state(row)
    assert row["ammo_in_magazine"]["value"] == 21
    assert row["ammo_reserve"]["value"] == 63
    assert row["is_reloading"]["value"] is True


def test_observed_zero_and_reload_false_are_not_unavailable() -> None:
    row = _row(magazine=0, reserve=0, reloading=False)
    validate_observed_weapon_state(row)
    assert row["ammo_in_magazine"] == _observed_fact(
        0, "demoinfocs_active_weapon_ammo"
    )
    assert row["ammo_reserve"] == _observed_fact(
        0, "demoinfocs_active_weapon_ammo"
    )
    assert row["is_reloading"]["status"] == "observed"
    assert row["is_reloading"]["value"] is False


def test_absent_weapon_and_missing_callbacks_remain_unavailable() -> None:
    row = _row()
    for field in (
        "active_weapon",
        "ammo_in_magazine",
        "ammo_reserve",
        "is_reloading",
    ):
        row[field] = _unavailable_fact()
    row["abstention_reasons"] = ["active_weapon_snapshot_unavailable"]
    validate_observed_weapon_state(row)
    assert all(
        row[field]["value"] is None
        for field in (
            "active_weapon",
            "ammo_in_magazine",
            "ammo_reserve",
            "is_reloading",
        )
    )


@pytest.mark.parametrize(
    "weapon", ["Knife", "C4", "HE Grenade", "Flashbang", "Molotov"]
)
def test_non_ammo_equipment_cannot_turn_sentinels_into_zero(weapon: str) -> None:
    row = _row(weapon=weapon)
    for field in ("ammo_in_magazine", "ammo_reserve", "is_reloading"):
        row[field] = _unavailable_fact("not_applicable")
    row["abstention_reasons"] = ["active_weapon_does_not_use_ammo"]
    validate_observed_weapon_state(row)

    fabricated = copy.deepcopy(row)
    fabricated["ammo_in_magazine"] = _observed_fact(
        0, "demoinfocs_active_weapon_ammo"
    )
    with pytest.raises(RuntimeContractError, match="cannot publish ammo/reload"):
        validate_observed_weapon_state(fabricated)

    mislabeled = copy.deepcopy(row)
    mislabeled["ammo_reserve"] = _unavailable_fact("unavailable")
    with pytest.raises(RuntimeContractError, match="not_applicable provenance"):
        validate_observed_weapon_state(mislabeled)


@pytest.mark.parametrize("sentinel", [-1, -999, math.nan, math.inf, -math.inf])
def test_negative_and_non_finite_ammo_sentinels_are_rejected(
    sentinel: float,
) -> None:
    row = _row()
    row["ammo_in_magazine"]["value"] = sentinel
    with pytest.raises(RuntimeContractError, match="non-negative observed integer"):
        validate_observed_weapon_state(row)


@pytest.mark.parametrize(
    "field", ["active_weapon", "ammo_in_magazine", "ammo_reserve", "is_reloading"]
)
def test_availability_after_t0_is_rejected_per_fact(field: str) -> None:
    row = _row()
    row[field]["availability_tick"] = 101
    with pytest.raises(RuntimeContractError, match="must be <= t0_tick"):
        validate_observed_weapon_state(row)


def test_source_tick_must_equal_availability() -> None:
    row = _row()
    row["ammo_reserve"]["provenance"]["source_tick"] = 101
    with pytest.raises(RuntimeContractError, match="must equal availability_tick"):
        validate_observed_weapon_state(row)


def test_null_status_and_provenance_combinations_fail_closed() -> None:
    unavailable_with_zero = _row()
    unavailable_with_zero["ammo_in_magazine"] = _unavailable_fact()
    unavailable_with_zero["ammo_in_magazine"]["value"] = 0
    unavailable_with_zero["abstention_reasons"] = ["ammo_snapshot_unavailable"]
    with pytest.raises(RuntimeContractError, match="require null value"):
        validate_observed_weapon_state(unavailable_with_zero)

    observed_without_value = _row()
    observed_without_value["ammo_in_magazine"]["value"] = None
    with pytest.raises(RuntimeContractError, match="non-negative observed integer"):
        validate_observed_weapon_state(observed_without_value)

    inferred = _row()
    inferred["ammo_in_magazine"]["status"] = "inferred"
    with pytest.raises(RuntimeContractError, match="observed or unavailable"):
        validate_observed_weapon_state(inferred)


@pytest.mark.parametrize(
    ("field", "extra_key"),
    [
        ("row", "future_ammo"),
        ("row", "oracle_weapon"),
        ("active_weapon", "nominal_capacity"),
        ("ammo_in_magazine.provenance", "source_event_id"),
        ("ammo_reserve", "player_id"),
    ],
)
def test_future_oracle_nominal_capacity_and_feature_ids_are_forbidden(
    field: str, extra_key: str
) -> None:
    row = _row()
    target: dict[str, object] = row
    for part in field.split("."):
        if part == "row":
            continue
        target = target[part]
    target[extra_key] = "forbidden"
    with pytest.raises(RuntimeContractError, match="unknown fields"):
        validate_observed_weapon_state(row)


def test_weapon_name_does_not_prove_ammo_reserve_or_reload() -> None:
    row = _row(weapon="AK-47")
    for field in ("ammo_in_magazine", "ammo_reserve", "is_reloading"):
        row[field] = _unavailable_fact()
    row["abstention_reasons"] = ["weapon_state_callbacks_unavailable"]
    validate_observed_weapon_state(row)
    assert row["active_weapon"]["status"] == "observed"
    assert row["ammo_in_magazine"]["value"] is None
    assert row["ammo_reserve"]["value"] is None
    assert row["is_reloading"]["value"] is None


def test_matching_fire_and_reload_reconciliation_are_accepted() -> None:
    fired = _row()
    for field in ("active_weapon", "ammo_in_magazine", "ammo_reserve"):
        fired[field]["provenance"]["reconciliation"] = "confirmed_weapon_fire"
    validate_observed_weapon_state(fired)

    reloaded = _row(magazine=30, reserve=60, reloading=True)
    for field in (
        "active_weapon",
        "ammo_in_magazine",
        "ammo_reserve",
        "is_reloading",
    ):
        reloaded[field]["provenance"]["reconciliation"] = (
            "confirmed_weapon_reload"
        )
    validate_observed_weapon_state(reloaded)


def test_mismatched_or_ambiguous_reconciliation_cannot_confirm_a_fact() -> None:
    mismatched = _row()
    mismatched["ammo_in_magazine"]["provenance"]["reconciliation"] = (
        "confirmed_weapon_fire"
    )
    with pytest.raises(RuntimeContractError, match="requires matching active weapon"):
        validate_observed_weapon_state(mismatched)

    ambiguous = _row()
    ambiguous["ammo_in_magazine"] = _unavailable_fact("ambiguous")
    ambiguous["abstention_reasons"] = ["weapon_fire_reconciliation_ambiguous"]
    validate_observed_weapon_state(ambiguous)
    assert ambiguous["ammo_in_magazine"]["status"] == "unavailable"


def test_ledger_requires_unique_canonically_ordered_rows_and_reasons() -> None:
    first = _row(tick=100, player_id="participant-001")
    second = _row(tick=104, player_id="participant-001")
    assert validate_observed_weapon_state_ledger(_ledger(first, second)) == 2

    with pytest.raises(RuntimeContractError, match="ordered deterministically"):
        validate_observed_weapon_state_ledger(_ledger(second, first))

    duplicate = copy.deepcopy(first)
    with pytest.raises(RuntimeContractError, match="duplicate observation_id"):
        validate_observed_weapon_state_ledger(_ledger(first, duplicate))

    unavailable = copy.deepcopy(first)
    unavailable["ammo_reserve"] = _unavailable_fact()
    unavailable["abstention_reasons"] = ["z_reason", "a_reason"]
    with pytest.raises(RuntimeContractError, match="sorted deterministically"):
        validate_observed_weapon_state(unavailable)


def test_identity_fields_are_confined_to_join_keys() -> None:
    row = _row()
    assert set(row["join_keys"]) == {
        "match_id",
        "round_id",
        "player_id",
        "observation_id",
    }
    for field in (
        "active_weapon",
        "ammo_in_magazine",
        "ammo_reserve",
        "is_reloading",
    ):
        assert all(not key.endswith("_id") for key in row[field])
        assert all(not key.endswith("_id") for key in row[field]["provenance"])
    validate_observed_weapon_state(row)


@pytest.mark.parametrize("raw_identity", ["76561198000000000", "Player Name"])
def test_raw_player_ids_and_names_are_rejected(raw_identity: str) -> None:
    row = _row(player_id=raw_identity)
    with pytest.raises(RuntimeContractError, match="opaque alias"):
        validate_observed_weapon_state(row)
