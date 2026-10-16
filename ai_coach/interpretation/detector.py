"""Provisional contextual hypotheses from a causal episode; no human answers.

The 2026-09-23 expert framing permits compatibility with preparation, never
confirmed intent, executed entry, effective cover, or a recommendation.
"""

import math
from copy import deepcopy
from itertools import pairwise

from ai_coach.interpretation import context as c
from ai_coach.interpretation import spatial_reference as spatial
from ai_coach.interpretation import temporal as t
from ai_coach.support.anchors import require

CONFIGURATION = "collective_c4_approach@1-provisional"
UTILITY = {
    "Smoke Grenade",
    "Flashbang",
    "HE Grenade",
    "Molotov",
    "Incendiary Grenade",
    "Decoy Grenade",
}


def point(sample):
    return sample["position"]["value"]


def distance(position, target):
    return math.hypot(position["x"] - target["x"], position["y"] - target["y"])


def progression(path, target):
    """Strict sampled facts: endpoint convergence plus latest nonzero direction.

    Zero is preserved as unresolved movement. No distance/time threshold certifies
    a tactical entry. Z remains in the input and is not a path/LOS calculation.
    """
    distances = [distance(point(row), target) for row in path]
    steps = [b - a for a, b in pairwise(distances)]
    nonzero = [step for step in steps if step != 0]
    dx, dy = (point(path[-1])[axis] - point(path[0])[axis] for axis in ("x", "y"))
    tx, ty = (target[axis] - point(path[0])[axis] for axis in ("x", "y"))
    norm = math.hypot(dx, dy) * math.hypot(tx, ty)
    return {
        "start_sequence": path[0]["cut"]["sequence"],
        "end_sequence": path[-1]["cut"]["sequence"],
        "start_distance": distances[0],
        "end_distance": distances[-1],
        "sample_count": len(path),
        "net_approach": distances[0] - distances[-1],
        "direction_cosine": (dx * tx + dy * ty) / norm if norm else None,
        "latest_resolved_direction": "toward"
        if nonzero and nonzero[-1] < 0
        else "away"
        if nonzero
        else "unresolved",
        "approaching": len(path) > 1
        and distances[-1] < distances[0]
        and bool(nonzero)
        and nonzero[-1] < 0,
    }


def signal(path, *, kind, value):
    return {
        "kind": kind,
        "actor_alias": path[-1]["actor_alias"],
        "value": value,
        "start_sequence": path[0]["cut"]["sequence"],
        "end_sequence": path[-1]["cut"]["sequence"],
        "references": [deepcopy(x["references"]) for x in path],
    }


def stationary_suffix(path):
    suffix = []
    active = path[-1]["physical"]["active_weapon"]
    for row in reversed(path):
        if any(point(row)[axis] != point(path[-1])[axis] for axis in ("x", "y")):
            break
        if row["physical"]["active_weapon"] != active:
            break
        suffix.append(row)
    return list(reversed(suffix))


