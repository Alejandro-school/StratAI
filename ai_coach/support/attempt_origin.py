"""Native planting transition origins, separate from baseline and sampled state."""

import argparse
import copy
import json
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import c4
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as c
from ai_coach.support import physical as p

DATA = "support_attempt_origin_observations.json"
REPORT = "support_attempt_origin_context.json"
PLAYER = (
    "planting_origin_state",
    "planting_duration_seconds",
    "planting_elapsed_seconds",
    "planting_progress",
)
FIELDS = tuple(role + "." + field for role in ("self", "ally") for field in PLAYER)
LIMITS = [
    *c.LIMITS,
    "planting_progress_is_native_delivery_tick_resolution_not_input_subtick_or_outcome",
]


def validate_source(data, support, clock):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == clock[key], "attempt source identity mismatch"
        )
    obs = data["observations"]
    deliveries, frames, cuts = obs["deliveries"], obs["frames"], obs["cuts"]
    last, lives, destroyed = {}, {}, set()
    previous_frame = 0
    for seq, d in enumerate(deliveries, 1):
        a.require(
            d["sequence"] == seq and d["frame"] >= previous_frame,
            "attempt delivery order mismatch",
        )
        previous_frame = d["frame"]
        life, prop, kind = d["entity_life"], d["property"], d["kind"]
        a.require(life not in destroyed, "attempt delivery after destruction")
        for value in (d["previous"], d["value"]):
            c.validate_scalar(value)
        if kind == "destroyed":
            a.require(life in lives and prop == "", "attempt destruction without life")
            destroyed.add(life)
            continue
        a.require(prop != "", "attempt property missing")
        key = life, prop
        if kind == "created":
            a.require(
                key not in last and lives.setdefault(life, d["frame"]) == d["frame"],
                "attempt repeated creation",
            )
            a.require(
                d["previous"]["status"] == "absent",
                "attempt baseline has a fabricated predecessor",
            )
        else:
            a.require(
                key in last and d["previous"] == last[key],
                "attempt delivery predecessor mismatch",
            )
            a.require(
                (kind == "creation_replay") == (d["frame"] == lives[life]),
                "attempt baseline replay misclassified",
            )
        last[key] = d["value"]
    previous_frame = frontier = 0
    available_values, ended = {}, set()
    identities = {}
    message_records = {}
    for seq, row in enumerate(frames, 1):
        a.require(
            row["sequence"] == seq and row["frame"] > previous_frame,
            "attempt frame order mismatch",
        )
        previous_frame = row["frame"]
        end = row["delivery_sequence"]
        a.require(
            frontier <= end <= len(deliveries), "attempt delivery frontier mismatch"
        )
        for d in deliveries[frontier:end]:
            a.require(
                d["frame"] == row["frame"] and d["tick"] == row["tick"],
                "attempt delivery frame mismatch",
            )
            if d["kind"] == "destroyed":
                ended.add(d["entity_life"])
            else:
                available_values[d["entity_life"], d["property"]] = d["value"]
        a.require(
            end == len(deliveries) or deliveries[end]["frame"] > row["frame"],
            "attempt omitted available delivery",
        )
        frontier = end
        for kind, messages in row["network"].items():
            previous_order = 0
            for m in messages:
                a.require(
                    m["frame"] == row["frame"]
                    and m["tick"] == row["tick"]
                    and m["order"] > previous_order,
                    "attempt native message frontier mismatch",
                )
                previous_order = m["order"]
                c.validate_scalar(m["value"])
                a.require(
                    message_records.setdefault((kind, m["order"]), m) == m,
                    "attempt message mutated",
                )
        if row["interval"]:
            c.validate_scalar(row["interval"]["value"])
            a.require(
                row["interval"]["frame"] <= row["frame"], "future attempt interval"
            )
        if row["rules"]:
            for value in row["rules"]["fields"].values():
                c.validate_scalar(value)
        current_handles = []
        for item in row["items"]:
            life = item["entity_life"]
            a.require(
                life in lives and item["created_frame"] == lives[life],
                "attempt item creation mismatch",
            )
            a.require(
                identities.setdefault(life, item["handle"]) == item["handle"],
                "attempt item identity changed",
            )
            a.require(item["handle"] & 16383 != 16383, "attempt item sentinel")
            if item["current"]:
                current_handles.append(item["handle"])
                a.require(life not in ended, "destroyed attempt item current")
            for name, value in item["fields"].items():
                c.validate_scalar(value)
                a.require(
                    available_values.get((life, name)) == value,
                    "attempt snapshot disagrees with native deliveries",
                )
            c.validate_scalar({**item["definition"], "declared_type": None})
            c4.validate_reference(item["owner"])
            for slot in item["inventory_slots"]:
                c4.validate_reference(slot)
        a.require(
            len(current_handles) == len(set(current_handles)),
            "duplicate attempt item handle",
        )
    a.require(frontier == len(deliveries), "unsealed attempt deliveries")
    native = {r["source_sequence"]: r for r in clock["observations"]["frames"]}
    times = [r["frame"] for r in frames]
    for cut in cuts:
        seq = cut["source_sequence"]
        a.require(
            1 <= seq <= len(support["observations"]["frames"]),
            "attempt source cut out of range",
        )
        base = support["observations"]["frames"][seq - 1]
        row = native.get(seq)
        a.require(
            row is not None
            and all(cut[k] == base[k] for k in ("frame", "tick", "instance", "epoch"))
            and cut["clock_order"] == row["order"],
            "attempt exact support cut mismatch",
        )
        available = bisect_right(times, cut["frame"])
        a.require(cut["frame_sequence"] == available, "attempt frame frontier mismatch")
        expected = frames[available - 1]["delivery_sequence"] if available else 0
        a.require(
            cut["delivery_sequence"] == expected,
            "attempt cut delivery frontier mismatch",
        )
        if available and frames[available - 1]["frame"] == cut["frame"]:
            exact = frames[available - 1]
            a.require(
                all(exact[k] == cut[k] for k in ("tick", "instance", "epoch")),
                "attempt active frame boundary mismatch",
            )
            clock_items = {item["entity_life"]: item for item in row["items"]}
            for item in exact["items"]:
                if not item["current"]:
                    continue
                other = clock_items.get(item["entity_life"])
                a.require(
                    other is not None
                    and other["handle"] == item["handle"]
                    and all(
                        other["fields"][k] == item["fields"][k]
                        for k in ("m_bStartedArming", "m_fArmedTime")
                    ),
                    "attempt and clock item mismatch",
                )
    a.require(
        [x["source_sequence"] for x in cuts]
        == list(range(1, len(support["observations"]["frames"]) + 1)),
        "attempt cut coverage mismatch",
    )


