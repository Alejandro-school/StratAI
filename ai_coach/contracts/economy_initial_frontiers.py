"""Candidate.2 initial frontiers and independently verified round correspondence.

Load with validate_candidate before using initial_team_state. Round links are
offline provenance gates; their end ticks, outcomes and reconciliation are not
features. A same-tick callback is ambiguous even when dispatched after the probe.
"""

from __future__ import annotations

import copy
import hashlib
from collections import Counter

from ai_coach.contracts import validate_economy_candidate as v1

RULE_SOURCES = {
    "match_started": "m_pGameRules.m_bHasMatchStarted",
    "warmup": "m_pGameRules.m_bWarmupPeriod",
    "freeze": "m_pGameRules.m_bFreezePeriod",
}
LINK_SOURCE = "canonical/core/rounds.json+derived/economy_players.json"


def frame_projection(data: dict) -> dict:
    """The unchanged first-FrameDone compartment retains the v1 semantics."""
    result = copy.deepcopy(data)
    result.pop("initial_frontiers")
    result["schema_id"] = "stratai.observed_economy_candidate@1"
    result["contract_version"] = v1.VERSION
    return result


def _events(round_: dict) -> list[dict]:
    return round_["execution"]["events"] + round_["reconciliation"]["other_events"]


def _competitive_native(frontier: dict) -> bool:
    return all(
        frontier["native_rules"][key]["status"] == "observed"
        and frontier["native_rules"][key]["value"] is value
        for key, value in {
            "match_started": True,
            "warmup": False,
            "freeze": True,
        }.items()
    )


def _complete(actor: dict | None) -> bool:
    return (
        actor is not None
        and actor["connected"]
        and all(fact["status"] == "observed" for fact in actor["facts"].values())
    )


def _selection(
    round_: dict, frontier: dict, alias: str
) -> tuple[dict | None, dict | None, str]:
    # Select a whole actor observation, never fill missing fields from a later
    # snapshot. Selection depends only on this actor, not opponent economy.
    for source, snapshot in (
        ("round_start_probe", frontier["round_start_probe"]),
        ("first_frame_done", round_["initial_state"]),
    ):
        if snapshot is None:
            continue
        actor = next((a for a in snapshot["actors"] if a["actor_alias"] == alias), None)
        if _complete(actor):
            return actor, snapshot, source
    return None, None, "required_initial_fact_unavailable"


def _initial_actor(
    round_: dict, frontier: dict, alias: str
) -> tuple[dict | None, dict | None, str]:
    actor, snapshot, source = _selection(round_, frontier, alias)
    if actor is None:
        return None, None, source
    if snapshot["tick"] != round_["start_tick"]:
        return None, None, "initial_sample_after_round_start"
    if not _competitive_native(frontier):
        return None, None, "native_competitive_state_unavailable"
    if (
        round_["freeze_end_tick"] is not None
        and round_["freeze_end_tick"] <= snapshot["tick"]
    ):
        return None, None, "freeze_closed_before_initial"
    if any(
        e["actor_alias"] == alias and e["tick"] <= snapshot["tick"]
        for e in [*_events(round_), *frontier["pre_probe_callbacks"]]
    ):
        return None, None, "activity_in_initial_tick"
    if actor["facts"]["spent_native"]["value"] != 0:
        return None, None, "spend_at_initial_observation"
    # Defensive checks for callers: observation and availability cannot be
    # backdated, even after the artifact has already passed validation.
    for fact in actor["facts"].values():
        v1._require(
            fact["observation_tick"] == fact["availability_tick"] == snapshot["tick"],
            "future initial fact",
        )
    return actor, snapshot, source


def initial_team_state(
    data: dict, instance: int, actor_alias: str, t0_tick: int
) -> dict:
    """Return tenable, same-side initial observations; grant no decision eligibility."""
    round_ = next((r for r in data["rounds"] if r["instance"] == instance), None)
    frontier = next(
        (f for f in data["initial_frontiers"] if f["instance"] == instance), None
    )
    v1._require(
        round_ is not None and frontier is not None and round_["start_tick"] == t0_tick,
        "unknown/changed economic T0",
    )
    v1._require(
        frontier["round_link"]["status"] == "verified",
        "competitive round correspondence unavailable",
    )
    probe = frontier["round_start_probe"]
    actor = next((a for a in probe["actors"] if a["actor_alias"] == actor_alias), None)
    v1._require(
        actor is not None and actor["side"] in {"CT", "T"},
        "actor absent from initial roster",
    )
    side = actor["side"]
    aliases = [a["actor_alias"] for a in probe["actors"] if a["side"] == side]
    v1._require(
        len(aliases) == len(set(aliases)) == 5, "complete initial team not observed"
    )
    team = []
    for alias in aliases:
        selected, _, source = _initial_actor(round_, frontier, alias)
        v1._require(
            selected is not None and selected["side"] == side,
            "complete prebuy team not observed",
        )
        team.append(
            {
                "actor_alias": alias,
                "side": side,
                "facts": selected["facts"],
                "observation_source": source,
            }
        )
    # Rules/scores come only from the initial probe, including explicit absence.
    for fact in probe["context"].values():
        v1._require(
            fact["status"] != "observed"
            or fact["observation_tick"] == fact["availability_tick"] == t0_tick,
            "future initial context",
        )
    return copy.deepcopy(
        {
            "instance": instance,
            "t0_tick": t0_tick,
            "side": side,
            "context": probe["context"],
            "actors": team,
        }
    )


