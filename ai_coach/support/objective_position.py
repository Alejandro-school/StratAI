"""Native objective geometry and separately accredited player availability."""

import argparse
import copy
import json
import math
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import attempt_origin as r
from ai_coach.support import c4
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as c
from ai_coach.support import physical as p

DATA = "support_objective_position_observations.json"
REPORT = "support_objective_position_context.json"
PLAYER = (
    "physical_objective_position",
    "objective_object_state",
    "position_availability",
)
FIELDS = tuple(role + "." + field for role in ("self", "ally") for field in PLAYER)
BODY = "CBodyComponent."
AXES = ("X", "Y", "Z")
PROBES = [
    *(BODY + "m_cell" + x for x in AXES),
    *(BODY + "m_vec" + x for x in AXES),
    *(
        BODY + x
        for x in (
            "m_hParent",
            "m_angRotation",
            "m_flScale",
            "m_name",
            "m_hierarchyAttachName",
            "m_hParent.m_hOwner",
            "m_hParent.m_name",
            "m_vecAbsOrigin",
            "m_nodeToWorld",
        )
    ),
    "m_hMoveParent",
    "m_iParentAttachment",
    "m_vecAbsOrigin",
    "m_vecOrigin",
    "m_hOwnerEntity",
    "m_hBombDefuser",
    "m_bBombTicking",
    "m_bBombDefused",
    "m_iItemDefinitionIndex",
    "m_fFlags",
    "m_bStartedArming",
    "m_entitySpottedState.m_bSpotted",
    "m_entitySpottedState.m_bSpottedByMask.0000",
    "m_entitySpottedState.m_bSpottedByMask.0001",
]
LIMITS = [
    *(
        x
        for x in r.LIMITS
        if x != "physical_c4_and_dropped_or_planted_position_not_accredited"
    ),
    "native_object_geometry_is_not_automatically_legal_player_information",
    "parented_object_requires_an_accredited_current_world_transform",
    "spotted_flag_past_carrier_and_global_plant_do_not_reveal_exact_position",
]


def validate_scalar(s):
    if s["native_type"] in ("[]float32", "[]float64", "[3]float32", "[3]float64"):
        a.require(
            (s["status"] == "observed") == (s["value"] is not None),
            "position vector presence mismatch",
        )
        if s["status"] == "observed":
            a.require(
                isinstance(s["value"], list)
                and len(s["value"]) == 3
                and all(
                    type(v) in (int, float) and math.isfinite(v) for v in s["value"]
                ),
                "invalid native position vector",
            )
        a.require(s["status"] != "absent", "vector has absent native type")
    else:
        c.validate_scalar(s)


def raw_integer(s):
    value = s["value"]
    return (
        value
        if s["status"] == "observed"
        and s["native_type"] in c.INTEGERS
        and type(value) is int
        and 0 <= value <= 4294967295
        else None
    )


def validate_reference(ref, scalar):
    c4.validate_reference(ref)
    value = raw_integer(scalar)
    if scalar["status"] == "absent":
        a.require(ref["status"] == "absent", "position reference without native value")
    elif value is None:
        a.require(
            ref["status"] == "unsupported_type",
            "position reference native type mismatch",
        )
    else:
        a.require(ref["handle"] == value, "position native handle mismatch")
        a.require(
            (ref["status"] == "invalid_sentinel") == (value & 16383 == 16383),
            "position native sentinel mismatch",
        )


