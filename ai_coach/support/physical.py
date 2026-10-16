"""Exact self/ally physical context; candidate research data, no tactical score."""

import argparse
import copy
import json
import math
from collections import Counter, defaultdict
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a

DATA = "support_physical_observations.json"
REPORT = "support_physical_context.json"
FIELDS = (
    "view_angles",
    "walking",
    "ducking",
    "active_weapon",
    "ammo_in_magazine",
    "ammo_reserve",
    "reloading",
)
MISSING = [
    "physical_entry",
    "tactical_commitment",
    "navigation_path",
    "line_of_sight",
    "legal_contact",
    "objective_context",
    "cooperation",
    "intent",
    "instantaneous_velocity",
    "full_weapon_readiness",
]


def validate_source(data, support):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(data[key] == support[key], "physical source identity mismatch")
    frames = support["observations"]["frames"]
    prior = 0
    lives = {}
    for frame in data["observations"]["frames"]:
        seq = frame["source_sequence"]
        a.require(prior < seq <= len(frames), "physical source sequence mismatch")
        prior = seq
        index = a.actor_index(frames[seq - 1])
        aliases = [p["actor_alias"] for p in frame["players"]]
        a.require(len(aliases) == len(set(aliases)), "duplicate physical actor")
        for p in frame["players"]:
            base = index.get(p["actor_alias"])
            a.require(base is not None, "physical actor not in support sample")
            ctx = p["context"]
            if ctx is None:
                continue
            a.require(a.usable(base), "physical context for unavailable actor")
            valid_identity = ctx["weapon_identity_status"] == "observed"
            if valid_identity:
                handle, life = ctx["weapon_handle"], ctx["weapon_life"]
                a.require(
                    handle is not None and handle & 16383 != 16383 and life is not None,
                    "physical weapon identity absent",
                )
                a.require(
                    lives.setdefault(life, handle) == handle,
                    "physical weapon lifetime changed handle",
                )
            else:
                a.require(
                    ctx["weapon_life"] is None,
                    "unresolved physical weapon has lifetime",
                )
                a.require(
                    all(
                        ctx[k] is None
                        for k in (
                            "raw_clip",
                            "raw_reserve",
                            "reserve_source",
                            "reload_signal",
                            "reload_signal_tick",
                        )
                    ),
                    "unresolved physical weapon has native facts",
                )
            for field, fact in ctx["facts"].items():
                observed = fact["status"] == "observed"
                a.require(
                    (fact["value"] is not None) == observed,
                    "physical fact presence mismatch",
                )
                a.require(
                    (fact["reason"] is None) == observed,
                    "physical fact reason mismatch",
                )
                if fact["status"] == "not_applicable":
                    a.require(
                        fact["reason"] == "non_ammunition_weapon",
                        "physical applicability reason mismatch",
                    )
                if field in FIELDS[3:]:
                    a.require(
                        not observed or valid_identity,
                        "physical fact without weapon identity",
                    )
                    a.require(
                        valid_identity or fact["status"] == "unavailable",
                        "unresolved weapon marked applicable",
                    )
                if field == "view_angles" and observed:
                    a.require(
                        all(math.isfinite(v) for v in fact["value"]),
                        "nonfinite physical angle",
                    )
            f = ctx["facts"]
            if f["ammo_in_magazine"]["status"] == "observed":
                a.require(
                    ctx["raw_clip"] is not None
                    and ctx["raw_clip"] > 0
                    and f["ammo_in_magazine"]["value"] == ctx["raw_clip"] - 1,
                    "physical magazine encoding mismatch",
                )
            if f["ammo_reserve"]["status"] == "observed":
                a.require(
                    ctx["reserve_source"] is not None
                    and ctx["raw_reserve"] is not None
                    and ctx["raw_reserve"] >= 0
                    and f["ammo_reserve"]["value"] == ctx["raw_reserve"],
                    "physical reserve mismatch",
                )
            a.require(
                (ctx["reload_signal"] is not None)
                == (f["reloading"]["status"] == "observed"),
                "physical reload signal mismatch",
            )
            a.require(
                (ctx["reload_signal_tick"] is not None)
                == (f["reloading"]["status"] == "observed"),
                "physical reload signal tick mismatch",
            )
            if ctx["reload_signal_tick"] is not None:
                a.require(
                    ctx["reload_signal_tick"] <= frames[seq - 1]["tick"],
                    "future physical reload signal",
                )
            nonammo = [f[k]["status"] == "not_applicable" for k in FIELDS[4:]]
            a.require(
                not any(nonammo)
                or (all(nonammo) and f["active_weapon"]["status"] == "observed"),
                "physical applicability mismatch",
            )
            if any(f[k]["status"] != "unavailable" for k in FIELDS[4:]):
                a.require(
                    f["active_weapon"]["status"] == "observed",
                    "physical ammo without active weapon",
                )


