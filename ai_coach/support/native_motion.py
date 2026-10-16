"""Native pawn motion with explicit presence, reference and causal cuts."""

import argparse
import json
import math
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import objective_position as z
from ai_coach.support import physical as p

DATA = "support_native_motion_observations.json"
REPORT = "support_native_motion_context.json"
PLAYER = (
    "native_velocity",
    "native_horizontal_speed",
    "native_speed",
    "ground_contact",
    "ground_reference_state",
    "movement_mode_code",
)
FIELDS = tuple(role + "." + field for role in ("self", "ally") for field in PLAYER)
PROBES = [
    "m_vecVelocity",
    "m_vecAbsVelocity",
    "m_vecBaseVelocity",
    "m_vecVelocity.m_vecX",
    "m_vecVelocity.m_vecY",
    "m_vecVelocity.m_vecZ",
    "m_vecAbsVelocity.m_vecX",
    "m_vecAbsVelocity.m_vecY",
    "m_vecAbsVelocity.m_vecZ",
    "m_vecVelocity[0]",
    "m_vecVelocity[1]",
    "m_vecVelocity[2]",
    "m_fFlags",
    "m_hGroundEntity",
    "m_nGroundBodyIndex",
    "m_MoveType",
    "m_nActualMoveType",
    "m_MoveCollide",
    "CBodyComponent.m_hParent",
    "m_hMoveParent",
    "m_iParentAttachment",
    "m_flSimulationTime",
    "m_nTickBase",
    "m_pMovementServices.m_flMaxspeed",
    "m_pMovementServices.m_flFallVelocity",
    "m_pMovementServices.m_nLastLandedTick",
    "m_pMovementServices.m_flLastLandedFrac",
    "m_pMovementServices.m_flLastLandedVelocityX",
    "m_pMovementServices.m_flLastLandedVelocityY",
    "m_pMovementServices.m_flLastLandedVelocityZ",
    "m_pMovementServices.m_nLastJumpTick",
    "m_pMovementServices.m_flLastJumpFrac",
    "m_pMovementServices.m_flLastJumpVelocityZ",
]
LIMITS = [
    *z.LIMITS,
    "base_and_historical_velocity_are_not_current_pawn_velocity",
    "native_motion_and_ground_contact_do_not_establish_complete_readiness",
    "movement_enum_code_does_not_establish_intention_or_physical_route",
]