def validate_source(data, support, ownership, clock):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == ownership[key] == clock[key],
            "position source identity mismatch",
        )
    obs = data["observations"]
    deliveries, births, states, cuts = (
        obs[k] for k in ("deliveries", "births", "states", "cuts")
    )
    created, last, destroyed = {}, {}, set()
    delivery_frames = []
    for seq, d in enumerate(deliveries, 1):
        a.require(
            d["sequence"] == seq
            and (not delivery_frames or d["frame"] >= delivery_frames[-1]),
            "position delivery order mismatch",
        )
        delivery_frames.append(d["frame"])
        life, name, kind = d["entity_life"], d["property"], d["kind"]
        a.require(life not in destroyed, "position delivery after destruction")
        validate_scalar(d["previous"])
        validate_scalar(d["value"])
        if kind == "destroyed":
            a.require(
                life in created and name == "" and d["value"]["status"] == "absent",
                "invalid position destruction",
            )
            destroyed.add(life)
            continue
        a.require(
            name in PROBES or name.startswith(BODY),
            "position field outside native probe",
        )
        key = life, name
        if kind == "created":
            a.require(
                key not in last and created.setdefault(life, d["frame"]) == d["frame"],
                "position repeated creation",
            )
            a.require(
                d["previous"]["status"] == "absent",
                "position fabricated creation predecessor",
            )
        else:
            a.require(
                key in last and d["previous"] == last[key],
                "position predecessor mismatch",
            )
            a.require(
                (kind == "creation_replay") == (d["frame"] == created[life]),
                "position creation replay misclassified",
            )
        last[key] = d["value"]
    identities, birth_by_life = {}, {}
    available, ended, frontier = {}, set(), 0

    def advance(end):
        nonlocal frontier
        a.require(frontier <= end <= len(deliveries), "position frontier regressed")
        for d in deliveries[frontier:end]:
            if d["kind"] == "destroyed":
                ended.add(d["entity_life"])
            else:
                available[d["entity_life"], d["property"]] = d
        frontier = end

    def check_object(item):
        life = item["entity_life"]
        a.require(
            life in created
            and life not in ended
            and item["created_frame"] == created[life],
            "position object lifetime mismatch",
        )
        identity = (
            item["handle"],
            item["class_name"],
            item["created_frame"],
            sorted(item["fields"]),
        )
        a.require(
            identities.setdefault(life, identity) == identity,
            "position object identity changed",
        )
        a.require(
            life.startswith(
                "support-item-"
                if item["class_name"] == "CC4"
                else "support-clock-object-"
            ),
            "position object class/life mismatch",
        )
        a.require(
            set(PROBES) <= item["fields"].keys() == item["field_sequences"].keys(),
            "position probe fields missing",
        )
        for name, value in item["fields"].items():
            d = available.get((life, name))
            a.require(
                d is not None
                and d["sequence"] == item["field_sequences"][name]
                and d["value"] == value,
                "position snapshot not equal to delivered prefix",
            )
        for key, field in (
            ("owner", "m_hOwnerEntity"),
            ("defuser", "m_hBombDefuser"),
            ("parent", BODY + "m_hParent"),
        ):
            validate_reference(item[key], item["fields"][field])

    # Merge birth seals and support cuts by their native frame. Both read the
    # settled frame, while separate sequence frontiers exclude future data.
    birth_frames = []
    for seq, birth in enumerate(births, 1):
        a.require(
            birth["sequence"] == seq
            and (not birth_frames or birth["frame"] >= birth_frames[-1]),
            "position birth order mismatch",
        )
        birth_frames.append(birth["frame"])
        item = birth["object"]
        a.require(
            item["entity_life"] not in birth_by_life
            and birth["frame"] == item["created_frame"],
            "position birth not sealed at creation frame",
        )
        birth_by_life[item["entity_life"]] = birth
    a.require(
        len(cuts) == len(support["observations"]["frames"]),
        "position cuts do not cover support",
    )
    clock_cuts = {x["source_sequence"]: x for x in clock["observations"]["frames"]}
    owner_cuts = {x["source_sequence"]: x for x in ownership["observations"]["cuts"]}
    expected_state, seen_states = 0, set()
    timeline = [(b["frame"], 0, b) for b in births] + [
        (cut["frame"], 1, cut) for cut in cuts
    ]
    for frame, kind, row in sorted(
        timeline,
        key=lambda x: (x[0], x[1], x[2].get("sequence", x[2].get("source_sequence"))),
    ):
        end = bisect_right(delivery_frames, frame)
        a.require(
            row["delivery_sequence"] == end, "position delivery frontier mismatch"
        )
        advance(end)
        if kind == 0:
            check_object(row["object"])
            continue
        seq = row["source_sequence"]
        a.require(1 <= seq <= len(cuts), "position source sequence invalid")
        base, old = support["observations"]["frames"][seq - 1], clock_cuts.get(seq)
        a.require(
            old is not None
            and all(
                row[k] == base[k] == old[k]
                for k in ("frame", "tick", "instance", "epoch")
            ),
            "position exact cut mismatch",
        )
        a.require(
            row["clock_order"] == old["order"]
            and row["birth_sequence"] == bisect_right(birth_frames, frame),
            "position common clock/birth frontier mismatch",
        )
        state_seq = row["state_sequence"]
        a.require(
            1 <= state_seq <= len(states)
            and state_seq in (expected_state, expected_state + 1),
            "position state order mismatch",
        )
        state = states[state_seq - 1]
        a.require(
            state["sequence"] == state_seq and state["first_cut"] <= seq,
            "position state before first observation",
        )
        if state_seq not in seen_states:
            a.require(state["first_cut"] == seq, "position state first-cut mismatch")
            seen_states.add(state_seq)
        elif seq > 1:
            prior = cuts[seq - 2]
            a.require(
                prior["instance"] == row["instance"] and prior["epoch"] == row["epoch"],
                "position state crosses round/epoch",
            )
        expected_state = state_seq
        expected = {x["entity_life"]: x for x in [*old["items"], *old["bombs"]]}
        objects = {x["entity_life"]: x for x in state["objects"]}
        a.require(
            len(objects) == len(state["objects"]) and objects.keys() == expected.keys(),
            "position native object census mismatch",
        )
        own = ownership["observations"]["states"][owner_cuts[seq]["state_sequence"] - 1]
        for life, item in objects.items():
            check_object(item)
            birth = birth_by_life.get(life)
            a.require(
                birth is not None and birth["frame"] <= frame,
                "position object without available birth",
            )
            a.require(
                (item["handle"], item["class_name"])
                == (expected[life]["handle"], expected[life]["class_name"]),
                "position common object identity mismatch",
            )
            for name in set(item["fields"]) & expected[life]["fields"].keys():
                a.require(
                    item["fields"][name] == expected[life]["fields"][name],
                    "position shared native field mismatch",
                )
            if item["class_name"] == "CC4":
                old_item = next(
                    (x for x in own["entities"] if x["entity_life"] == life), None
                )
                a.require(
                    old_item is not None and old_item["owner"] == item["owner"],
                    "position C4 owner mismatch",
                )
            else:
                a.require(
                    expected[life]["defuser"] == item["defuser"],
                    "position defuser mismatch",
                )
    a.require(
        [x["source_sequence"] for x in cuts] == list(range(1, len(cuts) + 1))
        and expected_state == len(states),
        "position trailing state or cut gap",
    )


