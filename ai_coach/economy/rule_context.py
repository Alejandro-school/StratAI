"""Native economic configuration with callback provenance and causal boundaries.

No default is imputed and a server variable is not player knowledge. Reports are
development evidence; native version metadata still does not bind shop assets.
"""

from __future__ import annotations

import argparse
import json
from collections import Counter
from decimal import Decimal, InvalidOperation
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts import validate_economy_candidate as base
from ai_coach.economy import native_sources, rule_assessment, rule_assets
from ai_coach.economy.candidate_io import (
    CONTRACTS,
    _write_new,
    sha,
    validate_output_directory,
)

require = base._require
PROBE = (
    Path(__file__).resolve().parents[2]
    / "backend/go-service/models/economy_rule_context_probe.json"
)
MANIFEST = "economy_rule_context_manifest.json"
DATA = "economy_rule_context.json"
REPORT = "economy_rule_context_assessment.json"
SCHEMAS = {
    "economy_rule_context.schema.json",
    "economy_rule_context_manifest.schema.json",
    "economy_rule_context_assessment.schema.json",
}
GATES = {
    "comparison_allowed": False,
    "training_allowed": False,
    "product_supported": False,
}
IDENTITY = ("source_alias", "source_sha256", "build_identifier")
EMPTY = {"value": None, "source_sequence": None, "source_tick": None}


def probe() -> tuple[dict, str]:
    raw = PROBE.read_bytes()
    value = json.loads(raw)
    require(
        set(value) == {"version", "convars", "rules"} and value["version"] == "1",
        "unknown rule context probe",
    )
    require(
        value["convars"] == sorted(set(value["convars"]))
        and len(value["convars"]) == 56,
        "rule probe names differ",
    )
    require(
        len(value["rules"]) == 4
        and all(type(x) is bool for x in value["rules"].values()),
        "rule property probes differ",
    )
    return value, sha(raw)


