"""Native attack deadlines with current ownership and explicit clock limits."""

import argparse
import json
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import native_motion as m
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as c
from ai_coach.support import objective_position as z
from ai_coach.support import physical as p

DATA = "support_weapon_timing_observations.json"
REPORT = "support_weapon_timing_context.json"
PAWN = [
    "m_pWeaponServices.m_hActiveWeapon",
    "m_pWeaponServices.m_flNextAttack",
    "m_flSimulationTime",
    "m_nTickBase",
]
WEAPON = [
    "m_hOwnerEntity",
    "m_nNextPrimaryAttackTick",
    "m_flNextPrimaryAttackTickRatio",
    "m_nNextSecondaryAttackTick",
    "m_flNextSecondaryAttackTickRatio",
    "m_nPostponeFireReadyTicks",
    "m_flPostponeFireReadyFrac",
]
PLAYER = (
    "primary_remaining_seconds",
    "primary_deadline_state",
    "secondary_remaining_seconds",
    "secondary_deadline_state",
    "postpone_remaining_seconds",
    "postpone_deadline_state",
    "pawn_remaining_seconds",
    "pawn_deadline_state",
)
FIELDS = tuple(role + "." + field for role in ("self", "ally") for field in PLAYER)
LIMITS = [
    *m.LIMITS,
    "weapon_tick_fraction_clock_semantics_require_positive_source",
    "elapsed_native_deadline_does_not_establish_can_fire_or_complete_readiness",
    "pawn_service_deadline_does_not_identify_the_cause_of_a_block",
]


