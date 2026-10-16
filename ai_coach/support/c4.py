"""Exact native C4 possession; no cached carrier or physical bomb coordinates."""

import argparse
import copy
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import physical as p

DATA = "support_c4_observations.json"
REPORT = "support_c4_context.json"
PLAYER = ("has_c4", "carrier_player_position", "physical_c4_position")
FIELDS = tuple(role + "." + key for role in ("self", "ally") for key in PLAYER)
LIMITS = [
    *[x for x in o.LIMITS if x != "objective_position_and_carrier_not_accredited"],
    "physical_c4_and_dropped_or_planted_position_not_accredited",
    "ownership_does_not_establish_tactical_responsibility",
]


def validate_reference(ref):
    status, handle, life = ref["status"], ref["handle"], ref["target_life"]
    a.require(
        (status == "resolved") == (life is not None), "C4 reference life mismatch"
    )
    a.require(
        (status in ("absent", "unsupported_type")) == (handle is None),
        "C4 reference presence mismatch",
    )
    if handle is not None:
        a.require(
            (status == "invalid_sentinel") == (handle & 16383 == 16383),
            "C4 reference sentinel mismatch",
        )


def validate_source(data, support, objective):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == objective[key], "C4 source identity mismatch"
        )
    frames = support["observations"]["frames"]
    objectives = {f["source_sequence"]: f for f in objective["observations"]["frames"]}
    states, cuts = data["observations"]["states"], data["observations"]["cuts"]
    identities = {}
    for seq, state in enumerate(states, 1):
        a.require(state["sequence"] == seq, "C4 state sequence mismatch")
        entities = {e["entity_life"]: e for e in state["entities"]}
        a.require(len(entities) == len(state["entities"]), "duplicate C4 entity life")
        a.require(
            len({e["handle"] for e in entities.values()}) == len(entities),
            "duplicate C4 live handle",
        )
        for e in entities.values():
            a.require(e["handle"] & 16383 != 16383, "invalid C4 entity handle")
            identity = e["handle"], e["class_name"]
            a.require(
                identities.setdefault(e["entity_life"], identity) == identity,
                "C4 object identity changed",
            )
            validate_reference(e["owner"])
            scalar = e["definition"]
            a.require(
                (scalar["status"] == "observed") == (scalar["value"] is not None),
                "C4 scalar presence mismatch",
            )
            a.require(
                (scalar["status"] == "absent") == (scalar["native_type"] is None),
                "C4 scalar type mismatch",
            )
        aliases = [x["actor_alias"] for x in state["players"]]
        a.require(len(aliases) == len(set(aliases)), "duplicate C4 actor")
        for player in state["players"]:
            status, length = player["inventory_status"], player["inventory_length"]
            a.require(
                (status in ("observed", "outside_probe")) == (length is not None),
                "C4 inventory length presence mismatch",
            )
            if status == "observed":
                a.require(
                    length <= 64 and len(player["slots"]) == length,
                    "C4 inventory length mismatch",
                )
            else:
                a.require(not player["slots"], "C4 slots without inventory shape")
                if status == "outside_probe":
                    a.require(length > 64, "C4 invalid probe boundary")
            a.require(
                (status == "pawn_unavailable") == (player["pawn_life"] is None),
                "C4 pawn availability mismatch",
            )
            a.require(
                (player["pawn_life"] is None) == (player["pawn_handle"] is None),
                "C4 pawn handle presence mismatch",
            )
            for ref in player["slots"]:
                validate_reference(ref)
                if ref["status"] == "resolved":
                    entity = entities.get(ref["target_life"])
                    a.require(
                        entity is not None and entity["handle"] == ref["handle"],
                        "C4 slot identity mismatch",
                    )
    last_seq = last_state = 0
    previous = None
    for cut in cuts:
        seq, state_seq = cut["source_sequence"], cut["state_sequence"]
        a.require(last_seq < seq <= len(frames), "C4 exact sequence mismatch")
        a.require(
            state_seq in (last_state, last_state + 1) and 1 <= state_seq <= len(states),
            "C4 state frontier mismatch",
        )
        base, state = frames[seq - 1], states[state_seq - 1]
        a.require(
            all(cut[k] == base[k] for k in ("frame", "tick", "instance", "epoch")),
            "C4 exact clock mismatch",
        )
        a.require(
            seq in objectives and cut["objective_order"] == objectives[seq]["order"],
            "C4 objective order mismatch",
        )
        if state_seq != last_state:
            a.require(state["first_cut"] == seq, "C4 first cut mismatch")
        else:
            a.require(
                previous is not None
                and all(cut[k] == previous[k] for k in ("instance", "epoch")),
                "C4 state crossed boundary",
            )
        a.require(state["first_cut"] <= seq, "C4 future state")
        actors = a.actor_index(base)
        a.require(
            set(actors) == {x["actor_alias"] for x in state["players"]},
            "C4 player coverage mismatch",
        )
        for player in state["players"]:
            if player["pawn_life"] is not None:
                original = actors[player["actor_alias"]]
                a.require(
                    a.usable(original)
                    and all(
                        player[k] == original[k] for k in ("pawn_life", "pawn_handle")
                    ),
                    "C4 pawn life mismatch",
                )
        last_seq, last_state, previous = seq, state_seq, cut
    a.require(last_state == len(states), "C4 unreferenced future state")


