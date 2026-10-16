"""Materialize and audit one explicit spatial candidate over an immutable 3.8 base.

No corpus discovery, publication, human labels or training entry points exist here.
"""

from __future__ import annotations

import argparse
import copy
import gzip
import json
import math
import stat
import struct
import tempfile
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path, PurePosixPath

from ai_coach.contracts.spatial_candidate import (
    BRIDGE_ID,
    EXAMPLE_ID,
    FIELDS,
    JOIN_ID,
    JOIN_POLICY,
    LOADER_ID,
    MANIFEST_ID,
    NAV_PHYSICAL_REASON,
    VALIDATOR_ID,
    VERSION,
    compact,
    decode,
    feature_values,
    require,
    scoped_key,
    sha256,
    validate_candidate_example,
    validate_schema,
    validate_spatial_ticks,
)
from ai_coach.contracts.validate_cover_full_demo import validate_cover_full_demo_report
from ai_coach.contracts.validate_exit_route_full_demo import (
    validate_exit_route_full_demo_report,
)
from ai_coach.contracts.validate_exposure_angle_full_demo import (
    validate_exposure_angle_full_demo_report,
)
from ai_coach.datasets.build_examples import _index_partition, process_manifest
from ai_coach.datasets.eligibility import evaluate_task_eligibility
from ai_coach.datasets.streaming import BundleManifest, load_direct_bundle_manifest

ROOT = Path(__file__).resolve().parents[2]
MANIFEST_FILENAME = "spatial_candidate_manifest.json"
JOIN_FILENAME = "spatial_fact_joins.jsonl"
EXAMPLES_FILENAME = "spatial_decision_examples.jsonl"
BRIDGE_FILENAME = "spatial_participant_bridge.json"
MAX_LINE_BYTES = 16 * 1024 * 1024


def safe_path(root: Path, relative: str) -> Path:
    parts = PurePosixPath(relative)
    require(
        not parts.is_absolute()
        and relative == parts.as_posix()
        and "\\" not in relative
        and ":" not in relative
        and all(part not in {".", ".."} for part in parts.parts),
        "unsafe candidate path",
    )
    current = root
    for part in parts.parts:
        current = current / part
        info = current.lstat()
        require(
            not current.is_symlink()
            and not getattr(info, "st_file_attributes", 0)
            & stat.FILE_ATTRIBUTE_REPARSE_POINT,
            "candidate reparse path forbidden",
        )
    require(
        current.resolve().is_relative_to(root.resolve()), "candidate path escapes root"
    )
    return current


def temporary_root(root: Path) -> Path:
    root = root.absolute()
    temp = Path(tempfile.gettempdir()).resolve(strict=True)
    require(
        root != temp and root.is_relative_to(temp),
        "candidate writes require a child of TEMP",
    )
    safe_path(temp, root.relative_to(temp).as_posix())
    return root


def descriptor(
    root: Path,
    path: str,
    schema_id: str,
    record_count: int,
    *,
    compressed: bool = False,
) -> dict:
    physical = safe_path(root, path)
    return {
        "path": path,
        "schema_id": schema_id,
        "descriptor_version": "1.0.0",
        "format": "jsonl" if path.endswith((".jsonl", ".jsonl.gz")) else "json",
        "compression": "gzip" if compressed else "none",
        "record_count": record_count,
        "bytes": physical.stat().st_size,
        "sha256": sha256(physical),
    }


def verify_descriptor(root: Path, entry: dict) -> Path:
    validate_schema(entry, "spatial_candidate_artifact.schema.json")
    path = safe_path(root, entry["path"])
    require(
        path.is_file()
        and path.stat().st_size == entry["bytes"]
        and sha256(path) == entry["sha256"],
        "candidate artifact hash/bytes mismatch",
    )
    return path


def read_rows(path: Path, *, compressed: bool = False):
    opener = gzip.open if compressed else open
    with opener(path, "rb") as stream:
        while raw := stream.readline(MAX_LINE_BYTES + 1):
            require(
                len(raw) <= MAX_LINE_BYTES and raw.endswith(b"\n"),
                "invalid candidate JSONL line",
            )
            yield decode(raw)


