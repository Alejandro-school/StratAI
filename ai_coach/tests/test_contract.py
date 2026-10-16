from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest

from ai_coach.contracts.runtime_contracts import validate_runtime_contract_files
from ai_coach.contracts.validate_contract import (
    ContractValidationError,
    validate_contract,
)

CONTRACT_PATH = Path(__file__).parents[1] / "contracts" / "capability_catalog.json"
NAVIGATION_SCHEMA_PATH = (
    Path(__file__).parents[1] / "contracts" / "observed_navigation_path.schema.json"
)


def load_contract() -> dict:
    return json.loads(CONTRACT_PATH.read_text(encoding="utf-8"))


def test_repository_contract_is_valid() -> None:
    counts = validate_contract(load_contract())
    assert counts == {
        "capabilities": 17,
        "decisions": 15,
        "actions": 51,
        "outcomes": 18,
        "fields": 58,
    }


def test_runtime_contract_files_are_valid_and_versioned() -> None:
    assert validate_runtime_contract_files() == {"schemas": 116, "reason_codes": 19}


def test_navigation_timing_is_internal_contextual_and_not_a_single_eta() -> None:
    schema = json.loads(NAVIGATION_SCHEMA_PATH.read_text(encoding="utf-8"))
    assert schema["$id"] == "stratai.observed_navigation_path@2"
    assert schema["contract_version"] == "2.0.0"
    assert "travel_time_estimate" not in schema["properties"]

    timing = schema["$defs"]["internalTiming"]["properties"]
    assert timing["intended_use"]["const"] == "internal_decision_support_only"
    assert timing["exact_user_facing_time_allowed"]["const"] is False
    assert timing["arrival_prediction"]["const"] is False
    assert timing["user_facing_mode"]["const"].startswith("qualitative_")
    assert set(timing["scenarios"]["properties"]) == {
        "optimistic_uninterrupted_250ups_baseline",
        "active_weapon_uninterrupted_baseline",
        "observed_pace_projection",
    }

    provenance = schema["$defs"]["provenance"]["properties"]
    assert provenance["weapon_speed_table_contract"]["const"] == (
        "cs2_weapon_max_speed_table@1"
    )


def test_duplicate_id_fails_clearly() -> None:
    contract = load_contract()
    contract["actions"].append(copy.deepcopy(contract["actions"][0]))
    with pytest.raises(ContractValidationError, match="duplicate id"):
        validate_contract(contract)


def test_broken_reference_fails_clearly() -> None:
    contract = load_contract()
    contract["capabilities"][0]["outcome_ids"].append("OUT-DOES-NOT-EXIST")
    with pytest.raises(
        ContractValidationError, match="unknown reference OUT-DOES-NOT-EXIST"
    ):
        validate_contract(contract)


def test_missing_field_provenance_key_fails_clearly() -> None:
    contract = load_contract()
    del contract["fields"][0]["producer_service"]
    with pytest.raises(
        ContractValidationError, match="FLD-MATCH: missing keys producer_service"
    ):
        validate_contract(contract)
