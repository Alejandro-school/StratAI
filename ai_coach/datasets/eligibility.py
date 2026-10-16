"""Deterministic TaskEligibility evaluation for the 17 catalog capabilities."""

from __future__ import annotations

from collections import Counter
from collections.abc import Mapping
from typing import Any

from ai_coach.contracts.runtime_contracts import (
    CONTRACT_VERSION,
    TASK_ELIGIBILITY_SCHEMA_ID,
    validate_task_eligibilities,
)

DECISION_FAMILY_BY_ID = {
    "DEC-TRADE-001": "spacing_or_trade_connection",
    "DEC-PEEK-001": "peek_hold_or_reposition",
}
HARD_UNAVAILABLE_STATUSES = {
    "FALTA_EXTRAER",
    "REQUIERE_CLASIFICACION",
    "NO_OBSERVABLE",
    "DESCONOCIDO",
}


def _field_totals(
    field_coverage: Mapping[str, Mapping[str, Mapping[str, int]]],
    field_id: str,
    families: set[str],
) -> dict[str, int]:
    totals = Counter({"present": 0, "missing": 0, "null": 0})
    for family in families:
        values = field_coverage.get(field_id, {}).get(family, {})
        for key in totals:
            value = values.get(key, 0)
            if isinstance(value, int) and not isinstance(value, bool):
                totals[key] += value
    return dict(totals)


def evaluate_task_eligibility(
    catalog: Mapping[str, Any],
    *,
    family_counts: Mapping[str, int],
    field_coverage: Mapping[str, Mapping[str, Mapping[str, int]]],
    bundles_processed: int,
    temporal_violations: int,
    join_failures: int,
    bundle_validation_reference: str = "root manifest validation_status=passed",
) -> list[dict[str, Any]]:
    """Return one validated, stable result for every catalog capability."""

    if (
        not isinstance(bundle_validation_reference, str)
        or not bundle_validation_reference
    ):
        raise TypeError("bundle_validation_reference must be a non-empty string")

    capabilities = catalog.get("capabilities")
    fields = catalog.get("fields")
    if not isinstance(capabilities, list) or not isinstance(fields, list):
        raise TypeError("capability catalog must contain capabilities[] and fields[]")
    field_by_id = {
        field["id"]: field
        for field in fields
        if isinstance(field, Mapping) and isinstance(field.get("id"), str)
    }
    catalog_schema_id = str(catalog.get("schema_id") or "")
    catalog_version = str(catalog.get("version") or "")
    results: list[dict[str, Any]] = []

    for capability in sorted(capabilities, key=lambda item: str(item.get("id", ""))):
        capability_id = str(capability.get("id") or "")
        decision_ids = capability.get("observed_action")
        eligibility = capability.get("task_eligibility")
        if not isinstance(decision_ids, list) or not isinstance(eligibility, Mapping):
            raise TypeError(
                f"{capability_id}: invalid decision or eligibility contract"
            )
        required_fields = eligibility.get("required_fields")
        if not isinstance(required_fields, list):
            raise TypeError(f"{capability_id}: required_fields must be a list")

        materialized_ids: list[str] = []
        materialized_families: set[str] = set()
        for decision_id in decision_ids:
            family = DECISION_FAMILY_BY_ID.get(str(decision_id))
            if family and family_counts.get(family, 0) > 0:
                materialized_ids.append(str(decision_id))
                materialized_families.add(family)

        relevant_examples = sum(
            family_counts.get(family, 0) for family in materialized_families
        )
        missing: set[str] = set()
        blockers: list[dict[str, Any]] = []
        reasons: set[str] = set()

        for field_id_value in required_fields:
            field_id = str(field_id_value)
            field = field_by_id.get(field_id)
            availability = (
                field.get("availability_status")
                if isinstance(field, Mapping)
                else "DESCONOCIDO"
            )
            totals = _field_totals(field_coverage, field_id, materialized_families)
            if availability == "PARCIAL":
                missing.add(field_id)
                reasons.add("PARTIAL_PROXY_ONLY")
                blockers.append(
                    {
                        "reason_code": "PARTIAL_PROXY_ONLY",
                        "field_id": field_id,
                        "detail": "El catálogo sólo autoriza esta señal como proxy parcial.",
                    }
                )
            elif availability in HARD_UNAVAILABLE_STATUSES:
                missing.add(field_id)
                reasons.add("REQUIRED_FIELD_UNAVAILABLE")
                blockers.append(
                    {
                        "reason_code": "REQUIRED_FIELD_UNAVAILABLE",
                        "field_id": field_id,
                        "detail": f"Disponibilidad declarada por el catálogo: {availability}.",
                    }
                )
            elif materialized_families and totals["present"] == 0:
                missing.add(field_id)
                reasons.add("REQUIRED_FIELD_MISSING")
                blockers.append(
                    {
                        "reason_code": "REQUIRED_FIELD_MISSING",
                        "field_id": field_id,
                        "detail": "No hubo ningún valor no nulo utilizable en los ejemplos materializados.",
                    }
                )

        if not materialized_ids:
            status = "not_materialized"
            reasons.add("DECISION_FAMILY_NOT_MATERIALIZED")
            blockers.append(
                {
                    "reason_code": "DECISION_FAMILY_NOT_MATERIALIZED",
                    "field_id": None,
                    "detail": "Los bundles no contienen ninguno de los tipos de decisión de esta capacidad.",
                }
            )
        else:
            if temporal_violations:
                reasons.add("T0_AVAILABILITY_VIOLATION")
                blockers.append(
                    {
                        "reason_code": "T0_AVAILABILITY_VIOLATION",
                        "field_id": None,
                        "detail": f"Se detectaron {temporal_violations} violaciones de disponibilidad temporal.",
                    }
                )
            if join_failures:
                reasons.add("JOIN_REFERENCE_MISSING")
                blockers.append(
                    {
                        "reason_code": "JOIN_REFERENCE_MISSING",
                        "field_id": None,
                        "detail": f"Se detectaron {join_failures} joins fallidos.",
                    }
                )
            if blockers:
                status = "ineligible"
            else:
                status = "eligible"
                reasons = {"ELIGIBLE_REQUIRED_EVIDENCE_PRESENT"}

        evidence_used = [
            {
                "kind": "bundle_validation",
                "reference": bundle_validation_reference,
                "count": bundles_processed,
            },
            {
                "kind": "catalog",
                "reference": "ai_coach/contracts/capability_catalog.json",
                "count": len(required_fields),
            },
        ]
        for family in sorted(materialized_families):
            evidence_used.append(
                {
                    "kind": "decision_family",
                    "reference": family,
                    "count": family_counts.get(family, 0),
                }
            )
        if materialized_families:
            evidence_used.append(
                {
                    "kind": "field_coverage",
                    "reference": "DecisionExample observed partitions",
                    "count": relevant_examples,
                }
            )

        results.append(
            {
                "schema_id": TASK_ELIGIBILITY_SCHEMA_ID,
                "contract_version": CONTRACT_VERSION,
                "catalog_schema_id": catalog_schema_id,
                "catalog_version": catalog_version,
                "capability_id": capability_id,
                "status": status,
                "reason_codes": sorted(reasons),
                "missing_required_fields": sorted(missing),
                "materialized_decision_types": sorted(materialized_ids),
                "blockers": sorted(
                    blockers,
                    key=lambda item: (
                        item["reason_code"],
                        item["field_id"] or "",
                        item["detail"],
                    ),
                ),
                "evidence_used": evidence_used,
            }
        )

    validate_task_eligibilities(results)
    return results