def coverage(data: dict) -> dict:
    reasons, sources = Counter(), Counter()
    available = competitive = teams = 0
    for round_, frontier in zip(data["rounds"], data["initial_frontiers"], strict=True):
        sides = Counter()
        for actor in frontier["round_start_probe"]["actors"]:
            selected, _, reason = _initial_actor(round_, frontier, actor["actor_alias"])
            if selected is None:
                reasons[reason] += 1
            else:
                available += 1
                sources[reason] += 1
                if frontier["round_link"]["status"] == "verified":
                    competitive += 1
                    if selected["side"] == actor["side"]:
                        sides[selected["side"]] += 1
        teams += sum(count == 5 for count in sides.values())
    return {
        "initial_actors_accredited": available,
        "competitive_initial_actors": competitive,
        "competitive_complete_teams": teams,
        "abstentions": dict(sorted(reasons.items())),
        "selected_sources": dict(sorted(sources.items())),
    }


def validate_data(data: dict) -> dict:
    v1._schema(data, "observed_economy_candidate_v2.schema.json")
    frame_data = frame_projection(data)
    summary = v1.validate_data(frame_data)
    frontiers = data["initial_frontiers"]
    v1._require(
        len(frontiers) == len(data["rounds"]), "initial frontier count mismatch"
    )
    probe_data = copy.deepcopy(frame_data)
    all_sequences = {
        item["callback_sequence"]
        for r in data["rounds"]
        for item in [
            r["initial_state"],
            r["execution"]["freeze_snapshot"],
            r["reconciliation"]["round_end_snapshot"],
            *_events(r),
        ]
        if item is not None
    }
    aliases = {p["participant_alias"] for p in data["participants"]}
    ledger_events = {
        e["callback_sequence"]: e for r in data["rounds"] for e in _events(r)
    }
    for round_, frontier, projection in zip(
        data["rounds"], frontiers, probe_data["rounds"], strict=True
    ):
        v1._require(
            frontier["instance"] == round_["instance"], "frontier instance mismatch"
        )
        probe = frontier["round_start_probe"]
        v1._require(probe["tick"] == round_["start_tick"], "probe is not at RoundStart")
        seq = probe["callback_sequence"]
        v1._require(seq not in all_sequences, "probe callback sequence reused")
        all_sequences.add(seq)
        frame = round_["initial_state"]
        v1._require(
            frame is None or seq < frame["callback_sequence"],
            "probe is after initial frame",
        )
        end = frontier["official_end_tick"]
        v1._require(
            end is None
            or (
                end >= round_["start_tick"]
                and (round_["end_tick"] is None or end >= round_["end_tick"])
            ),
            "official end precedes round end",
        )
        for key, fact in frontier["native_rules"].items():
            v1._fact(fact, probe["tick"], RULE_SOURCES[key])
        early = frontier["pre_probe_callbacks"]
        v1._require(
            [e["callback_sequence"] for e in early]
            == sorted({e["callback_sequence"] for e in early}),
            "duplicate/unsorted pre-probe callback",
        )
        for event in early:
            v1._require(
                event["actor_alias"] in aliases
                and event["tick"] == probe["tick"]
                and event["callback_sequence"] < seq,
                "pre-probe callback frontier mismatch",
            )
            old = ledger_events.get(event["callback_sequence"])
            v1._require(
                old is None or all(old[key] == value for key, value in event.items()),
                "pre-probe callback identity mismatch",
            )
            v1._require(
                old is not None or event["callback_sequence"] not in all_sequences,
                "pre-probe callback reuses snapshot sequence",
            )
        prior = [
            {k: e[k] for k in ("actor_alias", "kind", "tick", "callback_sequence")}
            for e in ledger_events.values()
            if e["tick"] == probe["tick"] and e["callback_sequence"] < seq
        ]
        v1._require(
            all(e in early for e in prior), "activity before round restart was omitted"
        )
        for actor in probe["actors"]:
            v1._require(
                actor["prepurchase_status"] == "unavailable"
                and actor["prepurchase_reason"]
                == (
                    "not_initial_snapshot"
                    if actor["connected"]
                    else "participant_unavailable"
                ),
                "raw probe claims prebuy before deferred callbacks",
            )
        # Reuse all v1 fact, inventory, identity and chronology checks on a
        # strictly validated projection. Synthetic statuses are never returned.
        projected = copy.deepcopy(probe)
        projected["phase"] = "initial_available"
        for actor in projected["actors"]:
            actor["prepurchase_status"], actor["prepurchase_reason"] = (
                v1._expected_prebuy(actor, projected, round_, _events(round_))
            )
        projection["initial_state"] = projected
    v1.validate_data(probe_data)
    summary.update(
        round_start_probes=len(frontiers),
        competitive_round_links=sum(
            f["round_link"]["status"] == "verified" for f in frontiers
        ),
        excluded_round_links=sum(
            f["round_link"]["status"] == "excluded" for f in frontiers
        ),
    )
    return summary