def fact_status(field_id: str, row: dict) -> tuple[str, list[str]]:
    if field_id != "FLD-NAV-EXIT-PATH":
        fact = row[
            "geometry_clearance"
            if field_id == "FLD-GEOMETRIC-CLEARANCE-SECTORS"
            else "geometry_occlusion"
        ]
        return (
            "present" if fact["status"] == "observed" else "null",
            fact["reason_codes"],
        )
    if row["sample"]["status"] == "unavailable":
        return "null", row["sample"]["reason_codes"]
    candidates = row["candidates"]
    # Presence describes the graph diagnostic, never physical-route evidence.
    known = [
        item
        for item in candidates
        if item["objective_viability"]["reachability"]["status"] == "derived"
    ]
    reasons = sorted(
        {
            reason
            for item in candidates
            for reason in item["objective_viability"]["reason_codes"]
        }
    )
    return ("present" if known else "null"), reasons


def index_spatial_rows(
    rows,
    *,
    field_id: str,
    source_alias: str,
    map_id: str,
    package: dict,
    aliases: set[str],
) -> dict:
    kind = FIELDS[field_id]
    result = {}
    previous = None
    for line, row in enumerate(rows, 1):
        validate_schema(row, f"observed_{kind}_diagnostic_v2.schema.json")
        validate_spatial_ticks(row, row["t0_tick"])
        key = (row["round_number"], row["t0_tick"], row["participant_alias"])
        require(
            previous is None or key > previous,
            "duplicate or non-deterministic spatial row order",
        )
        previous = key
        require(
            row["source_alias"] == source_alias
            and row["map_id"] == map_id
            and row["participant_alias"] in aliases,
            "spatial source/map/participant mismatch",
        )
        require(
            row["availability_tick"] == row["t0_tick"], "spatial temporal violation"
        )
        provenance = row["provenance"]
        require(
            provenance["package_sha256"] == package["package_sha256"]
            and provenance["geometry_revision"] == package["geometry_revision"]
            and provenance["package_contract_version"] == package["contract_version"]
            and row.get("asset_version", provenance.get("package_asset_version"))
            == package["asset_version"],
            "spatial package/source/hash mismatch",
        )
        result[key] = (line, row)
    require(bool(result), "empty spatial artifact")
    return result


def build_joins(
    decisions: dict, indexes: dict, bridge: dict, checksum: str
) -> list[dict]:
    joins = []
    for decision in decisions.values():
        value = decision.value
        key = scoped_key(checksum, value["actor_player_id"])
        require(key in bridge, "decision actor has no accredited alias bridge")
        alias = bridge[key]
        sample_key = (value["round_number"], value["t0_tick"], alias)
        for field_id in FIELDS:
            match = indexes[field_id].get(sample_key)
            status, reasons = (
                ("missing", ["no_exact_t0_sample"])
                if match is None
                else fact_status(field_id, match[1])
            )
            joins.append(
                {
                    "schema_id": JOIN_ID,
                    "contract_version": "1.0.0",
                    "join_policy": JOIN_POLICY,
                    "decision_key": scoped_key(checksum, value["decision_id"]),
                    "decision_family": value["decision_type"],
                    "source_alias": value["match_id"],
                    "round_number": value["round_number"],
                    "t0_tick": value["t0_tick"],
                    "participant_alias": alias,
                    "field_id": field_id,
                    "status": status,
                    "reason_codes": reasons,
                    "row_line": match[0] if match else None,
                    "sample_tick": match[1]["t0_tick"] if match else None,
                    "availability_tick": match[1]["availability_tick"]
                    if match
                    else None,
                    "training_allowed": False,
                }
            )
    return sorted(joins, key=lambda row: (row["decision_key"], row["field_id"]))


