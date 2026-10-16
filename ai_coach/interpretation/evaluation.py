"""Freeze replay episodes and score *separate* reviews grouped by match.

No labels enter the detector. Unreviewed abstentions are not true negatives.
Development agreement is not independent tactical accuracy.
"""

from collections import Counter
from copy import deepcopy

from ai_coach.interpretation.temporal import digest
from ai_coach.support.anchors import require


def episode_id(source_sha256, side, start, end):
    return digest({"match": source_sha256, "side": side, "start": start, "end": end})


def freeze(cases, *, development_matches, detector_configuration, exposure):
    """Expose uncertainty about historical access rather than certify independence."""
    ids, seen = set(), set()
    rows = []
    for case in cases:
        require(
            set(case) == {"source_sha256", "side", "start_sequence", "end_sequence"},
            "invalid evaluation case",
        )
        require(
            case["side"] in ("CT", "T")
            and type(case["start_sequence"]) is int
            and type(case["end_sequence"]) is int
            and 0 < case["start_sequence"] <= case["end_sequence"],
            "invalid evaluation interval",
        )
        match = case["source_sha256"]
        require(match in exposure, "missing match exposure record")
        id_ = episode_id(
            match, case["side"], case["start_sequence"], case["end_sequence"]
        )
        require(id_ not in ids, "duplicate evaluation episode")
        ids.add(id_)
        seen.add(match)
        known = match in development_matches or exposure[match] != "unexposed_verified"
        rows.append(
            {
                "episode_id": id_,
                **deepcopy(case),
                "split": "development"
                if match in development_matches
                else "review_reserve",
                "independence": "not_accredited"
                if known
                else "candidate_pending_overlap_audit",
            }
        )
    result = {
        "schema_id": "stratai.contextual_evaluation_manifest@1",
        "detector_configuration": detector_configuration,
        "cases": rows,
        "distinct_matches": len(seen),
        "exposure": {x: exposure[x] for x in sorted(seen)},
        "selection": "frozen_before_predictions_labels_and_adjustment",
        "training_allowed": False,
    }
    result["manifest_id"] = digest(result)
    return result


def evaluate(manifest, predictions, reviews):
    require(
        manifest["manifest_id"]
        == digest({k: v for k, v in manifest.items() if k != "manifest_id"}),
        "evaluation manifest changed",
    )
    cases = {case["episode_id"]: case for case in manifest["cases"]}
    require(len(cases) == len(manifest["cases"]), "duplicate manifest cases")
    require(
        set(predictions) == set(cases),
        "prediction coverage differs from frozen manifest",
    )
    review_index = {}
    for review in reviews:
        require(
            set(review)
            == {
                "episode_id",
                "reviewer",
                "blind",
                "status",
                "acceptable_destinations",
                "reason",
            },
            "invalid review",
        )
        id_ = review["episode_id"]
        require(
            id_ in cases and id_ not in review_index,
            "unknown/duplicate reviewed episode",
        )
        require(
            isinstance(review["reviewer"], str)
            and bool(review["reviewer"].strip())
            and isinstance(review["reason"], str)
            and bool(review["reason"].strip()),
            "review attribution required",
        )
        require(
            type(review["blind"]) is bool
            and review["status"] in ("compatible", "insufficient_information"),
            "invalid review status",
        )
        require(
            isinstance(review["acceptable_destinations"], list)
            and all(isinstance(x, str) and x for x in review["acceptable_destinations"])
            and len(set(review["acceptable_destinations"]))
            == len(review["acceptable_destinations"]),
            "invalid review destinations",
        )
        require(
            bool(review["acceptable_destinations"])
            == (review["status"] == "compatible"),
            "contradictory review",
        )
        review_index[id_] = review
    by_match = {}
    for id_, case in cases.items():
        pred = predictions[id_]
        require(
            pred["configuration"] == manifest["detector_configuration"],
            "detector changed during evaluation",
        )
        require(
            pred["source_sha256"] == case["source_sha256"]
            and pred["side"] == case["side"]
            and pred["interval"]
            == {k: case[k] for k in ("start_sequence", "end_sequence")},
            "prediction case mismatch",
        )
        require(
            pred["interpretation_id"]
            == digest({k: v for k, v in pred.items() if k != "interpretation_id"}),
            "prediction hash mismatch",
        )
        counts = by_match.setdefault(case["source_sha256"], Counter())
        counts["episodes"] += 1
        counts["abstained" if pred["status"] == "abstained" else "emitted"] += 1
        review = review_index.get(id_)
        if review is None:
            counts["unreviewed"] += 1
            continue
        counts["reviewed"] += 1
        counts["blind_reviews" if review["blind"] else "nonblind_reviews"] += 1
        if review["status"] == "insufficient_information":
            counts[
                "correct_abstentions"
                if pred["status"] == "abstained"
                else "unsupported_emissions"
            ] += 1
        elif pred["status"] == "abstained":
            counts["missed_compatible_readings"] += 1
        else:
            destinations = {h["destination"] for h in pred["hypotheses"]}
            counts[
                "compatible_readings"
                if destinations
                and destinations <= set(review["acceptable_destinations"])
                else "incorrect_readings"
            ] += 1
    totals = sum(by_match.values(), Counter())
    return {
        "schema_id": "stratai.contextual_evaluation@1",
        "manifest_id": manifest["manifest_id"],
        "distinct_matches": len(by_match),
        "counts": dict(totals),
        "by_match": {match: dict(counts) for match, counts in sorted(by_match.items())},
        "agreement": None
        if not totals["reviewed"]
        else (totals["compatible_readings"] + totals["correct_abstentions"])
        / totals["reviewed"],
        "independent_generalization_accredited": False,
        "independence_dependencies": [
            "verified_prior_exposure",
            "temporal_and_participant_overlap_audit",
            "independent_blind_reviews",
            "frozen_version_and_thresholds",
        ],
        "training_allowed": False,
    }
