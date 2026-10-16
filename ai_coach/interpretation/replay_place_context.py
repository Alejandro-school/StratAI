"""Validate projected place facts against the local reference and causal point."""
from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.support.anchors import require


def checked_place(player, *, start_tick, cut_tick):
    value = player.get("place")
    if not player["alive"]:
        require(value is None, "dead player has current place")
        return None
    require(isinstance(value, dict), "living player place context missing")
    tick, available = player["observation_tick"], player["availability_tick"]
    require(start_tick <= tick <= available <= cut_tick, "future place position")
    precision = .5 if player["knowledge"] == "own_team_snapshot" else 0
    expected = {**resolve(player, load_reference(), map_name="de_mirage", xy_precision=precision),
                "observation_tick":tick,"availability_tick":available,"xy_precision":precision}
    require(value == expected, "place differs from causal XYZ or map reference")
    return value
