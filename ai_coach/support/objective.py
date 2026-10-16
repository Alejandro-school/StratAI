"""Public objective delivery prefix and exact self/ally facts at support T0."""

import argparse
import bisect
import copy
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import navigation as n
from ai_coach.support import physical as p

DATA = "support_objective_observations.json"
REPORT = "support_objective_context.json"
RULES = {
    "bomb_planted": "m_bBombPlanted",
    "freeze_period": "m_bFreezePeriod",
    "warmup": "m_bWarmupPeriod",
    "round_duration_seconds": "m_iRoundTime",
}
PLAYER = {
    "in_bomb_zone": "m_bInBombZone",
    "defusing": "m_bIsDefusing",
    "has_defuse_kit": "m_pItemServices.m_bHasDefuser",
}
GLOBAL = (*RULES, "objective_phase", "round_elapsed_seconds", "bomb_elapsed_seconds")
FIELDS = tuple("global." + k for k in GLOBAL) + tuple(
    role + "." + k for role in ("self", "ally") for k in PLAYER
)
KEYS = (
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
KINDS = (
    "round_start",
    "freeze_end",
    "bomb_planted",
    "bomb_defused",
    "bomb_exploded",
    "round_end",
)
LIMITS = [
    "physical_entry_not_accredited",
    "physical_navigation_not_accredited",
    "line_of_sight_and_contact_not_accredited",
    "cooperation_and_intent_not_observable",
    "objective_position_and_carrier_not_accredited",
    "countdown_and_pause_clock_not_accredited",
    "instantaneous_velocity_and_full_weapon_readiness_not_accredited",
]


def fact(value=None, reason=None, status="unavailable"):
    return {
        "status": "observed" if value is not None else status,
        "value": value,
        "reason": reason,
    }


def native(scalar, *, integer=False):
    if scalar["status"] != "observed":
        return fact(reason="native_" + scalar["status"])
    value, typ = scalar["value"], scalar["native_type"]
    if integer:
        valid = (
            type(value) is int
            and typ in ("int", "int32", "uint32", "int64", "uint64")
            and 0 <= value <= 2**31 - 1
        )
    else:
        valid = type(value) is bool and typ == "bool"
    return fact(value) if valid else fact(reason="native_type_or_range_unsupported")


def validate_source(data, support):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(data[key] == support[key], "objective source identity mismatch")
    frames = support["observations"]["frames"]
    events = data["observations"]["events"]
    orders = []
    for i, event in enumerate(events, 1):
        a.require(event["sequence"] == i, "objective event sequence mismatch")
        a.require(
            not orders or event["order"] > orders[-1], "objective event order mismatch"
        )
        if i > 1:
            prev = events[i - 2]
            a.require(
                event["frame"] >= prev["frame"]
                and event["instance"] >= prev["instance"]
                and event["epoch"] >= prev["epoch"],
                "objective event clock regressed",
            )
            if event["epoch"] == prev["epoch"]:
                a.require(
                    event["tick"] >= prev["tick"],
                    "objective event tick regressed without epoch",
                )
        orders.append(event["order"])
    previous_seq = previous_order = 0
    all_orders = set(orders)
    for row in data["observations"]["frames"]:
        seq = row["source_sequence"]
        a.require(
            previous_seq < seq <= len(frames), "objective exact sequence mismatch"
        )
        base = frames[seq - 1]
        a.require(
            all(row[k] == base[k] for k in ("frame", "tick", "instance", "epoch")),
            "objective exact clock mismatch",
        )
        a.require(
            row["order"] > previous_order and row["order"] not in all_orders,
            "objective cut order mismatch",
        )
        previous_seq, previous_order = seq, row["order"]
        count = bisect.bisect_left(orders, row["order"])
        a.require(row["event_sequence"] == count, "objective event frontier mismatch")
        if count:
            event = events[count - 1]
            a.require(
                event["frame"] <= row["frame"]
                and event["instance"] <= row["instance"]
                and event["epoch"] <= row["epoch"],
                "objective future event at cut",
            )
            if event["epoch"] == row["epoch"]:
                a.require(event["tick"] <= row["tick"], "objective future tick at cut")
        if count < len(events):
            a.require(
                events[count]["frame"] >= row["frame"],
                "objective omitted earlier event",
            )
        actors = a.actor_index(base)
        aliases = [x["actor_alias"] for x in row["players"]]
        a.require(len(aliases) == len(set(aliases)), "duplicate objective actor")
        for player in row["players"]:
            original = actors.get(player["actor_alias"])
            a.require(original is not None, "objective actor absent from support")
            if player["pawn_life"] is not None:
                a.require(
                    a.usable(original) and player["pawn_life"] == original["pawn_life"],
                    "objective pawn life mismatch",
                )
            else:
                a.require(
                    all(s["status"] == "absent" for s in player["fields"].values()),
                    "objective fields without pawn identity",
                )
        for scalar in [
            *row["rules"].values(),
            *(s for x in row["players"] for s in x["fields"].values()),
        ]:
            a.require(
                (scalar["status"] == "observed") == (scalar["value"] is not None),
                "objective scalar presence mismatch",
            )
            a.require(
                (scalar["status"] == "absent") == (scalar["native_type"] is None),
                "objective native type presence mismatch",
            )


def public_prefixes(data):
    """One forward pass, scoped to the observed round instance and clock epoch."""
    states, result, cursor = {}, {}, 0
    events = data["observations"]["events"]
    for row in data["observations"]["frames"]:
        while cursor < row["event_sequence"]:
            event = events[cursor]
            key = event["instance"], event["epoch"]
            state = states.setdefault(key, {})
            kind = event["kind"]
            if kind == "round_start":
                state.clear()
                state.update(phase="preplant", phase_event=event)
            elif kind == "freeze_end":
                state["freeze"] = event
                if "phase" not in state:
                    state.update(phase="preplant", phase_event=event)
            elif kind == "bomb_planted":
                state.update(phase="planted", phase_event=event, plant=event)
            else:
                state.update(phase="resolved", phase_event=event)
            cursor += 1
        result[row["source_sequence"]] = dict(
            states.get((row["instance"], row["epoch"]), {})
        )
    return result


def elapsed(event, frame, support_frame):
    if event is None:
        return fact(reason="public_clock_origin_absent")
    rate = support_frame["tick_rate"]
    if rate is None or event["tick"] > frame["tick"]:
        return fact(reason="clock_unavailable")
    # This measures parser-tick elapsed time. It is not a paused game countdown.
    return fact((frame["tick"] - event["tick"]) / rate)


def initial_context(anchor, base, row, state):
    if row is None:
        return {
            role: {k: fact(reason="missing_exact_objective_sample") for k in fields}
            for role, fields in (("global", GLOBAL), ("self", PLAYER), ("ally", PLAYER))
        }
    global_facts = {
        k: native(row["rules"][source], integer=k == "round_duration_seconds")
        for k, source in RULES.items()
    }
    phase = state.get("phase")
    planted = global_facts["bomb_planted"]["value"]
    conflict = (phase == "preplant" and planted is True) or (
        phase == "planted" and planted is False
    )
    replay_conflict = phase == "resolved" and base["phase"] == "playing"
    conflict = conflict or replay_conflict
    global_facts["objective_phase"] = (
        fact(phase)
        if phase is not None and not conflict
        else fact(
            reason="public_replay_phase_conflict"
            if replay_conflict
            else "native_public_phase_conflict"
            if conflict
            else "public_phase_origin_absent"
        )
    )
    global_facts["round_elapsed_seconds"] = elapsed(state.get("freeze"), row, base)
    if conflict or phase is None:
        bomb_time = fact(reason="objective_phase_unavailable")
    elif phase != "planted":
        bomb_time = fact(reason="no_active_planted_phase", status="not_applicable")
    else:
        bomb_time = elapsed(state.get("plant"), row, base)
    global_facts["bomb_elapsed_seconds"] = bomb_time
    result = {"global": global_facts}
    players = {x["actor_alias"]: x for x in row["players"]}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        player = players.get(alias)
        result[role] = {
            k: native(player["fields"][source])
            if player is not None and player["pawn_life"] is not None
            else fact(reason="actor_objective_context_unavailable")
            for k, source in PLAYER.items()
        }
    return result


def initial_features(support, objective, anchor_id, actor, tick, frame):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    prefix = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:seq]},
    }
    cuts = [
        f for f in objective["observations"]["frames"] if f["source_sequence"] <= seq
    ]
    frontier = cuts[-1]["event_sequence"] if cuts else 0
    data = {
        **objective,
        "observations": {
            "frames": cuts,
            "events": objective["observations"]["events"][:frontier],
        },
    }
    validate_source(data, prefix)
    selected, _ = a.select_anchors(
        prefix["observations"]["frames"], support["source_alias"]
    )
    anchor = next(x for x in selected if x["anchor_id"] == anchor_id)
    row = next((f for f in cuts if f["source_sequence"] == seq), None)
    return copy.deepcopy(
        initial_context(
            anchor,
            prefix["observations"]["frames"][-1],
            row,
            public_prefixes(data).get(seq, {}),
        )
    )