def validate_source(data, support, position):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == position[key],
            "motion source identity mismatch",
        )
    obs = data["observations"]
    deliveries, states, cuts = (obs[k] for k in ("deliveries", "states", "cuts"))
    created, last, destroyed, delivery_frames = {}, {}, set(), []
    for seq, d in enumerate(deliveries, 1):
        a.require(
            d["sequence"] == seq
            and (not delivery_frames or d["frame"] >= delivery_frames[-1]),
            "motion delivery order mismatch",
        )
        delivery_frames.append(d["frame"])
        life, name, kind = d["entity_life"], d["property"], d["kind"]
        a.require(life not in destroyed, "motion delivery after destruction")
        z.validate_scalar(d["previous"])
        z.validate_scalar(d["value"])
        if kind == "destroyed":
            a.require(
                life in created
                and name == ""
                and d["previous"]["status"] == "absent"
                and d["value"]["status"] == "absent",
                "invalid motion destruction",
            )
            destroyed.add(life)
            continue
        a.require(name in PROBES, "motion field outside probe contract")
        key = life, name
        if kind == "created":
            a.require(
                key not in last
                and created.setdefault(life, d["frame"]) == d["frame"]
                and d["previous"]["status"] == "absent",
                "invalid motion creation",
            )
        else:
            a.require(
                key in last and d["previous"] == last[key],
                "motion predecessor mismatch",
            )
            a.require(
                (kind == "creation_replay") == (d["frame"] == created[life]),
                "motion creation replay misclassified",
            )
        last[key] = d["value"]
    available, ended, frontier, state_frontier, identities = {}, set(), 0, 0, {}
    previous_cuts = position["observations"]["cuts"]
    a.require(
        len(cuts) == len(support["observations"]["frames"]) == len(previous_cuts),
        "motion cut coverage mismatch",
    )
    for seq, (cut, base, previous) in enumerate(
        zip(cuts, support["observations"]["frames"], previous_cuts, strict=True), 1
    ):
        a.require(
            cut["source_sequence"]
            == seq
            == base["sequence"]
            == previous["source_sequence"]
            and all(
                cut[k] == base[k] == previous[k]
                for k in ("frame", "tick", "instance", "epoch")
            )
            and cut["clock_order"] == previous["clock_order"],
            "motion exact cut mismatch",
        )
        end = bisect_right(delivery_frames, cut["frame"])
        a.require(cut["delivery_sequence"] == end, "motion delivery frontier mismatch")
        for d in deliveries[frontier:end]:
            if d["kind"] == "destroyed":
                ended.add(d["entity_life"])
            else:
                available[d["entity_life"], d["property"]] = d
        frontier = end
        a.require(
            state_frontier <= cut["state_sequence"] <= len(states),
            "motion state frontier mismatch",
        )
        new_states = set(range(state_frontier + 1, cut["state_sequence"] + 1))
        used = set()
        a.require(
            len(cut["players"]) == len(base["players"]), "motion player census mismatch"
        )
        for player, actor in zip(cut["players"], base["players"], strict=True):
            a.require(
                player["actor_alias"] == actor["actor_alias"],
                "motion actor order mismatch",
            )
            present = [
                player[k] is not None
                for k in ("pawn_handle", "pawn_life", "state_sequence")
            ]
            a.require(all(present) or not any(present), "partial motion pawn reference")
            if not any(present):
                continue
            a.require(
                actor["connected"]
                and actor["identity_status"] == "observed"
                and actor["alive"] is True
                and actor["alive_status"] == "observed"
                and actor["position_status"] == "observed"
                and actor["position"] is not None
                and (player["pawn_life"], player["pawn_handle"])
                == (actor["pawn_life"], actor["pawn_handle"]),
                "motion pawn is not current support actor",
            )
            state_seq = player["state_sequence"]
            a.require(
                1 <= state_seq <= cut["state_sequence"], "motion state outside prefix"
            )
            used.add(state_seq)
            state = states[state_seq - 1]
            a.require(
                state["sequence"] == state_seq
                and state["first_cut"] <= seq
                and (state["instance"], state["epoch"])
                == (cut["instance"], cut["epoch"])
                and state["handle"] == player["pawn_handle"],
                "motion state provenance mismatch",
            )
            if state_seq in new_states:
                a.require(state["first_cut"] == seq, "motion first cut mismatch")
            life = state["entity_life"]
            a.require(
                life in created
                and life not in ended
                and state["created_frame"] == created[life],
                "motion state lifetime mismatch",
            )
            identity = (
                state["handle"],
                state["class_name"],
                state["created_frame"],
                player["pawn_life"],
            )
            a.require(
                identities.setdefault(life, identity) == identity,
                "motion pawn identity changed",
            )
            a.require(
                set(state["fields"]) == set(state["field_sequences"]) == set(PROBES),
                "motion probe fields mismatch",
            )
            for name, value in state["fields"].items():
                d = available.get((life, name))
                a.require(
                    d is not None
                    and value == d["value"]
                    and state["field_sequences"][name] == d["sequence"],
                    "motion snapshot not equal to delivered prefix",
                )
            for key, name in (
                ("ground", "m_hGroundEntity"),
                ("parent", "CBodyComponent.m_hParent"),
            ):
                ref = state[key]
                z.validate_reference(ref, state["fields"][name])
                a.require(
                    (ref["class_name"] is not None) == (ref["status"] == "resolved"),
                    "motion reference class mismatch",
                )
        a.require(new_states <= used, "unused motion state at first cut")
        state_frontier = cut["state_sequence"]
    a.require(state_frontier == len(states), "motion trailing state")


def missing(reason):
    return {k: o.fact(reason=reason) for k in PLAYER}


