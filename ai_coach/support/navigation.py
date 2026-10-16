"""Optional NAV diagnostics at unchanged support anchors; no physical claim.

The installed asset has no exact demo binding. Even a connected candidate
graph cannot establish a physically traversable route (nor absence of one).
"""

import argparse
import json
import re
import struct
import subprocess
from collections import Counter, defaultdict
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import physical as p
from ai_coach.support.navigation_verify import GraphVerifier

DATA = "support_navigation_observations.json"
REPORT = "support_navigation_context.json"
VERSION = "1.0.0-candidate.1"
LIMITS = [
    "exact_demo_asset_binding_unavailable",
    "endpoint_physical_association_unaccredited",
    "area_attributes_hull_and_mobile_transforms_not_modelled",
    "ladder_links_not_modelled",
    "target_edge_geometry_not_retained",
    "centre_polyline_not_collision_checked",
]
KEYS = (
    "anchor_id",
    "actor_alias",
    "ally_alias",
    "actor_life",
    "ally_life",
    "side",
    "instance",
    "round_number",
    "source_sequence",
    "epoch",
    "t0_tick",
    "t0_frame",
)


def encode(data):
    return (
        json.dumps(data, sort_keys=True, separators=(",", ":"), allow_nan=False) + "\n"
    ).encode()


def queries(anchors):
    return [
        {
            "origin": {
                k.upper(): row["observed"]["history"][-1]["self_position"][k]
                for k in "xyz"
            },
            "target": {
                k.upper(): row["observed"]["history"][-1]["ally_position"][k]
                for k in "xyz"
            },
        }
        for row in anchors
    ]


def physical_absence(tick, frame):
    return {
        "status": "insufficient_information",
        "reachable": None,
        "path_distance_world_units": None,
        "path": None,
        "availability_tick": tick,
        "availability_frame": frame,
        "reasons": LIMITS.copy(),
    }


def initial_features(support, anchor_id, actor, tick, frame):
    # No candidate asset, arbitrary graph, identity, or later outcome can enter
    # the initial factual API. A future accredited contract must be versioned.
    a.initial_features(support, anchor_id, actor, tick, frame)
    return physical_absence(tick, frame)