def validate_source(data, support, motion, physical, clock):
    a.schema_validator(DATA).validate(data)
    p.validate_source(physical, support)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            len({source[key] for source in (data, support, motion, physical, clock)})
            == 1,
            "timing source identity mismatch",
        )
    obs = data["observations"]
    deliveries, states, cuts = (obs[k] for k in ("deliveries", "states", "cuts"))
    created, last, destroyed, frames = {}, {}, set(), []
    for seq, delivery in enumerate(deliveries, 1):
        a.require(
            delivery["sequence"] == seq
            and (not frames or delivery["frame"] >= frames[-1]),
            "timing delivery order mismatch",
        )
        frames.append(delivery["frame"])
        life, name, kind = (delivery[k] for k in ("entity_life", "property", "kind"))
        a.require(life not in destroyed, "timing delivery after destruction")
        c.validate_scalar(delivery["previous"])
        c.validate_scalar(delivery["value"])
        if kind == "destroyed":
            a.require(
                life in created
                and name == ""
                and delivery["previous"]["status"] == "absent"
                and delivery["value"]["status"] == "absent",
                "invalid timing destruction",
            )
            destroyed.add(life)
            continue
        a.require(name in PAWN + WEAPON, "timing field outside probe contract")
        key = life, name
        if kind == "created":
            a.require(
                key not in last
                and created.setdefault(life, delivery["frame"]) == delivery["frame"]
                and delivery["previous"]["status"] == "absent",
                "invalid timing creation",
            )
        else:
            a.require(
                key in last and delivery["previous"] == last[key],
                "timing predecessor mismatch",
            )
            a.require(
                (kind == "creation_replay") == (delivery["frame"] == created[life]),
                "timing creation replay misclassified",
            )
        last[key] = delivery["value"]
    available, ended, frontier, state_frontier, identities = {}, set(), 0, 0, {}
    a.require(
        len(cuts)
        == len(support["observations"]["frames"])
        == len(motion["observations"]["cuts"])
        == len(clock["observations"]["frames"]),
        "timing cut coverage mismatch",
    )
    for seq, (cut, base, prior, cf, pf) in enumerate(
        zip(
            cuts,
            support["observations"]["frames"],
            motion["observations"]["cuts"],
            clock["observations"]["frames"],
            physical["observations"]["frames"],
            strict=True,
        ),
        1,
    ):
        a.require(
            cut["source_sequence"]
            == seq
            == base["sequence"]
            == prior["source_sequence"]
            == cf["source_sequence"]
            == pf["source_sequence"]
            and all(
                cut[k] == base[k] == prior[k] == cf[k]
                for k in ("frame", "tick", "instance", "epoch")
            )
            and cut["clock_order"] == prior["clock_order"] == cf["order"],
            "timing exact cut mismatch",
        )
        end = bisect_right(frames, cut["frame"])
        a.require(cut["delivery_sequence"] == end, "timing delivery frontier mismatch")
        for delivery in deliveries[frontier:end]:
            if delivery["kind"] == "destroyed":
                ended.add(delivery["entity_life"])
            else:
                available[delivery["entity_life"], delivery["property"]] = delivery
        frontier = end
        a.require(
            state_frontier <= cut["state_sequence"] <= len(states),
            "timing state frontier mismatch",
        )
        new_states = set(range(state_frontier + 1, cut["state_sequence"] + 1))
        used = set()

        def state_at(
            index, expected_kind, cut=cut, used=used, seq=seq, new_states=new_states
        ):
            a.require(
                1 <= index <= cut["state_sequence"], "timing state outside prefix"
            )
            state = states[index - 1]
            used.add(index)
            life = state["entity_life"]
            a.require(
                state["sequence"] == index
                and state["kind"] == expected_kind
                and state["first_cut"] <= seq
                and (state["instance"], state["epoch"])
                == (cut["instance"], cut["epoch"])
                and life in created
                and life not in ended
                and state["created_frame"] == created[life],
                "timing state lifetime/domain mismatch",
            )
            if index in new_states:
                a.require(state["first_cut"] == seq, "timing first cut mismatch")
            identity = (
                state["handle"],
                state["class_name"],
                state["kind"],
                state["created_frame"],
            )
            a.require(
                identities.setdefault(life, identity) == identity,
                "timing native identity changed",
            )
            names = PAWN if expected_kind == "pawn" else WEAPON
            a.require(
                set(state["fields"]) == set(state["field_sequences"]) == set(names),
                "timing probe fields mismatch",
            )
            for name, value in state["fields"].items():
                d = available.get((life, name))
                a.require(
                    d is not None
                    and value == d["value"]
                    and state["field_sequences"][name] == d["sequence"],
                    "timing snapshot not equal to delivered prefix",
                )
            z.validate_reference(state["reference"], state["fields"][names[0]])
            ref = state["reference"]
            if ref["status"] == "resolved":
                a.require(
                    ref["target_life"] in created and ref["target_life"] not in ended,
                    "timing reference to unavailable life",
                )
            return state

        a.require(
            len(cut["players"]) == len(base["players"]) == len(cf["players"]),
            "timing player census mismatch",
        )
        for player, actor, cp, pp in zip(
            cut["players"], base["players"], cf["players"], pf["players"], strict=True
        ):
            a.require(
                player["actor_alias"]
                == actor["actor_alias"]
                == cp["actor_alias"]
                == pp["actor_alias"],
                "timing actor order mismatch",
            )
            present = [
                player[k] is not None
                for k in ("pawn_handle", "pawn_life", "pawn_state_sequence")
            ]
            a.require(all(present) or not any(present), "partial timing pawn reference")
            if not any(present):
                a.require(
                    player["weapon_state_sequence"] is None
                    and player["weapon_life"] is None
                    and player["weapon_status"] == "current_pawn_unavailable",
                    "timing weapon without pawn",
                )
                continue
            a.require(
                actor["connected"]
                and actor["identity_status"] == "observed"
                and actor["alive"] is True
                and actor["alive_status"] == "observed"
                and actor["position_status"] == "observed"
                and actor["position"] is not None
                and player["pawn_life"] == actor["pawn_life"] == cp["pawn_life"]
                and player["pawn_handle"] == actor["pawn_handle"] == cp["pawn_handle"],
                "timing pawn is not current support actor",
            )
            pawn = state_at(player["pawn_state_sequence"], "pawn")
            a.require(
                pawn["handle"] == player["pawn_handle"]
                and pawn["class_name"] == "CCSPlayerPawn",
                "timing pawn identity mismatch",
            )
            identity = pawn["entity_life"], "support_pawn"
            a.require(
                identities.setdefault(identity, player["pawn_life"])
                == player["pawn_life"],
                "timing pawn life changed",
            )
            for name in ("m_flSimulationTime", "m_nTickBase"):
                a.require(
                    pawn["fields"][name] == cp["fields"][name],
                    "timing pawn clock source mismatch",
                )
            if player["weapon_state_sequence"] is None:
                a.require(
                    player["weapon_life"] is None
                    and player["weapon_status"] == "current_weapon_unavailable",
                    "timing unresolved weapon status mismatch",
                )
                continue
            weapon = state_at(player["weapon_state_sequence"], "weapon")
            active, owner = pawn["reference"], weapon["reference"]
            a.require(
                active["status"] == "resolved"
                and active["handle"] == weapon["handle"]
                and active["target_life"] == weapon["entity_life"],
                "timing active weapon mismatch",
            )
            owned = (
                owner["status"] == "resolved"
                and owner["handle"] == pawn["handle"]
                and owner["target_life"] == pawn["entity_life"]
            )
            physical_state = pp["context"]
            matched = (
                physical_state is not None
                and physical_state["weapon_identity_status"] == "observed"
                and physical_state["weapon_handle"] == weapon["handle"]
                and physical_state["weapon_life"] is not None
                and physical_state["facts"]["active_weapon"]["status"] == "observed"
            )
            expected = (
                "current_weapon_owner_unaccredited"
                if not owned
                else "physical_weapon_identity_unavailable"
                if not matched
                else "observed"
            )
            a.require(
                player["weapon_status"] == expected
                and player["weapon_life"]
                == (physical_state["weapon_life"] if expected == "observed" else None),
                "timing current owner/physical identity mismatch",
            )
            if expected == "observed":
                key = weapon["entity_life"], "physical_weapon"
                a.require(
                    identities.setdefault(key, player["weapon_life"])
                    == player["weapon_life"],
                    "timing physical weapon life changed",
                )
        a.require(new_states <= used, "unused timing state at first cut")
        state_frontier = cut["state_sequence"]
    a.require(state_frontier == len(states), "timing trailing state")


