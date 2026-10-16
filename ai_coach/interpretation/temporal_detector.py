"""Causal provisional persistence, separate from the frozen first detector.

A verified pause does not expire a hypothesis. Motion against its destination
must accumulate one second (stationary samples add none), or undo the carrier's
original approach. This engineering filter is not a universal tactical duration.
"""

import math
import json
from copy import deepcopy
from functools import lru_cache
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.interpretation import context, detector as frozen, temporal
from ai_coach.support.anchors import require

CONFIGURATION = "collective_c4_approach@2-temporal-provisional"
CONTRADICTION_SECONDS = 1.0
SCHEMA_ID = "stratai.contextual_interpretation@2"


@lru_cache(maxsize=1)
def validator():
    schema = json.loads(
        (
            Path(__file__).resolve().parents[1]
            / "contracts/contextual_temporal_interpretation.schema.json"
        ).read_bytes()
    )
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def validate(reading):
    validator().validate(reading)
    require(
        reading["interpretation_id"]
        == temporal.digest(
            {k: v for k, v in reading.items() if k != "interpretation_id"}
        ),
        "temporal reading hash mismatch",
    )
    state = reading["temporal_state"]
    require(
        state["last_sample"]["cut"] == reading["cut"], "temporal state cut mismatch"
    )
    require(
        len(state["active"]) == len(reading["hypotheses"]),
        "temporal hypothesis state mismatch",
    )
    count = len(reading["hypotheses"])
    require(
        reading["status"]
        == ("ambiguous" if count > 1 else "compatible" if count else "abstained"),
        "temporal status mismatch",
    )
    for h, support in zip(reading["hypotheses"], state["active"]):
        require(
            h["destination_landmark_id"] == support["site_id"]
            and h["map_reference_id"]
            == support["map_reference_id"]
            == state["map_reference_id"]
            and h["support_origin_cut"] == support["seed_cut"]
            and h["support_current_cut"] == reading["cut"],
            "temporal support mismatch",
        )
        for signal in h["signals"]:
            require(
                signal["start_sequence"]
                <= signal["end_sequence"]
                <= reading["cut"]["sequence"],
                "future temporal signal",
            )
            require(
                all(
                    r["source_sequence"] <= signal["end_sequence"]
                    for refs in signal["references"]
                    for r in refs
                ),
                "future temporal reference",
            )
        require(
            support["seed_cut"]["sequence"] <= reading["cut"]["sequence"],
            "future temporal seed",
        )
    return reading


def adjacent(before, after):
    a, b = before["cut"], after["cut"]
    return (
        after["continuity"] == "adjacent_samples"
        and b["phase"] == "playing"
        and all(
            a[k] == b[k]
            for k in (
                "epoch",
                "instance",
                "round_number",
                "tick_rate",
                "sample_step",
                "phase",
            )
        )
        and b["sequence"] == a["sequence"] + 1
        and b["tick"] - a["tick"] == b["sample_step"]
        and b["frame"] > a["frame"]
    )


def available(row):
    """Explicit deaths are facts; unavailable allies cannot be treated as dead."""
    if row["cut"]["phase"] != "playing":
        return "playing_phase_unavailable"
    if row["missing_previous_actors"]:
        return "current_ally_information_unavailable"
    for a in row["actors"]:
        if a["alive"]["status"] == "observed" and a["alive"]["value"] is False:
            continue
        if (
            a["alive"]["value"] is not True
            or a["identity"] is None
            or a["exclusion"] is not None
            or a["position"]["status"] != "observed"
            or a["has_c4"]["status"] != "observed"
        ):
            return "current_ally_information_unavailable"
    return None


def extend(paths, row):
    current = {}
    for a in row["actors"]:
        if a["alive"]["value"] is not True:
            continue
        alias = a["actor_alias"]
        old = paths.get(alias, [])
        if old and (
            old[-1]["identity"] != a["identity"] or old[-1]["has_c4"] != a["has_c4"]
        ):
            old = []
        current[alias] = old
        current[alias].append({"cut": row["cut"], **a})
    return current


def best(facts, others):
    cosine = facts["direction_cosine"]
    return cosine is not None and all(
        other["direction_cosine"] is None
        or cosine >= other["direction_cosine"]
        or math.isclose(cosine, other["direction_cosine"], rel_tol=0, abs_tol=1e-12)
        for other in others
    )