def validate_spatial_replay_sources(
    base: BundleManifest, indexes: dict, bridge: dict, checksum: str
) -> int:
    """Reconcile only self state at the exact source frame; no outcomes as inputs."""
    seen = set()
    required = set().union(*(set(index) for index in indexes.values()))
    for artifact in base.many("replay_round"):
        with gzip.open(artifact.physical_path, "rb") as stream:
            payload = stream.read(128 * 1024 * 1024 + 1)
        require(
            len(payload) <= 128 * 1024 * 1024,
            "replay round exceeds candidate validation bound",
        )
        data = decode(payload)
        require(data["match_id"] == base.bundle_id, "replay source match mismatch")
        round_data = data["round"]
        for frame in round_data["frames"]:
            for player in frame["players"]:
                actor = scoped_key(checksum, "steam:" + player["steam_id"])
                require(actor in bridge, "replay participant missing from bridge")
                key = (round_data["round"], frame["tick"], bridge[actor])
                if key not in required:
                    continue
                require(
                    key not in seen and player["alive"] is True,
                    "ambiguous or non-alive spatial source",
                )
                seen.add(key)
                for field_id, index in indexes.items():
                    if key not in index:
                        continue
                    row = index[key][1]
                    require(
                        row["position_world"]
                        == {axis: player[axis] for axis in ("x", "y", "z")},
                        "spatial position/source mismatch",
                    )
                    if row.get("view_yaw_degrees") is not None:
                        yaw = (
                            struct.unpack("f", struct.pack("f", player["yaw"]))[0] % 360
                        )
                        require(
                            math.isclose(row["view_yaw_degrees"], yaw, abs_tol=1e-6),
                            "spatial yaw/source mismatch",
                        )
                    if field_id == "FLD-GEOMETRIC-OCCLUSION":
                        require(
                            row["posture_id"]
                            == (
                                "crouching"
                                if player.get("is_ducking", False)
                                else "standing"
                            ),
                            "spatial posture/source mismatch",
                        )
                    if field_id == "FLD-NAV-EXIT-PATH":
                        for flag in ("is_ducking", "is_walking", "is_scoped"):
                            actual = row["posture_context"][flag]
                            require(
                                actual is None or actual == player.get(flag, False),
                                "spatial motion/source mismatch",
                            )
    require(seen == required, "spatial source frame missing")
    return len(seen)


