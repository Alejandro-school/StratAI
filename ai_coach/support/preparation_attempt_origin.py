"""Optional native planting origins and progress at exact prospective C0 cuts."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors as a
from ai_coach.support import attempt_origin as k
from ai_coach.support import physical as p
from ai_coach.support import preparation as c
from ai_coach.support import preparation_objective_clock as clock

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_attempt_origin_cuts.jsonl"
ROW_SCHEMA = "support_preparation_attempt_origin_cut.json"
MANIFEST = "support_preparation_attempt_origin_manifest.json"


def prefix_source(data, sequence):
    cuts = clock.delivered(data["observations"]["cuts"], sequence)
    frontier = cuts[-1] if cuts else {}
    return {
        **data,
        "observations": {
            "cuts": cuts,
            "frames": data["observations"]["frames"][
                : frontier.get("frame_sequence", 0)
            ],
            "deliveries": data["observations"]["deliveries"][
                : frontier.get("delivery_sequence", 0)
            ],
        },
    }


def actor_context(alias, previous, cut, row, states, owner, frames, times):
    initial, references = k.initial_context(
        {"actor_alias": alias, "ally_alias": alias, "t0_frame": cut["frame"]},
        {"self": previous, "ally": previous},
        row,
        states,
        owner,
    )
    proof = references["self"]
    if proof is not None:
        current = a.actor_index(cut)[alias]
        start = bisect_right(times, proof["frame"]) - 1
        continuous = start >= 0
        for sample in frames[max(0, start) : cut["sequence"]]:
            player = a.actor_index(sample).get(alias)
            continuous = (
                continuous
                and player is not None
                and all(
                    player[name] == current[name]
                    for name in ("side", "pawn_life", "pawn_handle")
                )
                and all(sample[name] == cut[name] for name in ("instance", "epoch"))
            )
        if not continuous:
            return {
                name: k.o.fact(reason="native_origin_c0_actor_continuity_unaccredited")
                for name in k.PLAYER
            }, None
    return copy.deepcopy(initial["self"]), copy.deepcopy(proof)


def initial_features(
    support,
    objective,
    ownership,
    clocks,
    data,
    identifier,
    actor,
    ally,
    sequence,
    tick,
    frame,
):
    # C0 and every dependent clock stream are clipped before their validators run.
    previous = clock.initial_features(
        support,
        objective,
        ownership,
        clocks,
        identifier,
        actor,
        ally,
        sequence,
        tick,
        frame,
    )
    support, objective, ownership, clocks = clock.prefix_inputs(
        support, objective, ownership, clocks, sequence
    )
    data = prefix_source(data, sequence)
    k.validate_source(data, support, clocks)
    cut = data["observations"]["cuts"][-1]
    row = (
        data["observations"]["frames"][cut["frame_sequence"] - 1]
        if cut["frame_sequence"]
        else None
    )
    states = k.origin_prefixes(support, objective, ownership, clocks, data).get(
        cut["frame_sequence"], {}
    )
    owner_cut = next(
        r for r in ownership["observations"]["cuts"] if r["source_sequence"] == sequence
    )
    owner = ownership["observations"]["states"][owner_cut["state_sequence"] - 1]
    frames = support["observations"]["frames"]
    times = [r["frame"] for r in frames]
    for role, alias in (("self", actor), ("ally", ally)):
        facts, _ = actor_context(
            alias,
            previous[role].pop("objective_clock"),
            frames[-1],
            row,
            states,
            owner,
            frames,
            times,
        )
        previous[role]["attempt_origin"] = facts
    return previous


def iter_rows(support, objective, ownership, clocks, data):
    origins = k.origin_prefixes(support, objective, ownership, clocks, data)
    index = {r["source_sequence"]: r for r in data["observations"]["cuts"]}
    owners = {
        r["source_sequence"]: ownership["observations"]["states"][
            r["state_sequence"] - 1
        ]
        for r in ownership["observations"]["cuts"]
    }
    frames = support["observations"]["frames"]
    times = [r["frame"] for r in frames]
    for previous in clock.iter_rows(support, objective, ownership, clocks):
        seq = previous["cut"]["sequence"]
        cut = index[seq]
        row = (
            data["observations"]["frames"][cut["frame_sequence"] - 1]
            if cut["frame_sequence"]
            else None
        )
        actors = []
        for actor in previous["actors"]:
            initial, proof = actor_context(
                actor["actor_alias"],
                actor["initial"],
                frames[seq - 1],
                row,
                origins.get(cut["frame_sequence"], {}),
                owners.get(seq),
                frames,
                times,
            )
            actors.append(
                {
                    **{
                        name: actor[name]
                        for name in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "initial": initial,
                    "origin_reference": proof,
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_attempt_origin_cut@1",
            "cut": previous["cut"],
            "provenance": {
                name: cut[name]
                for name in (
                    "source_sequence",
                    "clock_order",
                    "frame_sequence",
                    "delivery_sequence",
                )
            }
            if previous["candidates"]
            else None,
            "actors": actors,
            "candidates": previous["candidates"],
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


def stream_rows(support, objective, ownership, clocks, data, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, objective, ownership, clocks, data):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "attempt origin C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(
                row == expected, "attempt origin C0 row differs from exact source"
            )
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
        a.require(not stream.read(1), "extra attempt origin C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    data, clocks, ownership, objective, support, _ = k.load_source(source)
    return (support, objective, ownership, clocks, data), {
        **data["references"],
        "attempt_source": p.file_reference(source),
        "preparation_protocol": c.protocol()[1],
    }


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_attempt_origin_manifest@1",
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
    a.require(coverage["counts"]["candidates"] > 0, "no planting C0 candidates")
    for field in k.PLAYER:
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
        "attempt origin C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "attempt origin C0 manifest changed while reading",
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
