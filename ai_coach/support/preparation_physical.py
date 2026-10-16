"""Optional physical facts at exact C0 cuts; no new selection or eligibility."""

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

VERSION = "1.0.0-candidate.1"
ROWS = "support_preparation_physical_cuts.jsonl"
ROW_SCHEMA = "support_preparation_physical_cut.json"
MANIFEST = "support_preparation_physical_manifest.json"
COMPONENTS = (*p.FIELDS, "interval_average_velocity")
MOVEMENT_REASONS = (
    "no_previous_sample",
    "previous_cut_incompatible",
    "previous_actor_unavailable",
    "pawn_life_handle_or_side_changed",
)


def interval_movement(previous, current, alias):
    """Only the immediately preceding delivered sample can supply an interval."""
    reason = None
    if previous is None:
        reason = "no_previous_sample"
    elif not c.clock_ok(previous) or not a.compatible([previous, current]):
        reason = "previous_cut_incompatible"
    else:
        old, _, reasons = c.roster(previous)
        now = c.roster(current)[0][alias]
        if alias not in old or reasons[alias] is not None:
            reason = "previous_actor_unavailable"
        elif any(old[alias][k] != now[k] for k in ("pawn_life", "pawn_handle", "side")):
            reason = "pawn_life_handle_or_side_changed"
    if reason:
        return {"status": "unavailable", "reason": reason, "value": None}
    history = [
        {"tick": cut["tick"], "self_position": c.roster(cut)[0][alias]["position"]}
        for cut in (previous, current)
    ]
    value = p.movement(history, "self", current["tick_rate"])
    value.pop("status")
    return {"status": "observed", "reason": None, "value": value}


def actor_context(previous, cut, alias, index):
    ctx = index.get(cut["sequence"], {}).get(alias)
    reason = (
        "missing_exact_physical_sample"
        if cut["sequence"] not in index
        else "actor_physical_context_unavailable"
    )
    return {
        "availability_tick": cut["tick"],
        "availability_frame": cut["frame"],
        "facts": copy.deepcopy(ctx["facts"])
        if ctx is not None
        else {
            field: {"status": "unavailable", "value": None, "reason": reason}
            for field in p.FIELDS
        },
        "interval_average_velocity": interval_movement(previous, cut, alias),
    }


def initial_features(support, physical, identifier, actor, ally, sequence, tick, frame):
    """Validate only the delivered prefix, then return detached own/ally facts.

    The physical stream is ordered but may omit samples. Stop at the exact
    row (or first later locator); never scan or validate the remaining future.
    """
    initial = c.initial_features(
        support, identifier, actor, ally, sequence, tick, frame
    )
    frames = support["observations"]["frames"][:sequence]
    prefix = {**support, "observations": {"frames": frames}}
    rows = []
    for row in physical["observations"]["frames"]:
        if row["source_sequence"] > sequence:
            break
        rows.append(row)
        if row["source_sequence"] == sequence:
            break
    physical_prefix = {**physical, "observations": {"frames": rows}}
    p.validate_source(physical_prefix, prefix)
    index = p.physical_index(physical_prefix)
    for role, alias in zip(c.ROLES, (actor, ally)):
        initial[role].update(
            actor_context(
                frames[-2] if len(frames) > 1 else None, frames[-1], alias, index
            )
        )
    return initial


