"""Compare observed facts with historical references without approving game rules."""

from __future__ import annotations

import argparse
import json
from decimal import Decimal, InvalidOperation
from pathlib import Path

from ai_coach.economy import native_sources, rule_assets
from ai_coach.economy.candidate_io import _write_new, validate_output_directory
from ai_coach.economy.native_reconstruction import rules as native_identity
from ai_coach.economy.rule_asset_text import require

REPORT = "economy_rule_assessment.json"
BLOCKERS = [
    "exact_demo_to_asset_build_binding_unverified",
    "server_overrides_not_proved_by_reference_defaults",
    "prior_loss_bonus_state_unverified",
    "purchasable_loadout_and_limits_unverified",
    "complete_transactions_and_refund_rules_unverified",
    "teammate_acceptance_unobserved",
]


def identity_assessment(data, package):
    identity = native_identity(data)["game_identity"]
    patch, network = identity["patch_version"], identity["network_protocol"]
    candidates = []
    if any(row["status"] == "conflicting" for row in identity.values()) or (
        patch["value"] is not None
        and network["value"] is not None
        and patch["value"] != network["value"]
    ):
        status = "native_identity_conflicting"
    elif patch["status"] != "observed":
        status = "native_patch_unavailable"
    else:
        # Diagnostic lookup only. This normalization is NOT a verified build mapping.
        candidates = [
            r["revision"]
            for r in package["releases"]
            if int(r["version"]["PatchVersion"].replace(".", "")) == patch["value"]
        ]
        status = (
            "patch_label_candidates_only" if candidates else "no_patch_label_candidate"
        )
    return {
        "native": identity,
        "candidate_revisions": candidates,
        "lookup_status": status,
        "lookup_method": "remove_patch_label_dots_diagnostic_only_not_build_equivalence",
        "build_binding": "unverified",
        "compatible_for_comparison": False,
    }


def consensus(values):
    if not values or any(value is None for value in values):
        return "unavailable", None
    if len(set(values)) != 1:
        return "conflicting", None
    return "reference", values[0]


def reference_maps(package, revisions):
    releases = [r for r in package["releases"] if r["revision"] in revisions]
    return (
        [
            {r["item_definition_index"]: r["price"]["value"] for r in release["items"]}
            for release in releases
        ],
        [{r["name"]: r["value"] for r in release["defaults"]} for release in releases],
    )


def decimal_value(value):
    try:
        result = Decimal(value)
        return result if result.is_finite() else None
    except (InvalidOperation, ValueError, TypeError):
        return None


def native_convar(round_, name):
    fact = round_["initial_state"]["context"].get(name)
    if fact is None:
        return "not_captured_by_source_contract", None, None
    if fact["status"] != "observed":
        return "native_value_unavailable", None, None
    tick = fact["availability_tick"]
    if (
        fact["observation_tick"] is None
        or tick is None
        or tick > round_["start_tick"]
        or fact["observation_tick"] > round_["start_tick"]
    ):
        return "not_available_at_round_start", None, None
    return "observed", str(fact["value"]), tick