def item_type(entity):
    scalar = entity["definition"]
    value = scalar["value"]
    valid = (
        scalar["status"] == "observed"
        and type(value) is int
        and scalar["native_type"] in ("int", "int32", "uint32", "int64", "uint64")
        and 0 <= value <= 65535
    )
    if not valid:
        return "unavailable"
    c4_class, c4_index = entity["class_name"] == "CC4", value == 49
    if c4_class != c4_index:
        return "conflict"
    # Zero/unknown definitions do not classify a carried object as non-C4.
    if value == 0:
        return "unavailable"
    return "c4" if c4_class else "other"


def possession(player, entities):
    if player is None or player["pawn_life"] is None:
        return o.fact(reason="pawn_unavailable")
    by_life = {e["entity_life"]: e for e in entities}
    complete = player["inventory_status"] == "observed"
    c4, seen = [], set()
    conflict = False
    for ref in player["slots"]:
        if ref["status"] == "invalid_sentinel":
            continue
        if ref["status"] != "resolved":
            complete = False
            continue
        life = ref["target_life"]
        conflict |= life in seen
        seen.add(life)
        entity = by_life[life]
        kind = item_type(entity)
        conflict |= kind == "conflict"
        complete &= kind in ("c4", "other")
        if kind == "c4":
            c4.append(entity)
    claims = []
    unknown_claim = False
    for entity in entities:
        owner = entity["owner"]
        if (
            owner["status"] == "resolved"
            and owner["handle"] == player["pawn_handle"]
            and owner["target_life"] == player["pawn_life"]
        ):
            kind = item_type(entity)
            if kind == "c4":
                claims.append(entity["entity_life"])
            elif kind == "conflict":
                conflict = True
            elif kind == "unavailable" and entity["class_name"] == "CC4":
                unknown_claim = True
    conflict |= len(c4) > 1 or len(claims) > 1
    if complete and any(life not in seen for life in claims):
        conflict = True
    confirmed = []
    for entity in c4:
        owner = entity["owner"]
        if owner["status"] == "resolved":
            matches = (
                owner["handle"] == player["pawn_handle"]
                and owner["target_life"] == player["pawn_life"]
            )
            conflict |= not matches
            if matches:
                confirmed.append(entity)
        elif owner["status"] == "invalid_sentinel":
            conflict = True
    if conflict:
        return o.fact(reason="inventory_owner_or_type_conflict")
    if confirmed:
        return o.fact(True)
    if c4 or claims or unknown_claim:
        return o.fact(reason="ownership_not_corroborated")
    return o.fact(False) if complete else o.fact(reason="inventory_coverage_incomplete")


def initial_context(anchor, base, state):
    actors = a.actor_index(base)
    players = {x["actor_alias"]: x for x in state["players"]} if state else {}
    result = {}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        has = (
            possession(players.get(alias), state["entities"])
            if state
            else o.fact(reason="missing_exact_c4_sample")
        )
        position = o.fact(reason="ownership_unavailable")
        if has["value"] is True:
            position = o.fact(copy.deepcopy(actors[alias]["position"]))
        elif has["value"] is False:
            position = o.fact(reason="not_a_carrier", status="not_applicable")
        result[role] = {
            "has_c4": has,
            "carrier_player_position": position,
            "physical_c4_position": o.fact(reason="not_observed_in_legal_view"),
        }
    return result


