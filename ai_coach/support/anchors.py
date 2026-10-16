"""Causal onsets of increasing sampled ally separation, independent of deaths."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
from collections import Counter, defaultdict
from itertools import pairwise
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts.accelerated_support_schema import SCHEMAS, compiled_validator
from ai_coach.datasets.streaming import load_direct_bundle_manifest
from ai_coach.economy import candidate_io

DATA = "support_observations.json"
REPORT = "support_anchors.json"
PROTOCOL = "support_anchor_protocol.json"
CONTRACTS = Path(__file__).parents[1] / "contracts"
MISSING = [
    "physical_entry",
    "tactical_commitment",
    "navigation_path",
    "line_of_sight",
    "legal_contact",
    "objective_context",
    "cooperation",
    "intent",
    "orientation",
    "weapon_readiness",
]
REASONS = [
    "actor_unavailable",
    "no_live_ally",
    "insufficient_history",
    "incompatible_sample_window",
    "pair_history_unavailable",
    "pair_life_or_side_changed",
    "no_increase_onset",
    "observed_onset",
]


def require(ok, message):
    if not ok:
        raise ValueError(message)


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def schema_validator(name):
    raw = (CONTRACTS / name.replace(".json", ".schema.json")).read_bytes()
    if name in SCHEMAS:
        return compiled_validator(raw)
    schema = json.loads(raw)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def protocol():
    raw = (CONTRACTS / PROTOCOL).read_bytes()
    data = json.loads(raw)
    require(
        not list(schema_validator(PROTOCOL).iter_errors(data)),
        "invalid support protocol",
    )
    return data, {"path": PROTOCOL, "bytes": len(raw), "sha256": digest(raw)}


def validate_source(data):
    require(
        not list(schema_validator(DATA).iter_errors(data)),
        "invalid support source schema",
    )
    frames = data["observations"]["frames"]
    previous = None
    lives = {}
    for i, f in enumerate(frames, 1):
        require(f["sequence"] == i, "support sequence differs")
        if previous:
            require(f["frame"] > previous["frame"], "support frame order differs")
            require(
                f["instance"] >= previous["instance"]
                and f["epoch"] >= previous["epoch"],
                "support instance or epoch went backwards",
            )
            require(
                f["tick"] >= previous["tick"] or f["epoch"] > previous["epoch"],
                "support tick reset lacks epoch",
            )
        for p in f["players"]:
            require(
                (p["position"] is not None) == (p["position_status"] == "observed"),
                "support position presence differs",
            )
            require(
                (p["alive"] is not None) == (p["alive_status"] == "observed"),
                "support alive presence differs",
            )
            if p["identity_status"] == "observed":
                require(
                    p["pawn_handle"] is not None and p["pawn_life"] is not None,
                    "observed support identity lacks handle/life",
                )
                require(
                    p["pawn_handle"] % 16384 != 16383, "support identity uses sentinel"
                )
                old = lives.setdefault(p["pawn_life"], p["pawn_handle"])
                require(old == p["pawn_handle"], "support life handle changed")
            else:
                require(
                    p["pawn_life"] is None
                    and p["position"] is None
                    and p["alive"] is None,
                    "unavailable support identity acquired state",
                )
            if p["position"] is not None:
                require(
                    all(math.isfinite(v) for v in p["position"].values()),
                    "nonfinite support position",
                )
        previous = f


def actor_index(frame):
    grouped = defaultdict(list)
    claims = Counter(
        p["pawn_life"] for p in frame["players"] if p["pawn_life"] is not None
    )
    for p in frame["players"]:
        grouped[p["actor_alias"]].append(p)
    return {
        a: values[0]
        if len(values) == 1
        and (values[0]["pawn_life"] is None or claims[values[0]["pawn_life"]] == 1)
        else None
        for a, values in grouped.items()
    }


def usable(p, *, alive=True):
    return bool(
        p
        and p["side"] in ("CT", "T")
        and p["connected"]
        and p["identity_status"] == "observed"
        and p["position"] is not None
        and p["alive"] is not None
        and (p["alive"] or not alive)
    )


def compatible(frames):
    first = frames[0]
    return all(
        f["instance"] == first["instance"]
        and f["epoch"] == first["epoch"]
        and f["phase"] == "playing"
        and f["tick_rate"] is not None
        and f["tick_rate"] == first["tick_rate"]
        and f["sample_step"] == first["sample_step"]
        and f["sample_step"] == int(f["tick_rate"] / 16)
        for f in frames
    ) and all(
        b["sequence"] == a["sequence"] + 1
        and b["tick"] - a["tick"] == first["sample_step"]
        for a, b in pairwise(frames)
    )


def distance_squared(a, b):
    return sum((a[axis] - b[axis]) ** 2 for axis in ("x", "y", "z"))


def select_anchors(frames, source_alias):
    """Only the current and two preceding snapshots participate in selection."""
    anchors = []
    matrix = defaultdict(Counter)
    indexes = []
    for i, frame in enumerate(frames):
        index = actor_index(frame)
        indexes.append(index)
        if frame["phase"] != "playing":
            continue
        for actor in sorted(index):
            p = index[actor]
            counts = matrix[frame["instance"], actor]
            counts["sampled_frames"] += 1
            if not usable(p):
                counts["actor_unavailable"] += 1
                continue
            allies = [
                a
                for a, q in sorted(index.items())
                if a != actor and usable(q) and q["side"] == p["side"]
            ]
            if not allies:
                counts["no_live_ally"] += 1
            for ally in allies:
                counts["pairs_examined"] += 1
                if i < 2:
                    counts["insufficient_history"] += 1
                    continue
                window = frames[i - 2 : i + 1]
                if not compatible(window):
                    counts["incompatible_sample_window"] += 1
                    continue
                pairs = [(x.get(actor), x.get(ally)) for x in indexes[i - 2 : i + 1]]
                if not all(usable(x) for pair in pairs for x in pair):
                    counts["pair_history_unavailable"] += 1
                    continue
                identities = [
                    (
                        a["pawn_life"],
                        a["pawn_handle"],
                        a["side"],
                        b["pawn_life"],
                        b["pawn_handle"],
                        b["side"],
                    )
                    for a, b in pairs
                ]
                if len(set(identities)) != 1 or any(
                    a["side"] != b["side"] for a, b in pairs
                ):
                    counts["pair_life_or_side_changed"] += 1
                    continue
                d2 = [distance_squared(a["position"], b["position"]) for a, b in pairs]
                if not (d2[1] <= d2[0] and d2[2] > d2[1]):
                    counts["no_increase_onset"] += 1
                    continue
                counts["observed_onset"] += 1
                anchor_id = f"{source_alias}:support:{frame['instance']:05d}:{frame['sequence']:07d}:{actor}:{ally}"
                history = [
                    {
                        "source_sequence": f["sequence"],
                        "frame": f["frame"],
                        "tick": f["tick"],
                        "self_position": a["position"].copy(),
                        "ally_position": b["position"].copy(),
                        "self_alive": a["alive"],
                        "ally_alive": b["alive"],
                    }
                    for f, (a, b) in zip(window, pairs)
                ]
                anchors.append(
                    {
                        "anchor_id": anchor_id,
                        "instance": frame["instance"],
                        "round_number": frame["round_number"],
                        "actor_alias": actor,
                        "ally_alias": ally,
                        "side": p["side"],
                        "actor_life": pairs[-1][0]["pawn_life"],
                        "ally_life": pairs[-1][1]["pawn_life"],
                        "t0_tick": frame["tick"],
                        "t0_frame": frame["frame"],
                        "source_sequence": frame["sequence"],
                        "epoch": frame["epoch"],
                        "kind": "sampled_separation_increase_onset",
                        "decision_id": None,
                        "observed": {
                            "history": history,
                            "distance_squared_world_units": d2,
                            "availability_tick": frame["tick"],
                            "availability_frame": frame["frame"],
                        },
                        "missing_context": MISSING.copy(),
                        "physical_entry_status": "not_accredited",
                        "decision_eligible": False,
                    }
                )
    rows = [
        {
            "instance": instance,
            "actor_alias": actor,
            "sampled_frames": c["sampled_frames"],
            "pairs_examined": c["pairs_examined"],
            "reason_counts": {r: c[r] for r in REASONS},
        }
        for (instance, actor), c in sorted(matrix.items())
    ]
    return anchors, rows


def horizon(anchor, frames, indexes):
    start = anchor["source_sequence"] - 1
    first = frames[start]
    end_tick = first["tick"] + math.ceil(first["tick_rate"] * 2)
    result = {
        "status": "unavailable",
        "reason": "missing_exact_horizon_sample",
        "end_tick": end_tick,
        "availability_sequence": None,
        "actor_alive_at_end": None,
        "ally_alive_at_end": None,
        "not_alive_sample_observed": None,
    }
    prior = first
    any_dead = False
    for i in range(start + 1, len(frames)):
        f = frames[i]
        if not compatible([prior, f]):
            result["reason"] = "horizon_gap_or_boundary"
            return result
        if f["tick"] > end_tick:
            return result
        a, b = (
            indexes[i].get(anchor["actor_alias"]),
            indexes[i].get(anchor["ally_alias"]),
        )
        if not all(usable(p, alive=False) for p in (a, b)) or (
            a["pawn_life"],
            b["pawn_life"],
            a["side"],
            b["side"],
        ) != (
            anchor["actor_life"],
            anchor["ally_life"],
            anchor["side"],
            anchor["side"],
        ):
            result["reason"] = "horizon_pair_unavailable_or_changed"
            return result
        any_dead |= not a["alive"] or not b["alive"]
        if f["tick"] == end_tick:
            result.update(
                status="observed",
                reason=None,
                availability_sequence=f["sequence"],
                actor_alive_at_end=a["alive"],
                ally_alive_at_end=b["alive"],
                not_alive_sample_observed=any_dead,
            )
            return result
        prior = f
    return result


def initial_features(data, anchor_id, actor_alias, tick, frame):
    """Recompute the requested anchor from a delivery prefix; no future outcomes."""
    prefix = [f for f in data["observations"]["frames"] if f["frame"] <= frame]
    validate_source({**data, "observations": {"frames": prefix}})
    anchors, _ = select_anchors(prefix, data["source_alias"])
    found = [
        a
        for a in anchors
        if a["anchor_id"] == anchor_id
        and a["actor_alias"] == actor_alias
        and a["t0_tick"] == tick
        and a["t0_frame"] == frame
    ]
    require(len(found) == 1, "no exact causal support anchor for actor/tick/frame")
    # References, identities, missing tactical context and future labels stay out.
    return found[0]["observed"]


def build_report(data, ref, *, require_observed=False):
    validate_source(data)
    _, proto_ref = protocol()
    frames = data["observations"]["frames"]
    anchors, matrix = select_anchors(frames, data["source_alias"])
    indexes = [actor_index(f) for f in frames]
    for a in anchors:
        a["later_observation"] = horizon(a, frames, indexes)
    if require_observed:
        require(bool(anchors), "support smoke has no affected anchors")
    outcomes = Counter(a["later_observation"]["reason"] or "observed" for a in anchors)
    report = {
        k: data[k]
        for k in (
            "contract_version",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "comparison_allowed",
            "training_allowed",
            "product_supported",
        )
    }
    report.update(
        schema_id="stratai.support_anchors@1",
        protocol=proto_ref,
        source_observations=ref,
        anchors=anchors,
        matrix=matrix,
        summary={
            "source_frames": len(frames),
            "playing_frames": sum(f["phase"] == "playing" for f in frames),
            "anchors": len(anchors),
            "actor_instances": len(matrix),
            "pairs_examined": sum(r["pairs_examined"] for r in matrix),
            "reason_counts": {
                k: sum(r["reason_counts"][k] for r in matrix) for k in REASONS
            },
            "later_observation_counts": dict(sorted(outcomes.items())),
            "complete_horizons_without_not_alive_samples": sum(
                a["later_observation"]["not_alive_sample_observed"] is False
                for a in anchors
            ),
            "physical_entries_accredited": 0,
            "eligible_decisions": 0,
        },
    )
    require(
        not list(schema_validator(REPORT).iter_errors(report)),
        "invalid support report schema",
    )
    return report


def load_source(path):
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 256 * 1024 * 1024,
        "invalid support source path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    validate_source(data)
    alias, checksum = data["source_alias"], data["source_sha256"]
    expected = f"match_{alias}/canonical/manifest.json"
    require(data["base_manifest"]["path"] == expected, "support base path differs")
    base = load_direct_bundle_manifest(
        path.parent / f"match_{alias}",
        expected_match_id=alias,
        expected_demo_checksum=checksum,
    )
    base_path = path.parent / expected
    base_raw = base_path.read_bytes()
    require(
        data["base_manifest"]
        == {"path": expected, "bytes": len(base_raw), "sha256": digest(base_raw)},
        "support base hash differs",
    )
    require(path.read_bytes() == raw, "support source changed while loading")
    return data, {"path": DATA, "bytes": len(raw), "sha256": digest(raw)}, base


def validate_report(report, data, ref):
    require(report == build_report(data, ref), "support report differs from source")


def validate_replay_overlap(data, base):
    """Cross-check source fields against existing replay without using outcomes."""
    frames = defaultdict(list)
    for f in data["observations"]["frames"]:
        frames[f["round_number"], f["tick"]].append(f)
    checked = ambiguous = 0
    seen = set()
    for artifact in base.many("replay_round"):
        with gzip.open(artifact.physical_path, "rb") as stream:
            payload = stream.read(128 * 1024 * 1024 + 1)
        require(len(payload) <= 128 * 1024 * 1024, "support replay bound exceeded")
        replay = json.loads(payload)["round"]
        for f in replay["frames"]:
            key = replay["round"], f["tick"]
            candidates = frames.get(key, [])
            if len(candidates) != 1:
                ambiguous += 1
                continue
            source = candidates[0]
            index = actor_index(source)
            for p in f["players"]:
                actor = (
                    "support-actor-"
                    + digest(
                        (data["source_sha256"] + "\nsteam:" + p["steam_id"]).encode()
                    )[:20]
                )
                row = index.get(actor)
                require(
                    row is not None and row["side"] == p["team"],
                    "support replay actor mismatch",
                )
                if row["position"] is not None:
                    require(
                        row["position"] == {k: p[k] for k in ("x", "y", "z")},
                        "support replay position/alive mismatch",
                    )
                    checked += 1
                if row["alive"] is not None:
                    require(
                        row["alive"] == p["alive"],
                        "support replay position/alive mismatch",
                    )
            seen.add(source["sequence"])
    return {
        "player_samples_checked": checked,
        "source_frames_matched": len(seen),
        "source_frames_without_unique_replay_match": len(data["observations"]["frames"])
        - len(seen),
        "ambiguous_or_missing_replay_frames": ambiguous,
    }


def materialize(source, output, *, require_observed=False):
    output = candidate_io.validate_output_directory(output)
    data, ref, base = load_source(source)
    overlap = validate_replay_overlap(data, base)
    report = build_report(data, ref, require_observed=require_observed)
    candidate_io._write_new(output / REPORT, report)
    return {"summary": report["summary"], "replay_validation": overlap}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    print(
        json.dumps(
            materialize(
                args.source, args.output, require_observed=args.require_observed
            )
        )
    )


if __name__ == "__main__":
    main()