def build_report(support, data, anchor_report, physical_report, references):
    validate_source(data, support)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        selected
        == [
            {k: v for k, v in x.items() if k != "later_observation"}
            for x in anchor_report["anchors"]
        ],
        "objective v1 anchor mismatch",
    )
    a.require(
        len(physical_report["rows"]) == len(selected),
        "objective physical row count mismatch",
    )
    index = {f["source_sequence"]: f for f in data["observations"]["frames"]}
    states = public_prefixes(data)
    matrix, counts, rows = {}, {f: Counter() for f in FIELDS}, []
    for anchor, physical in zip(selected, physical_report["rows"], strict=True):
        a.require(
            all(anchor[k] == physical[k] for k in KEYS),
            "objective physical anchor mismatch",
        )
        seq = anchor["source_sequence"]
        initial = initial_context(
            anchor,
            support["observations"]["frames"][seq - 1],
            index.get(seq),
            states.get(seq, {}),
        )
        competitive = physical["competitive_round_status"]
        key = anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]
        entry = matrix.setdefault(
            key,
            {
                "anchors": 0,
                "canonical_interval_matched": 0,
                "phase_observed": 0,
                "fact_status_counts": {f: Counter() for f in FIELDS},
            },
        )
        entry["anchors"] += 1
        entry["canonical_interval_matched"] += (
            competitive == "canonical_interval_matched"
        )
        entry["phase_observed"] += (
            initial["global"]["objective_phase"]["status"] == "observed"
        )
        for field in FIELDS:
            role, name = field.split(".")
            f = initial[role][name]
            status = f["status"] + (":" + f["reason"] if f["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        rows.append(
            {
                **{k: anchor[k] for k in KEYS},
                "competitive_round_status": competitive,
                "initial": initial,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    matrix_rows = [
        {"instance": key[0], "actor_alias": key[1], "ally_alias": key[2], **entry}
        for key, entry in sorted(matrix.items())
    ]
    result = {
        "schema_id": "stratai.support_objective_context@1",
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
        "summary": {
            "anchors": len(rows),
            "matrix_rows": len(matrix_rows),
            "phase_observed": sum(v["phase_observed"] for v in matrix.values()),
            "canonical_interval_matched": sum(
                v["canonical_interval_matched"] for v in matrix.values()
            ),
            "fact_status_counts": counts,
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(result)
    return result


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 512 * 1024 * 1024,
        "invalid objective source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    support, ref, base = a.load_source(path.parent / a.DATA)
    a.require(data["support_source"] == ref, "objective support hash mismatch")
    validate_source(data, support)
    a.require(path.read_bytes() == raw, "objective source changed")
    return data, support, ref, base


def report_inputs(source, output):
    data, support, support_ref, _ = load_source(source)
    anchor_report = json.loads((output / a.REPORT).read_bytes())
    physical_report = json.loads((output / p.REPORT).read_bytes())
    refs = {
        "support_source": support_ref,
        "objective_source": p.file_reference(source),
        **{
            key: p.file_reference(output / name)
            for key, name in (
                ("anchors", a.REPORT),
                ("physical_context", p.REPORT),
                ("navigation_context", n.REPORT),
            )
        },
    }
    return support, data, anchor_report, physical_report, refs


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    # Validate new input before the expensive, unchanged previous materializer.
    load_source(source)
    previous = n.materialize(source.parent / p.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["phase_observed"] > 0, "no observed objective phases")
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    n.validate_artifact(output, source.parent / p.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "objective report recomputation mismatch",
    )
    return report["summary"]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--evaluator", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    result = (
        validate_artifact(args.output, args.source, args.evaluator, args.maps_root)
        if args.validate
        else materialize(args.source, args.output, args.evaluator, args.maps_root)
    )
    print(json.dumps(result))


if __name__ == "__main__":
    main()