def native_facts(state):
    fields = state["fields"]
    result = missing("native_current_velocity_source_absent")
    mode = fields["m_MoveType"]
    code = z.raw_integer(mode)
    result["movement_mode_code"] = (
        o.fact(code)
        if code is not None and mode["declared_type"] == "MoveType_t"
        else o.fact(reason="native_movement_mode_unavailable")
    )
    flags, ground = fields["m_fFlags"], fields["m_hGroundEntity"]
    bits, handle = z.raw_integer(flags), z.raw_integer(ground)
    reason = "native_ground_fields_unavailable"
    if (
        bits is not None
        and flags["declared_type"] == "uint32"
        and handle is not None
        and (ground["declared_type"] or "").replace(" ", "") == "CHandle<CBaseEntity>"
    ):
        if not bits & 1 and state["ground"]["status"] == "invalid_sentinel":
            result["ground_contact"], result["ground_reference_state"] = (
                o.fact(False),
                o.fact("none"),
            )
            reason = None
        elif bits & 1 and state["ground"]["status"] == "resolved":
            result["ground_contact"] = o.fact(True)
            result["ground_reference_state"] = o.fact(
                "world" if state["ground"]["class_name"] == "CWorld" else "entity"
            )
            reason = None
        else:
            reason = (
                "native_ground_reference_unresolved"
                if state["ground"]["status"] not in ("resolved", "invalid_sentinel")
                else "native_ground_signals_conflict"
            )
    if reason:
        result["ground_contact"] = o.fact(reason=reason)
        result["ground_reference_state"] = o.fact(reason=reason)
    velocity = [fields[name] for name in ("m_vecAbsVelocity", "m_vecVelocity")]
    if all(v["status"] == "absent" for v in velocity):
        return result
    reason = "native_velocity_reference_unaccredited"
    vectors = [
        v["value"]
        for v in velocity
        if v["status"] == "observed"
        and v["declared_type"] == "VectorWS"
        and v["native_type"] in ("[]float32", "[]float64", "[3]float32", "[3]float64")
        and isinstance(v["value"], list)
        and len(v["value"]) == 3
    ]
    if vectors:
        parent = fields["CBodyComponent.m_hParent"]
        if any(v != vectors[0] for v in vectors):
            reason = "native_velocity_sources_conflict"
        elif (
            parent["declared_type"] != "CGameSceneNodeHandle"
            or state["parent"]["status"] != "invalid_sentinel"
        ) or any(
            fields[name]["status"] != "absent"
            and (
                z.raw_integer(fields[name]) is None
                or (
                    z.raw_integer(fields[name]) & 16383 != 16383
                    if name == "m_hMoveParent"
                    else z.raw_integer(fields[name]) != 0
                )
            )
            for name in ("m_hMoveParent", "m_iParentAttachment")
        ):
            reason = "native_velocity_parent_reference_unaccredited"
        else:
            vector = vectors[0]
            result["native_velocity"] = o.fact(dict(zip("xyz", vector, strict=True)))
            result["native_horizontal_speed"] = o.fact(math.hypot(*vector[:2]))
            result["native_speed"] = o.fact(math.hypot(*vector))
            return result
    for name in PLAYER[:3]:
        result[name] = o.fact(reason=reason)
    return result


def initial_context(anchor, data, cut):
    result, references = {}, {}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        player = next((x for x in cut["players"] if x["actor_alias"] == alias), None)
        if player is None or player["state_sequence"] is None:
            result[role], references[role] = (
                missing("current_motion_pawn_unavailable"),
                None,
            )
            continue
        state = data["observations"]["states"][player["state_sequence"] - 1]
        result[role] = native_facts(state)
        references[role] = {
            "state_sequence": state["sequence"],
            "entity_life": state["entity_life"],
            "pawn_life": player["pawn_life"],
            "source_sequence": cut["source_sequence"],
            "frame": cut["frame"],
        }
    return result, references


def prefix_source(data, seq):
    obs = data["observations"]
    cuts = [c for c in obs["cuts"] if c["source_sequence"] <= seq]
    a.require(cuts and cuts[-1]["source_sequence"] == seq, "missing exact motion cut")
    cut = cuts[-1]
    return {
        **{k: v for k, v in data.items() if k != "observations"},
        "observations": {
            "cuts": cuts,
            "states": [
                s for s in obs["states"] if s["sequence"] <= cut["state_sequence"]
            ],
            "deliveries": [
                d
                for d in obs["deliveries"]
                if d["sequence"] <= cut["delivery_sequence"]
            ],
        },
    }