def geometry(item):
    fields = item["fields"]
    unavailable = lambda reason: o.fact(reason=reason)
    own = item["owner"]["status"]
    state = None
    if (
        item["class_name"] == "CC4"
        and raw_integer(fields["m_iItemDefinitionIndex"]) == 49
    ):
        state = (
            "carried"
            if own == "resolved"
            else "unowned"
            if own == "invalid_sentinel"
            else None
        )
    elif item["class_name"] == "CPlantedC4":
        ticking = o.native(fields["m_bBombTicking"])["value"]
        defused = o.native(fields["m_bBombDefused"])["value"]
        state = (
            "planted"
            if ticking is True and defused is False
            else "resolved"
            if ticking is False or defused is True
            else None
        )
    result = {
        "entity_life": item["entity_life"],
        "class_name": item["class_name"],
        "native_object_state": o.fact(state)
        if state
        else unavailable("native_object_state_unavailable"),
        "cell_position": unavailable("native_coordinates_unavailable"),
        "world_position": unavailable("native_coordinates_unavailable"),
        "parent_state": unavailable("native_parent_unavailable"),
    }
    parent = fields[BODY + "m_hParent"]
    handle = raw_integer(parent)
    if handle is not None and parent["declared_type"] == "CGameSceneNodeHandle":
        result["parent_state"] = o.fact(
            "root_unparented" if handle & 16383 == 16383 else "parented"
        )
    xyz = {}
    for axis in AXES:
        cell, offset = fields[BODY + "m_cell" + axis], fields[BODY + "m_vec" + axis]
        value = raw_integer(cell)
        if value is None or value > 65535 or cell["declared_type"] != "uint16":
            return result
        if (
            offset["status"] != "observed"
            or offset["native_type"] not in ("float32", "float64")
            or offset["declared_type"] not in ("CNetworkedQuantizedFloat", "float32")
            or type(offset["value"]) not in (int, float)
            or not math.isfinite(offset["value"])
        ):
            return result
        xyz[axis.lower()] = value * 512 - 16384 + offset["value"]
    result["cell_position"] = o.fact(xyz)
    if result["parent_state"]["value"] != "root_unparented":
        result["world_position"] = unavailable(
            "parent_transform_unavailable"
            if result["parent_state"]["value"] == "parented"
            else "native_parent_unavailable"
        )
        return result
    if item["parent"]["status"] != "invalid_sentinel":
        result["world_position"] = unavailable("native_parent_reference_conflict")
        return result
    alternate = raw_integer(fields["m_hMoveParent"])
    attachment = raw_integer(fields["m_iParentAttachment"])
    named_attachment = fields[BODY + "m_hierarchyAttachName"]
    if (
        (alternate is not None and alternate & 16383 != 16383)
        or (attachment is not None and attachment != 0)
        or (
            named_attachment["status"] != "absent"
            and raw_integer(named_attachment) != 0
        )
    ):
        result["world_position"] = unavailable("native_parent_attachment_conflict")
        return result
    for name in ("m_vecAbsOrigin", BODY + "m_vecAbsOrigin"):
        value = fields[name]
        if value["status"] == "observed" and (
            not isinstance(value["value"], list)
            or any(
                not math.isclose(xyz[axis.lower()], v, rel_tol=0, abs_tol=1e-6)
                for axis, v in zip(AXES, value["value"], strict=True)
            )
        ):
            result["world_position"] = unavailable("native_absolute_origin_conflict")
            return result
    result["world_position"] = o.fact(xyz)
    return result