def schema_validator(filename: str) -> Draft202012Validator:
    require(filename in SCHEMAS, "unknown rule context schema")
    schema = json.loads((CONTRACTS / filename).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_schema(value: dict, filename: str) -> None:
    error = next(schema_validator(filename).iter_errors(value), None)
    require(
        error is None,
        f"invalid {filename}: {list(error.absolute_path) if error else ''}",
    )


def validate_data(data: dict, source: dict) -> dict:
    check_schema(data, "economy_rule_context.schema.json")
    spec, spec_hash = probe()
    require(data["probe_sha256"] == spec_hash, "rule probe hash differs")
    require(
        all(data[k] == source[k] for k in IDENTITY),
        "rule context source identity differs",
    )
    rounds = {r["instance"]: r for r in source["rounds"]}
    observations = data["observations"]
    for rows in observations.values():
        require(
            [r["sequence"] for r in rows] == sorted(r["sequence"] for r in rows),
            "native source channel order differs",
        )
    stream = sorted(
        [
            (row["sequence"], kind, row)
            for kind, rows in observations.items()
            for row in rows
        ]
    )
    latest, seen = {}, set()
    for expected, (sequence, kind, row) in enumerate(stream, 1):
        require(
            sequence == expected, "native callback sequence duplicated or incomplete"
        )
        if kind == "messages":
            for entry in row["entries"]:
                require(entry["name"] in spec["convars"], "unprobed convar retained")
                latest[entry["name"]] = {
                    "value": entry["value"],
                    "source_sequence": sequence,
                    "source_tick": row["tick"],
                }
        elif kind == "boundaries":
            key = (row["instance"], row["phase"])
            require(
                key not in seen and row["instance"] in rounds,
                "rule boundary identity differs",
            )
            seen.add(key)
            round_ = rounds[row["instance"]]
            expected_tick = (
                round_["start_tick"]
                if row["phase"] == "round_start_probe"
                else round_["freeze_end_tick"]
            )
            require(row["tick"] == expected_tick, "rule boundary tick differs from v2")
            require(
                row["convars"]
                == {name: latest.get(name, EMPTY) for name in spec["convars"]},
                "rule context is not the observed message prefix",
            )
            require(
                set(row["rules"]) == set(spec["rules"]),
                "native rule property coverage differs",
            )
            for name, boolean in spec["rules"].items():
                fact = row["rules"][name]
                native_sources.validate_fact(fact, row["tick"], name)
                if fact["status"] == "observed":
                    require(
                        type(fact["value"]) is (bool if boolean else int),
                        "native rule property type differs",
                    )
                    require(
                        boolean or 0 <= fact["value"] <= 2147483647,
                        "native rule counter out of range",
                    )
        else:
            for name in ("manifest", "configuration_data"):
                field = row[name]
                if field["present"]:
                    require(
                        field["bytes"] is not None and field["sha256"] is not None,
                        "present opaque field lacks fingerprint",
                    )
                    if field["bytes"] == 0:
                        require(
                            field["sha256"] == sha(b""),
                            "empty opaque field hash differs",
                        )
                else:
                    require(
                        field["bytes"] is field["sha256"] is None,
                        "absent opaque field became observed",
                    )
            require(
                row["configuration_present"]
                or not row["configuration_data"]["present"],
                "data without session configuration",
            )
    expected_boundaries = {
        (r["instance"], "round_start_probe") for r in rounds.values()
    }
    expected_boundaries |= {
        (r["instance"], "freeze_end_probe")
        for r in rounds.values()
        if r["freeze_end_tick"] is not None
    }
    require(seen == expected_boundaries, "rule boundary coverage incomplete")
    return {key: len(value) for key, value in observations.items()}


def load_candidate(
    path: Path, *, require_observed: bool = False
) -> tuple[dict, dict, dict, dict]:
    path = base._safe_file(path.parent.absolute(), path.name)
    require(path.name == MANIFEST, "unexpected rule context manifest")
    manifest, manifest_raw = base._read(path)
    check_schema(manifest, "economy_rule_context_manifest.schema.json")
    files = {}
    for key in ("source_native", "artifact"):
        reference = manifest[key]
        file = base._safe_file(path.parent, reference["path"])
        raw = file.read_bytes()
        require(
            len(raw) == reference["bytes"] and sha(raw) == reference["sha256"],
            "rule context file integrity differs",
        )
        files[key] = (file, raw)
    native, source, native_bindings = native_sources.load_candidate(
        files["source_native"][0], require_observed=require_observed
    )
    require(
        native_bindings["source_manifest_sha256"]
        == manifest["source_native"]["sha256"],
        "native source changed during load",
    )
    data = json.loads(files["artifact"][1])
    for key in (*IDENTITY, *GATES, "contract_version", "probe_sha256"):
        require(
            data.get(key) == manifest[key], "rule context manifest metadata differs"
        )
    require(
        data["source_native_manifest_sha256"] == manifest["source_native"]["sha256"],
        "rule native binding differs",
    )
    counts = validate_data(data, source)
    require(counts == manifest["summary"], "rule context summary differs")
    require(
        path.read_bytes() == manifest_raw, "rule context manifest changed during load"
    )
    if require_observed:
        require(
            all(counts.values())
            and any(r["entries"] for r in data["observations"]["messages"]),
            "native rule context smoke is empty",
        )
    bindings = {
        "context_manifest_sha256": sha(manifest_raw),
        "context_data_sha256": sha(files["artifact"][1]),
        **native_bindings,
    }
    return data, native, source, bindings


def numeric(value: str | None) -> Decimal | None:
    if value is None:
        return None
    if value in ("true", "false"):
        return Decimal(int(value == "true"))
    try:
        result = Decimal(value)
        return result if result.is_finite() else None
    except InvalidOperation:
        return None


def configuration_row(
    boundary: dict, name: str, fact: dict, references: list[str]
) -> dict:
    if fact["source_sequence"] is None:
        status = "not_seen_in_probed_message_prefix"
    elif fact["source_tick"] > boundary["tick"]:
        status = "source_tick_after_boundary"
    elif fact["value"] is None:
        status = "native_value_absent"
    elif numeric(fact["value"]) is None:
        status = "native_value_unsupported"
    else:
        status = "observed_native_parameter"
    comparison = "unavailable"
    if status == "observed_native_parameter" and references:
        comparison = (
            "conflicting_references"
            if len({numeric(x) for x in references}) > 1
            else "equal_reference"
            if numeric(fact["value"]) == numeric(references[0])
            else "different_reference"
        )
    return {k: boundary[k] for k in ("instance", "phase", "tick", "sequence")} | {
        "name": name,
        **fact,
        "status": status,
        "reference_values": references,
        "comparison": comparison,
    }


def armor_groups(native: dict) -> list[dict]:
    groups = Counter()
    for row in native["observations"]["snapshots"]:
        if not row["connected"] or row["sellback"]["status"] != "observed":
            continue
        for entry in row["sellback"]["entries"]:
            if entry["m_unDefIdx"]["value"] in (50, 51):
                groups[
                    tuple(
                        entry[key]["value"]
                        for key in (
                            "m_unDefIdx",
                            "m_nCost",
                            "m_nPrevArmor",
                            "m_bPrevHelmet",
                        )
                    )
                ] += 1
    return [
        dict(
            zip(
                (
                    "item_definition_index",
                    "recorded_cost",
                    "previous_armor_native",
                    "previous_helmet_native",
                    "snapshot_occurrences",
                ),
                (*key, count),
                strict=True,
            )
        )
        for key, count in sorted(groups.items())
    ]


def summary(report: dict) -> dict:
    return {
        "configuration_checks": len(report["configuration_checks"]),
        "native_property_checks": len(report["native_property_checks"]),
        "observed_parameters": sum(
            r["status"] == "observed_native_parameter"
            for r in report["configuration_checks"]
        ),
        "observed_properties": sum(
            r["fact"]["status"] == "observed" for r in report["native_property_checks"]
        ),
        "approved_rule_packages": 0,
    }


def build_report(
    data: dict, native: dict, package: dict, package_hash: str, bindings: dict
) -> dict:
    references = {}
    for release in package["releases"]:
        for default in release["defaults"]:
            references.setdefault(default["name"], set()).add(default["value"])
    report = {
        "schema_id": "stratai.economy_rule_context_assessment@1",
        "contract_version": "1.0.0-candidate.1",
        **{k: data[k] for k in IDENTITY},
        **GATES,
        "source_bindings": bindings,
        "package_sha256": package_hash,
        "identity": rule_assessment.identity_assessment(native, package),
        "configuration_checks": [
            configuration_row(b, name, fact, sorted(references.get(name, [])))
            for b in data["observations"]["boundaries"]
            for name, fact in sorted(b["convars"].items())
        ],
        "native_property_checks": [
            {
                "instance": b["instance"],
                "phase": b["phase"],
                "tick": b["tick"],
                "name": name,
                "fact": fact,
            }
            for b in data["observations"]["boundaries"]
            for name, fact in sorted(b["rules"].items())
        ],
        "sessions": data["observations"]["sessions"],
        "armor_snapshot_groups": armor_groups(native),
    }
    report["summary"] = summary(report)
    return report


def validate_report(
    report: dict,
    data: dict,
    native: dict,
    source: dict,
    package: dict,
    package_hash: str,
    bindings: dict,
) -> dict:
    """Recompute joins from raw messages, not the producer's context snapshots."""
    validate_data(data, source)
    check_schema(report, "economy_rule_context_assessment.schema.json")
    require(
        report["source_bindings"] == bindings
        and report["package_sha256"] == package_hash,
        "rule assessment bindings differ",
    )
    require(
        all(report[k] == data[k] for k in IDENTITY), "rule assessment identity differs"
    )
    require(
        report["identity"] == rule_assessment.identity_assessment(native, package),
        "rule asset identity was inferred",
    )
    spec, _ = probe()
    expected, properties = [], []
    for boundary in data["observations"]["boundaries"]:
        for name in spec["convars"]:
            assignments = [
                (m, e)
                for m in data["observations"]["messages"]
                if m["sequence"] < boundary["sequence"]
                for e in m["entries"]
                if e["name"] == name
            ]
            fact = EMPTY.copy()
            if assignments:
                message, entry = assignments[-1]
                fact = {
                    "value": entry["value"],
                    "source_sequence": message["sequence"],
                    "source_tick": message["tick"],
                }
            values = sorted(
                {
                    d["value"]
                    for r in package["releases"]
                    for d in r["defaults"]
                    if d["name"] == name
                }
            )
            expected.append(configuration_row(boundary, name, fact, values))
        for name in sorted(spec["rules"]):
            properties.append(
                {
                    "instance": boundary["instance"],
                    "phase": boundary["phase"],
                    "tick": boundary["tick"],
                    "name": name,
                    "fact": boundary["rules"][name],
                }
            )
    require(
        report["configuration_checks"] == expected,
        "rule assessment message prefix differs",
    )
    require(
        report["native_property_checks"] == properties,
        "native property assessment differs",
    )
    require(
        report["sessions"] == data["observations"]["sessions"],
        "session presence differs",
    )
    # Independent multiset recount, retaining raw signed armor metadata.
    counts = Counter()
    for row in native["observations"]["snapshots"]:
        if row["connected"] and row["sellback"]["status"] == "observed":
            for entry in row["sellback"]["entries"]:
                item, cost, armor, helmet = [
                    entry[k]["value"]
                    for k in ("m_unDefIdx", "m_nCost", "m_nPrevArmor", "m_bPrevHelmet")
                ]
                if item in (50, 51):
                    counts[(item, cost, armor, helmet)] += 1
    observed = {
        (
            r["item_definition_index"],
            r["recorded_cost"],
            r["previous_armor_native"],
            r["previous_helmet_native"],
        ): r["snapshot_occurrences"]
        for r in report["armor_snapshot_groups"]
    }
    require(
        len(observed) == len(report["armor_snapshot_groups"]) and observed == counts,
        "armor native grouping differs",
    )
    require(report["summary"] == summary(report), "rule assessment counts differ")
    return {"ok": True, "summary": report["summary"]}


def materialize(source: Path, package: Path, sources: Path, output: Path) -> dict:
    output = validate_output_directory(output)
    data, native, original, bindings = load_candidate(source, require_observed=True)
    assets, package_hash = rule_assets.load_package(package, sources)
    report = build_report(data, native, assets, package_hash, bindings)
    result = validate_report(
        report, data, native, original, assets, package_hash, bindings
    )
    _write_new(output / REPORT, report)
    return result


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("source", "package", "sources", "output"):
        parser.add_argument("--" + name, required=True, type=Path)
    args = parser.parse_args()
    print(json.dumps(materialize(args.source, args.package, args.sources, args.output)))


if __name__ == "__main__":
    main()