def physical_index(data):
    return {
        f["source_sequence"]: {p["actor_alias"]: p["context"] for p in f["players"]}
        for f in data["observations"]["frames"]
    }


def movement(history, role, rate):
    before, current = history[-2:]
    key = "self_position" if role == "self" else "ally_position"
    seconds = (current["tick"] - before["tick"]) / rate
    velocity = [(current[key][k] - before[key][k]) / seconds for k in ("x", "y", "z")]
    return {
        "status": "observed",
        "method": "quantized_position_interval_average@1",
        "start_tick": before["tick"],
        "end_tick": current["tick"],
        "seconds": seconds,
        "velocity_world_units_per_second": velocity,
        "horizontal_speed_world_units_per_second": math.hypot(*velocity[:2]),
    }


def initial_context(anchor, frame, index):
    pair = index.get(anchor["source_sequence"], {})
    result = {
        "availability_tick": anchor["t0_tick"],
        "availability_frame": anchor["t0_frame"],
    }
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        ctx = pair.get(alias)
        why = (
            "missing_exact_physical_sample"
            if anchor["source_sequence"] not in index
            else "actor_physical_context_unavailable"
        )
        facts = (
            ctx["facts"]
            if ctx is not None
            else {
                k: {"status": "unavailable", "value": None, "reason": why}
                for k in FIELDS
            }
        )
        result[role] = {
            "movement": movement(
                anchor["observed"]["history"], role, frame["tick_rate"]
            ),
            "facts": copy.deepcopy(facts),
        }
    return result


def initial_features(support, physical, anchor_id, actor, tick, frame):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    prefix = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:seq]},
    }
    physical_prefix = {
        **physical,
        "observations": {
            "frames": [
                f
                for f in physical["observations"]["frames"]
                if f["source_sequence"] <= seq
            ]
        },
    }
    validate_source(physical_prefix, prefix)
    candidates, _ = a.select_anchors(
        prefix["observations"]["frames"], support["source_alias"]
    )
    anchor = next(x for x in candidates if x["anchor_id"] == anchor_id)
    return initial_context(
        anchor, prefix["observations"]["frames"][-1], physical_index(physical_prefix)
    )