def planting_witnesses(support, objective, ownership, clock, attempt, data):
    """A native new planted object corroborates a previously observed own plant.

    A public plant event alone grants no position. This witness additionally
    needs the exact pawn's accredited native attempt in the preceding frame,
    its destruction at creation, the new object's owner and a root position.
    """
    prefixes = r.origin_prefixes(support, objective, ownership, clock, attempt)
    alignments, _ = c.alignment_prefixes(support, objective, clock)
    clock_frames = clock["observations"]["frames"]
    clock_times = [x["frame"] for x in clock_frames]
    frames = {x["frame"]: x for x in attempt["observations"]["frames"]}
    ends = {
        (x["frame"], x["entity_life"])
        for x in attempt["observations"]["deliveries"]
        if x["kind"] == "destroyed"
    }
    events = {
        (x["frame"], x["instance"], x["epoch"])
        for x in objective["observations"]["events"]
        if x["kind"] == "bomb_planted"
    }
    births = data["observations"]["births"]
    result = {}
    for birth in births:
        item = birth["object"]
        if item["class_name"] != "CPlantedC4" or item["owner"]["status"] != "resolved":
            continue
        frame = birth["frame"]
        if (frame, birth["instance"], birth["epoch"]) not in events:
            continue
        previous = frames.get(frame - 1)
        if previous is None or (previous["instance"], previous["epoch"]) != (
            birth["instance"],
            birth["epoch"],
        ):
            continue
        owner = item["owner"]
        candidates = [
            state
            for life, state in prefixes.get(previous["sequence"], {}).items()
            if state["proof"] is not None
            and (frame, life) in ends
            and (state["proof"]["pawn_life"], state["proof"]["pawn_handle"])
            == (owner["target_life"], owner["handle"])
        ]
        simultaneous = [
            x
            for x in births
            if x["frame"] == frame
            and x["object"]["class_name"] == "CPlantedC4"
            and x["object"]["owner"] == owner
        ]
        geo = geometry(item)
        if (
            len(candidates) != 1
            or len(simultaneous) != 1
            or geo["native_object_state"]["value"] != "planted"
            or geo["world_position"]["status"] != "observed"
        ):
            continue
        current = frames.get(frame)
        index = bisect_right(clock_times, frame) - 1
        alignment = (
            alignments.get(clock_frames[index]["source_sequence"])
            if index >= 0
            else None
        )
        now = r.frame_clock(current, alignment)[0] if current else None
        candidate = candidates[0]
        proof = candidate["proof"]
        if (
            now is None
            or now[3] != candidate["key"]
            or now[0] < proof["deadline_seconds"]
        ):
            continue
        result[item["entity_life"]] = {
            "birth_sequence": birth["sequence"],
            "frame": frame,
            "instance": birth["instance"],
            "epoch": birth["epoch"],
            "pawn_life": owner["target_life"],
            "pawn_handle": owner["handle"],
            "position": geo["world_position"]["value"],
            "delivery_sequence": birth["delivery_sequence"],
            "attempt_delivery_sequence": proof["delivery_sequence"],
        }
    return result


