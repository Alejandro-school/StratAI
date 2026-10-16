"""ICT-I1.A: deterministic own/ally prefix ledger, without inferred intentions.

The model-facing API projects before validation. Full-file integrity belongs to
the offline reader, and its hashes never become a temporal model feature.
"""

from __future__ import annotations

from copy import deepcopy
import hashlib
import json
from functools import lru_cache

from ai_coach.support import anchors as a
from ai_coach.support import c4, physical, preparation as p

VERSION = "1.0.0-candidate.1"
SCHEMA = "temporal_description.json"
CONFIGURATION = "own_ally_sampled_description@1"
HEAD = (
    "schema_id",
    "contract_version",
    "source_alias",
    "source_sha256",
    "build_identifier",
    "comparison_allowed",
    "training_allowed",
    "product_supported",
    "capture_contract",
)
CLOCK = (
    "sequence",
    "frame",
    "tick",
    "epoch",
    "instance",
    "round_number",
    "tick_rate",
    "sample_step",
    "phase",
)
PLAYER = (
    "actor_alias",
    "side",
    "connected",
    "pawn_handle",
    "pawn_life",
    "identity_status",
    "position",
    "alive",
    "position_status",
    "alive_status",
)
UNKNOWN = (
    "shared_knowledge",
    "legal_enemy_contact",
    "semantic_zone",
    "physical_route",
    "instantaneous_velocity",
    "full_weapon_readiness",
    "intent",
    "tactical_plan",
)
REQUIREMENTS = {
    "ICT-I1-E01": [
        "attributed_contact_or_utility_indication",
        "recipient_and_time",
        "distinct_defenders_if_counted",
        "spatial_link",
    ],
    "ICT-I1-E02": [
        "attributed_kill_information",
        "recipient_and_time",
        "mid_spatial_link",
        "ally_opportunity",
        "all_team_at_B_exception",
        "route_feasibility",
    ],
    "ICT-I1-E03": [
        "attributed_pressure_and_utility",
        "spatial_link",
        "prior_context",
        "fake_intent_unobserved",
        "CT_rotation_unobserved",
        "A_access_feasibility",
    ],
    "ICT-I1-E04": [
        "attributed_death_information",
        "recipient_and_time",
        "prior_information_role_context",
    ],
}


def digest(value):
    return hashlib.sha256(
        json.dumps(
            value, sort_keys=True, separators=(",", ":"), allow_nan=False
        ).encode()
    ).hexdigest()


def pick(value, keys):
    return {key: deepcopy(value[key]) for key in keys}


def ordered_prefix(rows, sequence):
    """Consume locators only through the first later row, never its payload."""
    for row in rows:
        if row["source_sequence"] > sequence:
            break
        yield row
        if row["source_sequence"] == sequence:
            break


def support_prefix(source, side, sequence):
    a.require(side in ("CT", "T"), "unsupported perspective side")
    a.require(type(sequence) is int and sequence > 0, "invalid exact sequence")
    frames = source["observations"]["frames"][:sequence]
    a.require(len(frames) == sequence, "missing exact cut")
    projected = []
    for frame in frames:
        row = pick(frame, CLOCK)
        row["players"] = [
            pick(player, PLAYER)
            for player in frame["players"]
            if player["side"] == side
        ]
        projected.append(row)
    result = pick(source, (*HEAD, "base_manifest"))
    result["observations"] = {"frames": projected}
    a.validate_source(result)
    return result


def physical_prefix(source, support):
    frames = support["observations"]["frames"]
    rows = []
    for row in ordered_prefix(source["observations"]["frames"], len(frames)):
        seq = row["source_sequence"]
        a.require(
            type(seq) is int and 0 < seq <= len(frames), "invalid physical locator"
        )
        aliases = {x["actor_alias"] for x in frames[seq - 1]["players"]}
        rows.append(
            {
                "source_sequence": seq,
                "players": [
                    pick(player, ("actor_alias", "context"))
                    for player in row["players"]
                    if player["actor_alias"] in aliases
                ],
            }
        )
    result = pick(source, (*HEAD, "support_source"))
    result["observations"] = {"frames": rows}
    physical.validate_source(result, support)
    return result