def frame_clock(row, witness):
    if witness is None or row["rules"] is None or row["interval"] is None:
        return None, "native_origin_clock_unaccredited"
    interval = row["interval"]
    step = c.native(interval["value"])
    start = c.native(row["rules"]["fields"]["m_fRoundStartTime"], game_time=True)
    key = (
        row["instance"],
        row["epoch"],
        row["rules"]["entity_life"],
        interval["order"],
        step["value"],
        start["value"],
    )
    expected = (
        witness["instance"],
        witness["epoch"],
        witness["rules_life"],
        witness["interval_order"],
        witness["native_interval_seconds"],
        witness["native_round_start_seconds"],
    )
    if key != expected or row["frame"] < witness["frame"] or row["phase"] != "playing":
        return None, "native_origin_domain_or_round_changed"
    values = []
    for kind in ("tick", "packet_tick"):
        messages = row["network"][kind]
        if not messages or any(
            c.native(m["value"], "integer")["status"] != "observed" for m in messages
        ):
            return None, "native_origin_tick_absent"
        unique = {m["value"]["value"] for m in messages}
        if len(unique) != 1:
            return None, "native_origin_packet_tick_ambiguous"
        values.append(next(iter(unique)))
    if values[0] != values[1] or values[0] < witness["server_tick"]:
        return None, "native_origin_tick_disagreement_or_rewind"
    # Every message in the frame must use the one interval/session. A frame
    # containing a transition between interval messages is not time-localized.
    if any(m != interval for m in row["network"]["interval"]):
        return None, "native_origin_interval_transition"
    pause = c.pause_state({"rules": row["rules"], "network": {"paused": None}})
    if pause["value"] != "running":
        return None, "native_origin_paused" if pause[
            "value"
        ] == "paused" else "native_origin_pause_unavailable"
    return (values[0] * step["value"], values[0], step["value"], key), None