def position_changes(data):
    result = {}
    for d in data["observations"]["deliveries"]:
        if d["kind"] == "destroyed" or (
            d["kind"] == "update"
            and (
                d["value"] != d["previous"]
                or d["value"]["status"] in ("unsupported_type", "nonfinite")
            )
            and (
                d["property"].startswith(BODY)
                or d["property"]
                in (
                    "m_hMoveParent",
                    "m_iParentAttachment",
                    "m_vecAbsOrigin",
                    "m_vecOrigin",
                    "m_hOwnerEntity",
                )
            )
        ):
            result.setdefault(d["entity_life"], []).append(d["sequence"])
    return result


def unavailable(reason, status="unavailable"):
    return {k: o.fact(reason=reason, status=status) for k in PLAYER}


def initial_context(
    anchor, objective_initial, ownership_state, state, cut, witnesses, changes
):
    phase = objective_initial["global"]["objective_phase"]
    result, references = {}, {role: None for role in ("self", "ally")}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        if phase["status"] != "observed":
            result[role] = unavailable("objective_phase_unavailable")
            continue
        if phase["value"] == "resolved":
            result[role] = unavailable("objective_resolved", "not_applicable")
            continue
        player = next(
            (x for x in ownership_state["players"] if x["actor_alias"] == alias), None
        )
        if player is None or player["pawn_life"] is None:
            result[role] = unavailable("current_pawn_unavailable")
            continue
        pawn = player["pawn_life"], player["pawn_handle"]
        candidates = []
        for item in state["objects"]:
            basis, witness = None, None
            if (
                phase["value"] == "preplant"
                and item["class_name"] == "CC4"
                and c4.possession(player, ownership_state["entities"])["value"] is True
                and item["owner"]["status"] == "resolved"
                and (item["owner"]["target_life"], item["owner"]["handle"]) == pawn
            ):
                basis = "current_owned_c4"
            elif phase["value"] == "planted" and item["class_name"] == "CPlantedC4":
                if (
                    objective_initial[role]["defusing"]["value"] is True
                    and item["defuser"]["status"] == "resolved"
                    and (item["defuser"]["target_life"], item["defuser"]["handle"])
                    == pawn
                ):
                    basis = "current_defuse_contact"
                else:
                    proof = witnesses.get(item["entity_life"])
                    if (
                        proof
                        and (proof["pawn_life"], proof["pawn_handle"]) == pawn
                        and (proof["instance"], proof["epoch"])
                        == (cut["instance"], cut["epoch"])
                        and proof["frame"] <= cut["frame"]
                    ):
                        updates = changes.get(item["entity_life"], [])
                        if bisect_right(
                            updates, cut["delivery_sequence"]
                        ) == bisect_right(updates, proof["delivery_sequence"]):
                            basis, witness = "witnessed_stationary_plant", proof
            # Geometry of unobserved objects is not consulted to choose an
            # initial absence reason or to expose hidden ownership/location.
            if basis:
                candidates.append((item, basis, witness))
        if len(candidates) != 1:
            result[role] = unavailable(
                "legal_object_position_not_observed"
                if not candidates
                else "legal_object_identity_conflict"
            )
            continue
        item, basis, witness = candidates[0]
        geo = geometry(item)
        position = geo["world_position"]
        if position["status"] != "observed":
            result[role] = unavailable(position["reason"])
            continue
        expected = "carried" if basis == "current_owned_c4" else "planted"
        if geo["native_object_state"]["value"] != expected:
            result[role] = unavailable("current_object_state_conflict")
            continue
        if witness and witness["position"] != position["value"]:
            result[role] = unavailable("witnessed_position_changed")
            continue
        result[role] = {
            "physical_objective_position": copy.deepcopy(position),
            "objective_object_state": o.fact(expected),
            "position_availability": o.fact(basis),
        }
        references[role] = {
            "basis": basis,
            "entity_life": item["entity_life"],
            "source_sequence": cut["source_sequence"],
            "frame": cut["frame"],
            "availability_frame": witness["frame"] if witness else cut["frame"],
            "birth_sequence": witness["birth_sequence"] if witness else None,
        }
    return result, references