@dataclass
class SpatialCandidate:
    manifest: dict
    base: BundleManifest
    indexes: dict
    joins: dict
    validations: dict

    def attach(self, example: dict) -> dict:
        result = copy.deepcopy(example)
        checksum = self.manifest["source_sha256"]
        decision_key = scoped_key(checksum, example["example_id"])
        result.update(
            schema_id=EXAMPLE_ID,
            contract_version="3.0.0",
            candidate_version=VERSION,
            training_allowed=False,
            map_id=self.manifest["map_id"],
            spatial_evidence={},
        )
        result["example_id"] = "decision-" + decision_key
        result["join_keys"]["decision_id"] = result["example_id"]
        result["join_keys"]["observed_state_ref"] = "state-" + scoped_key(
            checksum, example["join_keys"]["observed_state_ref"]
        )
        result["observed"]["spatial_facts"] = {}
        for field_id, kind in FIELDS.items():
            join = self.joins[(decision_key, field_id)]
            result["join_keys"]["actor_player_id"] = join["participant_alias"]
            metadata = {
                "status": join["status"],
                "reason_codes": join["reason_codes"],
                "sample_tick": join["sample_tick"],
                "row_reference": None,
                "provenance": None,
                "package": self.manifest["facts"][field_id]["package"],
                "units": self.manifest["units"],
            }
            values = {}
            if field_id == "FLD-NAV-EXIT-PATH":
                metadata["diagnostic"] = {
                    "scope": "partial_nav_graph",
                    "status": join["status"],
                    "reason_codes": list(join["reason_codes"]),
                    "values": {},
                }
                metadata["status"] = (
                    "missing" if join["status"] == "missing" else "null"
                )
                metadata["reason_codes"] = sorted(
                    set(join["reason_codes"]) | {NAV_PHYSICAL_REASON}
                )
            if join["row_line"] is not None:
                sample_key = (
                    join["round_number"],
                    join["t0_tick"],
                    join["participant_alias"],
                )
                line, row = self.indexes[field_id][sample_key]
                metadata["row_reference"] = {
                    "artifact_type": kind,
                    "schema_id": row["schema_id"],
                    "path": self.manifest["facts"][field_id]["rows"]["path"],
                    "line": line,
                }
                metadata["provenance"] = row["provenance"]
                if field_id == "FLD-NAV-EXIT-PATH":
                    # Preserve numeric values and local IDs outside observed features.
                    diagnostic_values = copy.deepcopy(
                        {
                            key: row[key]
                            for key in ("sample", "candidates", "posture_context")
                        }
                    )
                    for item in diagnostic_values["candidates"]:
                        item.pop("tactical_assessment")
                    metadata["diagnostic"]["values"] = diagnostic_values
                elif join["status"] == "present":
                    keys = (
                        ("geometry_clearance", "view_yaw_degrees")
                        if kind == "exposure_angle"
                        else (
                            ("geometry_occlusion", "posture_id", "view_yaw_degrees")
                            if kind == "cover"
                            else ("sample", "candidates", "posture_context")
                        )
                    )
                    values = feature_values({key: row[key] for key in keys})
            result["spatial_evidence"][field_id] = metadata
            result["observed"]["spatial_facts"][field_id] = {
                "availability_tick": join["availability_tick"]
                if join["availability_tick"] is not None
                else example["t0_tick"],
                "status": "derived"
                if metadata["status"] == "present"
                else "unavailable",
                "values": values,
            }
        validate_candidate_example(result)
        return result

    @staticmethod
    def field_status(example: dict, field_id: str) -> str:
        return example["spatial_evidence"][field_id]["status"]