def seed(paths, sites, reference, *, after_review=False):
    carriers = [a for a, p in paths.items() if p[-1]["has_c4"]["value"] is True]
    if len(carriers) != 1:
        return [], "unique_current_c4_carrier_unavailable"
    carrier = carriers[0]
    if len(paths) < 2:
        return [], "fewer_than_two_traceable_survivors"
    if not all(r["has_c4"]["value"] is True for r in paths[carrier]):
        return [], "c4_trajectory_not_continuously_observed"
    movement = {
        s["id"]: {a: frozen.progression(p, s["position"]) for a, p in paths.items()}
        for s in sites
    }
    candidates, reasons = [], []
    for site in sites:
        facts = movement[site["id"]]
        if after_review and (
            len(paths[carrier]) < 2
            or not frozen.progression(paths[carrier][-2:], site["position"])[
                "approaching"
            ]
        ):
            reasons.append("fresh_carrier_approach_required_after_review")
            continue
        advances = sorted(
            a
            for a, f in facts.items()
            if f["approaching"] and best(f, [other[a] for other in movement.values()])
        )
        if carrier not in advances:
            reasons.append("carrier_has_no_joint_approach_evidence")
            continue
        if len(advances) < 2:
            reasons.append("no_second_ally_approaching_same_reference")
            continue
        if any(
            a not in advances
            and len(frozen.stationary_suffix(p)) < 2
            and any(other[a]["approaching"] for other in movement.values())
            for a, p in paths.items()
        ):
            reasons.append("divergent_ally_movement")
            continue
        candidates.append(
            {
                "site_id": site["id"],
                "carrier": carrier,
                "contributors": advances,
                "identities": {
                    a: deepcopy(p[-1]["identity"]) for a, p in paths.items()
                },
                "carrier_origin_distance": facts[carrier]["start_distance"],
                "seed_cut": deepcopy(paths[carrier][-1]["cut"]),
                "seed_signals": [
                    frozen.signal(
                        paths[a], kind="sampled_approach_to_landmark", value=facts[a]
                    )
                    for a in advances
                ],
                "map_reference_id": reference["reference_id"],
                "motion": {
                    a: {"away_seconds": 0.0, "divergent_seconds": 0.0} for a in paths
                },
            }
        )
    return candidates, reasons[0] if reasons else "continuous_trajectory_unavailable"


def advance(h, before, row, sites):
    """Check current facts and accumulate only observed contrary motion time."""
    actors = {a["actor_alias"]: a for a in row["actors"] if a["alive"]["value"] is True}
    carriers = [a for a, v in actors.items() if v["has_c4"]["value"] is True]
    if carriers != [h["carrier"]]:
        return "c4_carrier_changed_or_unavailable", None
    if any(h["identities"].get(a) != v["identity"] for a, v in actors.items()):
        return "ally_identity_changed", None
    h["contributors"] = [a for a in h["contributors"] if a in actors]
    if len(h["contributors"]) < 2:
        return "fewer_than_two_supporting_survivors", None
    site = next(s for s in sites if s["id"] == h["site_id"])
    old = {a["actor_alias"]: a for a in before["actors"]}
    seconds = (row["cut"]["tick"] - before["cut"]["tick"]) / row["cut"]["tick_rate"]
    for alias, actor in actors.items():
        previous = old[alias]
        path = [{"cut": before["cut"], **previous}, {"cut": row["cut"], **actor}]
        change = frozen.progression(path, site["position"])
        other = [frozen.progression(path, s["position"]) for s in sites]
        counter = h["motion"][alias]
        delta = change["end_distance"] - change["start_distance"]
        if delta > 0:
            counter["away_seconds"] += seconds
        elif delta < 0:
            counter["away_seconds"] = 0.0
        # A stationary sample neither adds contrary time nor erases it.
        if change["direction_cosine"] is not None:
            counter["divergent_seconds"] = (
                0.0 if best(change, other) else counter["divergent_seconds"] + seconds
            )
        reason = None
        if (
            alias == h["carrier"]
            and delta > 0
            and change["end_distance"] >= h["carrier_origin_distance"]
        ):
            reason = "carrier_approach_fully_reversed"
        elif counter["away_seconds"] >= CONTRADICTION_SECONDS:
            reason = (
                "sustained_carrier_retreat"
                if alias == h["carrier"]
                else "sustained_ally_retreat"
            )
        elif counter["divergent_seconds"] >= CONTRADICTION_SECONDS:
            reason = "sustained_divergent_ally_movement"
        if reason:
            return reason, frozen.signal(
                path,
                kind="contradictory_motion",
                value={
                    **change,
                    **counter,
                    "confirmation_seconds": CONTRADICTION_SECONDS,
                },
            )
    return None, None


