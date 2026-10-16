"""Optional legal objective positions at exact prospective C0 cuts."""

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
from ai_coach.support import objective_position as k
from ai_coach.support import physical as p
from ai_coach.support import preparation as c
from ai_coach.support import preparation_attempt_origin as origin
from ai_coach.support import preparation_objective as objective_c0
from ai_coach.support import preparation_objective_clock as clock

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_objective_position_cuts.jsonl"
ROW_SCHEMA = "support_preparation_objective_position_cut.json"
MANIFEST = "support_preparation_objective_position_manifest.json"


def prefix_source(data, sequence):
    cuts = clock.delivered(data["observations"]["cuts"], sequence)
    frontier = cuts[-1] if cuts else {}
    return {
        **data,
        "observations": {
            "cuts": cuts,
            **{
                name: data["observations"][name][: frontier.get(key, 0)]
                for name, key in (
                    ("states", "state_sequence"),
                    ("births", "birth_sequence"),
                    ("deliveries", "delivery_sequence"),
                )
            },
        },
    }


def actor_context(
    alias, initial, owner, state, cut, witnesses, changes, frames, times, deliveries
):
    facts, refs = k.initial_context(
        {"actor_alias": alias, "ally_alias": alias},
        initial,
        owner,
        state,
        cut,
        witnesses,
        changes,
    )
    reference = refs["self"]
    if reference and reference["basis"] == "witnessed_stationary_plant":
        proof = witnesses[reference["entity_life"]]
        onset = deliveries[proof["attempt_delivery_sequence"] - 1]["frame"]
        start = bisect_right(times, onset) - 1
        current_cut = frames[cut["source_sequence"] - 1]
        current = a.actor_index(current_cut)[alias]
        continuous = start >= 0
        for sample in frames[max(0, start) : cut["source_sequence"]]:
            player = a.actor_index(sample).get(alias)
            continuous = (
                continuous
                and player is not None
                and all(
                    player[n] == current[n]
                    for n in ("side", "pawn_life", "pawn_handle")
                )
                and all(sample[n] == current_cut[n] for n in ("instance", "epoch"))
            )
        if not continuous:
            return k.unavailable(
                "native_position_c0_actor_continuity_unaccredited"
            ), None
    return copy.deepcopy(facts["self"]), copy.deepcopy(reference)


def initial_features(
    support,
    objective,
    ownership,
    clocks,
    attempt,
    data,
    identifier,
    actor,
    ally,
    sequence,
    tick,
    frame,
):
    initial = clock.initial_features(
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
    attempt, data = (
        origin.prefix_source(attempt, sequence),
        prefix_source(data, sequence),
    )
    k.r.validate_source(attempt, support, clocks)
    k.validate_source(data, support, ownership, clocks)
    frames = support["observations"]["frames"]
    cut = data["observations"]["cuts"][-1]
    public = k.o.initial_context(
        {"actor_alias": actor, "ally_alias": ally},
        frames[-1],
        objective["observations"]["frames"][-1],
        k.o.public_prefixes(objective).get(sequence, {}),
    )
    owner_cut = ownership["observations"]["cuts"][-1]
    owner = ownership["observations"]["states"][owner_cut["state_sequence"] - 1]
    state = data["observations"]["states"][cut["state_sequence"] - 1]
    witnesses = k.planting_witnesses(
        support, objective, ownership, clocks, attempt, data
    )
    changes = k.position_changes(data)
    times = [r["frame"] for r in frames]
    for role, alias in (("self", actor), ("ally", ally)):
        facts, _ = actor_context(
            alias,
            {"global": public["global"], "self": public[role], "ally": public[role]},
            owner,
            state,
            cut,
            witnesses,
            changes,
            frames,
            times,
            attempt["observations"]["deliveries"],
        )
        initial[role].pop("objective_clock")
        initial[role]["objective_position"] = facts
    return initial


def iter_rows(support, objective, ownership, clocks, attempt, data):
    witnesses = k.planting_witnesses(
        support, objective, ownership, clocks, attempt, data
    )
    changes = k.position_changes(data)
    index = {r["source_sequence"]: r for r in data["observations"]["cuts"]}
    owners = {
        r["source_sequence"]: ownership["observations"]["states"][
            r["state_sequence"] - 1
        ]
        for r in ownership["observations"]["cuts"]
    }
    frames = support["observations"]["frames"]
    times = [r["frame"] for r in frames]
    for previous in objective_c0.iter_rows(support, objective):
        seq = previous["cut"]["sequence"]
        cut = index[seq]
        state = data["observations"]["states"][cut["state_sequence"] - 1]
        actors = []
        for actor in previous["actors"]:
            initial, reference = actor_context(
                actor["actor_alias"],
                {
                    "global": previous["global"],
                    "self": actor["initial"],
                    "ally": actor["initial"],
                },
                owners[seq],
                state,
                cut,
                witnesses,
                changes,
                frames,
                times,
                attempt["observations"]["deliveries"],
            )
            actors.append(
                {
                    **{
                        n: actor[n]
                        for n in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "initial": initial,
                    "position_reference": reference,
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_objective_position_cut@1",
            "cut": previous["cut"],
            "provenance": {
                n: cut[n]
                for n in (
                    "source_sequence",
                    "clock_order",
                    "state_sequence",
                    "birth_sequence",
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


def stream_rows(
    support, objective, ownership, clocks, attempt, data, stream, *, reading
):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, objective, ownership, clocks, attempt, data):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "objective position C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(
                row == expected, "objective position C0 row differs from exact source"
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
        a.require(not stream.read(1), "extra objective position C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    data, attempt, clocks, ownership, objective, support, _ = k.load_source(source)
    return (support, objective, ownership, clocks, attempt, data), {
        **data["references"],
        "position_source": p.file_reference(source),
        "preparation_protocol": c.protocol()[1],
    }


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_objective_position_manifest@1",
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
    a.require(
        coverage["counts"]["candidates"] > 0, "no objective position C0 candidates"
    )
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
        "objective position C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "objective position C0 manifest changed while reading",
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
