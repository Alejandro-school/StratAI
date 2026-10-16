"""Narrow, provisional destination hypothesis, separate from preparing an entry.

The native place is a value read at the cut, not an accredited geometric region.
No case ID, human response, C4 carrier, trajectory score or future decides a site.
"""

from copy import deepcopy
from functools import lru_cache

from ai_coach.interpretation import candidate_context, native_place_context
from ai_coach.support.anchors import require, schema_validator

SCHEMA_ID = "stratai.destination_orientation@1"
RULE_ID = "ally_site_middle_orientation@1-provisional"
LIMITATIONS = [
    "ordinal_hypothesis_not_calibrated_probability",
    "native_place_property_not_accredited_geometric_zone",
    "delivery_age_is_not_a_spatial_expiry_threshold",
    "intent_and_next_action_not_observed",
    "alternative_destination_not_excluded",
    "route_accessibility_time_and_support_effectiveness_unverified",
    "preparing_entry_not_assessed",
    "only_reviewed_two_alive_site_and_middle_configuration",
]


@lru_cache(maxsize=1)
def _validator():
    return schema_validator("destination_orientation.json")


def _seal(value):
    value["orientation_id"] = candidate_context.content_hash(
        {k: v for k, v in value.items() if k != "orientation_id"}
    )
    return validate(value)


def _base(candidate, native):
    snapshot = candidate["snapshot"]
    return {
        "schema_id": SCHEMA_ID,
        "rule_id": RULE_ID,
        "candidate_context_id": candidate["candidate_context_id"],
        "native_place_context_id": native["native_place_context_id"],
        "cut": deepcopy(snapshot["cut"]),
        "status": "abstain",
        "preferred_site": None,
        "preferred_qualification": None,
        "alternatives": [],
        "reasons": [],
        "observed_facts": {
            "perspective_side": candidate["packet"]["context"]["perspective"]["side"],
            "objective_phase": snapshot["objective"]["phase"],
            "eligibility_kind": snapshot["round_context"]["eligibility"]["kind"],
            "allies": deepcopy(snapshot["team_status"]["allies"]),
            "native_source_status": native["status"],
            "native_places": [],
        },
        "inference": None,
        "limitations": list(LIMITATIONS),
        "prepare_entry_assessed": False,
        "training_allowed": False,
        "product_supported": False,
    }


def _finish(value, reason, *, outside=False):
    value["status"] = "out_of_scope" if outside else "abstain"
    value["reasons"] = [reason]
    return _seal(value)