def owner_key(item, items):
    owner = item["owner"]
    if (
        not item["current"]
        or owner["status"] != "resolved"
        or owner["target_life"] is None
    ):
        return None
    raw_owner = item["fields"]["m_hOwnerEntity"]
    if (
        raw_owner["status"] != "observed"
        or raw_owner["native_type"] not in c.INTEGERS
        or raw_owner["value"] != owner["handle"]
    ):
        return None
    definition = item["definition"]
    if (
        definition["status"] != "observed"
        or definition["native_type"] not in c.INTEGERS
        or definition["value"] != 49
    ):
        return None
    if item["inventory_status"] != "observed":
        return None
    slots = item["inventory_slots"]
    resolved = [s["handle"] for s in slots if s["status"] == "resolved"]
    if len(resolved) != len(set(resolved)):
        return None
    matching = [
        s
        for s in slots
        if s["status"] == "resolved"
        and s["handle"] == item["handle"]
        and s["target_life"] == item["entity_life"]
    ]
    claimed = [
        x
        for x in items
        if x["current"]
        and x["owner"]["status"] == "resolved"
        and x["owner"]["handle"] == owner["handle"]
    ]
    if len(matching) != 1 or len(claimed) != 1:
        return None
    return owner["target_life"], owner["handle"]


def origin_prefixes(support, objective, ownership, clock, data):
    alignments, _ = c.alignment_prefixes(support, objective, clock)
    clock_frames = clock["observations"]["frames"]
    times = [r["frame"] for r in clock_frames]
    deliveries = data["observations"]["deliveries"]
    states, result = {}, {}
    frontier = 0
    for row in data["observations"]["frames"]:
        index = bisect_right(times, row["frame"]) - 1
        witness = (
            alignments.get(clock_frames[index]["source_sequence"])
            if index >= 0
            else None
        )
        now, clock_reason = frame_clock(row, witness)
        delivered = deliveries[frontier : row["delivery_sequence"]]
        frontier = row["delivery_sequence"]
        current = {}
        for item in row["items"]:
            life = item["entity_life"]
            previous = states.get(life)
            own = owner_key(item, row["items"])
            arming = c.native(item["fields"]["m_bStartedArming"], "bool")
            deadline = c.native(item["fields"]["m_fArmedTime"], game_time=True)
            updates = [d for d in delivered if d["entity_life"] == life]
            starts = [
                d
                for d in updates
                if d["kind"] == "update"
                and d["property"] == "m_bStartedArming"
                and c.native(d["previous"], "bool")["value"] is False
                and c.native(d["value"], "bool")["value"] is True
            ]
            proof, reason = None, "native_start_transition_unavailable"
            if not item["current"]:
                reason = "native_attempt_life_ended"
            elif arming["value"] is not True:
                reason = (
                    "native_attempt_cancelled"
                    if previous and previous["active"] and arming["value"] is False
                    else "no_active_planting_attempt"
                )
            elif row["frame"] == item["created_frame"]:
                reason = "baseline_active_without_start_transition"
            elif own is None:
                reason = "native_origin_owner_uncorroborated"
            elif now is None:
                reason = clock_reason
            elif deadline["status"] != "observed":
                reason = deadline["reason"]
            elif len(starts) > 1:
                reason = "multiple_native_starts_in_delivery_frame"
            elif len(starts) == 1:
                start = starts[0]
                # A new owner after the flag or multiple distinct deadline
                # writes cannot be assigned to that onset without subtick data.
                later_owner = any(
                    d["sequence"] > start["sequence"]
                    and d["property"] == "m_hOwnerEntity"
                    and d["previous"] != d["value"]
                    for d in updates
                )
                deadline_updates = [
                    d
                    for d in updates
                    if d["kind"] == "update"
                    and d["property"] == "m_fArmedTime"
                    and d["previous"] != d["value"]
                ]
                duration = deadline["value"] - now[0]
                if later_owner:
                    reason = "native_owner_changed_after_start"
                elif not deadline_updates or any(
                    d["value"] != item["fields"]["m_fArmedTime"]
                    for d in deadline_updates
                ):
                    reason = "native_start_deadline_assignment_unaccredited"
                elif not 0 < duration <= 120:
                    reason = "native_start_duration_nonpositive_or_outside_range"
                else:
                    proof = {
                        "delivery_sequence": start["sequence"],
                        "frame": row["frame"],
                        "server_tick": now[1],
                        "interval_seconds": now[2],
                        "start_seconds": now[0],
                        "deadline_seconds": deadline["value"],
                        "duration_seconds": duration,
                        "entity_life": life,
                        "pawn_life": own[0],
                        "pawn_handle": own[1],
                    }
                    reason = None
            elif previous and previous["proof"]:
                prior = previous["proof"]
                if row["frame"] != previous["frame"] + 1:
                    reason = "native_active_frame_gap"
                elif now[3] != previous["key"] or now[0] < previous["now"]:
                    reason = "native_origin_domain_or_round_changed"
                elif own != (prior["pawn_life"], prior["pawn_handle"]):
                    reason = "native_attempt_owner_changed"
                elif deadline["value"] != prior["deadline_seconds"]:
                    reason = "native_attempt_deadline_changed"
                elif any(
                    d["property"] == "m_hOwnerEntity" and d["previous"] != d["value"]
                    for d in updates
                ):
                    reason = "native_attempt_owner_changed"
                else:
                    proof, reason = prior, None
            elif previous and previous["active"]:
                reason = previous["reason"]
            state = {
                "proof": proof,
                "reason": reason,
                "active": arming["value"] is True,
                "frame": row["frame"],
                "now": now[0] if now else None,
                "key": now[3] if now else None,
            }
            states[life] = current[life] = state
        result[row["sequence"]] = current
    return result