def build_report(data, source, package, package_hash, bindings):
    identity = identity_assessment(data, package)
    prices, defaults = reference_maps(package, identity["candidate_revisions"])
    price_rows = []
    for snapshot in data["observations"]["snapshots"]:
        if snapshot["sellback"]["status"] != "observed" or not snapshot["connected"]:
            continue
        for index, entry in enumerate(snapshot["sellback"]["entries"]):
            item, cost = entry["m_unDefIdx"]["value"], entry["m_nCost"]["value"]
            status, reference = consensus([table.get(item) for table in prices])
            result = (
                ("equal_reference" if cost == reference else "different_reference")
                if status == "reference"
                else "reference_" + status
            )
            price_rows.append(
                {
                    "observation_id": snapshot["observation_id"],
                    "entry_index": index,
                    "instance": snapshot["instance"],
                    "actor_alias": snapshot["actor_alias"],
                    "tick": snapshot["tick"],
                    "item_definition_index": item,
                    "recorded_cost": cost,
                    "reference_price": reference,
                    "result": result,
                    "effective_shop_price": None,
                    "player_error": False,
                }
            )
    names = sorted(
        {r["name"] for release in package["releases"] for r in release["defaults"]}
        | {
            name
            for round_ in source["rounds"]
            for name in round_["initial_state"]["context"]
            if name.startswith("mp_")
        }
    )
    configuration = []
    for round_ in source["rounds"]:
        for name in names:
            native_status, value, tick = native_convar(round_, name)
            ref_status, ref = consensus([table.get(name) for table in defaults])
            left, right = decimal_value(value), decimal_value(ref)
            result = "not_comparable"
            if (
                native_status == "observed"
                and ref_status == "reference"
                and left is not None
                and right is not None
            ):
                result = "equal_reference" if left == right else "different_reference"
            configuration.append(
                {
                    "instance": round_["instance"],
                    "t0_tick": round_["start_tick"],
                    "name": name,
                    "native_status": native_status,
                    "native_value": value,
                    "availability_tick": tick,
                    "native_source": "convar:" + name,
                    "reference_status": ref_status,
                    "reference_default": ref,
                    "result": result,
                    "default_applied_to_demo": False,
                }
            )
    report = {
        "schema_id": "stratai.economy_rule_assessment@1",
        "contract_version": rule_assets.VERSION,
        **{k: data[k] for k in ("source_alias", "source_sha256", "build_identifier")},
        **bindings,
        "rule_package_sha256": package_hash,
        "rule_registry_sha256": package["registry_sha256"],
        **rule_assets.GATES,
        "status": "reference_audit_not_rule_approval",
        "identity": identity,
        "price_partition": "execution_diagnostics_only_never_initial_features",
        "price_checks": price_rows,
        "source_coverage": source_coverage(data),
        "configuration_checks": configuration,
        "blockers": BLOCKERS.copy(),
    }
    report["summary"] = summary(report)
    return report


def source_coverage(data):
    rows = data["observations"]["snapshots"]
    return {
        "snapshots": len(rows),
        "connected_complete_lists": sum(
            row["connected"] and row["sellback"]["status"] == "observed" for row in rows
        ),
        "disconnected_snapshots": sum(not row["connected"] for row in rows),
        "connected_incomplete_lists": sum(
            row["connected"] and row["sellback"]["status"] != "observed" for row in rows
        ),
    }


def summary(report):
    return {
        "price_checks": len(report["price_checks"]),
        "equal_reference_prices": sum(
            r["result"] == "equal_reference" for r in report["price_checks"]
        ),
        "different_reference_prices": sum(
            r["result"] == "different_reference" for r in report["price_checks"]
        ),
        "unavailable_reference_prices": sum(
            r["reference_price"] is None for r in report["price_checks"]
        ),
        "configuration_checks": len(report["configuration_checks"]),
        "native_configuration_observed": sum(
            r["native_status"] == "observed" for r in report["configuration_checks"]
        ),
        "different_reference_defaults": sum(
            r["result"] == "different_reference" for r in report["configuration_checks"]
        ),
        "approved_rule_packages": 0,
        "classified_player_errors": 0,
    }