def infer(candidate, native_places=None):
    """Use exact-cut factual inputs; invalid provenance raises, absence abstains.

    A positive ranks the ally's named site above the other destination. This is
    the reviewed support/reunion hypothesis, not evidence of intent, a command,
    a probability estimate, or a positive preparing-entry classification.
    """
    candidate_context.validate(candidate)
    native = (
        native_place_context.unavailable(candidate)
        if native_places is None
        else native_place_context.validate(native_places, candidate)
    )
    value = _base(candidate, native)
    snapshot = candidate["snapshot"]
    if value["observed_facts"]["perspective_side"] != "T":
        return _finish(value, "t_side_only", outside=True)
    if snapshot["objective"]["phase"] == "postplant":
        return _finish(value, "postplant_outside_preplant_orientation", outside=True)
    eligibility = snapshot["round_context"]["eligibility"]
    if eligibility["kind"] in ("knife", "warmup"):
        return _finish(value, "confirmed_round_kind_excluded", outside=True)
    if eligibility["kind"] != "bomb_objective" or not eligibility["analysis_allowed"]:
        return _finish(value, "round_eligibility_unknown")
    if snapshot["objective"]["phase"] != "preplant":
        return _finish(value, "objective_phase_unknown_or_conflicting")
    own = snapshot["team_status"]["allies"]
    if own["status"] != "observed" or own["unknown"] != 0:
        return _finish(value, "ally_survivor_count_incomplete")
    if own["alive"] != 2:
        return _finish(value, "configuration_not_covered_more_observation_required")
    if native["status"] != "observed":
        return _finish(value, "native_source_not_materialized")
    if native["provenance"]["map_name"] != "de_mirage":
        return _finish(value, "map_outside_reviewed_scope", outside=True)
    alive = [
        actor
        for actor in snapshot["actors"]
        if actor["alive"]["status"] == "observed" and actor["alive"]["value"] is True
    ]
    if len(alive) != 2 or any(
        a["exclusion"] is not None
        or a["connected"] is not True
        or a["identity"] is None
        or a["identity"]["identity_status"] != "observed"
        or a["position"]["status"] != "observed"
        for a in alive
    ):
        return _finish(value, "current_ally_identity_life_or_position_unavailable")
    observations = {o["actor_alias"]: o for o in native["observations"]}
    places = [observations[a["actor_alias"]] for a in alive]
    value["observed_facts"]["native_places"] = [
        {
            "actor_alias": row["actor_alias"],
            "label": row["label"],
            "property": row["property"],
            "value_status": row["value_status"],
            "value": row["value"],
            "last_delivery": deepcopy(row["last_delivery"]),
        }
        for row in places
    ]
    if any(row["value_status"] != "observed" for row in places):
        return _finish(value, "current_native_place_value_unavailable")
    sites = [row for row in places if row["value"] in ("BombsiteA", "BombsiteB")]
    middle = [row for row in places if row["value"] == "Middle"]
    if len(sites) != 1 or len(middle) != 1:
        return _finish(value, "configuration_not_covered_more_observation_required")
    preferred = {"BombsiteA": "A", "BombsiteB": "B"}[sites[0]["value"]]
    value.update(
        status="hypothesis",
        preferred_site=preferred,
        preferred_qualification="more_likely",
        alternatives=[
            {
                "site": "B" if preferred == "A" else "A",
                "qualification": "possible_not_excluded",
            }
        ],
        reasons=["ally_site_and_middle_support_reunion_hypothesis"],
        inference={
            "basis": "site_ally_as_possible_support_reunion_destination",
            "site_ally": sites[0]["actor_alias"],
            "middle_ally": middle[0]["actor_alias"],
        },
    )
    return _seal(value)


def validate(value):
    """Validate output structure and internal consistency, not input authenticity."""
    _validator().validate(value)
    require(
        value["orientation_id"]
        == candidate_context.content_hash(
            {k: v for k, v in value.items() if k != "orientation_id"}
        ),
        "orientation content hash differs",
    )
    if value["status"] == "hypothesis":
        require(value["preferred_site"] in ("A", "B"), "preferred site absent")
        require(
            value["preferred_qualification"] == "more_likely", "ordinal rank absent"
        )
        require(
            value["alternatives"]
            == [
                {
                    "site": "B" if value["preferred_site"] == "A" else "A",
                    "qualification": "possible_not_excluded",
                }
            ],
            "alternative destination must remain possible",
        )
        require(value["inference"] is not None, "hypothesis explanation absent")
        facts = value["observed_facts"]
        own = facts["allies"]
        require(
            facts["perspective_side"] == "T"
            and facts["objective_phase"] == "preplant"
            and facts["eligibility_kind"] == "bomb_objective"
            and facts["native_source_status"] == "observed"
            and own["status"] == "observed"
            and own["alive"] == own["observed_alive"] == 2
            and own["observed_dead"] == 3
            and own["unknown"] == 0,
            "hypothesis factual gates differ",
        )
        places = facts["native_places"]
        sites = [
            row
            for row in places
            if row["value"] == "Bombsite" + value["preferred_site"]
        ]
        middle = [row for row in places if row["value"] == "Middle"]
        require(
            len(places) == 2
            and all(row["value_status"] == "observed" for row in places)
            and len(sites) == len(middle) == 1
            and sites[0]["actor_alias"] != middle[0]["actor_alias"],
            "hypothesis native facts differ",
        )
        require(
            value["inference"]
            == {
                "basis": "site_ally_as_possible_support_reunion_destination",
                "site_ally": sites[0]["actor_alias"],
                "middle_ally": middle[0]["actor_alias"],
            }
            and value["reasons"] == ["ally_site_and_middle_support_reunion_hypothesis"],
            "hypothesis explanation differs",
        )
    else:
        require(
            value["preferred_site"] is None
            and value["preferred_qualification"] is None
            and value["alternatives"] == []
            and value["inference"] is None,
            "non-emission asserts a destination",
        )
    return value