def initial_context(anchor, previous, row, states, ownership_state):
    result, references = {}, {"self": None, "ally": None}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):

        def unavailable(reason, status="unavailable"):
            return {k: o.fact(reason=reason, status=status) for k in PLAYER}

        attempt = previous[role]["attempt_state"]
        if attempt["status"] != "observed":
            result[role] = unavailable(attempt["reason"])
            continue
        if attempt["value"] not in ("planting", "paused_planting", "expired_planting"):
            result[role] = unavailable(
                "not_a_current_planting_attempt", "not_applicable"
            )
            continue
        if row is None or row["frame"] != anchor["t0_frame"]:
            result[role] = unavailable("exact_attempt_frame_unavailable")
            continue
        player = (
            next(
                (x for x in ownership_state["players"] if x["actor_alias"] == alias),
                None,
            )
            if ownership_state
            else None
        )
        if (
            player is None
            or c4.possession(player, ownership_state["entities"])["value"] is not True
        ):
            result[role] = unavailable("current_planter_ownership_uncorroborated")
            continue
        candidates = [
            x
            for x in row["items"]
            if x["current"]
            and x["owner"]["status"] == "resolved"
            and x["owner"]["target_life"] == player["pawn_life"]
            and x["owner"]["handle"] == player["pawn_handle"]
        ]
        if len(candidates) != 1:
            result[role] = unavailable("current_attempt_identity_uncorroborated")
            continue
        state = states.get(candidates[0]["entity_life"])
        proof = state["proof"] if state else None
        if proof is None:
            result[role] = unavailable(
                state["reason"] if state else "native_start_transition_unavailable"
            )
            continue
        if (proof["pawn_life"], proof["pawn_handle"]) != (
            player["pawn_life"],
            player["pawn_handle"],
        ):
            result[role] = unavailable("native_attempt_pawn_changed")
            continue
        left = previous[role]["attempt_remaining_seconds"]
        if left["status"] != "observed" or state["now"] is None:
            result[role] = unavailable("current_attempt_clock_unavailable")
            continue
        duration = proof["duration_seconds"]
        remaining = max(0, proof["deadline_seconds"] - state["now"])
        if remaining != left["value"] or not 0 <= remaining <= duration:
            result[role] = unavailable("current_attempt_clock_conflict")
            continue
        result[role] = {
            "planting_origin_state": o.fact(
                "expired" if remaining == 0 else "native_transition"
            ),
            "planting_duration_seconds": o.fact(duration),
            "planting_elapsed_seconds": o.fact(duration - remaining),
            "planting_progress": o.fact(1 - remaining / duration),
        }
        references[role] = copy.deepcopy(proof)
    return result, references