def validate_data(data, support, selected, manifest):
    a.schema_validator(DATA).validate(data)
    a.require(
        all(
            data[k] == support[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "NAV source identity mismatch",
    )
    a.require(data["map_name"] == manifest["lineage"]["map_name"], "NAV map mismatch")
    reference = data["map_name"] + "/" + data["map_name"] + ".nav"
    a.require(data["asset"]["reference"] == reference, "NAV asset reference mismatch")
    a.require(
        data["queries"] == queries(selected),
        "NAV queries differ from exact causal anchors",
    )
    a.require(
        data["anchor_keys"] == [{k: row[k] for k in KEYS} for row in selected],
        "NAV anchor identity/clock mismatch",
    )
    graph = data["graph"]
    asset = data["asset"]
    supported = (asset["magic"], asset["format_version"], asset["subversion"]) == (
        0xFEEDFACE,
        35,
        1,
    )
    if asset["status"] == "asset_absent":
        a.require(
            all(
                asset[k] is None
                for k in ("sha256", "bytes", "magic", "format_version", "subversion")
            ),
            "NAV absent asset has metadata",
        )
    else:
        a.require(
            asset["sha256"] is not None and asset["bytes"] is not None,
            "NAV asset metadata missing",
        )
        a.require(
            supported == (asset["status"] == "candidate_loaded"),
            "NAV format status mismatch",
        )
    if graph is None:
        a.require(
            data["results"] == [] and data["asset"]["status"] != "candidate_loaded",
            "absent NAV has results",
        )
        return []
    a.require(
        data["asset"]["status"] == "candidate_loaded",
        "NAV graph without candidate asset",
    )
    a.require(
        data["asset"]["format_version"] == 35 and data["asset"]["subversion"] == 1,
        "unsupported NAV format",
    )
    observed = manifest["lineage"]["input_hashes"].get("nav_mesh", {})
    a.require(
        observed.get("status") == "observed"
        and observed.get("value") == data["asset"]["sha256"]
        and observed.get("source") == "map_manager:" + reference,
        "NAV parser asset provenance mismatch",
    )
    a.require(
        graph["AssetSHA256"] == data["asset"]["sha256"]
        and graph["AssetReference"] == reference
        and graph["AvailabilityTick"] == 0,
        "NAV graph asset provenance mismatch",
    )
    a.require(len(data["results"]) == len(selected), "missing exact NAV sample")
    verifier = GraphVerifier(graph)
    return [verifier.verify(q, row) for q, row in zip(data["queries"], data["results"])]


def build_report(data, support, anchor_report, physical_report, references, manifest):
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        selected
        == [
            {k: v for k, v in row.items() if k != "later_observation"}
            for row in anchor_report["anchors"]
        ],
        "NAV v1 selection mismatch",
    )
    associations = validate_data(data, support, selected, manifest)
    physical_rows = physical_report["rows"]
    a.require(len(physical_rows) == len(selected), "NAV physical rows mismatch")
    rows, matrix = [], defaultdict(Counter)
    for i, anchor in enumerate(selected):
        prev = physical_rows[i]
        a.require(
            all(
                prev.get(k) == anchor[k]
                for k in KEYS
                if k not in ("actor_life", "ally_life", "side")
            ),
            "NAV physical exact join mismatch",
        )
        if data["graph"] is None:
            diagnostic = {
                "status": "unavailable",
                "reason": data["asset"]["status"],
                "measurement": None,
                "points": [],
                "association": None,
            }
        else:
            trace = data["results"][i]
            measurement = trace["Measurement"]
            status = (
                "candidate_connected"
                if measurement["Reachable"] is True
                else (
                    "candidate_no_directed_path"
                    if measurement["Reachable"] is False
                    else "candidate_outside_coverage"
                )
            )
            diagnostic = {
                "status": status,
                "reason": measurement["UnavailableReason"] or None,
                "measurement": measurement,
                "points": trace["Points"],
                "association": associations[i],
            }
        row = {
            **{k: anchor[k] for k in KEYS},
            "endpoints": data["queries"][i],
            "competitive_round_status": prev["competitive_round_status"],
            "initial": physical_absence(anchor["t0_tick"], anchor["t0_frame"]),
            "candidate_diagnostic": diagnostic,
            "decision_eligible": False,
        }
        rows.append(row)
        c = matrix[anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]]
        c["anchors"] += 1
        c[diagnostic["status"]] += 1
        c["canonical_interval_matched"] += (
            prev["competitive_round_status"] == "canonical_interval_matched"
        )
        c["insufficient_information"] += 1
    categories = (
        "anchors",
        "canonical_interval_matched",
        "insufficient_information",
        "candidate_connected",
        "candidate_no_directed_path",
        "candidate_outside_coverage",
        "unavailable",
    )
    matrix_rows = [
        {
            "instance": i,
            "actor_alias": actor,
            "ally_alias": ally,
            "counts": {k: c[k] for k in categories},
        }
        for (i, actor, ally), c in sorted(matrix.items())
    ]
    result = {
        "schema_id": "stratai.support_navigation_context@1",
        "contract_version": VERSION,
        **{
            k: data[k]
            for k in ("source_alias", "source_sha256", "build_identifier", "map_name")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
        "rows": rows,
        "matrix": matrix_rows,
        "summary": {
            **{k: sum(c[k] for c in matrix.values()) for k in categories},
            "matrix_rows": len(matrix_rows),
            "physical_connections_observed": 0,
            "physical_absences_accredited": 0,
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(result)
    return result


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)  # Before reading sources or running tools.
    # Preserve v1/3E.2 byte serialization and validations, in the same output.
    p.materialize(source, output, require_observed=True)
    anchor_report = json.loads((output / a.REPORT).read_bytes())
    physical_report = json.loads((output / p.REPORT).read_bytes())
    physical_ref = p.file_reference(source)
    support_ref = p.file_reference(source.parent / a.DATA)
    a.require(
        physical_report["references"]["physical_source"] == physical_ref
        and physical_report["references"]["support_source"] == support_ref,
        "physical sources changed after materialization",
    )
    support = json.loads((source.parent / a.DATA).read_bytes())
    manifest_path = source.parent / support["base_manifest"]["path"]
    manifest = json.loads(manifest_path.read_bytes())
    selected = [
        {k: v for k, v in row.items() if k != "later_observation"}
        for row in anchor_report["anchors"]
    ]
    map_name = manifest["lineage"]["map_name"]
    a.require(re.fullmatch(r"de_[a-z0-9_]+", map_name), "invalid NAV map name")
    reference = map_name + "/" + map_name + ".nav"
    nav = maps_root / reference
    raw = nav.read_bytes() if nav.is_file() else None
    asset = {
        "reference": reference,
        "sha256": a.digest(raw) if raw is not None else None,
        "bytes": len(raw) if raw is not None else None,
        "magic": None,
        "format_version": None,
        "subversion": None,
        "status": "asset_absent",
        "demo_binding": "not_accredited",
    }
    graph, results = None, []
    query_rows = queries(selected)
    if raw is not None:
        asset["status"] = "unsupported_format"
        if len(raw) >= 16:
            magic, version, subversion, _ = struct.unpack("<4I", raw[:16])
            asset.update(magic=magic, format_version=version, subversion=subversion)
            if magic == 0xFEEDFACE and version == 35 and subversion == 1:
                observed = manifest["lineage"]["input_hashes"].get("nav_mesh", {})
                a.require(
                    observed.get("status") == "observed"
                    and observed.get("value") == asset["sha256"]
                    and observed.get("source") == "map_manager:" + reference,
                    "NAV asset differs from parser input",
                )
                request = {
                    "maps_root": str(maps_root.resolve()),
                    "map_name": map_name,
                    "asset_sha256": asset["sha256"],
                    "queries": query_rows,
                }
                process = subprocess.run(
                    [str(evaluator.resolve())],
                    input=encode(request),
                    capture_output=True,
                    check=True,
                )
                reply = json.loads(process.stdout)
                a.require(
                    set(reply) == {"graph", "results"},
                    "unexpected NAV evaluator output",
                )
                graph, results = reply["graph"], reply["results"]
                a.require(nav.read_bytes() == raw, "NAV asset changed")
                asset["status"] = "candidate_loaded"
    refs = {
        "physical_source": physical_ref,
        "support_source": support_ref,
        "anchors": p.file_reference(output / a.REPORT),
        "physical_context": p.file_reference(output / p.REPORT),
        "base_manifest": p.file_reference(manifest_path),
        "evaluator": p.file_reference(evaluator),
    }
    data = {
        "schema_id": "stratai.support_navigation_observations@1",
        "contract_version": VERSION,
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "map_name": map_name,
        "references": refs,
        "asset": asset,
        "limitations": LIMITS.copy(),
        "algorithm": "directed_dijkstra_stable_ties@1",
        "units": "world_units",
        "association_method": "containing_nav_polygon_xy_then_nearest_center_3d@2",
        "endpoint_tolerance": 96,
        "endpoint_vertical_tolerance": 128,
        "anchor_keys": [{k: row[k] for k in KEYS} for row in selected],
        "queries": query_rows,
        "graph": graph,
        "results": results,
    }
    data_raw = encode(data)
    report_refs = {
        **refs,
        "navigation_source": {
            "path": DATA,
            "bytes": len(data_raw),
            "sha256": a.digest(data_raw),
        },
    }
    report = build_report(
        data, support, anchor_report, physical_report, report_refs, manifest
    )
    for name, payload in ((DATA, data_raw), (REPORT, encode(report))):
        with (output / name).open("xb") as f:
            f.write(payload)
    return report["summary"]


def validate_artifact(output, source, evaluator, maps_root):
    physical, support, physical_ref, support_ref, base = p.load_source(source)
    anchor_report = json.loads((output / a.REPORT).read_bytes())
    physical_report = json.loads((output / p.REPORT).read_bytes())
    a.validate_report(anchor_report, support, support_ref)
    rounds = base.one("rounds").physical_path
    physical_refs = {
        "physical_source": physical_ref,
        "support_source": support_ref,
        "anchors": p.file_reference(output / a.REPORT),
        "rounds": p.file_reference(rounds),
    }
    p.validate_report(
        physical_report,
        support,
        physical,
        anchor_report,
        physical_refs,
        json.loads(rounds.read_bytes())["rounds"],
    )
    manifest_path = source.parent / support["base_manifest"]["path"]
    manifest = json.loads(manifest_path.read_bytes())
    data = json.loads((output / DATA).read_bytes())
    report = json.loads((output / REPORT).read_bytes())
    refs = {
        "physical_source": physical_ref,
        "support_source": support_ref,
        "anchors": p.file_reference(output / a.REPORT),
        "physical_context": p.file_reference(output / p.REPORT),
        "base_manifest": p.file_reference(manifest_path),
        "evaluator": p.file_reference(evaluator),
    }
    a.require(data["references"] == refs, "NAV source hash references mismatch")
    nav = maps_root / data["map_name"] / (data["map_name"] + ".nav")
    if data["asset"]["sha256"] is not None:
        raw = nav.read_bytes()
        header = (
            struct.unpack("<3I", raw[:12]) if len(raw) >= 16 else (None, None, None)
        )
        a.require(
            header
            == tuple(
                data["asset"][k] for k in ("magic", "format_version", "subversion")
            ),
            "NAV header provenance mismatch",
        )
        a.require(
            a.digest(raw) == data["asset"]["sha256"]
            and len(raw) == data["asset"]["bytes"],
            "NAV retained asset hash mismatch",
        )
        # Rebuild from the pinned asset too; a forged but internally consistent
        # graph cannot pass solely by copying the asset's hash label.
        if data["graph"] is not None:
            request = {
                "maps_root": str(maps_root.resolve()),
                "map_name": data["map_name"],
                "asset_sha256": data["asset"]["sha256"],
                "queries": data["queries"],
            }
            reply = json.loads(
                subprocess.run(
                    [str(evaluator.resolve())],
                    input=encode(request),
                    capture_output=True,
                    check=True,
                ).stdout
            )
            a.require(
                reply == {"graph": data["graph"], "results": data["results"]},
                "NAV producer reconstruction mismatch",
            )
    else:
        a.require(not nav.exists(), "NAV asset absence mismatch")
    refs["navigation_source"] = p.file_reference(output / DATA)
    a.require(
        report
        == build_report(data, support, anchor_report, physical_report, refs, manifest),
        "NAV report recomputation mismatch",
    )
    return report["summary"]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--evaluator", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    fn = validate_artifact if args.validate else materialize
    print(
        json.dumps(
            fn(args.output, args.source, args.evaluator, args.maps_root)
            if args.validate
            else fn(args.source, args.output, args.evaluator, args.maps_root)
        )
    )


if __name__ == "__main__":
    main()