def initial_features(support, objective, data, anchor_id, actor, tick, frame):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    prefix = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:seq]},
    }
    cuts = [c for c in data["observations"]["cuts"] if c["source_sequence"] <= seq]
    states = (
        data["observations"]["states"][: cuts[-1]["state_sequence"]] if cuts else []
    )
    objective_cuts = [
        c for c in objective["observations"]["frames"] if c["source_sequence"] <= seq
    ]
    objective_prefix = {
        **objective,
        "observations": {
            "frames": objective_cuts,
            "events": objective["observations"]["events"][
                : objective_cuts[-1]["event_sequence"] if objective_cuts else 0
            ],
        },
    }
    o.validate_source(objective_prefix, prefix)
    validate_source(
        {**data, "observations": {"cuts": cuts, "states": states}},
        prefix,
        objective_prefix,
    )
    selected, _ = a.select_anchors(
        prefix["observations"]["frames"], support["source_alias"]
    )
    anchor = next(x for x in selected if x["anchor_id"] == anchor_id)
    state = (
        states[cuts[-1]["state_sequence"] - 1]
        if cuts and cuts[-1]["source_sequence"] == seq
        else None
    )
    return initial_context(anchor, prefix["observations"]["frames"][-1], state)


def build_report(support, objective, data, anchor_report, objective_report, references):
    validate_source(data, support, objective)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        selected
        == [
            {k: v for k, v in x.items() if k != "later_observation"}
            for x in anchor_report["anchors"]
        ],
        "C4 v1 anchor mismatch",
    )
    a.require(
        len(objective_report["rows"]) == len(selected), "C4 objective rows mismatch"
    )
    cuts = {c["source_sequence"]: c for c in data["observations"]["cuts"]}
    states = data["observations"]["states"]
    matrix, counts, rows = {}, {f: Counter() for f in FIELDS}, []
    for anchor, objective_row in zip(selected, objective_report["rows"], strict=True):
        a.require(
            all(anchor[k] == objective_row[k] for k in o.KEYS),
            "C4 objective anchor mismatch",
        )
        seq = anchor["source_sequence"]
        cut = cuts.get(seq)
        state = states[cut["state_sequence"] - 1] if cut else None
        initial = initial_context(
            anchor, support["observations"]["frames"][seq - 1], state
        )
        competitive = objective_row["competitive_round_status"]
        key = anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]
        entry = matrix.setdefault(
            key,
            {
                "anchors": 0,
                "canonical_interval_matched": 0,
                "fact_status_counts": {f: Counter() for f in FIELDS},
                "possession_counts": {r: Counter() for r in ("self", "ally")},
            },
        )
        entry["anchors"] += 1
        entry["canonical_interval_matched"] += (
            competitive == "canonical_interval_matched"
        )
        for field in FIELDS:
            role, name = field.split(".")
            fact = initial[role][name]
            status = fact["status"] + (":" + fact["reason"] if fact["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        for role in ("self", "ally"):
            value = initial[role]["has_c4"]["value"]
            entry["possession_counts"][role][
                "positive"
                if value is True
                else "negative"
                if value is False
                else "insufficient"
            ] += 1
        rows.append(
            {
                **{k: anchor[k] for k in o.KEYS},
                "competitive_round_status": competitive,
                "initial": initial,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    result = {
        "schema_id": "stratai.support_c4_context@1",
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
            "possession_counts": {
                r: dict(
                    sum((v["possession_counts"][r] for v in matrix.values()), Counter())
                )
                for r in ("self", "ally")
            },
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(result)
    return result


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 512 * 1024 * 1024,
        "invalid C4 source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    objective, support, support_ref, base = o.load_source(path.parent / o.DATA)
    a.require(
        data["support_source"] == support_ref
        and data["objective_source"] == p.file_reference(path.parent / o.DATA),
        "C4 source hash mismatch",
    )
    validate_source(data, support, objective)
    a.require(path.read_bytes() == raw, "C4 source changed")
    return data, objective, support, support_ref, base


def report_inputs(source, output):
    data, objective, support, support_ref, _ = load_source(source)
    refs = {
        "support_source": support_ref,
        "objective_source": p.file_reference(source.parent / o.DATA),
        "c4_source": p.file_reference(source),
        **{
            k: p.file_reference(output / name)
            for k, name in (
                ("anchors", a.REPORT),
                ("physical_context", p.REPORT),
                ("navigation_context", n.REPORT),
                ("objective_context", o.REPORT),
            )
        },
    }
    return (
        support,
        objective,
        data,
        json.loads((output / a.REPORT).read_bytes()),
        json.loads((output / o.REPORT).read_bytes()),
        refs,
    )


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = o.materialize(source.parent / o.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(
        sum(
            report["summary"]["possession_counts"][r].get("positive", 0)
            for r in ("self", "ally")
        )
        > 0,
        "no confirmed C4 carriers",
    )
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    o.validate_artifact(output, source.parent / o.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "C4 report recomputation mismatch",
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