def _canonical_artifact(data: dict, base: dict, root, name: str, rows_key: str) -> list:
    refs = [r for r in base["artifacts"] if r["path"] == name]
    v1._require(len(refs) == 1, "canonical round-link reference missing/duplicated")
    ref = refs[0]
    value, payload = v1._read(
        v1._safe_file(root, f"match_{data['source_alias']}/canonical/{name}")
    )
    v1._require(
        len(payload) == ref["bytes"]
        and hashlib.sha256(payload).hexdigest() == ref["sha256"],
        "round-link canonical integrity mismatch",
    )
    v1._require(
        value.get("match_id") == data["source_alias"]
        and len(value.get(rows_key, [])) == ref["record_count"],
        "round-link canonical identity/count mismatch",
    )
    return value[rows_key]


def validate_round_links(data: dict, base: dict, root) -> dict:
    canonical = _canonical_artifact(data, base, root, "core/rounds.json", "rounds")
    actors = _canonical_artifact(
        data, base, root, "derived/economy_players.json", "players"
    )
    bridge = {p["actor_key"]: p["participant_alias"] for p in data["participants"]}
    expected = []
    for round_, frontier in zip(data["rounds"], data["initial_frontiers"], strict=True):
        reason, identity = "native_competitive_state_unavailable", None
        if _competitive_native(frontier):
            end = frontier["official_end_tick"]
            matches = (
                [
                    r
                    for r in canonical
                    if r["round_number"] == round_["round_number"]
                    and r["start_tick"] == round_["start_tick"]
                    and r["end_tick"] == end
                ]
                if end is not None
                else []
            )
            reason = (
                "official_end_unavailable"
                if end is None
                else "canonical_boundaries_unmatched"
            )
            if len(matches) > 1:
                reason = "ambiguous_canonical_round"
            elif len(matches) == 1:
                row = matches[0]
                if (
                    row["round_id"]
                    != f"{data['source_alias']}:round:{round_['round_number']:03d}"
                ):
                    reason = "canonical_identity_mismatch"
                else:
                    roster, valid = {}, True
                    for item in actors:
                        if item["round_id"] != row["round_id"]:
                            continue
                        key = hashlib.sha256(
                            (data["source_sha256"] + "\n" + item["player_id"]).encode()
                        ).hexdigest()
                        alias = bridge.get(key)
                        if (
                            alias is None
                            or alias in roster
                            or item["round_number"] != round_["round_number"]
                            or item["side"] not in {"ct", "t"}
                        ):
                            valid = False
                            break
                        roster[alias] = item["side"].upper()
                    probe = frontier["round_start_probe"]["actors"]
                    if not valid:
                        reason = "canonical_participants_mismatch"
                    elif len(roster) != 10 or len(probe) != 10:
                        reason = "initial_roster_unavailable"
                    elif roster != {
                        a["actor_alias"]: a["side"] for a in probe
                    } or Counter(roster.values()) != {"CT": 5, "T": 5}:
                        reason = "canonical_participants_mismatch"
                    else:
                        reason, identity = (
                            "unique_identity_start_official_end_and_roster",
                            row["round_id"],
                        )
        expected.append(
            {
                "status": "verified" if identity else "excluded",
                "reason": reason,
                "canonical_round_id": identity,
                "source": LINK_SOURCE,
            }
        )
    uses = Counter(
        link["canonical_round_id"] for link in expected if link["canonical_round_id"]
    )
    for link, frontier in zip(expected, data["initial_frontiers"], strict=True):
        if link["canonical_round_id"] and uses[link["canonical_round_id"]] > 1:
            link.update(
                status="excluded", reason="ambiguous_instance", canonical_round_id=None
            )
        v1._require(
            link == frontier["round_link"],
            "round correspondence differs from independent computation",
        )
    return {
        "independently_recomputed_round_links": len(expected),
        "round_link_reasons": dict(
            sorted(Counter(link["reason"] for link in expected).items())
        ),
        "initial_coverage": coverage(data),
    }