def missing(reason):
    return {k: o.fact(reason=reason) for k in PLAYER}


def tick_pair_reason(tick, fraction):
    """Presence/type checks do not prove a tick epoch or a fraction formula."""
    if tick["status"] != "observed" or fraction["status"] != "observed":
        return "native_weapon_tick_or_fraction_unavailable"
    value = z.raw_integer(tick)
    if value is None or tick["declared_type"] != "GameTick_t":
        return "native_weapon_tick_type_unaccredited"
    if value == 0 or value >= 2147483647:
        return "native_weapon_tick_zero_or_sentinel_unaccredited"
    if (
        fraction["declared_type"] != "float32"
        or fraction["native_type"] not in ("float32", "float64")
        or type(fraction["value"]) not in (int, float)
        or not 0 <= fraction["value"] < 1
    ):
        return "native_weapon_fraction_unaccredited"
    return "native_weapon_tick_fraction_clock_unaccredited"


def native_facts(player, pawn, weapon, physical, clock_row, clock_player, witness):
    result = missing(player["weapon_status"])
    for prefix, ti, fr in (
        ("primary", WEAPON[1], WEAPON[2]),
        ("secondary", WEAPON[3], WEAPON[4]),
        ("postpone", WEAPON[5], WEAPON[6]),
    ):
        if player["weapon_status"] != "observed":
            continue
        ammo = physical["facts"]["ammo_in_magazine"]
        if ammo["status"] == "not_applicable":
            fact = {
                "status": "not_applicable",
                "value": None,
                "reason": "non_ammunition_weapon",
            }
        elif ammo["reason"] == "unknown_weapon_class":
            fact = o.fact(reason="unknown_weapon_class")
        else:
            fact = o.fact(
                reason=tick_pair_reason(weapon["fields"][ti], weapon["fields"][fr])
            )
        for field in (prefix + "_remaining_seconds", prefix + "_deadline_state"):
            result[field] = dict(fact)
    deadline = pawn["fields"]["m_pWeaponServices.m_flNextAttack"]
    value = c.native(deadline, game_time=True)
    now, pause = (
        c.aligned_now(clock_row, clock_player, witness),
        c.pause_state(clock_row),
    )
    reason = value["reason"] or now["reason"] or pause["reason"]
    if value["status"] == "observed" and value["value"] == 0:
        reason = "native_pawn_zero_deadline_semantics_unaccredited"
    if reason:
        for field in PLAYER[-2:]:
            result[field] = o.fact(reason=reason)
    else:
        remaining = max(0, value["value"] - now["value"])
        result["pawn_remaining_seconds"] = o.fact(remaining)
        result["pawn_deadline_state"] = o.fact(
            "paused"
            if pause["value"] == "paused"
            else "pending"
            if remaining > 0
            else "elapsed"
        )
    return result


def initial_context(anchor, data, physical, clock, witness):
    seq = anchor["source_sequence"]
    cut, cf, pf = (
        data["observations"]["cuts"][seq - 1],
        clock["observations"]["frames"][seq - 1],
        physical["observations"]["frames"][seq - 1],
    )
    result, refs = {}, {}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        index = next(
            i for i, x in enumerate(cut["players"]) if x["actor_alias"] == alias
        )
        player = cut["players"][index]
        if player["pawn_state_sequence"] is None:
            result[role], refs[role] = missing("current_timing_pawn_unavailable"), None
            continue
        pawn = data["observations"]["states"][player["pawn_state_sequence"] - 1]
        weapon = (
            data["observations"]["states"][player["weapon_state_sequence"] - 1]
            if player["weapon_state_sequence"]
            else None
        )
        result[role] = native_facts(
            player,
            pawn,
            weapon,
            pf["players"][index]["context"],
            cf,
            cf["players"][index],
            witness,
        )
        refs[role] = {
            "source_sequence": seq,
            "frame": cut["frame"],
            "pawn_state_sequence": pawn["sequence"],
            "pawn_life": player["pawn_life"],
            "weapon_state_sequence": player["weapon_state_sequence"],
            "weapon_life": player["weapon_life"],
            "clock_witness_sequence": witness["source_sequence"] if witness else None,
        }
    return result, refs