def hypothesis(h, paths, sites, row):
    site = next(s for s in sites if s["id"] == h["site_id"])
    carrier = h["carrier"]
    signals = deepcopy(h["seed_signals"])
    signals.append(
        frozen.signal([paths[carrier][-1]], kind="current_c4_possession", value=True)
    )
    roles = [
        {
            "actor_alias": carrier,
            "function": "c4_carrier",
            "status": "observed",
            "signal_indices": [len(signals) - 1],
        }
    ]
    for alias, path in sorted(paths.items()):
        signals.append(
            frozen.signal(
                [path[-1]],
                kind="temporal_possibility_maintained",
                value={
                    "seed_sequence": h["seed_cut"]["sequence"],
                    "current_distance": frozen.distance(
                        frozen.point(path[-1]), site["position"]
                    ),
                    **h["motion"][alias],
                    "confirmation_seconds": CONTRADICTION_SECONDS,
                },
            )
        )
        active = path[-1]["physical"]["active_weapon"]
        suffix = frozen.stationary_suffix(path)
        if (
            alias != carrier
            and active["status"] == "observed"
            and active["value"] in frozen.UTILITY
            and len(suffix) >= 2
            and frozen.distance(frozen.point(path[-1]), site["position"])
            > frozen.distance(frozen.point(paths[carrier][-1]), site["position"])
        ):
            signals.append(
                frozen.signal(
                    suffix,
                    kind="stationary_behind_with_utility_equipped",
                    value={
                        "weapon": active["value"],
                        "movement": frozen.progression(suffix, site["position"]),
                    },
                )
            )
            roles.append(
                {
                    "actor_alias": alias,
                    "function": "possible_utility_preparation",
                    "status": "compatible",
                    "signal_indices": [len(signals) - 1],
                }
            )
    return {
        "plan_id": "prepare_entry",
        "destination": site["label"],
        "destination_landmark_id": site["id"],
        "status": "compatible",
        "spatial_scope": "development_landmark_not_verified_zone_or_route",
        "map_reference_id": h["map_reference_id"],
        "signals": signals,
        "possible_functions": roles,
        "statement": "Organización compatible con preparar una entrada hacia la referencia "
        + site["label"]
        + " del mapa local. La posibilidad se conserva mientras no haya contradicción y los datos sigan disponibles.",
        "support_origin_cut": deepcopy(h["seed_cut"]),
        "support_current_cut": deepcopy(row["cut"]),
        "supporting_survivors": list(h["contributors"]),
    }


def restore(previous, value, reference):
    if previous is None:
        return [], None, False
    require(
        previous["interpretation_id"]
        == temporal.digest(
            {k: v for k, v in previous.items() if k != "interpretation_id"}
        ),
        "prior reading hash mismatch",
    )
    state = previous.get("temporal_state")
    if previous["configuration"] == CONFIGURATION:
        validate(previous)
    if not (
        previous["configuration"] == CONFIGURATION
        and state
        and reference
        and previous["source_sha256"] == value["source_sha256"]
        and previous["side"] == value["perspective"]["side"]
        and state["build_identifier"] == value["build_identifier"]
        and previous["temporal_policy"]["contradiction_motion_seconds"]
        == CONTRADICTION_SECONDS
        and state["map_reference_id"] == reference["reference_id"]
        and state["last_sample"]["cut"] == previous["cut"]
        and previous["cut"]["sequence"] < value["requested_interval"]["end_sequence"]
    ):
        return [], None, False
    last = state["last_sample"]
    overlap = next(
        (
            r
            for r in value["samples"]
            if r["cut"]["sequence"] == last["cut"]["sequence"]
        ),
        None,
    )
    if overlap != last and not (
        overlap is None and adjacent(last, value["samples"][0])
    ):
        return [], None, False
    return deepcopy(state["active"]), deepcopy(last) if state["active"] else None, True