def _inputs(root: Path, manifest: dict, maps_root: Path, spatial_root: Path) -> tuple:
    checksum, alias, map_id = (
        manifest[key] for key in ("source_sha256", "source_alias", "map_id")
    )
    require(
        alias == "demo-" + checksum[:20], "candidate source checksum/alias mismatch"
    )
    base_path = verify_descriptor(root, manifest["base_manifest"])
    require(
        manifest["base_manifest"]["path"] == f"match_{alias}/canonical/manifest.json",
        "candidate base path mismatch",
    )
    base = load_direct_bundle_manifest(
        base_path.parent.parent,
        expected_match_id=alias,
        expected_demo_checksum=checksum,
    )
    match = decode(base.one("match").physical_path.read_bytes())
    require(match["map_name"] == map_id, "base map mismatch")
    bridge_path = verify_descriptor(root, manifest["participant_bridge"])
    bridge = decode(bridge_path.read_bytes())
    validate_schema(bridge, "spatial_participant_bridge.schema.json")
    require(
        bridge["source_alias"] == alias
        and bridge["source_sha256"] == checksum
        and bridge["map_id"] == map_id,
        "bridge source mismatch",
    )
    players = decode(base.one("participants").physical_path.read_bytes())["players"]
    actor_ids = [player["player_id"] for player in players]
    require(len(actor_ids) == len(set(actor_ids)), "duplicate canonical participants")
    # Canonical IDs are verified locally; only match-scoped hashes leave this loader.
    require(
        all(actor.startswith("steam:") and actor[6:].isdigit() for actor in actor_ids),
        "unrecognized canonical actor ID",
    )
    expected_bridge = [
        {
            "actor_key": scoped_key(checksum, actor),
            "participant_alias": f"participant-{index:03d}",
        }
        for index, actor in enumerate(
            sorted(actor_ids, key=lambda item: int(item[6:])), 1
        )
    ]
    require(
        bridge["participants"] == expected_bridge,
        "participant bridge mismatch or ambiguous aliases",
    )
    aliases = {entry["participant_alias"] for entry in expected_bridge}
    by_actor = {
        entry["actor_key"]: entry["participant_alias"] for entry in expected_bridge
    }
    indexes, validations = {}, {}
    validators = {
        "exposure_angle": validate_exposure_angle_full_demo_report,
        "cover": validate_cover_full_demo_report,
        "exit_route": validate_exit_route_full_demo_report,
    }
    seen_paths = {
        manifest["base_manifest"]["path"],
        manifest["participant_bridge"]["path"],
    }
    builds = set()
    for field_id, kind in FIELDS.items():
        fact = manifest["facts"][field_id]
        for role in ("report", "rows"):
            require(
                fact[role]["path"] not in seen_paths,
                "ambiguous candidate artifact paths",
            )
            seen_paths.add(fact[role]["path"])
            verify_descriptor(root, fact[role])
        report_path = safe_path(root, fact["report"]["path"])
        report = decode(report_path.read_bytes())
        require(
            fact["package"] == report["package_resolution"],
            "descriptor package mismatch",
        )
        require(fact["sampling"] == report["sampling"], "descriptor sampling mismatch")
        require(fact["coverage"] == report["coverage"], "descriptor coverage mismatch")
        require(
            fact["rows"]["path"] == report["rows_artifact"]["path"]
            and fact["rows"]["sha256"] == report["rows_artifact"]["sha256"]
            and fact["rows"]["schema_id"] == report["candidate_schema_id"]
            and fact["rows"]["record_count"]
            == report["rows_artifact"].get(
                "record_count", report["rows_artifact"].get("logical_records")
            ),
            "report/row descriptor mismatch",
        )
        builds.add(report["build_identifier"])
        validations[field_id] = validators[kind](
            report_path,
            registry_root=spatial_root
            / ("exposure" if kind == "exposure_angle" else kind),
            maps_root=maps_root,
            expected_alias=alias,
            expected_sha256=checksum,
            expected_map_id=map_id,
        )
        indexes[field_id] = index_spatial_rows(
            read_rows(safe_path(root, fact["rows"]["path"]), compressed=True),
            field_id=field_id,
            source_alias=alias,
            map_id=map_id,
            package=fact["package"],
            aliases=aliases,
        )
        require(
            len(indexes[field_id]) == fact["rows"]["record_count"],
            "spatial record count mismatch",
        )
    require(builds == {manifest["build_identifier"]}, "producer build mismatch")
    validations["source_reconciliation"] = {
        "frames_with_participant": validate_spatial_replay_sources(
            base, indexes, by_actor, checksum
        ),
        "position_yaw_posture_mismatches": 0,
        "policy": "exact replay self frame; outcomes ignored",
    }
    return base, indexes, by_actor, validations


def load_spatial_candidate(
    manifest_path: Path,
    *,
    maps_root: Path,
    spatial_root: Path = ROOT / "ai_coach/spatial",
) -> SpatialCandidate:
    root = manifest_path.absolute().parent
    safe_path(root, manifest_path.name)
    manifest = decode(manifest_path.read_bytes())
    validate_schema(manifest, "spatial_candidate_manifest_v2.schema.json")
    base, indexes, bridge, validations = _inputs(
        root, manifest, maps_root, spatial_root
    )
    joins_path = verify_descriptor(root, manifest["joins"])
    verify_descriptor(root, manifest["decision_examples"])
    joins = list(read_rows(joins_path))
    for join in joins:
        validate_schema(join, "spatial_fact_join.schema.json")
    expected = build_joins(
        _index_partition(base, "decisions"), indexes, bridge, manifest["source_sha256"]
    )
    require(
        joins == expected and len(joins) == manifest["joins"]["record_count"],
        "joins missing, duplicate, unordered, ambiguous or not causal",
    )
    return SpatialCandidate(
        manifest,
        base,
        indexes,
        {(row["decision_key"], row["field_id"]): row for row in joins},
        validations,
    )


