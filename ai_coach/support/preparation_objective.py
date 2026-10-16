"""Optional public and own/ally objective facts at exact prospective C0 cuts."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import physical as p
from ai_coach.support import preparation as c

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_objective_cuts.jsonl"
ROW_SCHEMA = "support_preparation_objective_cut.json"
MANIFEST = "support_preparation_objective_manifest.json"


def initial_features(
    support, objective, identifier, actor, ally, sequence, tick, frame
):
    """Clip both delivery streams before validation; expose detached facts only."""
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    frames = support["observations"]["frames"][:sequence]
    prefix = {**support, "observations": {"frames": frames}}
    cuts = []
    for row in objective["observations"]["frames"]:
        if row["source_sequence"] > sequence:
            break
        cuts.append(row)
        if row["source_sequence"] == sequence:
            break
    frontier = cuts[-1]["event_sequence"] if cuts else 0
    # Do not inspect a later event, even its tick/type, after the recorded frontier.
    data = {
        **objective,
        "observations": {
            "frames": cuts,
            "events": objective["observations"]["events"][:frontier],
        },
    }
    o.validate_source(data, prefix)
    row = cuts[-1] if cuts and cuts[-1]["source_sequence"] == sequence else None
    facts = o.initial_context(
        {"actor_alias": actor, "ally_alias": ally},
        frames[-1],
        row,
        o.public_prefixes(data).get(sequence, {}),
    )
    initial["global"] = copy.deepcopy(facts["global"])
    for role in c.ROLES:
        initial[role]["objective"] = copy.deepcopy(facts[role])
    return initial


def iter_rows(support, objective):
    """Normalize global facts per cut and player facts per selected actor/cut."""
    index = {row["source_sequence"]: row for row in objective["observations"]["frames"]}
    states = o.public_prefixes(objective)
    for cut in support["observations"]["frames"]:
        pairs = c.select_pairs(cut)
        roster = c.roster(cut)[0]
        row = index.get(cut["sequence"])
        state = states.get(cut["sequence"], {})
        players = {v["actor_alias"]: v for v in row["players"]} if row else {}
        actors = []
        global_facts = None
        for alias in sorted({actor for actor, _ in pairs}):
            player = roster[alias]
            facts = o.initial_context(
                {"actor_alias": alias, "ally_alias": alias}, cut, row, state
            )
            global_facts = facts["global"]
            actors.append(
                {
                    **{
                        k: player[k]
                        for k in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "native_player": copy.deepcopy(players.get(alias)),
                    "initial": facts["self"],
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_objective_cut@1",
            "cut": c.cut_reference(cut),
            # Excluded cuts remain in the population, without initial features.
            "global": global_facts,
            "provenance": {
                "source_sequence": row["source_sequence"],
                "order": row["order"],
                "event_sequence": row["event_sequence"],
                "rules": copy.deepcopy(row["rules"]),
                "public_state": copy.deepcopy(state),
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
            cell = self.matrix.setdefault(
                key,
                {
                    "candidates": 0,
                    "phase_observed": 0,
                    "both_available": Counter({k: 0 for k in o.PLAYER}),
                    "fact_status_counts": {k: Counter() for k in o.FIELDS},
                },
            )
            cell["candidates"] += 1
            cell["phase_observed"] += (
                row["global"]["objective_phase"]["status"] == "observed"
            )
            facts = {
                "global": row["global"],
                "self": actors[actor]["initial"],
                "ally": actors[ally]["initial"],
            }
            for field in o.FIELDS:
                role, name = field.split(".")
                fact = facts[role][name]
                cell["fact_status_counts"][field][
                    fact["status"] + ":" + (fact["reason"] or "present")
                ] += 1
            for name in o.PLAYER:
                cell["both_available"][name] += all(
                    facts[role][name]["status"] == "observed" for role in c.ROLES
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


def stream_rows(support, objective, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, objective):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "objective C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(row == expected, "objective C0 row differs from exact source")
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
        a.require(not stream.read(1), "extra objective C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, o.DATA)
    objective, support, support_ref, _ = o.load_source(source)
    return (
        objective,
        support,
        {
            "objective_source": p.file_reference(source),
            "support_source": support_ref,
            "preparation_protocol": c.protocol()[1],
        },
    )


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_objective_manifest@1",
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
    a.require(coverage["counts"]["candidates"] > 0, "no objective C0 candidates")
    a.require(
        any(cell["phase_observed"] for cell in coverage["matrix"]),
        "no observed objective C0 phase",
    )
    a.require(
        any(cell["both_available"]["in_bomb_zone"] for cell in coverage["matrix"]),
        "no observed own/ally objective C0 facts",
    )


def materialize(source: Path, output: Path, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    objective, support, refs = load_inputs(source)
    with (output / ROWS).open("xb") as stream:
        artifact, coverage = stream_rows(support, objective, stream, reading=False)
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
    objective, support, refs = load_inputs(source)
    with safe(output, ROWS).open("rb") as stream:
        artifact, coverage = stream_rows(support, objective, stream, reading=True)
    a.require(
        result == manifest(support, refs, artifact, coverage),
        "objective C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "objective C0 manifest changed while reading",
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