def prefix_source(data, seq):
    obs = data["observations"]
    cuts = [x for x in obs["cuts"] if x["source_sequence"] <= seq]
    a.require(cuts and cuts[-1]["source_sequence"] == seq, "missing exact position cut")
    cut = cuts[-1]
    return {
        **{k: v for k, v in data.items() if k != "observations"},
        "observations": {
            "cuts": cuts,
            "states": [
                x for x in obs["states"] if x["sequence"] <= cut["state_sequence"]
            ],
            "births": [
                x for x in obs["births"] if x["sequence"] <= cut["birth_sequence"]
            ],
            "deliveries": [
                x
                for x in obs["deliveries"]
                if x["sequence"] <= cut["delivery_sequence"]
            ],
        },
    }


def initial_features(
    support, objective, ownership, clock, attempt, data, anchor_id, actor, tick, frame
):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support, objective, ownership, clock = c.prefix_inputs(
        support, objective, ownership, clock, seq
    )
    attempt, data = r.prefix_source(attempt, frame, seq), prefix_source(data, seq)
    c4.validate_source(ownership, support, objective)
    c.validate_source(clock, support, objective, ownership)
    r.validate_source(attempt, support, clock)
    validate_source(data, support, ownership, clock)
    initial = o.initial_features(support, objective, anchor_id, actor, tick, frame)
    anchor = next(
        x
        for x in a.select_anchors(
            support["observations"]["frames"], support["source_alias"]
        )[0]
        if x["anchor_id"] == anchor_id
    )
    cut = data["observations"]["cuts"][-1]
    owner_cut = next(
        x for x in ownership["observations"]["cuts"] if x["source_sequence"] == seq
    )
    state = data["observations"]["states"][cut["state_sequence"] - 1]
    owner = ownership["observations"]["states"][owner_cut["state_sequence"] - 1]
    return initial_context(
        anchor,
        initial,
        owner,
        state,
        cut,
        planting_witnesses(support, objective, ownership, clock, attempt, data),
        position_changes(data),
    )[0]


