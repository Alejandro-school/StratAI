"""Validate the machine-readable AI Coach capability contract.

This validator is intentionally dependency-free so it can run in CI before any
dataset or model job.  It never mutates the contract.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


ALLOWED_AVAILABILITY = {
    "EXISTE_VERIFICADO",
    "DERIVABLE",
    "PARCIAL",
    "FALTA_EXTRAER",
    "REQUIERE_CLASIFICACION",
    "NO_OBSERVABLE",
}
REQUIRED_CAPABILITY_KEYS = {
    "id",
    "category",
    "question",
    "affected_players",
    "start_condition",
    "t0",
    "end_condition",
    "required_history",
    "label_outcome_window",
    "observed_action",
    "physically_possible_alternatives",
    "required_observed_fields",
    "permitted_oracle_for_labels",
    "outcome_ids",
    "confounders",
    "exclusions",
    "task_eligibility",
    "initial_method",
    "evaluation_metrics",
    "minimum_support",
    "confidence",
    "abstention_conditions",
    "user_evidence",
    "implementation_status",
    "required_tests",
}
REQUIRED_DECISION_KEYS = {
    "id",
    "type",
    "start_condition",
    "t0_definition",
    "end_condition",
    "deduplication",
    "history_window",
    "action_label_window",
}
REQUIRED_ACTION_KEYS = {"id", "family", "required_physical_parameters", "candidate_valid_only_if"}
REQUIRED_OUTCOME_KEYS = {
    "id",
    "partition",
    "meaning",
    "horizon_seconds",
    "unit",
    "label_only",
    "not_a_correctness_label",
}
REQUIRED_FIELD_KEYS = {
    "id",
    "partition",
    "availability_status",
    "artifact",
    "json_path",
    "producer_service",
    "producer_code",
    "unit",
    "frequency",
    "meaning",
    "evidence_ref",
}


class ContractValidationError(ValueError):
    """Raised when a contract violates a structural or reference invariant."""


def _indexed(items: Any, collection: str, errors: list[str]) -> dict[str, dict[str, Any]]:
    if not isinstance(items, list):
        errors.append(f"{collection}: expected a list")
        return {}
    result: dict[str, dict[str, Any]] = {}
    for index, item in enumerate(items):
        if not isinstance(item, dict):
            errors.append(f"{collection}[{index}]: expected an object")
            continue
        item_id = item.get("id")
        if not isinstance(item_id, str) or not item_id:
            errors.append(f"{collection}[{index}]: missing non-empty id")
            continue
        if item_id in result:
            errors.append(f"{collection}: duplicate id {item_id}")
        result[item_id] = item
    return result


def _check_refs(
    source_id: str,
    key: str,
    values: Any,
    targets: dict[str, Any],
    errors: list[str],
) -> None:
    if not isinstance(values, list) or not values:
        errors.append(f"{source_id}.{key}: expected a non-empty list")
        return
    for value in values:
        if value not in targets:
            errors.append(f"{source_id}.{key}: unknown reference {value}")


def validate_contract(contract: dict[str, Any]) -> dict[str, int]:
    """Return collection counts or raise with every detected violation."""

    errors: list[str] = []
    if contract.get("schema_id") != "stratai.ai_coach_capability_catalog@1":
        errors.append("schema_id: expected stratai.ai_coach_capability_catalog@1")

    capabilities = _indexed(contract.get("capabilities"), "capabilities", errors)
    decisions = _indexed(contract.get("decisions"), "decisions", errors)
    actions = _indexed(contract.get("actions"), "actions", errors)
    outcomes = _indexed(contract.get("outcomes"), "outcomes", errors)
    fields = _indexed(contract.get("fields"), "fields", errors)

    all_ids: dict[str, str] = {}
    for collection, items in (
        ("capabilities", capabilities),
        ("decisions", decisions),
        ("actions", actions),
        ("outcomes", outcomes),
        ("fields", fields),
    ):
        for item_id in items:
            previous = all_ids.get(item_id)
            if previous:
                errors.append(f"global duplicate id {item_id} in {previous} and {collection}")
            all_ids[item_id] = collection

    for capability_id, capability in capabilities.items():
        missing = sorted(REQUIRED_CAPABILITY_KEYS - capability.keys())
        if missing:
            errors.append(f"{capability_id}: missing keys {', '.join(missing)}")
        _check_refs(capability_id, "observed_action", capability.get("observed_action"), decisions, errors)
        _check_refs(
            capability_id,
            "physically_possible_alternatives",
            capability.get("physically_possible_alternatives"),
            actions,
            errors,
        )
        _check_refs(
            capability_id,
            "required_observed_fields",
            capability.get("required_observed_fields"),
            fields,
            errors,
        )
        permitted_oracle = capability.get("permitted_oracle_for_labels")
        if permitted_oracle:
            _check_refs(capability_id, "permitted_oracle_for_labels", permitted_oracle, fields, errors)
        _check_refs(capability_id, "outcome_ids", capability.get("outcome_ids"), outcomes, errors)

        eligibility = capability.get("task_eligibility")
        if not isinstance(eligibility, dict):
            errors.append(f"{capability_id}.task_eligibility: expected an object")
        else:
            _check_refs(
                capability_id,
                "task_eligibility.required_fields",
                eligibility.get("required_fields"),
                fields,
                errors,
            )

    declared_partitions = contract.get("partitions")
    if not isinstance(declared_partitions, dict):
        errors.append("partitions: expected an object")
        allowed_partitions: set[str] = set()
    else:
        allowed_partitions = set(declared_partitions)
        for required_partition in ("observed", "oracle", "outcomes"):
            if required_partition not in allowed_partitions:
                errors.append(f"partitions: missing {required_partition}")

    for field_id, field in fields.items():
        missing = sorted(REQUIRED_FIELD_KEYS - field.keys())
        if missing:
            errors.append(f"{field_id}: missing keys {', '.join(missing)}")
        status = field.get("availability_status")
        if status not in ALLOWED_AVAILABILITY:
            errors.append(f"{field_id}.availability_status: invalid value {status!r}")
        if field.get("partition") not in allowed_partitions:
            errors.append(f"{field_id}.partition: invalid value {field.get('partition')!r}")

    for action_id, action in actions.items():
        missing = sorted(REQUIRED_ACTION_KEYS - action.keys())
        if missing:
            errors.append(f"{action_id}: missing keys {', '.join(missing)}")
        params = action.get("required_physical_parameters")
        if not isinstance(params, list) or not params:
            errors.append(f"{action_id}.required_physical_parameters: expected a non-empty list")

    for decision_id, decision in decisions.items():
        missing = sorted(REQUIRED_DECISION_KEYS - decision.keys())
        if missing:
            errors.append(f"{decision_id}: missing keys {', '.join(missing)}")

    for outcome_id, outcome in outcomes.items():
        missing = sorted(REQUIRED_OUTCOME_KEYS - outcome.keys())
        if missing:
            errors.append(f"{outcome_id}: missing keys {', '.join(missing)}")

    if errors:
        raise ContractValidationError("\n".join(errors))
    return {
        "capabilities": len(capabilities),
        "decisions": len(decisions),
        "actions": len(actions),
        "outcomes": len(outcomes),
        "fields": len(fields),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "contract",
        nargs="?",
        type=Path,
        default=Path(__file__).with_name("capability_catalog.json"),
    )
    args = parser.parse_args()
    try:
        contract = json.loads(args.contract.read_text(encoding="utf-8"))
        counts = validate_contract(contract)
    except (OSError, json.JSONDecodeError, ContractValidationError) as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False, sort_keys=True))
        return 1
    print(json.dumps({"ok": True, "counts": counts}, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
