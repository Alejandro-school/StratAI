"""Offline ICT-I1.A ledger writer/reader. Output is confined to empty TEMP."""

from __future__ import annotations

import argparse
from collections import Counter
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import temporal as t
from ai_coach.support import anchors as a, c4, objective, physical

ROWS = "temporal_descriptions.jsonl"
MANIFEST = "temporal_description_manifest.json"


def load_sources(directory):
    """Artifact integrity is distinct from the model's causal projection."""
    support, support_ref, base = a.load_source(directory / a.DATA)
    sources, refs = {}, {a.DATA: support_ref}
    for module, keyword in (
        (physical, "physical_source"),
        (objective, "objective_source"),
        (c4, "c4_source"),
    ):
        path = directory / module.DATA
        a.candidate_io.source_io._safe_file(directory.absolute(), module.DATA)
        a.require(path.stat().st_size <= 512 * 1024 * 1024, "oversized temporal source")
        raw = path.read_bytes()
        data = json.loads(raw)
        a.require(
            data["support_source"] == support_ref, "temporal support hash mismatch"
        )
        if module is c4:
            a.require(
                data["objective_source"] == refs[objective.DATA],
                "temporal objective hash mismatch",
            )
            module.validate_source(data, support, sources["objective_source"])
        else:
            module.validate_source(data, support)
        a.require(path.read_bytes() == raw, "temporal source changed during read")
        sources[keyword] = data
        refs[module.DATA] = {
            "path": module.DATA,
            "bytes": len(raw),
            "sha256": a.digest(raw),
        }
    return support, sources, refs, base


def count_row(counts, row):
    counts["records"] += 1
    counts["actors"] += len(row["actors"])
    counts["missing_previous_actors"] += len(row["missing_previous_actors"])
    counts["hypotheses_unavailable"] += len(row["hypotheses"])
    for actor in row["actors"]:
        counts["excluded:" + (actor["exclusion"] or "none")] += 1
        for field in ("position", "alive", "has_c4", "sampled_displacement"):
            fact = actor[field]
            counts[
                field + ":" + fact["status"] + ":" + (fact["reason"] or "present")
            ] += 1
        for field, fact in actor["physical"].items():
            counts[
                field + ":" + fact["status"] + ":" + (fact["reason"] or "present")
            ] += 1


def record_stream(support, sources, sides, sequence):
    for side in sides:
        yield from t.iter_records(
            support, side=side, through_sequence=sequence, **sources
        )


def materialize(source, output, *, sides=("CT", "T"), through_sequence=None):
    output = validate_output_directory(output)
    a.require(
        sides and len(sides) == len(set(sides)) and set(sides) <= {"CT", "T"},
        "invalid sides",
    )
    support, sources, refs, _ = load_sources(source)
    sequence = (
        through_sequence
        if through_sequence is not None
        else len(support["observations"]["frames"])
    )
    counts = Counter()
    with (output / ROWS).open("x", encoding="utf-8", newline="\n") as stream:
        for row in record_stream(support, sources, sides, sequence):
            t.validate_record(row)
            count_row(counts, row)
            stream.write(
                json.dumps(row, sort_keys=True, separators=(",", ":"), allow_nan=False)
                + "\n"
            )
    a.require(
        counts["position:observed:present"] > 0
        and counts["sampled_displacement:observed:present"] > 0,
        "empty affected temporal records",
    )
    manifest = {
        "schema_id": "stratai.temporal_description_manifest@1",
        "contract_version": t.VERSION,
        "source_alias": support["source_alias"],
        "source_sha256": support["source_sha256"],
        "configuration": t.CONFIGURATION,
        "sides": list(sides),
        "through_sequence": sequence,
        "source_artifacts": refs,
        "artifact": physical.file_reference(output / ROWS),
        "counts": dict(sorted(counts.items())),
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }
    with (output / MANIFEST).open("x", encoding="utf-8") as stream:
        json.dump(manifest, stream, indent=2, sort_keys=True)
        stream.write("\n")
    return manifest


def validate(source, output):
    manifest = json.loads((output / MANIFEST).read_bytes())
    support, sources, refs, _ = load_sources(source)
    a.require(manifest["source_artifacts"] == refs, "temporal input artifacts changed")
    a.require(
        manifest["artifact"] == physical.file_reference(output / ROWS),
        "temporal artifact changed",
    )
    counts = Counter()
    expected = record_stream(
        support, sources, manifest["sides"], manifest["through_sequence"]
    )
    with (output / ROWS).open(encoding="utf-8") as stream:
        for row in expected:
            actual = json.loads(next(stream, "null"))
            a.require(actual == row, "temporal causal recomputation differs")
            t.validate_record(actual)
            count_row(counts, actual)
        a.require(not stream.read(), "extra temporal rows")
    a.require(dict(counts) == manifest["counts"], "temporal counts differ")
    a.require(
        manifest
        == {
            "schema_id": "stratai.temporal_description_manifest@1",
            "contract_version": t.VERSION,
            "source_alias": support["source_alias"],
            "source_sha256": support["source_sha256"],
            "configuration": t.CONFIGURATION,
            "sides": manifest["sides"],
            "through_sequence": manifest["through_sequence"],
            "source_artifacts": refs,
            "artifact": manifest["artifact"],
            "counts": dict(sorted(counts.items())),
            "comparison_allowed": False,
            "training_allowed": False,
            "product_supported": False,
        },
        "temporal manifest differs",
    )
    a.require(
        manifest["sides"] and len(set(manifest["sides"])) == len(manifest["sides"]),
        "invalid sides",
    )
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("materialize", "validate"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--through-sequence", type=int)
    args = parser.parse_args()
    result = (
        materialize(args.source, args.output, through_sequence=args.through_sequence)
        if args.action == "materialize"
        else validate(args.source, args.output)
    )
    print(json.dumps({"ok": True, "counts": result["counts"]}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