def interpret(value, *, map_name, map_reference=None, previous=None):
    """All plan generation consumes allowed fields, never source/case IDs as features."""
    paths = c.trajectories(value)
    if map_reference is not None:
        spatial.validate(map_reference)
        require(map_reference["map_name"] == map_name, "map reference mismatch")
    hypotheses, missing = [], []
    current = value["samples"][-1]
    sites = (
        [x for x in map_reference["landmarks"] if x["kind"] == "site_landmark"]
        if map_reference
        else []
    )
    carriers = [
        alias for alias, path in paths.items() if path[-1]["has_c4"]["value"] is True
    ]
    if value["perspective"]["side"] != "T":
        missing.append("criterion_scoped_to_T_preparation")
    if not sites:
        missing.append("missing_site_reference")
    if len(carriers) != 1:
        missing.append("unique_current_c4_carrier_unavailable")
    if not any(len(path) > 1 for path in paths.values()):
        missing.append("continuous_trajectory_unavailable")
    if not missing:
        carrier = carriers[0]
        by_site = {
            site["id"]: {
                alias: progression(path, site["position"])
                for alias, path in paths.items()
            }
            for site in sites
        }
        for site in sites:
            movement = by_site[site["id"]]
            # Prefer the most directionally aligned reference for each actor.
            # A numerical tie supports both; never use site IDs to break it.
            advances = sorted(
                alias
                for alias, facts in movement.items()
                if facts["approaching"]
                and all(
                    other[alias]["direction_cosine"] is None
                    or facts["direction_cosine"] >= other[alias]["direction_cosine"]
                    or math.isclose(
                        facts["direction_cosine"],
                        other[alias]["direction_cosine"],
                        rel_tol=0,
                        abs_tol=1e-12,
                    )
                    for other in by_site.values()
                )
            )
            # "Several" means more than one; this is a provisional sufficient
            # pattern, not a claim that other entries cannot exist.
            if carrier not in advances or len(advances) < 2:
                continue
            conflicting = [
                alias
                for alias, path in paths.items()
                if alias not in advances
                and len(stationary_suffix(path)) < 2
                and any(other[alias]["approaching"] for other in by_site.values())
            ]
            if conflicting:
                missing.append("divergent_ally_movement")
                continue
            # C4 must be observed throughout its actual advancing suffix, not
            # retrospectively attached to movement before an unknown pickup.
            carrier_path = paths[carrier]
            if not all(row["has_c4"]["value"] is True for row in carrier_path):
                continue
            signals = [
                signal(
                    paths[alias],
                    kind="sampled_approach_to_landmark",
                    value=movement[alias],
                )
                for alias in advances
            ]
            signals.append(
                signal([carrier_path[-1]], kind="current_c4_possession", value=True)
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
                active = path[-1]["physical"]["active_weapon"]
                if (
                    alias == carrier
                    or active["status"] != "observed"
                    or active["value"] not in UTILITY
                ):
                    continue
                # Behind along the current approach, and no resolved lateral
                # movement in this available suffix. Weapon alone is insufficient.
                suffix = stationary_suffix(path)
                if (
                    len(suffix) < 2
                    or movement[alias]["end_distance"]
                    <= movement[carrier]["end_distance"]
                ):
                    continue
                signals.append(
                    signal(
                        suffix,
                        kind="stationary_behind_with_utility_equipped",
                        value={
                            "weapon": active["value"],
                            "movement": progression(suffix, site["position"]),
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
            hypotheses.append(
                {
                    "plan_id": "prepare_entry",
                    "destination": site["label"],
                    "destination_landmark_id": site["id"],
                    "status": "compatible",
                    "spatial_scope": "development_landmark_not_verified_zone_or_route",
                    "map_reference_id": map_reference["reference_id"],
                    "signals": signals,
                    "possible_functions": roles,
                    "statement": "Organización compatible con preparar una entrada hacia la referencia "
                    + site["label"]
                    + " del mapa local.",
                }
            )
    if not hypotheses and not missing:
        missing.append("no_joint_continuous_approach_with_c4")
    state = (
        "compatible"
        if len(hypotheses) == 1
        else "ambiguous"
        if hypotheses
        else "abstained"
    )
    result = {
        "schema_id": "stratai.contextual_interpretation@1",
        "configuration": CONFIGURATION,
        "context_id": value["context_id"],
        "source_alias": value["source_alias"],
        "source_sha256": value["source_sha256"],
        "side": value["perspective"]["side"],
        "cut": deepcopy(current["cut"]),
        "interval": deepcopy(value["requested_interval"]),
        "status": state,
        "hypotheses": hypotheses,
        "abstention_reasons": missing,
        "unknowns": [
            "intent",
            "cooperation",
            "opponent_information",
            "physical_route",
            "zone_containment",
            "map_build_compatibility",
            "full_inventory",
            "weapon_readiness",
            "best_action",
        ],
        "alternatives": [
            {
                "plan_id": "fake_or_other_plan",
                "status": "not_excluded_not_supported",
                "reason": "own_ally_organization_does_not_reveal_intent",
            }
        ],
        "update": {"kind": "initial", "previous_interpretation_id": None},
        "recommendations": [],
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }
    if previous is not None:
        require(
            previous["interpretation_id"]
            == t.digest(
                {k: v for k, v in previous.items() if k != "interpretation_id"}
            ),
            "prior reading hash mismatch",
        )
        compatible = (
            previous["source_sha256"] == value["source_sha256"]
            and previous["side"] == result["side"]
            and previous["configuration"] == CONFIGURATION
            and all(
                previous["cut"][k] == current["cut"][k]
                for k in ("epoch", "instance", "round_number")
            )
            and value["requested_interval"]["start_sequence"]
            <= previous["cut"]["sequence"]
            < current["cut"]["sequence"]
        )
        if compatible:
            retained = [
                row
                for row in value["samples"]
                if row["cut"]["sequence"] >= previous["cut"]["sequence"]
            ]
            compatible = all(
                row["continuity"] == "adjacent_samples" for row in retained[1:]
            )
            identities = lambda row: {
                a["actor_alias"]: a["identity"] for a in row["actors"]
            }
            compatible = compatible and all(
                identities(row) == identities(retained[0]) for row in retained[1:]
            )
            compatible = (
                compatible
                and all(
                    h["map_reference_id"] == map_reference["reference_id"]
                    for h in previous["hypotheses"]
                )
                if map_reference
                else False
            )
        if not compatible:
            kind = "prior_context_unavailable"
        elif not hypotheses:
            kind = "abstained"
        else:
            key = lambda row: {
                (x["plan_id"], x["destination"]) for x in row["hypotheses"]
            }
            kind = "maintained" if key(previous) == key(result) else "revised"
        result["update"] = {
            "kind": kind,
            "previous_interpretation_id": previous["interpretation_id"],
        }
    result["interpretation_id"] = t.digest(result)
    return result
