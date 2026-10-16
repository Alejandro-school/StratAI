"""Optional native own/ally attack deadlines at exact prospective C0 cuts."""

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
from ai_coach.support import preparation_objective_clock as clock_c0
from ai_coach.support import weapon_timing as k

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_weapon_timing_cuts.jsonl"
ROW_SCHEMA = "support_preparation_weapon_timing_cut.json"
MANIFEST = "support_preparation_weapon_timing_manifest.json"


def prefix_inputs(support, motion, physical, objective, ownership, clock, data, seq):
    support, objective, ownership, clock = clock_c0.prefix_inputs(
        support, objective, ownership, clock, seq
    )
    motion, data = (
        motion_c0.prefix_source(motion, seq),
        motion_c0.prefix_source(data, seq),
    )
    physical = {
        **physical,
        "observations": {
            "frames": clock_c0.delivered(physical["observations"]["frames"], seq)
        },
    }
    return support, motion, physical, objective, ownership, clock, data


def actor_context(alias, support, data, physical, clock, sequence, witness):
    facts, refs = k.initial_context(
        {"actor_alias": alias, "ally_alias": alias, "source_sequence": sequence},
        data,
        physical,
        clock,
        witness,
    )
    reference = refs["self"]
    if reference:
        state = data["observations"]["states"][reference["pawn_state_sequence"] - 1]
        frames = support["observations"]["frames"]
        current_cut = frames[sequence - 1]
        current = a.actor_index(current_cut)[alias]
        for sample in frames[state["first_cut"] - 1 : sequence]:
            players = [x for x in sample["players"] if x["actor_alias"] == alias]
            if (
                len(players) != 1
                or any(
                    players[0][n] != current[n]
                    for n in ("side", "pawn_life", "pawn_handle")
                )
                or any(sample[n] != current_cut[n] for n in ("instance", "epoch"))
            ):
                return k.missing("native_timing_c0_actor_continuity_unaccredited"), None
    return copy.deepcopy(facts["self"]), copy.deepcopy(reference)


def initial_features(
    support,
    motion,
    physical,
    objective,
    ownership,
    clock,
    data,
    identifier,
    actor,
    ally,
    sequence,
    tick,
    frame,
):
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    support, motion, physical, objective, ownership, clock, data = prefix_inputs(
        support, motion, physical, objective, ownership, clock, data, sequence
    )
    k.o.validate_source(objective, support)
    clock_c0.c4.validate_source(ownership, support, objective)
    k.c.validate_source(clock, support, objective, ownership)
    a.schema_validator(k.m.DATA).validate(motion)
    k.validate_source(data, support, motion, physical, clock)
    alignment, _ = k.c.alignment_prefixes(support, objective, clock)
    for role, alias in (("self", actor), ("ally", ally)):
        facts, _ = actor_context(
            alias, support, data, physical, clock, sequence, alignment.get(sequence)
        )
        initial[role]["weapon_timing"] = facts
    return initial


def iter_rows(support, motion, physical, objective, ownership, clock, data):
    alignment, _ = k.c.alignment_prefixes(support, objective, clock)
    for base in support["observations"]["frames"]:
        seq = base["sequence"]
        cut = data["observations"]["cuts"][seq - 1]
        pairs, roster = c.select_pairs(base), c.roster(base)[0]
        actors = []
        for alias in sorted({actor for actor, _ in pairs}):
            initial, reference = actor_context(
                alias, support, data, physical, clock, seq, alignment.get(seq)
            )
            actors.append(
                {
                    **{
                        n: roster[alias][n]
                        for n in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "initial": initial,
                    "timing_reference": reference,
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_weapon_timing_cut@1",
            "cut": c.cut_reference(base),
            "provenance": {
                **{
                    n: cut[n]
                    for n in (
                        "source_sequence",
                        "clock_order",
                        "state_sequence",
                        "delivery_sequence",
                    )
                },
                "clock_witness": copy.deepcopy(alignment.get(seq)),
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


def stream_rows(
    support, motion, physical, objective, ownership, clock, data, stream, *, reading
):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(
        support, motion, physical, objective, ownership, clock, data
    ):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "weapon timing C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(row == expected, "weapon timing C0 row differs from exact source")
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
        a.require(not stream.read(1), "extra weapon timing C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    *inputs, _ = k.load_source(source)
    return tuple(inputs), {
        **inputs[-1]["references"],
        "timing_source": p.file_reference(source),
        "preparation_protocol": c.protocol()[1],
    }


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_weapon_timing_manifest@1",
        contract_version=VERSION,
        references=refs,
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
    a.require(coverage["counts"]["candidates"] > 0, "no weapon timing C0 candidates")
    for field in k.PLAYER[-2:]:
        a.require(
            any(
                cell["fact_status_counts"]["self." + field].get("observed:present", 0)
                for cell in coverage["matrix"]
            ),
            "no observed C0 " + field,
        )


def materialize(source: Path, output: Path, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    inputs, refs = load_inputs(source)
    support = inputs[0]
    with (output / ROWS).open("xb") as stream:
        artifact, coverage = stream_rows(*inputs, stream, reading=False)
    if require_observed:
        require_observed_facts(coverage)
    result = manifest(support, refs, artifact, coverage)
    candidate_io._write_new(output / MANIFEST, result)
    return result


def validate_artifact(source: Path, output: Path):
    safe = candidate_io.source_io._safe_file
    output = output.absolute()
    raw_manifest = safe(output, MANIFEST).read_bytes()
    result = json.loads(raw_manifest)
    a.schema_validator(MANIFEST).validate(result)
    inputs, refs = load_inputs(source)
    support = inputs[0]
    with safe(output, ROWS).open("rb") as stream:
        artifact, coverage = stream_rows(*inputs, stream, reading=True)
    a.require(
        result == manifest(support, refs, artifact, coverage),
        "weapon timing C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "weapon timing C0 manifest changed while reading",
    )
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args(argv)
    result = (
        validate_artifact(args.source, args.output)
        if args.validate
        else materialize(
            args.source, args.output, require_observed=args.require_observed
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