def interpret(value, *, map_name, map_reference=None, previous=None):
    context.validate(value)
    # Reuse only the public envelope. The frozen implementation stays reproducible.
    result = frozen.interpret(value, map_name=map_name, map_reference=map_reference)
    result.pop("interpretation_id")
    result["schema_id"] = SCHEMA_ID
    result["configuration"] = CONFIGURATION
    sites = [
        s
        for s in (map_reference or {}).get("landmarks", [])
        if s["kind"] == "site_landmark"
    ]
    active, before, restored = restore(previous, value, map_reference)
    paths = extend({}, before) if before and available(before) is None else {}
    events, reason = [], "continuous_trajectory_unavailable"
    after_review = False
    scoped = bool(sites) and value["perspective"]["side"] == "T"
    for row in value["samples"]:
        if before and row["cut"]["sequence"] <= before["cut"]["sequence"]:
            continue
        issue = available(row)
        broken = before is not None and not adjacent(before, row)
        if broken or issue:
            reason = issue or "temporal_continuity_lost"
            if active:
                events.append(
                    {
                        "sequence": row["cut"]["sequence"],
                        "reason": reason,
                        "signal": None,
                    }
                )
            paths, active = {}, []
        if issue or not scoped:
            before = row
            continue
        paths = extend(paths, row)
        rejected = False
        retained = []
        reset_actors = set()
        reset_group = False
        for h in active:
            contradiction, signal = advance(h, before, row, sites)
            if contradiction:
                rejected = True
                if signal is None:
                    reset_group = True
                else:
                    reset_actors.add(signal["actor_alias"])
                reason = contradiction
                events.append(
                    {
                        "sequence": row["cut"]["sequence"],
                        "destination_landmark_id": h["site_id"],
                        "reason": reason,
                        "signal": signal,
                    }
                )
            else:
                retained.append(h)
        active = retained
        if rejected:
            # Preserve unaffected allies' factual paths. Only contrary movement
            # loses its earlier approach; a new seed needs fresh carrier motion.
            after_review = True
            if reset_group:
                paths = extend({}, row)
            else:
                for alias in reset_actors:
                    paths[alias] = paths[alias][-1:]
        elif not active:
            active, reason = seed(
                paths, sites, map_reference, after_review=after_review
            )
            if active:
                after_review = False
        before = row
    current = value["samples"][-1]
    result["hypotheses"] = (
        [hypothesis(h, paths, sites, current) for h in active] if scoped else []
    )
    result["status"] = (
        "ambiguous" if len(active) > 1 else "compatible" if active else "abstained"
    )
    if not scoped:
        result["status"] = "abstained"
        reason = (
            "missing_site_reference"
            if not sites
            else "criterion_scoped_to_T_preparation"
        )
    result["abstention_reasons"] = (
        []
        if active
        else list(dict.fromkeys(([events[-1]["reason"]] if events else []) + [reason]))
    )
    key = lambda r: {(h["plan_id"], h["destination"]) for h in r["hypotheses"]}
    result["update"] = {
        "kind": "initial"
        if previous is None
        else "prior_context_unavailable"
        if not restored
        else "abstained"
        if not active
        else "maintained"
        if key(previous) == key(result)
        else "revised",
        "previous_interpretation_id": previous["interpretation_id"]
        if previous
        else None,
    }
    result["temporal_events"] = events
    result["temporal_policy"] = {
        "contradiction_motion_seconds": CONTRADICTION_SECONDS,
        "pause_expiry_seconds": None,
        "parameter_status": "provisional_engineering_filter_not_expert_threshold",
    }
    result["temporal_state"] = {
        "build_identifier": value["build_identifier"],
        "map_reference_id": map_reference["reference_id"] if map_reference else None,
        "last_sample": deepcopy(current),
        "active": active,
    }
    result["interpretation_id"] = temporal.digest(result)
    return validate(result)