def initial_features(support, position, data, anchor_id, actor, tick, frame):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support = {
        **support,
        "observations": {
            "frames": [
                f for f in support["observations"]["frames"] if f["sequence"] <= seq
            ]
        },
    }
    position, data = z.prefix_source(position, seq), prefix_source(data, seq)
    validate_source(data, support, position)
    anchor = next(
        x
        for x in a.select_anchors(
            support["observations"]["frames"], support["source_alias"]
        )[0]
        if x["anchor_id"] == anchor_id
    )
    return initial_context(anchor, data, data["observations"]["cuts"][-1])[0]


def build_report(support, position, data, previous, references):
    validate_source(data, support, position)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(len(selected) == len(previous["rows"]), "motion previous rows mismatch")
    counts, matrix, rows = {k: Counter() for k in FIELDS}, {}, []
    for anchor, old in zip(selected, previous["rows"], strict=True):
        a.require(
            all(anchor[k] == old[k] for k in o.KEYS), "motion previous anchor mismatch"
        )
        cut = data["observations"]["cuts"][anchor["source_sequence"] - 1]
        initial, ref = initial_context(anchor, data, cut)
        key = anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]
        entry = matrix.setdefault(
            key,
            {
                "anchors": 0,
                "canonical_interval_matched": 0,
                "fact_status_counts": {k: Counter() for k in FIELDS},
            },
        )
        entry["anchors"] += 1
        entry["canonical_interval_matched"] += (
            old["competitive_round_status"] == "canonical_interval_matched"
        )
        for field in FIELDS:
            role, name = field.split(".")
            f = initial[role][name]
            status = f["status"] + (":" + f["reason"] if f["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        rows.append(
            {
                **{k: anchor[k] for k in o.KEYS},
                "competitive_round_status": old["competitive_round_status"],
                "initial": initial,
                "motion_reference": ref,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    source_counts = {name: Counter() for name in PROBES}
    for state in data["observations"]["states"]:
        for name, value in state["fields"].items():
            source_counts[name][value["status"]] += 1
    report = {
        "schema_id": "stratai.support_native_motion_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
        "rows": rows,
        "matrix": [
            {"instance": k[0], "actor_alias": k[1], "ally_alias": k[2], **v}
            for k, v in sorted(matrix.items())
        ],
        "summary": {
            "anchors": len(rows),
            "matrix_rows": len(matrix),
            "canonical_interval_matched": sum(
                v["canonical_interval_matched"] for v in matrix.values()
            ),
            "fact_status_counts": counts,
            "native_field_state_counts": source_counts,
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(report)
    return report


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 1024 * 1024 * 1024,
        "invalid native motion source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    position, _attempt, _clock, _ownership, _objective, support, base = z.load_source(
        path.parent / z.DATA
    )
    a.require(
        data["references"]
        == {
            **position["references"],
            "position_source": p.file_reference(path.parent / z.DATA),
        },
        "motion source hash mismatch",
    )
    validate_source(data, support, position)
    a.require(path.read_bytes() == raw, "motion source changed")
    return data, position, support, base


def report_inputs(source, output):
    data, position, support, _ = load_source(source)
    previous = json.loads((output / z.REPORT).read_bytes())
    references = {
        **previous["references"],
        "motion_source": p.file_reference(source),
        "position_context": p.file_reference(output / z.REPORT),
    }
    return support, position, data, previous, references


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = z.materialize(source.parent / z.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["anchors"] > 0, "empty native motion context")
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    z.validate_artifact(output, source.parent / z.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "motion report recomputation mismatch",
    )
    return report["summary"]


def main():
    parser = argparse.ArgumentParser()
    for name in ("source", "output", "evaluator", "maps-root"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    result = (
        validate_artifact(args.output, args.source, args.evaluator, args.maps_root)
        if args.validate
        else materialize(args.source, args.output, args.evaluator, args.maps_root)
    )
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()
