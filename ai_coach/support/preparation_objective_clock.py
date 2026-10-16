"""Optional native objective clocks at exact prospective C0 deliveries."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors as a
from ai_coach.support import c4
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as k
from ai_coach.support import physical as p
from ai_coach.support import preparation as c

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_objective_clock_cuts.jsonl"
ROW_SCHEMA = "support_preparation_objective_clock_cut.json"
MANIFEST = "support_preparation_objective_clock_manifest.json"


def delivered(rows, sequence):
    """Stop at the exact delivery or first later locator, without scanning future."""
    result = []
    for row in rows:
        if row["source_sequence"] > sequence:
            break
        result.append(row)
        if row["source_sequence"] == sequence:
            break
    return result


def prefix_inputs(support, objective, ownership, data, sequence):
    support = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:sequence]},
    }
    frames = delivered(objective["observations"]["frames"], sequence)
    objective = {
        **objective,
        "observations": {
            "frames": frames,
            "events": objective["observations"]["events"][
                : frames[-1]["event_sequence"] if frames else 0
            ],
        },
    }
    cuts = delivered(ownership["observations"]["cuts"], sequence)
    ownership = {
        **ownership,
        "observations": {
            "cuts": cuts,
            "states": ownership["observations"]["states"][
                : cuts[-1]["state_sequence"] if cuts else 0
            ],
        },
    }
    data = {
        **data,
        "observations": {"frames": delivered(data["observations"]["frames"], sequence)},
    }
    return support, objective, ownership, data


def actor_context(alias, current, prior, witness):
    # An interruption is meaningful only across the same own/ally identity and side.
    if prior is not None:
        before = a.actor_index(prior[0]).get(alias)
        now = a.actor_index(current[0]).get(alias)
        if (
            before is None
            or now is None
            or any(
                before[key] != now[key] for key in ("side", "pawn_life", "pawn_handle")
            )
        ):
            prior = None
    return k.initial_context(
        {"actor_alias": alias, "ally_alias": alias},
        *current,
        prior=prior,
        witness=witness,
    )["self"]


def initial_features(
    support, objective, ownership, data, identifier, actor, ally, sequence, tick, frame
):
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    support, objective, ownership, data = prefix_inputs(
        support, objective, ownership, data, sequence
    )
    o.validate_source(objective, support)
    c4.validate_source(ownership, support, objective)
    k.validate_source(data, support, objective, ownership)
    public = o.public_prefixes(objective)
    alignments, _ = k.alignment_prefixes(support, objective, data)
    current = k.context_inputs(support, objective, ownership, data, public, sequence)
    prior = (
        k.context_inputs(support, objective, ownership, data, public, sequence - 1)
        if sequence > 1
        else None
    )
    for role, alias in (("self", actor), ("ally", ally)):
        initial[role]["objective_clock"] = copy.deepcopy(
            actor_context(alias, current, prior, alignments.get(sequence))
        )
    return initial


def iter_rows(support, objective, ownership, data):
    index = {r["source_sequence"]: r for r in data["observations"]["frames"]}
    objectives = {r["source_sequence"]: r for r in objective["observations"]["frames"]}
    owners = {
        r["source_sequence"]: ownership["observations"]["states"][
            r["state_sequence"] - 1
        ]
        for r in ownership["observations"]["cuts"]
    }
    public = o.public_prefixes(objective)
    alignments, _ = k.alignment_prefixes(support, objective, data)
    prior = None
    for cut in support["observations"]["frames"]:
        seq = cut["sequence"]
        pairs = c.select_pairs(cut)
        roster = c.roster(cut)[0]
        row = index.get(seq)
        current = cut, objectives.get(seq), public.get(seq, {}), owners.get(seq), row
        actors = []
        for alias in sorted({actor for actor, _ in pairs}):
            actors.append(
                {
                    **{
                        name: roster[alias][name]
                        for name in ("actor_alias", "side", "pawn_handle", "pawn_life")
                    },
                    "initial": actor_context(
                        alias, current, prior, alignments.get(seq)
                    ),
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_objective_clock_cut@1",
            "cut": c.cut_reference(cut),
            "provenance": {
                **{
                    name: row[name]
                    for name in (
                        "source_sequence",
                        "order",
                        "objective_order",
                        "c4_state_sequence",
                    )
                },
                "clock_witness": copy.deepcopy(alignments.get(seq)),
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
        prior = current


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


def stream_rows(support, objective, ownership, data, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, objective, ownership, data):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "objective clock C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(
                row == expected, "objective clock C0 row differs from exact source"
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
        a.require(not stream.read(1), "extra objective clock C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, k.DATA)
    data, ownership, objective, support, _ = k.load_source(source)
    return (support, objective, ownership, data), {
        "clock_source": p.file_reference(source),
        "c4_source": p.file_reference(source.parent / c4.DATA),
        "objective_source": p.file_reference(source.parent / o.DATA),
        "support_source": p.file_reference(source.parent / a.DATA),
        "preparation_protocol": c.protocol()[1],
    }


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_objective_clock_manifest@1",
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
    a.require(coverage["counts"]["candidates"] > 0, "no clock C0 candidates")
    for field in (
        "round_remaining_seconds",
        "bomb_remaining_seconds",
        "attempt_remaining_seconds",
        "attempt_progress",
        "defuse_deadline_margin_seconds",
    ):
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
        "objective clock C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "objective clock C0 manifest changed while reading",
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