def ownership_prefix(source, objective, support):
    """C4 inventory/owner evidence only. Objective orders are join locators.

    No objective events, enemy inventory, outcome or location enter the ledger.
    The offline reader separately validates the original objective stream.
    """
    frames = support["observations"]["frames"]
    cuts = [
        pick(
            row,
            (
                "source_sequence",
                "frame",
                "tick",
                "epoch",
                "instance",
                "objective_order",
                "state_sequence",
            ),
        )
        for row in ordered_prefix(source["observations"]["cuts"], len(frames))
    ]
    objectives = []
    for row in ordered_prefix(objective["observations"]["frames"], len(frames)):
        seq = row["source_sequence"]
        a.require(
            type(seq) is int and 0 < seq <= len(frames), "invalid objective locator"
        )
        a.require(
            all(
                row[k] == frames[seq - 1][k]
                for k in ("frame", "tick", "epoch", "instance")
            ),
            "objective clock mismatch",
        )
        objectives.append(pick(row, ("source_sequence", "order")))
    a.require(
        len({r["source_sequence"] for r in objectives}) == len(objectives),
        "duplicate objective locator",
    )
    states = []
    for state in source["observations"]["states"][
        : cuts[-1]["state_sequence"] if cuts else 0
    ]:
        first = state["first_cut"]
        a.require(type(first) is int and 0 < first <= len(frames), "future C4 state")
        aliases = {x["actor_alias"] for x in frames[first - 1]["players"]}
        players = [deepcopy(x) for x in state["players"] if x["actor_alias"] in aliases]
        lives = {(x["pawn_handle"], x["pawn_life"]) for x in players if x["pawn_life"]}
        referenced = {
            s["target_life"]
            for x in players
            for s in x["slots"]
            if s["status"] == "resolved"
        }
        entities = [
            deepcopy(e)
            for e in state["entities"]
            if e["entity_life"] in referenced
            or (e["owner"]["handle"], e["owner"]["target_life"]) in lives
        ]
        states.append(
            {
                "sequence": state["sequence"],
                "first_cut": first,
                "players": players,
                "entities": entities,
            }
        )
    result = pick(source, (*HEAD, "support_source", "objective_source"))
    result["observations"] = {"cuts": cuts, "states": states}
    obj = pick(objective, HEAD)
    obj["observations"] = {"frames": objectives}
    c4.validate_source(result, support, obj)
    return result


def unavailable(reason):
    return {"status": "unavailable", "value": None, "reason": reason}


def fact(value):
    return {"status": "observed", "value": deepcopy(value), "reason": None}


def reference(source, cut, alias):
    return {
        "artifact": source,
        "source_sequence": cut["sequence"],
        "availability_frame": cut["frame"],
        "availability_tick": cut["tick"],
        "epoch": cut["epoch"],
        "instance": cut["instance"],
        "actor_alias": alias,
    }


