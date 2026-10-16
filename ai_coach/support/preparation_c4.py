"""Optional own/ally C4 ownership at exact prospective C0 cuts."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors as a
from ai_coach.support import c4 as k
from ai_coach.support import objective as o
from ai_coach.support import physical as p
from ai_coach.support import preparation as c

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_c4_cuts.jsonl"
ROW_SCHEMA = "support_preparation_c4_cut.json"
MANIFEST = "support_preparation_c4_manifest.json"


def initial_features(
    support, objective, data, identifier, actor, ally, sequence, tick, frame
):
    """Clip all delivered prefixes before validation; return detached own/ally facts."""
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    frames = support["observations"]["frames"][:sequence]
    prefix = {**support, "observations": {"frames": frames}}
    objective_cuts = []
    for row in objective["observations"]["frames"]:
        if row["source_sequence"] > sequence:
            break
        objective_cuts.append(row)
        if row["source_sequence"] == sequence:
            break
    frontier = objective_cuts[-1]["event_sequence"] if objective_cuts else 0
    objective_prefix = {
        **objective,
        "observations": {
            "frames": objective_cuts,
            "events": objective["observations"]["events"][:frontier],
        },
    }
    cuts = []
    for row in data["observations"]["cuts"]:
        if row["source_sequence"] > sequence:
            break
        cuts.append(row)
        if row["source_sequence"] == sequence:
            break
    states = (
        data["observations"]["states"][: cuts[-1]["state_sequence"]] if cuts else []
    )
    clipped = {**data, "observations": {"cuts": cuts, "states": states}}
    o.validate_source(objective_prefix, prefix)
    k.validate_source(clipped, prefix, objective_prefix)
    state = (
        states[cuts[-1]["state_sequence"] - 1]
        if cuts and cuts[-1]["source_sequence"] == sequence
        else None
    )
    facts = k.initial_context(
        {"actor_alias": actor, "ally_alias": ally}, frames[-1], state
    )
    for role in c.ROLES:
        initial[role]["c4"] = copy.deepcopy(facts[role])
    return initial


def iter_rows(support, data):
    """One source cut and one record per selected actor; every C0 pair retained."""
    index = {row["source_sequence"]: row for row in data["observations"]["cuts"]}
    states = data["observations"]["states"]
    for cut in support["observations"]["frames"]:
        pairs = c.select_pairs(cut)
        roster = c.roster(cut)[0]
        row = index.get(cut["sequence"])
        state = states[row["state_sequence"] - 1] if row else None
        players = {v["actor_alias"]: v for v in state["players"]} if state else {}
        actors = []
        for alias in sorted({actor for actor, _ in pairs}):
            player = roster[alias]
            facts = k.initial_context(
                {"actor_alias": alias, "ally_alias": alias}, cut, state
            )
            actors.append(
                {
                    **{
                        name: player[name]
                        for name in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "native_player": copy.deepcopy(players.get(alias)),
                    "initial": facts["self"],
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_c4_cut@1",
            "cut": c.cut_reference(cut),
            # Native entities remain in the hash-bound source, not in initial features.
            "provenance": {
                **{
                    name: row[name]
                    for name in ("source_sequence", "state_sequence", "objective_order")
                },
                "first_cut": state["first_cut"],
            }
            if row is not None and pairs
            else None,
            "actors": actors,
            "candidates": [
                {
                    "candidate_id": c.candidate_id(
                        support["source_alias"], cut, actor, ally
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
                    "possession_counts": {
                        role: Counter(positive=0, negative=0, insufficient=0)
                        for role in c.ROLES
                    },
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
            for role in c.ROLES:
                value = facts[role]["has_c4"]["value"]
                cell["possession_counts"][role][
                    "positive"
                    if value is True
                    else "negative"
                    if value is False
                    else "insufficient"
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


def stream_rows(support, data, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, data):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "C4 C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(row == expected, "C4 C0 row differs from exact source")
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
        a.require(not stream.read(1), "extra C4 C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    data, _, support, support_ref, _ = k.load_source(source)
    return (
        data,
        support,
        {
            "c4_source": p.file_reference(source),
            "objective_source": p.file_reference(source.parent / o.DATA),
            "support_source": support_ref,
            "preparation_protocol": c.protocol()[1],
        },
    )


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_c4_manifest@1",
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
    a.require(coverage["counts"]["candidates"] > 0, "no C4 C0 candidates")
    a.require(
        any(
            cell["possession_counts"][role]["positive"]
            for cell in coverage["matrix"]
            for role in c.ROLES
        ),
        "no confirmed C4 C0 carrier",
    )


def materialize(source: Path, output: Path, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    data, support, refs = load_inputs(source)
    with (output / ROWS).open("xb") as stream:
        artifact, coverage = stream_rows(support, data, stream, reading=False)
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
    data, support, refs = load_inputs(source)
    with safe(output, ROWS).open("rb") as stream:
        artifact, coverage = stream_rows(support, data, stream, reading=True)
    a.require(
        result == manifest(support, refs, artifact, coverage),
        "C4 C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "C4 C0 manifest changed while reading",
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