def build_report(
    support,
    objective,
    ownership,
    clock,
    attempt,
    data,
    previous_report,
    objective_report,
    references,
):
    validate_source(data, support, ownership, clock)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        len(selected) == len(previous_report["rows"]) == len(objective_report["rows"]),
        "position previous row count mismatch",
    )
    witnesses = planting_witnesses(support, objective, ownership, clock, attempt, data)
    changes = position_changes(data)
    cuts = {x["source_sequence"]: x for x in data["observations"]["cuts"]}
    owner_cuts = {x["source_sequence"]: x for x in ownership["observations"]["cuts"]}
    rows, matrix, counts = [], {}, {k: Counter() for k in FIELDS}
    for anchor, previous, public in zip(
        selected, previous_report["rows"], objective_report["rows"], strict=True
    ):
        a.require(
            all(anchor[k] == previous[k] == public[k] for k in o.KEYS),
            "position previous anchor mismatch",
        )
        cut = cuts[anchor["source_sequence"]]
        state = data["observations"]["states"][cut["state_sequence"] - 1]
        owner = ownership["observations"]["states"][
            owner_cuts[anchor["source_sequence"]]["state_sequence"] - 1
        ]
        initial, proof = initial_context(
            anchor, public["initial"], owner, state, cut, witnesses, changes
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
                "position_reference": proof,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    native_geometry, native_counts = [], Counter()
    for state in data["observations"]["states"]:
        objects = [geometry(item) for item in state["objects"]]
        native_geometry.append(
            {
                "state_sequence": state["sequence"],
                "first_cut": state["first_cut"],
                "objects": objects,
            }
        )
        for item in objects:
            for name in (
                "native_object_state",
                "cell_position",
                "world_position",
                "parent_state",
            ):
                fact = item[name]
                status = fact["status"] + (
                    ":" + fact["reason"]
                    if fact["reason"]
                    else ":" + str(fact["value"])
                    if name in ("native_object_state", "parent_state")
                    else ""
                )
                native_counts[item["class_name"] + "." + name + "." + status] += 1
    report = {
        "schema_id": "stratai.support_objective_position_context@1",
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
        "native_geometry": native_geometry,
        "native_geometry_is_initial_information": False,
        "summary": {
            "anchors": len(rows),
            "matrix_rows": len(matrix),
            "canonical_interval_matched": sum(
                v["canonical_interval_matched"] for v in matrix.values()
            ),
            "fact_status_counts": counts,
            "native_geometry_counts": native_counts,
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(report)
    return report


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 1024 * 1024 * 1024,
        "invalid objective position source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    attempt, clock, ownership, objective, support, base = r.load_source(
        path.parent / r.DATA
    )
    expected = {
        **attempt["references"],
        "attempt_source": p.file_reference(path.parent / r.DATA),
    }
    a.require(data["references"] == expected, "position source hash mismatch")
    validate_source(data, support, ownership, clock)
    a.require(path.read_bytes() == raw, "position source changed")
    return data, attempt, clock, ownership, objective, support, base


def report_inputs(source, output):
    data, attempt, clock, ownership, objective, support, _ = load_source(source)
    previous = json.loads((output / r.REPORT).read_bytes())
    public = json.loads((output / o.REPORT).read_bytes())
    refs = {
        **previous["references"],
        "position_source": p.file_reference(source),
        "attempt_context": p.file_reference(output / r.REPORT),
    }
    return support, objective, ownership, clock, attempt, data, previous, public, refs


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = r.materialize(source.parent / r.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(
        report["summary"]["anchors"] > 0 and report["native_geometry"],
        "empty objective position context",
    )
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    r.validate_artifact(output, source.parent / r.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "position report recomputation mismatch",
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