def validate_report(report, data, source, package, package_hash, bindings):
    """Recompute joins and comparisons from inputs; never call build_report."""
    rule_assets.check_schema(report, "economy_rule_assessment.schema.json")
    for key, value in {
        **bindings,
        **{k: data[k] for k in ("source_alias", "source_sha256", "build_identifier")},
        "rule_package_sha256": package_hash,
        "rule_registry_sha256": package["registry_sha256"],
    }.items():
        require(report[key] == value, "rule report binding differs")
    require(
        report["identity"] == identity_assessment(data, package),
        "rule identity assessment differs",
    )
    require(
        report["source_coverage"] == source_coverage(data),
        "rule source coverage differs",
    )
    prices, defaults = reference_maps(
        package, report["identity"]["candidate_revisions"]
    )
    expected_entries = [
        (s, i, entry)
        for s in data["observations"]["snapshots"]
        if s["connected"] and s["sellback"]["status"] == "observed"
        for i, entry in enumerate(s["sellback"]["entries"])
    ]
    require(
        len(report["price_checks"]) == len(expected_entries), "price coverage differs"
    )
    for row, (snapshot, index, entry) in zip(
        report["price_checks"], expected_entries, strict=True
    ):
        item, cost = entry["m_unDefIdx"]["value"], entry["m_nCost"]["value"]
        for key, value in {
            "observation_id": snapshot["observation_id"],
            "entry_index": index,
            "instance": snapshot["instance"],
            "actor_alias": snapshot["actor_alias"],
            "tick": snapshot["tick"],
            "item_definition_index": item,
            "recorded_cost": cost,
        }.items():
            require(row[key] == value, "price provenance differs")
        candidates = [p.get(item) for p in prices]
        available = bool(candidates) and None not in candidates
        unique = set(candidates)
        reference = candidates[0] if available and len(unique) == 1 else None
        result = (
            ("equal_reference" if reference == cost else "different_reference")
            if reference is not None
            else "reference_conflicting"
            if available and len(unique) > 1
            else "reference_unavailable"
        )
        require(
            row["reference_price"] == reference and row["result"] == result,
            "price comparison differs",
        )
    names = sorted(
        {r["name"] for release in package["releases"] for r in release["defaults"]}
        | {
            name
            for round_ in source["rounds"]
            for name in round_["initial_state"]["context"]
            if name.startswith("mp_")
        }
    )
    expected_keys = [(r["instance"], name) for r in source["rounds"] for name in names]
    require(
        [(r["instance"], r["name"]) for r in report["configuration_checks"]]
        == expected_keys,
        "configuration coverage differs",
    )
    rounds = {r["instance"]: r for r in source["rounds"]}
    for row in report["configuration_checks"]:
        round_ = rounds[row["instance"]]
        status, value, tick = native_convar(round_, row["name"])
        require(
            (row["native_status"], row["native_value"], row["availability_tick"])
            == (status, value, tick)
            and row["t0_tick"] == round_["start_tick"]
            and row["native_source"] == "convar:" + row["name"],
            "native configuration provenance differs",
        )
        ref_status, ref = consensus([d.get(row["name"]) for d in defaults])
        require(
            row["reference_status"] == ref_status and row["reference_default"] == ref,
            "reference default differs",
        )
        left, right = decimal_value(value), decimal_value(ref)
        expected = (
            "not_comparable"
            if status != "observed"
            or ref_status != "reference"
            or left is None
            or right is None
            else "equal_reference"
            if left == right
            else "different_reference"
        )
        require(row["result"] == expected, "configuration comparison differs")
    require(report["summary"] == summary(report), "rule summary differs")
    return {
        "ok": True,
        "summary": report["summary"],
        "independently_recomputed_price_checks": len(expected_entries),
        "independently_recomputed_configuration_checks": len(expected_keys),
    }


def materialize(native_path: Path, package_path: Path, sources: Path, output: Path):
    output = validate_output_directory(output)
    data, source, bindings = native_sources.load_candidate(
        native_path, require_observed=True
    )
    package, package_hash = rule_assets.load_package(package_path, sources)
    report = build_report(data, source, package, package_hash, bindings)
    validation = validate_report(report, data, source, package, package_hash, bindings)
    require(
        validation["summary"]["price_checks"] > 0
        and validation["summary"]["configuration_checks"] > 0,
        "rule smoke has no affected records",
    )
    return {**validation, "artifact": _write_new(output / REPORT, report)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("native", "package", "sources", "output"):
        parser.add_argument("--" + name, required=True, type=Path)
    args = parser.parse_args()
    print(
        json.dumps(
            materialize(args.native, args.package, args.sources, args.output),
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