def build_report(support, physical, anchor_report, references, rounds):
    validate_source(physical, support)
    # Reconstruct v1 selection; later observations never select context rows.
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        selected
        == [
            {k: v for k, v in row.items() if k != "later_observation"}
            for row in anchor_report["anchors"]
        ],
        "physical v1 anchor mismatch",
    )
    index = physical_index(physical)
    rows = []
    matrix = defaultdict(Counter)
    statuses = {role: {k: Counter() for k in FIELDS} for role in ("self", "ally")}
    for anchor in selected:
        frame = support["observations"]["frames"][anchor["source_sequence"] - 1]
        initial = initial_context(anchor, frame, index)
        joined = [
            r
            for r in rounds
            if r["round_number"] == anchor["round_number"]
            and r["start_tick"]
            <= anchor["observed"]["history"][0]["tick"]
            <= anchor["t0_tick"]
            <= r["end_tick"]
        ]
        competitive = (
            "canonical_interval_matched" if len(joined) == 1 else "not_accredited"
        )
        joint = {
            k: all(
                initial[role]["facts"][k]["status"] in ("observed", "not_applicable")
                for role in ("self", "ally")
            )
            for k in FIELDS
        }
        c = matrix[anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]]
        c["anchors"] += 1
        c["canonical_interval_matched"] += competitive == "canonical_interval_matched"
        c["all_components_available"] += all(joint.values())
        for field in FIELDS:
            c[field] += joint[field]
            for role in statuses:
                f = initial[role]["facts"][field]
                statuses[role][field][
                    f["status"] + (":" + f["reason"] if f["reason"] else "")
                ] += 1
        rows.append(
            {
                **{
                    k: anchor[k]
                    for k in (
                        "anchor_id",
                        "actor_alias",
                        "ally_alias",
                        "instance",
                        "round_number",
                        "source_sequence",
                        "epoch",
                        "t0_tick",
                        "t0_frame",
                    )
                },
                "competitive_round_status": competitive,
                "initial": initial,
                "joint_components_available": joint,
                "all_components_available": all(joint.values()),
                "missing_context": MISSING.copy(),
                "decision_eligible": False,
            }
        )
    matrix_rows = [
        {
            "instance": i,
            "actor_alias": actor,
            "ally_alias": ally,
            "anchors": c["anchors"],
            "canonical_interval_matched": c["canonical_interval_matched"],
            "all_components_available": c["all_components_available"],
            "joint_component_counts": {k: c[k] for k in FIELDS},
        }
        for (i, actor, ally), c in sorted(matrix.items())
    ]
    summary = {
        "anchors": len(rows),
        "matrix_rows": len(matrix_rows),
        "canonical_interval_matched": sum(
            c["canonical_interval_matched"] for c in matrix.values()
        ),
        "all_components_available": sum(
            c["all_components_available"] for c in matrix.values()
        ),
        "joint_component_counts": {
            k: sum(c[k] for c in matrix.values()) for k in FIELDS
        },
        "fact_status_counts": {
            role: {k: dict(v) for k, v in fields.items()}
            for role, fields in statuses.items()
        },
        "eligible_decisions": 0,
    }
    result = {
        "schema_id": "stratai.support_physical_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
        "rows": rows,
        "matrix": matrix_rows,
        "summary": summary,
    }
    a.schema_validator(REPORT).validate(result)
    return result


def validate_report(report, support, physical, anchor_report, references, rounds):
    a.require(
        report == build_report(support, physical, anchor_report, references, rounds),
        "physical report recomputation mismatch",
    )


def file_reference(path):
    raw = path.read_bytes()
    return {"path": path.name, "bytes": len(raw), "sha256": a.digest(raw)}


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 512 * 1024 * 1024,
        "invalid physical source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    a.require(
        data["support_source"] == file_reference(path.parent / a.DATA),
        "physical support hash mismatch",
    )
    support, ref, base = a.load_source(path.parent / a.DATA)
    validate_source(data, support)
    a.require(path.read_bytes() == raw, "physical source changed")
    return (
        data,
        support,
        {"path": DATA, "bytes": len(raw), "sha256": a.digest(raw)},
        ref,
        base,
    )


def materialize(source, output, require_observed=False):
    validate_output_directory(output)
    physical, support, ref, support_ref, base = load_source(source)
    overlap = a.validate_replay_overlap(support, base)
    anchor_report = a.build_report(
        support, support_ref, require_observed=require_observed
    )
    anchor_bytes = (
        json.dumps(anchor_report, sort_keys=True, separators=(",", ":")) + "\n"
    ).encode()
    rounds_artifact = base.one("rounds")
    rounds = json.loads(rounds_artifact.physical_path.read_bytes())["rounds"]
    refs = {
        "physical_source": ref,
        "support_source": support_ref,
        "anchors": {
            "path": a.REPORT,
            "bytes": len(anchor_bytes),
            "sha256": a.digest(anchor_bytes),
        },
        "rounds": file_reference(rounds_artifact.physical_path),
    }
    report = build_report(support, physical, anchor_report, refs, rounds)
    if require_observed:
        a.require(
            report["summary"]["all_components_available"] > 0,
            "no complete physical pairs observed",
        )
    with (output / a.REPORT).open("xb") as stream:
        stream.write(anchor_bytes)
    with (output / REPORT).open("x", encoding="utf-8") as stream:
        json.dump(report, stream, separators=(",", ":"), sort_keys=True)
        stream.write("\n")
    return {"summary": report["summary"], "replay_validation": overlap}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    print(json.dumps(materialize(args.source, args.output, args.require_observed)))


if __name__ == "__main__":
    main()