def iter_rows(support, physical):
    """One normalized row per source cut, with every directed C0 candidate ID."""
    index = p.physical_index(physical)
    previous = None
    for cut in support["observations"]["frames"]:
        pairs = c.select_pairs(cut)
        roster = c.roster(cut)[0]
        actors = []
        for alias in sorted({actor for actor, _ in pairs}):
            player = roster[alias]
            actors.append(
                {
                    "actor_alias": alias,
                    "side": player["side"],
                    "pawn_handle": player["pawn_handle"],
                    "pawn_life": player["pawn_life"],
                    "physical_source_sequence": cut["sequence"]
                    if cut["sequence"] in index
                    else None,
                    "native_context": copy.deepcopy(
                        index.get(cut["sequence"], {}).get(alias)
                    ),
                    "initial": actor_context(previous, cut, alias, index),
                }
            )
        yield {
            "schema_id": "stratai.support_preparation_physical_cut@1",
            "cut": c.cut_reference(cut),
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
        previous = cut


def component_states(initial):
    return {
        **{
            field: fact["status"] + ":" + (fact["reason"] or "present")
            for field, fact in initial["facts"].items()
        },
        "interval_average_velocity": initial["interval_average_velocity"]["status"]
        + ":"
        + (initial["interval_average_velocity"]["reason"] or "present"),
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
        states = {alias: component_states(v["initial"]) for alias, v in actors.items()}
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
                    "both_available": Counter({k: 0 for k in COMPONENTS}),
                    "self": {k: Counter() for k in COMPONENTS},
                    "ally": {k: Counter() for k in COMPONENTS},
                }
            cell = self.matrix[key]
            cell["candidates"] += 1
            for field in COMPONENTS:
                values = [states[alias][field] for alias in (actor, ally)]
                cell["both_available"][field] += all(
                    not v.startswith("unavailable:") for v in values
                )
                for role, value in zip(c.ROLES, values):
                    cell[role][field][value] += 1

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


def stream_rows(support, physical, stream, *, reading):
    validator = a.schema_validator(ROW_SCHEMA)
    checksum, size, count = hashlib.sha256(), 0, 0
    coverage = Coverage()
    for expected in iter_rows(support, physical):
        if reading:
            raw = stream.readline(1024 * 1024 + 1)
            a.require(
                bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                "physical C0 row missing or oversized",
            )
            row = json.loads(raw)
            a.require(row == expected, "physical C0 row differs from exact source")
        else:
            row = expected
            raw = c.encoded_row(row)
        validator.validate(row)
        if not reading:
            stream.write(raw)
        coverage.add(row)
        checksum.update(raw)
        size += len(raw)
        count += 1
    if reading:
        a.require(not stream.read(1), "extra physical C0 rows")
    return {
        "path": ROWS,
        "bytes": size,
        "records": count,
        "sha256": checksum.hexdigest(),
    }, coverage.result()


def load_inputs(source):
    # Safe, immutable paths and complete canonical/source provenance are checked
    # separately from the initial prefix API.
    safe = candidate_io.source_io._safe_file
    source = safe(source.absolute().parent, p.DATA)
    physical, support, physical_ref, support_ref, _ = p.load_source(source)
    return (
        physical,
        support,
        {
            "physical_source": physical_ref,
            "support_source": support_ref,
            "preparation_protocol": c.protocol()[1],
        },
    )


def manifest(support, refs, artifact, coverage):
    result = {
        k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
    }
    result.update(
        schema_id="stratai.support_preparation_physical_manifest@1",
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


def materialize(source: Path, output: Path, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    physical, support, refs = load_inputs(source)
    with (output / ROWS).open("xb") as stream:
        artifact, coverage = stream_rows(support, physical, stream, reading=False)
    if require_observed:
        a.require(coverage["counts"]["candidates"] > 0, "no physical C0 candidates")
        a.require(
            any(cell["both_available"]["view_angles"] for cell in coverage["matrix"]),
            "no observed physical C0 facts",
        )
    result = manifest(support, refs, artifact, coverage)
    candidate_io._write_new(output / MANIFEST, result)
    return result


def validate_artifact(source: Path, output: Path):
    safe = candidate_io.source_io._safe_file
    output = output.absolute()
    raw_manifest = safe(output, MANIFEST).read_bytes()
    result = json.loads(raw_manifest)
    a.schema_validator(MANIFEST).validate(result)
    physical, support, refs = load_inputs(source)
    with safe(output, ROWS).open("rb") as stream:
        artifact, coverage = stream_rows(support, physical, stream, reading=True)
    a.require(
        result == manifest(support, refs, artifact, coverage),
        "physical C0 manifest differs",
    )
    a.require(
        safe(output, MANIFEST).read_bytes() == raw_manifest,
        "physical C0 manifest changed while reading",
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
        a.require(
            result["coverage"]["counts"]["candidates"] > 0, "no physical C0 candidates"
        )
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
