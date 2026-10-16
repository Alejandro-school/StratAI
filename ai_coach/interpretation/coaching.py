"""Prescriptive boundary for the contextual prototype.

Interpretation is useful evidence but supplies neither viable alternatives nor
a superiority comparison. Do not turn an agreement label into a mission.
"""

from ai_coach.interpretation.temporal import digest
from ai_coach.support.anchors import require


def assess_readiness(reading):
    require(
        reading["interpretation_id"]
        == digest({k: v for k, v in reading.items() if k != "interpretation_id"}),
        "coaching input hash mismatch",
    )
    require(
        reading["schema_id"]
        in (
            "stratai.contextual_interpretation@1",
            "stratai.contextual_interpretation@2",
        )
        and reading["comparison_allowed"] is False,
        "unsupported coaching source",
    )
    return {
        "schema_id": "stratai.contextual_coaching_readiness@1",
        "interpretation_id": reading["interpretation_id"],
        "status": "abstained",
        "recommendations": [],
        "priorities": [],
        "missions": [],
        "unmet_requirements": [
            "legal_information_for_decision",
            "physically_viable_parameterized_alternatives",
            "supported_contextual_comparison",
            "independent_tactical_evaluation",
            "secure_player_identity_and_prior_profile",
            "reviewed_practice_and_opportunity_criteria",
        ],
        "profile_effect": "no_skill_or_error_inferred_from_plan_hypothesis",
        "follow_up": "count_future_reviewed_opportunities_not_wins_or_completed_tasks",
        "training_allowed": False,
        "product_supported": False,
    }