def materialize_spatial_candidate(
    root: Path,
    *,
    source_alias: str,
    source_sha256: str,
    map_id: str,
    maps_root: Path,
    spatial_root: Path = ROOT / "ai_coach/spatial",
) -> Path:
    root = temporary_root(root)
    for name in (MANIFEST_FILENAME, JOIN_FILENAME, EXAMPLES_FILENAME):
        require(
            not (root / name).exists(),
            "candidate output already exists; overwrites forbidden",
        )
    # The Go producer has already enforced an empty TEMP destination before parsing.
    manifest = {
        "schema_id": MANIFEST_ID,
        "export_format_version": VERSION,
        "contract_version": "2.0.0",
        "loader_schema_id": LOADER_ID,
        "validator_schema_id": VALIDATOR_ID,
        "join_policy": JOIN_POLICY,
        "source_alias": source_alias,
        "source_sha256": source_sha256,
        "map_id": map_id,
        "training_allowed": False,
        "product_supported": False,
        "units": {
            "position": "world_units",
            "distance": "world_units",
            "angle": "degrees",
            "time": "milliseconds",
        },
        "base_manifest": descriptor(
            root,
            f"match_{source_alias}/canonical/manifest.json",
            "stratai.canonical_manifest@3",
            1,
        ),
        "participant_bridge": descriptor(root, BRIDGE_FILENAME, BRIDGE_ID, 1),
        "facts": {},
    }
    for field_id, kind in FIELDS.items():
        report_name = f"observed_{kind}_full_demo.json"
        report = decode(safe_path(root, report_name).read_bytes())
        manifest["build_identifier"] = report["build_identifier"]
        artifact = report["rows_artifact"]
        manifest["facts"][field_id] = {
            "report": descriptor(root, report_name, report["schema_id"], 1),
            "rows": descriptor(
                root,
                artifact["path"],
                report["candidate_schema_id"],
                artifact.get("record_count", artifact.get("logical_records")),
                compressed=True,
            ),
            "package": report["package_resolution"],
            "sampling": report["sampling"],
            "coverage": report["coverage"],
        }
    base, indexes, bridge, validations = _inputs(
        root, manifest, maps_root, spatial_root
    )
    joins = build_joins(
        _index_partition(base, "decisions"), indexes, bridge, source_sha256
    )
    for join in joins:
        validate_schema(join, "spatial_fact_join.schema.json")
    with (root / JOIN_FILENAME).open("xb") as stream:
        for join in joins:
            stream.write(compact(join))
    manifest["joins"] = descriptor(root, JOIN_FILENAME, JOIN_ID, len(joins))
    candidate = SpatialCandidate(
        manifest,
        base,
        indexes,
        {(row["decision_key"], row["field_id"]): row for row in joins},
        validations,
    )
    count = 0
    with (root / EXAMPLES_FILENAME).open("xb") as stream:

        def emit(example):
            nonlocal count
            stream.write(compact(example))
            count += 1

        audit_spatial_candidate(candidate, example_sink=emit, verify_examples=False)
    manifest["decision_examples"] = descriptor(
        root, EXAMPLES_FILENAME, EXAMPLE_ID, count
    )
    validate_schema(manifest, "spatial_candidate_manifest_v2.schema.json")
    with (root / MANIFEST_FILENAME).open("xb") as stream:
        stream.write(compact(manifest))
    return root / MANIFEST_FILENAME


