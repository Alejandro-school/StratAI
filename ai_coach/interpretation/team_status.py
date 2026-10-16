"""Causal 5v5 alive counts for review, independent of hidden spatial data."""

from collections import Counter

from ai_coach.support.anchors import require

SCOPE = "five_per_side_current_native_alive@1"
FIELDS = (
    "actor_alias",
    "side",
    "connected",
    "identity_status",
    "pawn_life",
    "pawn_handle",
    "alive_status",
    "alive",
)


def sequence(context, support):
    """Use prior roster identity and current alive flags; never carry alive forward."""
    require(
        all(context[k] == support[k] for k in ("source_alias", "source_sha256")),
        "team status source differs",
    )
    cuts = {s["cut"]["sequence"]: s["cut"] for s in context["samples"]}
    end = max(cuts)
    side = context["perspective"]["side"]
    require(side in ("T", "CT"), "unsupported team perspective")
    previous = 0
    boundary = None
    output = []
    for frame in support["observations"]["frames"]:
        seq = frame["sequence"]
        if seq > end:
            break
        require(seq == previous + 1, "team status source sequence differs")
        previous = seq
        key = (frame["epoch"], frame["instance"], frame["round_number"])
        if key != boundary:
            boundary, start = key, seq
            rosters = {"T": set(), "CT": set()}
            sides, lives, conflicts = {}, {}, set()
        # This projection deliberately never reads rival positions or resources.
        players = [{k: p.get(k) for k in FIELDS} for p in frame["players"]]
        aliases = Counter(p["actor_alias"] for p in players)
        pawns = Counter(p["pawn_life"] for p in players if p["pawn_life"])
        handles = Counter(
            p["pawn_handle"] for p in players if p["pawn_handle"] is not None
        )
        for p in players:
            alias, team = p["actor_alias"], p["side"]
            if alias and team in rosters:
                rosters[team].add(alias)
                if sides.setdefault(alias, team) != team:
                    conflicts.add(alias)
            life, handle = p["pawn_life"], p["pawn_handle"]
            if life and handle is not None and lives.setdefault(life, handle) != handle:
                conflicts.add(alias)
        if seq not in cuts:
            continue
        require(
            all(frame[k] == v for k, v in cuts[seq].items()), "team status cut differs"
        )
        counts = {}
        for team, roster in rosters.items():
            current = [p for p in players if p["side"] == team]
            valid = [
                p
                for p in current
                if bool(p["actor_alias"])
                and p["actor_alias"] not in conflicts
                and aliases[p["actor_alias"]] == 1
                and p["connected"] is True
                and p["identity_status"] == "observed"
                and bool(p["pawn_life"])
                and pawns[p["pawn_life"]] == 1
                and type(p["pawn_handle"]) is int
                and p["pawn_handle"] >= 0
                and p["pawn_handle"] % 16384 != 16383
                and handles[p["pawn_handle"]] == 1
                and p["alive_status"] == "observed"
                and type(p["alive"]) is bool
            ]
            invalid_roster = len(roster) > 5 or len(current) > 5
            unavailable = frame["phase"] != "playing" or invalid_roster
            if unavailable:
                valid = []
            alive = sum(p["alive"] for p in valid)
            exact = not unavailable and len(roster) == len(current) == len(valid) == 5
            counts[team] = {
                "side": team,
                "status": "observed"
                if exact
                else "unavailable"
                if unavailable
                else "partial",
                "alive": alive if exact else None,
                "observed_alive": alive,
                "observed_dead": len(valid) - alive,
                "unknown": None if invalid_roster else 5 - len(valid),
                "reason": None
                if exact
                else "roster_exceeds_5v5_scope"
                if invalid_roster
                else "not_playing"
                if unavailable
                else "incomplete_current_roster_or_alive",
            }
        output.append(
            {
                "source_sequence": seq,
                "roster_start_sequence": start,
                "allies": counts[side],
                "opponents": counts["CT" if side == "T" else "T"],
            }
        )
    require(len(output) == len(cuts), "team status prefix incomplete")
    return output
