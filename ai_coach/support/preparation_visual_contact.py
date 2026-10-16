"""Optional own/ally native visual inputs at exact prospective C0 cuts."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors as a
from ai_coach.support import physical as p
from ai_coach.support import preparation as c
from ai_coach.support import preparation_native_motion as motion_c0
from ai_coach.support import visual_contact as k

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_visual_contact_cuts.jsonl"
ROW_SCHEMA = "support_preparation_visual_contact_cut.json"
MANIFEST = "support_preparation_visual_contact_manifest.json"


def prefix_inputs(support, timing, data, sequence):
    support = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:sequence]},
    }
    return (
        support,
        motion_c0.prefix_source(timing, sequence),
        motion_c0.prefix_source(data, sequence),
    )


def actor_context(alias, support, data, sequence):
    cut = data["observations"]["cuts"][sequence - 1]
    player = next(x for x in cut["players"] if x["actor_alias"] == alias)
    state_sequence = player["state_sequence"]
    if state_sequence is None:
        return k.native_facts(None), None
    state = data["observations"]["states"][state_sequence - 1]
    frames = support["observations"]["frames"]
    current_cut = frames[sequence - 1]
    current = a.actor_index(current_cut)[alias]
    for sample in frames[state["first_cut"] - 1 : sequence]:
        players = [x for x in sample["players"] if x["actor_alias"] == alias]
        if (
            len(players) != 1
            or any(
                players[0][name] != current[name]
                for name in ("side", "pawn_life", "pawn_handle")
            )
            or any(sample[name] != current_cut[name] for name in ("instance", "epoch"))
        ):
            facts = k.native_facts(None)
            for name in k.PLAYER[:2]:
                facts[name] = k.o.fact(
                    reason="native_visual_c0_actor_continuity_unaccredited"
                )
            return facts, None
    return copy.deepcopy(k.native_facts(state)), {
        "source_sequence": sequence,
        "frame": cut["frame"],
        "state_sequence": state_sequence,
        "pawn_life": player["pawn_life"],
    }


def initial_features(
    support, timing, data, identifier, actor, ally, sequence, tick, frame
):
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    support, timing, data = prefix_inputs(support, timing, data, sequence)
    a.schema_validator(k.t.DATA).validate(timing)
    k.validate_source(data, support, timing)
    for role, alias in (("self", actor), ("ally", ally)):
        initial[role]["visual_inputs"] = actor_context(alias, support, data, sequence)[
            0
        ]
    return initial


def iter_rows(support, timing, data):
    for base in support["observations"]["frames"]:
        sequence = base["sequence"]
        cut = data["observations"]["cuts"][sequence - 1]
        pairs, roster = c.select_pairs(base), c.roster(base)[0]
        actors = []
        for alias in sorted({actor for actor, _ in pairs}):
            initial, reference = actor_context(alias, support, data, sequence)
            actors.append(
                {
                    **{
                        name: roster[alias][name]
                        for name in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "initial": initial,
                    "visual_reference": reference,
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_visual_contact_cut@1",
            "cut": c.cut_reference(base),
            "provenance": {
                name: cut[name]
                for name in (
                    "source_sequence",
                    "clock_order",
                    "state_sequence",
                    "delivery_sequence",
                )
            }
            if pairs
            else None,
            "actors": actors,
            "candidates": [
                {
                    "candidate_id": c.candidate_id(
                        support["source_alias"], base, actor, ally
                    ),
                    "actor_alias": actor,
                    "ally_alias": ally,
                }
                for actor, ally in pairs
            ],
        }


class Coverage:
    def __init__(self):
        self.counts = Counter(cuts=0, actor_cuts=0, candidates=0)
        self.matrix = {}

    def add(self, row):
        self.counts.update(
            cuts=1, actor_cuts=len(row["actors"]), candidates=len(row["candidates"])
        )
        actors = {v["actor_alias"]: v for v in row["actors"]}
        for pair in row["candidates"]:
            actor, ally = pair["actor_alias"], pair["ally_alias"]
            key = (
                actor,
                ally,
                actors[actor]["side"],
                row["cut"]["instance"],
                row["cut"]["epoch"],
            )
            if key not in self.matrix:
                self.matrix[key] = {
                    "candidates": 0,
                    "both_available": Counter({field: 0 for field in k.PLAYER}),
                    "fact_status_counts": {field: Counter() for field in k.FIELDS},
                }
            cell = self.matrix[key]
            cell["candidates"] += 1
            facts = {"self": actors[actor]["initial"], "ally": actors[ally]["initial"]}
            for field in k.FIELDS:
                role, name = field.split(".")
                fact = facts[role][name]
                cell["fact_status_counts"][field][
                    fact["status"] + ":" + (fact["reason"] or "present")
                ] += 1
            for name in k.PLAYER:
                cell["both_available"][name] += all(
                    facts[role][name]["status"] != "unavailable" for role in c.ROLES
                )

    def result(self):
        return {
            "counts": dict(self.counts),
            "matrix": [
                {
                    "actor_alias": key[0],
                    "ally_alias": key[1],
                    "side": key[2],
                    "instance": key[3],
                    "epoch": key[4],
                    **self.matrix[key],
                }
                for key in sorted(self.matrix)
            ],
        }


def stream_rows(support, timing, data, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, timing, data):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "visual input C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(row == expected, "visual input C0 row differs from exact source")
        else:
            row, raw = expected, c.encoded_row(expected)
        validator.validate(row)
        if not reading:
            stream.write(raw)
        coverage.add(row)
        checksum.update(raw)
        size += len(raw)
        count += 1
    if reading:
        a.require(not stream.read(1), "extra visual input C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source, visual_evaluator, maps_root):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    support, _, timing, data, _ = k.load_source(source)
    physics = k.verify_physics(source, data, maps_root, visual_evaluator)
    return (
        (support, timing, data),
        {
            **data["references"],
            "visual_source": p.file_reference(source),
            "preparation_protocol": c.protocol()[1],
        },
        physics,
    )


def manifest(support, refs, artifact, coverage, physics):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_visual_contact_manifest@1",
        contract_version=VERSION,
        references=refs,
        evaluator_diagnostics={
            "visibility_scope": "internal_physics_diagnostic_not_initial_information",
            "static_ray_recomputation": physics,
        },
        artifact=artifact,
        coverage=coverage,
        comparison_allowed=False,
        training_allowed=False,
        product_supported=False,
        decision_eligible=False,
        physical_entry_status="not_accredited",
        tactical_commitment_status="not_accredited",
    )
    a.schema_validator(MANIFEST).validate(result)
    return result


def require_observed_facts(coverage):
    a.require(coverage["counts"]["candidates"] > 0, "no visual input C0 candidates")
    for field in k.PLAYER[:2]:
        a.require(
            any(
                cell["fact_status_counts"]["self." + field].get("observed:present", 0)
                for cell in coverage["matrix"]
            ),
            "no observed C0 " + field,
        )


def materialize(
    source: Path, output: Path, visual_evaluator, maps_root, *, require_observed=False
):
    output = candidate_io.validate_output_directory(output)
    inputs, refs, physics = load_inputs(source, visual_evaluator, maps_root)
    support = inputs[0]
    with (output / ROWS).open("xb") as stream:
        artifact, coverage = stream_rows(*inputs, stream, reading=False)
    if require_observed:
        require_observed_facts(coverage)
    result = manifest(support, refs, artifact, coverage, physics)
    candidate_io._write_new(output / MANIFEST, result)
    return result


def validate_artifact(source: Path, output: Path, visual_evaluator, maps_root):
    safe = candidate_io.source_io._safe_file
    output = output.absolute()
    raw_manifest = safe(output, MANIFEST).read_bytes()
    result = json.loads(raw_manifest)
    a.schema_validator(MANIFEST).validate(result)
    inputs, refs, physics = load_inputs(source, visual_evaluator, maps_root)
    support = inputs[0]
    with safe(output, ROWS).open("rb") as stream:
        artifact, coverage = stream_rows(*inputs, stream, reading=True)
    a.require(
        result == manifest(support, refs, artifact, coverage, physics),
        "visual input C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "visual input C0 manifest changed while reading",
    )
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    parser.add_argument("--require-observed", action="store_true")
    parser.add_argument("--visual-evaluator", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    args = parser.parse_args(argv)
    result = (
        validate_artifact(
            args.source, args.output, args.visual_evaluator, args.maps_root
        )
        if args.validate
        else materialize(
            args.source,
            args.output,
            args.visual_evaluator,
            args.maps_root,
            require_observed=args.require_observed,
        )
    )
    if args.require_observed:
        require_observed_facts(result["coverage"])
    print(
        json.dumps(
            {
                "ok": True,
                "counts": result["coverage"]["counts"],
                "artifact": result["artifact"],
            }
        )
    )


if __name__ == "__main__":
    main()