def audit_spatial_candidate(
    candidate: SpatialCandidate, *, example_sink=None, verify_examples=True
) -> dict:
    catalog = decode((ROOT / "ai_coach/contracts/capability_catalog.json").read_bytes())
    fields = sorted(
        {
            field
            for cap in catalog["capabilities"]
            for field in cap["task_eligibility"]["required_fields"]
        }
        | set(FIELDS)
    )
    stored = None
    example_count = 0
    if verify_examples:
        path = verify_descriptor(
            candidate.base.bundle_dir.parent, candidate.manifest["decision_examples"]
        )
        stored = read_rows(path)

    def consume(example):
        nonlocal example_count
        example_count += 1
        if stored is not None:
            require(
                next(stored, None) == example, "materialized DecisionExample mismatch"
            )
        if example_sink is not None:
            example_sink(example)

    try:
        summary = process_manifest(
            candidate.base,
            catalog_required_fields=fields,
            spatial_candidate=candidate,
            example_sink=consume,
        )
        if stored is not None:
            require(
                next(stored, None) is None
                and example_count
                == candidate.manifest["decision_examples"]["record_count"],
                "extra or missing materialized DecisionExample",
            )
    finally:
        if stored is not None:
            stored.close()
    coverage = {
        field: {family: metric.as_dict() for family, metric in families.items()}
        for field, families in summary.field_coverage.items()
    }
    spatial = defaultdict(
        lambda: defaultdict(
            lambda: {
                "present": 0,
                "missing": 0,
                "null": 0,
                "unavailable": 0,
                "physical_reason_counts": Counter(),
                "join_reason_counts": Counter(),
                "temporal_violations": 0,
                "joins_failed": 0,
            }
        )
    )
    for join in candidate.joins.values():
        counts = spatial[join["decision_family"]][join["field_id"]]
        counts[join["status"]] += 1
        if join["status"] == "missing":
            counts["joins_failed"] += 1
            counts["join_reason_counts"].update(join["reason_codes"])
        else:
            counts["unavailable"] += join["status"] == "null"
            counts["physical_reason_counts"].update(join["reason_codes"])
        if join["field_id"] == "FLD-NAV-EXIT-PATH":
            counts["scope"] = "partial_nav_graph"
            counts["physical_evidence_status"] = "unavailable"
    for family in spatial.values():
        if "FLD-NAV-EXIT-PATH" in family:
            counts = family["FLD-NAV-EXIT-PATH"]
            counts["diagnostic_reason_counts"] = counts.pop("physical_reason_counts")
    eligibility = evaluate_task_eligibility(
        catalog,
        family_counts=summary.family_counts,
        field_coverage=coverage,
        bundles_processed=1,
        temporal_violations=0,
        join_failures=0,
        bundle_validation_reference=f"{MANIFEST_ID}; {VALIDATOR_ID}; {JOIN_POLICY}",
    )
    return {
        "schema_id": "stratai.spatial_candidate_audit@1",
        "validator_schema_id": VALIDATOR_ID,
        "candidate_version": VERSION,
        "training_allowed": False,
        "product_supported": False,
        "source_alias": candidate.manifest["source_alias"],
        "status": "passed",
        "family_counts": dict(summary.family_counts),
        "base_joins_succeeded": summary.joins_succeeded,
        "spatial_by_family": dict(spatial),
        "field_coverage": coverage,
        "task_eligibility": eligibility,
        "sidecar_validations": candidate.validations,
        "temporal_violations": 0,
        "eligibility_counts": dict(Counter(row["status"] for row in eligibility)),
        "interpretation": "Partial exact-T0 coverage only. Missing joins are abstentions; no training or support authorization.",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("materialize", "validate"))
    parser.add_argument("root", type=Path)
    parser.add_argument("--alias", required=True)
    parser.add_argument("--checksum", required=True)
    parser.add_argument("--map-id", required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--spatial-root", type=Path, default=ROOT / "ai_coach/spatial")
    args = parser.parse_args()
    manifest_path = args.root / MANIFEST_FILENAME
    if args.mode == "materialize":
        manifest_path = materialize_spatial_candidate(
            args.root,
            source_alias=args.alias,
            source_sha256=args.checksum,
            map_id=args.map_id,
            maps_root=args.maps_root,
            spatial_root=args.spatial_root,
        )
    candidate = load_spatial_candidate(
        manifest_path, maps_root=args.maps_root, spatial_root=args.spatial_root
    )
    require(
        (args.alias, args.checksum, args.map_id)
        == tuple(
            candidate.manifest[key]
            for key in ("source_alias", "source_sha256", "map_id")
        ),
        "requested candidate identity mismatch",
    )
    audit = audit_spatial_candidate(candidate)
    print(json.dumps(audit, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