def describe_cut(support, cut, previous, prior_record, side, physical_index, c4_index):
    index, _, reasons = p.roster(cut)
    old, old_reasons = (
        (p.roster(previous)[0], p.roster(previous)[2]) if previous else ({}, {})
    )
    continuous = bool(
        previous
        and p.clock_ok(previous)
        and p.clock_ok(cut)
        and a.compatible([previous, cut])
    )
    actors = []
    for alias in sorted(index):
        player, reason = index[alias], reasons[alias]
        usable = reason is None and cut["phase"] == "playing" and p.clock_ok(cut)
        state_reason = reason or ("cut_unavailable" if not usable else None)
        native = physical_index.get(cut["sequence"], {}).get(alias) if usable else None
        facts = (
            deepcopy(native["facts"])
            if native
            else {
                key: unavailable(state_reason or "missing_exact_physical_sample")
                for key in physical.FIELDS
            }
        )
        state = c4_index.get(cut["sequence"]) if usable else None
        ownership_player = (
            next((x for x in state["players"] if x["actor_alias"] == alias), None)
            if state
            else None
        )
        has_c4 = (
            c4.possession(ownership_player, state["entities"])
            if state
            else unavailable(state_reason or "missing_exact_c4_sample")
        )
        prior = old.get(alias)
        change_reason = (
            "no_previous_sample"
            if previous is None
            else "sample_gap_or_boundary"
            if not continuous
            else "previous_actor_unavailable"
            if not prior or old_reasons[alias]
            else "identity_changed"
            if player
            and any(prior[k] != player[k] for k in ("pawn_life", "pawn_handle", "side"))
            else state_reason
        )
        displacement = (
            unavailable(change_reason)
            if change_reason
            else fact(
                {
                    axis: player["position"][axis] - prior["position"][axis]
                    for axis in ("x", "y", "z")
                }
            )
        )
        actors.append(
            {
                "actor_alias": alias,
                "exclusion": state_reason,
                "identity": pick(
                    player, ("side", "pawn_handle", "pawn_life", "identity_status")
                )
                if player
                else None,
                "connected": player["connected"] if player else None,
                "alive": fact(player["alive"])
                if player
                and player["alive_status"] == "observed"
                and reason
                not in (
                    "ambiguous_actor_alias",
                    "shared_pawn_reference",
                    "native_identity_unavailable",
                )
                else unavailable(reason or "alive_unavailable"),
                "position": fact(player["position"])
                if usable
                else unavailable(state_reason),
                "physical": facts,
                "has_c4": has_c4,
                "sampled_displacement": displacement,
                "references": [
                    reference(a.DATA, cut, alias),
                    *([reference(physical.DATA, cut, alias)] if native else []),
                    *([reference(c4.DATA, cut, alias)] if state else []),
                ],
            }
        )
    missing = sorted(set(old) - set(index)) if continuous else []
    # Hash only allowed, perspective-filtered facts; whole artifact hashes contain future.
    row = {
        "schema_id": "stratai.temporal_description@1",
        "contract_version": VERSION,
        "configuration": CONFIGURATION,
        "source_alias": support["source_alias"],
        "source_sha256": support["source_sha256"],
        "build_identifier": support["build_identifier"],
        "cut": pick(cut, CLOCK),
        "perspective": {
            "side": side,
            "kind": "analytical_own_ally",
            "shared_knowledge_accredited": False,
        },
        "previous_record_id": prior_record["record_id"] if prior_record else None,
        "continuity": "adjacent_samples" if continuous else "unavailable",
        "actors": actors,
        "missing_previous_actors": missing,
        "missing_actor_meaning": "no_current_sample_not_a_death_or_retreat",
        "resolution": "source_sampled_world_coordinates_no_extra_precision_or_zone",
        "hypotheses": [
            {
                "criterion_id": key,
                "status": "unavailable",
                "unmet_requirements": list(value),
                "supporting_signals": [],
            }
            for key, value in REQUIREMENTS.items()
        ],
        "unknowns": list(UNKNOWN),
        "recommendations": [],
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }
    row["record_id"] = digest(row)
    return row


def iter_records(
    source,
    *,
    side,
    through_sequence,
    physical_source=None,
    c4_source=None,
    objective_source=None,
):
    """Read each permitted prefix once, then construct records in arrival order."""
    support = support_prefix(source, side, through_sequence)
    phy = (
        physical_prefix(physical_source, support)
        if physical_source is not None
        else None
    )
    a.require(
        (c4_source is None) == (objective_source is None),
        "C4 requires objective join source",
    )
    own = (
        ownership_prefix(c4_source, objective_source, support)
        if c4_source is not None
        else None
    )
    physical_index = physical.physical_index(phy) if phy else {}
    c4_index = (
        {
            r["source_sequence"]: own["observations"]["states"][r["state_sequence"] - 1]
            for r in own["observations"]["cuts"]
        }
        if own
        else {}
    )
    previous = prior_record = None
    for cut in support["observations"]["frames"]:
        row = describe_cut(
            support, cut, previous, prior_record, side, physical_index, c4_index
        )
        previous, prior_record = cut, {"record_id": row["record_id"]}
        yield row


@lru_cache(maxsize=1)
def validator():
    return a.schema_validator(SCHEMA)


def validate_record(row):
    validator().validate(row)
    a.require(
        row["record_id"] == digest({k: v for k, v in row.items() if k != "record_id"}),
        "temporal record hash mismatch",
    )
    for actor in row["actors"]:
        for ref in actor["references"]:
            a.require(
                all(ref[key] == row["cut"][key] for key in ("epoch", "instance"))
                and ref["source_sequence"] == row["cut"]["sequence"]
                and ref["availability_tick"] == row["cut"]["tick"]
                and ref["availability_frame"] == row["cut"]["frame"]
                and ref["actor_alias"] == actor["actor_alias"],
                "temporal provenance mismatch",
            )
