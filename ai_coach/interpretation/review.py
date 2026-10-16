"""Multidimensional human review, separate from detection and from v1 metrics."""

from collections import Counter
from copy import deepcopy

from ai_coach.interpretation import evaluation, temporal
from ai_coach.support.anchors import require, schema_validator


def validate(value):
    schema_validator("contextual_review.json").validate(value)
    require(
        bool(value["reviewer"].strip()) and bool(value["literal"].strip()),
        "review attribution required",
    )
    names = [d["destination"] for d in value["destinations"]]
    require(len(set(names)) == len(names), "duplicate review destination")
    strategy = value["strategy"]
    require(
        bool(strategy["compatible_plans"]) == (strategy["status"] == "interpretable"),
        "contradictory strategy review",
    )
    require(
        len(set(strategy["compatible_plans"])) == len(strategy["compatible_plans"]),
        "duplicate review plan",
    )
    return value


def compare(manifest, predictions, reviews):
    """Possible destination is neither a confirmed plan nor an automatic hit."""
    # Preserve all provenance/configuration checks of the frozen v1 evaluator.
    baseline = evaluation.evaluate(manifest, predictions, [])
    cases = {c["episode_id"]: c for c in manifest["cases"]}
    seen, rows, groups = set(), [], {}
    for value in reviews:
        validate(value)
        episode = value["episode_id"]
        require(
            episode in cases and (episode, value["reviewer"]) not in seen,
            "unknown or duplicate review",
        )
        seen.add((episode, value["reviewer"]))
        prediction = predictions[episode]
        require(
            value["context_id"] == prediction["context_id"], "review context mismatch"
        )
        hypotheses = prediction["hypotheses"]
        destinations = {d["destination"]: d["judgement"] for d in value["destinations"]}
        plan_status = value["strategy"]["status"]
        if plan_status == "unreviewed":
            plan = "unreviewed"
        elif plan_status == "undetermined":
            plan = (
                "abstained_with_undetermined_strategy"
                if not hypotheses
                else "plan_not_confirmed_by_review"
            )
        elif not hypotheses:
            plan = "compatible_plan_not_emitted"
        else:
            accepted = set(value["strategy"]["compatible_plans"])
            plan = (
                "plan_compatible"
                if all(h["plan_id"] in accepted for h in hypotheses)
                else "plan_outside_reviewed_set"
            )
        directions = [
            {
                "destination": h["destination"],
                "review": destinations.get(h["destination"], "unreviewed"),
            }
            for h in hypotheses
        ]
        row = {
            "episode_id": episode,
            "reviewer": value["reviewer"],
            "strategy_comparison": plan,
            "emitted_destinations": directions,
            "review_destinations": deepcopy(value["destinations"]),
            "review_roles": deepcopy(value["roles"]),
            "exposure": deepcopy(value["exposure"]),
            "literal": value["literal"],
            "agreement": None,
        }
        row["comparison_id"] = temporal.digest(row)
        rows.append(row)
        counts = groups.setdefault(cases[episode]["source_sha256"], Counter())
        counts["reviews"] += 1
        counts[plan] += 1
        counts.update("destination:" + d["review"] for d in directions)
    reviewed_episodes = len({x[0] for x in seen})
    return {
        "schema_id": "stratai.contextual_review_comparison@2",
        "manifest_id": manifest["manifest_id"],
        "episodes": len(cases),
        "reviewed_episodes": reviewed_episodes,
        "unreviewed_episodes": len(cases) - reviewed_episodes,
        "rows": rows,
        "by_match": {k: dict(v) for k, v in sorted(groups.items())},
        "technical_counts": {
            k: baseline["counts"].get(k, 0)
            for k in ("episodes", "emitted", "abstained")
        },
        "agreement": None,
        "independent_generalization_accredited": False,
        "training_allowed": False,
    }