def prefix_source(data, seq):
    return m.prefix_source(data, seq)


def initial_features(
    support,
    motion,
    physical,
    objective,
    ownership,
    clock,
    data,
    anchor_id,
    actor,
    tick,
    frame,
):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support, objective, ownership, clock = c.prefix_inputs(
        support, objective, ownership, clock, seq
    )
    motion, data = m.prefix_source(motion, seq), prefix_source(data, seq)
    physical = {
        **physical,
        "observations": {
            "frames": [
                f
                for f in physical["observations"]["frames"]
                if f["source_sequence"] <= seq
            ]
        },
    }
    o.validate_source(objective, support)
    c.validate_source(clock, support, objective, ownership)
    validate_source(data, support, motion, physical, clock)
    alignment, _ = c.alignment_prefixes(support, objective, clock)
    anchor = next(
        x
        for x in a.select_anchors(
            support["observations"]["frames"], support["source_alias"]
        )[0]
        if x["anchor_id"] == anchor_id
    )
    return initial_context(anchor, data, physical, clock, alignment[seq])[0]


def build_report(
    support, motion, physical, objective, ownership, clock, data, previous, references
):
    o.validate_source(objective, support)
    c.validate_source(clock, support, objective, ownership)
    validate_source(data, support, motion, physical, clock)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    alignment, witnesses = c.alignment_prefixes(support, objective, clock)
    a.require(len(selected) == len(previous["rows"]), "timing previous rows mismatch")
    counts, matrix, rows = {k: Counter() for k in FIELDS}, {}, []
    for anchor, old in zip(selected, previous["rows"], strict=True):
        a.require(
            all(anchor[k] == old[k] for k in o.KEYS), "timing previous anchor mismatch"
        )
        initial, ref = initial_context(
            anchor, data, physical, clock, alignment[anchor["source_sequence"]]
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
            old["competitive_round_status"] == "canonical_interval_matched"
        )
        for field in FIELDS:
            role, name = field.split(".")
            fact = initial[role][name]
            status = fact["status"] + (":" + fact["reason"] if fact["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        rows.append(
            {
                **{k: anchor[k] for k in o.KEYS},
                "competitive_round_status": old["competitive_round_status"],
                "initial": initial,
                "timing_reference": ref,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    source_counts = {name: Counter() for name in PAWN + WEAPON}
    for state in data["observations"]["states"]:
        for name, value in state["fields"].items():
            source_counts[name][value["status"]] += 1
    report = {
        "schema_id": "stratai.support_weapon_timing_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
        "rows": rows,
        "clock_witnesses": witnesses,
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
        path.name == DATA and path.stat().st_size <= 1024**3,
        "invalid weapon timing source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    motion, _, support, base = m.load_source(path.parent / m.DATA)
    physical = json.loads((path.parent / p.DATA).read_bytes())
    objective = json.loads((path.parent / o.DATA).read_bytes())
    clock = json.loads((path.parent / c.DATA).read_bytes())
    ownership = json.loads((path.parent / "support_c4_observations.json").read_bytes())
    expected = {
        **motion["references"],
        "motion_source": p.file_reference(path.parent / m.DATA),
        "physical_source": p.file_reference(path.parent / p.DATA),
    }
    a.require(data["references"] == expected, "timing source hash mismatch")
    validate_source(data, support, motion, physical, clock)
    a.require(path.read_bytes() == raw, "timing source changed")
    return support, motion, physical, objective, ownership, clock, data, base


def report_inputs(source, output):
    *sources, _ = load_source(source)
    previous = json.loads((output / m.REPORT).read_bytes())
    references = {
        **previous["references"],
        "timing_source": p.file_reference(source),
        "motion_context": p.file_reference(output / m.REPORT),
    }
    return *sources, previous, references


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = m.materialize(source.parent / m.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["anchors"] > 0, "empty weapon timing context")
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    m.validate_artifact(output, source.parent / m.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "timing report recomputation mismatch",
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
