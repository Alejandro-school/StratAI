from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest

from ai_coach.contracts.validate_contract import ContractValidationError, validate_contract


CONTRACT_PATH = Path(__file__).parents[1] / "contracts" / "capability_catalog.json"


def load_contract() -> dict:
    return json.loads(CONTRACT_PATH.read_text(encoding="utf-8"))


def test_repository_contract_is_valid() -> None:
    counts = validate_contract(load_contract())
    assert counts == {
        "capabilities": 17,
        "decisions": 15,
        "actions": 51,
        "outcomes": 18,
        "fields": 55,
    }


def test_duplicate_id_fails_clearly() -> None:
    contract = load_contract()
    contract["actions"].append(copy.deepcopy(contract["actions"][0]))
    with pytest.raises(ContractValidationError, match="duplicate id"):
        validate_contract(contract)


def test_broken_reference_fails_clearly() -> None:
    contract = load_contract()
    contract["capabilities"][0]["outcome_ids"].append("OUT-DOES-NOT-EXIST")
    with pytest.raises(ContractValidationError, match="unknown reference OUT-DOES-NOT-EXIST"):
        validate_contract(contract)


def test_missing_field_provenance_key_fails_clearly() -> None:
    contract = load_contract()
    del contract["fields"][0]["producer_service"]
    with pytest.raises(ContractValidationError, match="FLD-MATCH: missing keys producer_service"):
        validate_contract(contract)
