"""Prospective C0 preparation checkpoints, not physical entry or intent.

Selection sees one current own/ally cut. Execution descriptors are separate;
the optional JSONL artifact is a research census, never a DecisionExample.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
from collections import Counter, defaultdict
from copy import deepcopy
from pathlib import Path

from ai_coach.economy import candidate_io
from ai_coach.support import anchors

VERSION = "1.0.0-candidate.1"
PROTOCOL_ID = "support_preparation_census/1.0.0-candidate.1"
PROTOCOL = "support_preparation_protocol.json"
CUTS = "support_preparation_cuts.jsonl"
PAIRS = "support_preparation_pairs.jsonl"
MANIFEST = "support_preparation_manifest.json"
SCHEMAS = (
    PROTOCOL,
    "support_preparation_cut.json",
    "support_preparation_pair.json",
    MANIFEST,
)
ACTOR_REASONS = (
    "ambiguous_actor_alias",
    "shared_pawn_reference",
    "unsupported_side",
    "disconnected",
    "native_identity_unavailable",
    "alive_unavailable",
    "not_alive",
    "position_unavailable",
)
EXECUTION_STATUSES = (
    "sampled_displacement",
    "no_sampled_displacement",
    "censored:missing_exact_horizon_sample",
    "censored:horizon_gap_or_boundary",
    "censored:identity_unavailable_or_changed",
    *("censored:" + reason for reason in ACTOR_REASONS),
)
PAIR_REASONS = ("selected", "excluded_actor", "excluded_ally")
ROLES = ("self", "ally")
require = anchors.require


def numeric(value):
    return type(value) in (int, float) and math.isfinite(value)


def clock_ok(cut):
    return (
        all(
            type(cut.get(k)) is int
            for k in ("sequence", "frame", "tick", "epoch", "instance")
        )
        and min(cut["sequence"], cut["frame"]) >= 1
        and min(cut["epoch"], cut["instance"]) >= 0
        and numeric(cut.get("tick_rate"))
        and cut["tick_rate"] > 0
        and type(cut.get("sample_step")) is int
        and cut["sample_step"] >= 1
        and cut["sample_step"] == int(cut["tick_rate"] / 16)
    )


def actor_reason(player):
    if player["side"] not in ("CT", "T"):
        return "unsupported_side"
    if player["connected"] is not True:
        return "disconnected"
    handle = player["pawn_handle"]
    if (
        player["identity_status"] != "observed"
        or not player["pawn_life"]
        or type(handle) is not int
        or not 0 <= handle < 2**32
        or handle % 16384 == 16383
    ):
        return "native_identity_unavailable"
    if player["alive_status"] != "observed" or type(player["alive"]) is not bool:
        return "alive_unavailable"
    if not player["alive"]:
        return "not_alive"
    position = player["position"]
    if (
        player["position_status"] != "observed"
        or not isinstance(position, dict)
        or set(position) != {"x", "y", "z"}
        or not all(numeric(v) for v in position.values())
    ):
        return "position_unavailable"
    return None


def roster(cut):
    """Ambiguity is local to this cut; never resolve it from future frames."""
    grouped = defaultdict(list)
    for player in cut["players"]:
        grouped[player["actor_alias"]].append(player)
    index, sides, reasons = {}, {}, {}
    claims = {field: Counter() for field in ("pawn_handle", "pawn_life")}
    for players in grouped.values():
        if len(players) == 1 and players[0]["side"] in ("CT", "T"):
            player = players[0]
            for field, counts in claims.items():
                if player[field] is not None:
                    counts[player["side"], player[field]] += 1
    for alias, players in grouped.items():
        observed_sides = {player["side"] for player in players}
        sides[alias] = next(iter(observed_sides)) if len(observed_sides) == 1 else None
        player = players[0] if len(players) == 1 else None
        index[alias] = player
        if player is None:
            reasons[alias] = "ambiguous_actor_alias"
        elif player["side"] in ("CT", "T") and any(
            player[field] is not None and counts[player["side"], player[field]] > 1
            for field, counts in claims.items()
        ):
            reasons[alias] = "shared_pawn_reference"
        else:
            reasons[alias] = actor_reason(player)
    return index, sides, reasons


def select_pairs(cut):
    """C0 uses neither history nor execution; unknown fields are not predicates."""
    if cut["phase"] != "playing" or not clock_ok(cut):
        return []
    index, sides, reasons = roster(cut)
    return [
        (a, b)
        for a in sorted(index)
        for b in sorted(index)
        if a != b
        and sides[a] in ("CT", "T")
        and sides[a] == sides[b]
        and reasons[a] is None
        and reasons[b] is None
    ]


def candidate_id(source_alias, cut, actor, ally):
    key = [
        source_alias,
        PROTOCOL_ID,
        cut["epoch"],
        cut["instance"],
        cut["sequence"],
        actor,
        ally,
    ]
    return "support-preparation-" + anchors.digest(
        json.dumps(key, separators=(",", ":")).encode()
    )


def cut_reference(cut):
    return {
        key: cut[key]
        for key in (
            "sequence",
            "frame",
            "tick",
            "epoch",
            "instance",
            "tick_rate",
            "sample_step",
        )
    }


def current_features(cut, index, actor, ally):
    return {
        role: {
            "position": deepcopy(index[alias]["position"]),
            "alive": True,
            "availability_tick": cut["tick"],
            "availability_frame": cut["frame"],
        }
        for role, alias in zip(ROLES, (actor, ally))
    }


def initial_features(data, identifier, actor, ally, sequence, tick, frame):
    """Slice by delivery ordinal BEFORE validating; never read a future row.

    Exact actor/ally/T0 and the recomputed candidate ID are required. Full-file
    integrity is a separate reader concern, not an initial model feature.
    """
    require(
        type(sequence) is int
        and sequence >= 1
        and type(tick) is int
        and type(frame) is int,
        "no exact preparation cut",
    )
    prefix = data["observations"]["frames"][:sequence]
    require(len(prefix) == sequence, "no exact preparation cut")
    anchors.validate_source({**data, "observations": {"frames": prefix}})
    cut = prefix[-1]
    require(
        (cut["sequence"], cut["tick"], cut["frame"]) == (sequence, tick, frame)
        and (actor, ally) in select_pairs(cut)
        and candidate_id(data["source_alias"], cut, actor, ally) == identifier,
        "no exact preparation candidate for actor/ally/tick/frame",
    )
    return current_features(cut, roster(cut)[0], actor, ally)


def execution_descriptor(frames, rosters, start_index, alias):
    """Two-second sampled-position descriptor, including explicit censures."""
    start = frames[start_index]
    length = math.ceil(2 * start["tick_rate"])
    result = {
        "status": "censored:missing_exact_horizon_sample",
        "end_tick": start["tick"] + length,
        "availability_sequence": None,
        "availability_frame": None,
        "availability_tick": None,
        "sampled_displacement": None,
    }
    steps, remainder = divmod(length, start["sample_step"])
    if remainder or start_index + steps >= len(frames):
        return result
    original = rosters[start_index][0][alias]
    identity = tuple(original[k] for k in ("side", "pawn_handle", "pawn_life"))
    previous = start
    old_position, moved = original["position"], False
    for offset in range(1, steps + 1):
        cut = frames[start_index + offset]
        result.update(
            availability_sequence=cut["sequence"],
            availability_frame=cut["frame"],
            availability_tick=cut["tick"],
        )
        if (
            not clock_ok(cut)
            or cut["phase"] != "playing"
            or cut["frame"] <= previous["frame"]
            or cut["sequence"] != start["sequence"] + offset
            or any(
                cut[k] != start[k]
                for k in ("epoch", "instance", "tick_rate", "sample_step")
            )
            or cut["tick"] != start["tick"] + offset * start["sample_step"]
        ):
            result["status"] = "censored:horizon_gap_or_boundary"
            return result
        index, _, reasons = rosters[start_index + offset]
        player = index.get(alias)
        if (
            player is None
            or tuple(player[k] for k in ("side", "pawn_handle", "pawn_life"))
            != identity
        ):
            result["status"] = "censored:identity_unavailable_or_changed"
            return result
        if reasons[alias] is not None:
            result["status"] = "censored:" + reasons[alias]
            return result
        moved |= player["position"] != old_position
        old_position, previous = player["position"], cut
    result.update(
        status="sampled_displacement" if moved else "no_sampled_displacement",
        sampled_displacement=moved,
    )
    return result


def iter_census(data):
    """Bounded rows per cut; retain exclusions even when no pair is selected."""
    frames = data["observations"]["frames"]
    rosters = [roster(cut) for cut in frames]
    for i, cut in enumerate(frames):
        index, sides, reasons = rosters[i]
        admission = (
            "excluded_phase"
            if cut["phase"] != "playing"
            else "admissible"
            if clock_ok(cut)
            else "excluded_clock"
        )
        pairs = select_pairs(cut)
        selected_actors = {actor for actor, _ in pairs}
        actors = [
            {
                "actor_alias": alias,
                "side": sides[alias],
                "roster_rows": sum(p["actor_alias"] == alias for p in cut["players"]),
                "reason": reasons[alias],
                "has_selected_ally": alias in selected_actors,
            }
            for alias in sorted(index)
        ]
        examined = [
            (a, b)
            for a in sorted(index)
            for b in sorted(index)
            if admission == "admissible"
            and a != b
            and sides[a] in ("CT", "T")
            and sides[b] == sides[a]
        ]
        counts = Counter(
            "excluded_actor"
            if reasons[a]
            else "excluded_ally"
            if reasons[b]
            else "selected"
            for a, b in examined
        )
        cut_row = {
            "schema_id": "stratai.support_preparation_cut@1",
            "cut": cut_reference(cut),
            "phase": cut["phase"],
            "admission": admission,
            "actors": actors,
            "counts": {
                "roster_rows": len(cut["players"]),
                "actor_aliases": len(index),
                "usable_actors": sum(v is None for v in reasons.values()),
                "selected_actor_cuts": len(selected_actors),
                "pairs_examined": len(examined),
                **{key: counts[key] for key in PAIR_REASONS},
            },
        }
        execution = {
            alias: execution_descriptor(frames, rosters, i, alias)
            for alias in {p for pair in pairs for p in pair}
        }
        rows = []
        for actor, ally in pairs:
            rows.append(
                {
                    "schema_id": "stratai.support_preparation_pair@1",
                    "candidate_id": candidate_id(
                        data["source_alias"], cut, actor, ally
                    ),
                    "t0": cut_reference(cut),
                    "actor_alias": actor,
                    "ally_alias": ally,
                    "side": sides[actor],
                    "pawn_references": {
                        role: {
                            "handle": index[alias]["pawn_handle"],
                            "life": index[alias]["pawn_life"],
                        }
                        for role, alias in zip(ROLES, (actor, ally))
                    },
                    "initial": current_features(cut, index, actor, ally),
                    "execution": {
                        role: execution[alias].copy()
                        for role, alias in zip(ROLES, (actor, ally))
                    },
                    "decision_id": None,
                    "decision_eligible": False,
                    "physical_entry_status": "not_accredited",
                    "tactical_commitment_status": "not_accredited",
                }
            )
        yield cut_row, rows


def protocol():
    raw = (anchors.CONTRACTS / PROTOCOL).read_bytes()
    value = json.loads(raw)
    anchors.schema_validator(PROTOCOL).validate(value)
    require(value["protocol_id"] == PROTOCOL_ID, "preparation protocol differs")
    return value, {"path": PROTOCOL, "bytes": len(raw), "sha256": anchors.digest(raw)}


class Coverage:
    """Keep cut, actor and directed-pair denominators distinct."""

    def __init__(self):
        self.counts = Counter()
        self.actor_reasons = Counter()
        self.execution = {role: Counter() for role in ROLES}
        self.joint = Counter()
        self.matrix = defaultdict(
            lambda: {"counts": Counter(), **{role: Counter() for role in ROLES}}
        )

    def add(self, cut, rows):
        c = self.counts
        c["cuts_total"] += 1
        c["cuts_" + cut["admission"]] += 1
        c["roster_rows_total"] += cut["counts"]["roster_rows"]
        if cut["admission"] != "admissible":
            return
        c["actor_alias_cuts"] += len(cut["actors"])
        c["actor_usable"] += cut["counts"]["usable_actors"]
        c["selected_actor_cuts"] += cut["counts"]["selected_actor_cuts"]
        c["pairs_examined"] += cut["counts"]["pairs_examined"]
        for reason in PAIR_REASONS:
            c["pairs_" + reason] += cut["counts"][reason]
        for a in cut["actors"]:
            if a["reason"]:
                self.actor_reasons[a["reason"]] += 1
            for b in cut["actors"]:
                if (
                    a["actor_alias"] == b["actor_alias"]
                    or a["side"] not in ("CT", "T")
                    or a["side"] != b["side"]
                ):
                    continue
                key = (
                    cut["cut"]["instance"],
                    cut["cut"]["epoch"],
                    a["side"],
                    a["actor_alias"],
                    b["actor_alias"],
                )
                reason = (
                    "excluded_actor"
                    if a["reason"]
                    else "excluded_ally"
                    if b["reason"]
                    else "selected"
                )
                self.matrix[key]["counts"][reason] += 1
        for row in rows:
            key = (
                row["t0"]["instance"],
                row["t0"]["epoch"],
                row["side"],
                row["actor_alias"],
                row["ally_alias"],
            )
            for role in ROLES:
                status = row["execution"][role]["status"]
                self.execution[role][status] += 1
                self.matrix[key][role][status] += 1
            self.joint[
                " / ".join(row["execution"][role]["status"] for role in ROLES)
            ] += 1

    def result(self):
        c = self.counts
        names = (
            "cuts_total",
            "cuts_admissible",
            "cuts_excluded_phase",
            "cuts_excluded_clock",
            "roster_rows_total",
            "actor_alias_cuts",
            "actor_usable",
            "selected_actor_cuts",
            "pairs_examined",
            "pairs_selected",
            "pairs_excluded_actor",
            "pairs_excluded_ally",
        )
        require(
            c["cuts_total"]
            == sum(
                c["cuts_" + x]
                for x in ("admissible", "excluded_phase", "excluded_clock")
            ),
            "cut counts differ",
        )
        require(
            c["actor_alias_cuts"]
            == c["actor_usable"] + sum(self.actor_reasons.values()),
            "actor counts differ",
        )
        require(
            c["pairs_examined"] == sum(c["pairs_" + x] for x in PAIR_REASONS),
            "pair counts differ",
        )
        require(
            all(
                sum(v.values()) == c["pairs_selected"] for v in self.execution.values()
            ),
            "execution counts differ",
        )
        return {
            "counts": {name: c[name] for name in names},
            "actor_unavailable": {
                reason: self.actor_reasons[reason] for reason in ACTOR_REASONS
            },
            "execution_by_role": {
                role: {s: self.execution[role][s] for s in EXECUTION_STATUSES}
                for role in ROLES
            },
            "joint_execution": dict(sorted(self.joint.items())),
            "matrix": [
                {
                    "instance": key[0],
                    "epoch": key[1],
                    "side": key[2],
                    "actor_alias": key[3],
                    "ally_alias": key[4],
                    "counts": {
                        reason: value["counts"][reason] for reason in PAIR_REASONS
                    },
                    "execution_counts": {
                        role: {s: value[role][s] for s in EXECUTION_STATUSES}
                        for role in ROLES
                    },
                }
                for key, value in sorted(self.matrix.items())
            ],
        }


def encoded_row(row):
    return (
        json.dumps(row, ensure_ascii=False, separators=(",", ":"), allow_nan=False)
        + "\n"
    ).encode()


def materialize(source: Path, output: Path, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    data, source_ref, _ = anchors.load_source(source)
    _, protocol_ref = protocol()
    validators = [anchors.schema_validator(name) for name in SCHEMAS[1:3]]
    coverage = Coverage()
    hashes, sizes, lines = [hashlib.sha256(), hashlib.sha256()], [0, 0], [0, 0]
    with (
        (output / CUTS).open("xb") as cut_file,
        (output / PAIRS).open("xb") as pair_file,
    ):
        for cut, pairs in iter_census(data):
            coverage.add(cut, pairs)
            for i, rows in enumerate(([cut], pairs)):
                for row in rows:
                    validators[i].validate(row)
                    raw = encoded_row(row)
                    (cut_file if i == 0 else pair_file).write(raw)
                    hashes[i].update(raw)
                    sizes[i] += len(raw)
                    lines[i] += 1
    if require_observed:
        require(lines[1] > 0, "preparation smoke has no selected pairs")
    result = {
        key: data[key]
        for key in (
            "source_alias",
            "source_sha256",
            "build_identifier",
            "comparison_allowed",
            "training_allowed",
            "product_supported",
        )
    }
    result.update(
        schema_id="stratai.support_preparation_manifest@1",
        contract_version=VERSION,
        protocol=protocol_ref,
        source_observations=source_ref,
        artifacts=[
            {
                "path": name,
                "bytes": sizes[i],
                "sha256": hashes[i].hexdigest(),
                "records": lines[i],
            }
            for i, name in enumerate((CUTS, PAIRS))
        ],
        coverage=coverage.result(),
        decision_id=None,
        eligible_decisions=0,
        physical_entries_accredited=0,
        tactical_commitments_accredited=0,
    )
    anchors.schema_validator(MANIFEST).validate(result)
    with (output / PROTOCOL).open("xb") as handle:
        handle.write((anchors.CONTRACTS / PROTOCOL).read_bytes())
    candidate_io._write_new(output / MANIFEST, result)
    return result


def validate_artifact(source: Path, output: Path):
    """Stream, hash and recompute every row, including exclusions and coverage."""
    safe = candidate_io.source_io._safe_file
    output = output.absolute()
    manifest_raw = safe(output, MANIFEST).read_bytes()
    manifest = json.loads(manifest_raw)
    anchors.schema_validator(MANIFEST).validate(manifest)
    data, ref, _ = anchors.load_source(source)
    require(manifest["source_observations"] == ref, "preparation source hash differs")
    for key in ("source_alias", "source_sha256", "build_identifier"):
        require(manifest[key] == data[key], "preparation source identity differs")
    proto_raw = safe(output, PROTOCOL).read_bytes()
    require(
        manifest["protocol"]
        == {
            "path": PROTOCOL,
            "bytes": len(proto_raw),
            "sha256": anchors.digest(proto_raw),
        }
        and json.loads(proto_raw) == protocol()[0],
        "preparation protocol differs",
    )
    validators = [anchors.schema_validator(name) for name in SCHEMAS[1:3]]
    coverage = Coverage()
    hashes, sizes, lines = [hashlib.sha256(), hashlib.sha256()], [0, 0], [0, 0]
    with (
        safe(output, CUTS).open("rb") as cut_file,
        safe(output, PAIRS).open("rb") as pair_file,
    ):
        for cut, pairs in iter_census(data):
            coverage.add(cut, pairs)
            for i, rows in enumerate(([cut], pairs)):
                for expected in rows:
                    raw = (cut_file if i == 0 else pair_file).readline(1024 * 1024 + 1)
                    require(
                        bool(raw) and len(raw) <= 1024 * 1024 and raw.endswith(b"\n"),
                        "preparation row missing or oversized",
                    )
                    row = json.loads(raw)
                    validators[i].validate(row)
                    require(row == expected, "preparation row differs from source")
                    hashes[i].update(raw)
                    sizes[i] += len(raw)
                    lines[i] += 1
        require(
            not cut_file.read(1) and not pair_file.read(1), "extra preparation rows"
        )
    expected_refs = [
        {
            "path": name,
            "bytes": sizes[i],
            "sha256": hashes[i].hexdigest(),
            "records": lines[i],
        }
        for i, name in enumerate((CUTS, PAIRS))
    ]
    require(
        manifest["artifacts"] == expected_refs,
        "preparation artifact hash/count differs",
    )
    require(manifest["coverage"] == coverage.result(), "preparation coverage differs")
    require(
        safe(output, MANIFEST).read_bytes() == manifest_raw,
        "preparation manifest changed while reading",
    )
    return manifest


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
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
        require(
            result["coverage"]["counts"]["pairs_selected"] > 0,
            "preparation smoke has no selected pairs",
        )
    print(
        json.dumps(
            {
                "ok": True,
                "counts": result["coverage"]["counts"],
                "artifacts": result["artifacts"],
            }
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