def prefix_source(data, frame, seq):
    obs = data["observations"]
    return {
        **{k: v for k, v in data.items() if k != "observations"},
        "observations": {
            "cuts": [x for x in obs["cuts"] if x["source_sequence"] <= seq],
            "frames": [x for x in obs["frames"] if x["frame"] <= frame],
            "deliveries": [x for x in obs["deliveries"] if x["frame"] <= frame],
        },
    }


def initial_features(
    support, objective, ownership, clock, data, anchor_id, actor, tick, frame
):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support, objective, ownership, clock = c.prefix_inputs(
        support, objective, ownership, clock, seq
    )
    data = prefix_source(data, frame, seq)
    previous = c.initial_features(
        support, objective, ownership, clock, anchor_id, actor, tick, frame
    )
    validate_source(data, support, clock)
    anchor = next(
        x
        for x in a.select_anchors(
            support["observations"]["frames"], support["source_alias"]
        )[0]
        if x["anchor_id"] == anchor_id
    )
    cut = data["observations"]["cuts"][-1]
    row = (
        data["observations"]["frames"][cut["frame_sequence"] - 1]
        if cut["frame_sequence"]
        else None
    )
    states = origin_prefixes(support, objective, ownership, clock, data).get(
        cut["frame_sequence"], {}
    )
    owner_cut = next(
        x for x in ownership["observations"]["cuts"] if x["source_sequence"] == seq
    )
    owner = ownership["observations"]["states"][owner_cut["state_sequence"] - 1]
    return initial_context(anchor, previous, row, states, owner)[0]


def build_report(
    support, objective, ownership, clock, data, previous_report, references
):
    validate_source(data, support, clock)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        len(selected) == len(previous_report["rows"]),
        "attempt previous row count mismatch",
    )
    prefixes = origin_prefixes(support, objective, ownership, clock, data)
    cuts = {r["source_sequence"]: r for r in data["observations"]["cuts"]}
    owner_cuts = {r["source_sequence"]: r for r in ownership["observations"]["cuts"]}
    rows, matrix, counts = [], {}, {k: Counter() for k in FIELDS}
    for anchor, previous in zip(selected, previous_report["rows"], strict=True):
        a.require(
            all(anchor[k] == previous[k] for k in o.KEYS),
            "attempt previous anchor mismatch",
        )
        seq = anchor["source_sequence"]
        cut = cuts[seq]
        row = (
            data["observations"]["frames"][cut["frame_sequence"] - 1]
            if cut["frame_sequence"]
            else None
        )
        owner = ownership["observations"]["states"][
            owner_cuts[seq]["state_sequence"] - 1
        ]
        initial, proof = initial_context(
            anchor,
            previous["initial"],
            row,
            prefixes.get(cut["frame_sequence"], {}),
            owner,
        )
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
            previous["competitive_round_status"] == "canonical_interval_matched"
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
                "competitive_round_status": previous["competitive_round_status"],
                "initial": initial,
                "origin_reference": proof,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    report = {
        "schema_id": "stratai.support_attempt_origin_context@1",
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
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(report)
    return report


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 512 * 1024 * 1024,
        "invalid attempt origin source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    clock, ownership, objective, support, base = c.load_source(path.parent / c.DATA)
    expected = {
        **clock["references"],
        "clock_source": p.file_reference(path.parent / c.DATA),
    }
    a.require(data["references"] == expected, "attempt source hash mismatch")
    validate_source(data, support, clock)
    a.require(path.read_bytes() == raw, "attempt source changed")
    return data, clock, ownership, objective, support, base


def report_inputs(source, output):
    data, clock, ownership, objective, support, _ = load_source(source)
    previous = json.loads((output / c.REPORT).read_bytes())
    refs = {
        **previous["references"],
        "attempt_source": p.file_reference(source),
        "clock_context": p.file_reference(output / c.REPORT),
    }
    return support, objective, ownership, clock, data, previous, refs


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = c.materialize(source.parent / c.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["anchors"] > 0, "empty attempt origin context")
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    c.validate_artifact(output, source.parent / c.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "attempt report recomputation mismatch",
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
