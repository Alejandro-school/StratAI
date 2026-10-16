"""Strict invariants for input@6, without rewriting historical input receipts."""
from copy import deepcopy
import math

from ai_coach.support.anchors import require

# Exact Equipment.String vocabulary from the pinned demoinfocs v5.2.0 producer.
# An inventory item is a name from this vocabulary, never an extra sentence.
NATIVE_EQUIPMENT = frozenset((
    'AK-47','AUG','AWP','PP-Bizon','C4','Desert Eagle','Decoy Grenade','Dual Berettas',
    'FAMAS','Five-SeveN','Flashbang','G3SG1','Galil AR','Glock-18','HE Grenade','P2000',
    'Incendiary Grenade','M249','M4A4','MAC-10','MAG-7','Molotov','MP7','MP5-SD','MP9',
    'Negev','Nova','P250','P90','Sawed-Off','SCAR-20','SG 553','Smoke Grenade','SSG 08',
    'Zeus x27','Tec-9','UMP-45','XM1014','M4A1','CZ75 Auto','USP-S','World','R8 Revolver',
    'Kevlar Vest','Kevlar + Helmet','Defuse Kit','Knife','UNKNOWN',
))


def finite(value):
    return type(value) in (int, float) and math.isfinite(value)


def checked_players(view, start):
    players = view["frame"]["players"]
    ids = [str(p.get("steam_id")) for p in players]
    require(all(i.isdigit() and 1 <= int(i) <= 10 for i in ids), "nonlocal participant identity")
    require(len({int(i) for i in ids}) == len(ids), "duplicate participant identity")
    require(sum(p.get("team") == "T" for p in players) <= 5, "more than five own players")
    for p in players:
        require(type(p.get("alive")) is bool, "invalid alive flag")
        require(all(finite(p.get(k)) for k in ("x", "y", "z")), "invalid player position")
        observed, available = p.get("observation_tick"), p.get("availability_tick")
        require(type(observed) is int and type(available) is int
                and start <= observed <= available <= view["tick"], "invalid player observation time")
        if p["team"] == "T":
            require(observed == available == view["availability_tick"], "own snapshot date differs")
            require(p.get('source') == {'kind':'demo_replay_frame',
                    'reference':f"round:{view['round']}:frame:{observed}:player:{p['steam_id']}"}, 'own snapshot source differs')
            require(type(p.get("health")) is int and 0 <= p["health"] <= 100
                    and p["alive"] == (p["health"] > 0), "incoherent player health")
            for key in ("armor", "money"):
                require(key not in p or type(p[key]) is int and p[key] >= 0, "invalid player resource")
            require(type(p.get("has_c4", False)) is bool, "invalid C4 flag")
            if p.get('life_context') is not None:
                checked_life(p, view)


def checked_life(player, view):
    value = player['life_context']
    require(isinstance(value, dict) and set(value) == {'schema_id','status','alive','health',
            'observation_tick','availability_tick','source'}, 'invalid life evidence fields')
    require(value['schema_id'] == 'stratai.replay_life_context@1' and value['status'] == 'observed'
            and value['alive'] is player['alive'] is False
            and type(value['health']) is int and value['health'] == player['health'] == 0, 'invalid death state')
    tick = value['observation_tick']
    require(type(tick) is int and type(value['availability_tick']) is int
            and view['availability_tick'] < tick == value['availability_tick'] <= view['tick'], 'invalid life date')
    source = value['source']
    require(isinstance(source, dict) and set(source) == {'kind','reference'}
            and source['kind'] == 'demo_kill_event' and isinstance(source['reference'], str)
            and bool(source['reference']), 'invalid life source')
    witnesses = [e for e in view.get('events', []) if e.get('id') == source['reference']
                 and e.get('type') == 'kill' and e.get('knowledge') == 'public_killfeed'
                 and e.get('tick') == e.get('availability_tick') == tick
                 and str(e.get('victim_id')) == str(player['steam_id']) and e.get('victim_team') == player['team']]
    require(len(witnesses) == 1, 'life callback witness differs')
    return deepcopy(value)


def checked_inventory(player, view, start):
    value = player.get("inventory_context")
    require(isinstance(value, dict) and value.get("schema_id") == "stratai.replay_inventory@1",
            "inventory provenance absent")
    require(set(value) <= {'schema_id','status','observation_tick','availability_tick','source','weapon','weapons','has_helmet'},
            'unsupported inventory field')
    status = value.get("status")
    require(status in ("observed", "last_observed", "unavailable"), "invalid inventory status")
    if status == "unavailable":
        require(value.get('source') in (None, {}), 'unavailable inventory has a source claim')
        require(value.get("observation_tick") is None and value.get("availability_tick") is None
                and value.get("weapon") is None and value.get("weapons") == []
                and value.get("has_helmet") is None, "unavailable inventory contains claims")
    else:
        observed, available = value.get("observation_tick"), value.get("availability_tick")
        # The round starts in freeze time; the selected window may start later.
        # A dated inventory from before that window remains history, never current.
        require(type(observed) is int and type(available) is int
                and 0 <= observed == available <= view["availability_tick"], "invalid inventory time")
        require(value.get("source") == {"kind": "demo_inventory_snapshot", "reference": f"frame:{observed}:inventory"},
                "inventory source differs")
        require(status != "observed" or player["alive"] and observed == view["availability_tick"],
                "stale or dead observed inventory")
        require(status != "last_observed" or observed < view["availability_tick"]
                or player.get('life_context') is not None and observed < checked_life(player, view)['observation_tick'],
                "historical inventory date not prior")
        require(isinstance(value.get("weapons"), list) and bool(value["weapons"])
                and all(isinstance(w, str) and w in NATIVE_EQUIPMENT for w in value["weapons"]),
                "invalid inventory items")
        require(value.get("weapon") is None or isinstance(value["weapon"], str) and value['weapon'] in NATIVE_EQUIPMENT,
                "invalid observed active weapon")
        require(type(value.get("has_helmet")) is bool, "invalid inventory helmet")
    return deepcopy(value)


def checked_objective(view):
    bomb = view["frame"].get("bomb")
    carriers = [str(p["steam_id"]) for p in view["frame"]["players"] if p["team"] == "T" and p.get("has_c4")]
    require(len(carriers) <= 1, "multiple C4 carriers")
    if not bomb:
        require(not carriers, "carrier without objective evidence")
        return
    require(bomb.get("state") in ("unknown", "carried", "dropped", "planting", "planted",
                                 "defusing", "defused", "exploded", "resolved"), "invalid objective state")
    expected = [str(bomb["carrier_id"])] if bomb.get("carrier_id") is not None else []
    require(carriers == expected, "objective carrier mismatch")
    require(not carriers or bomb["state"] in ("carried", "planting")
            and any(str(p["steam_id"]) == carriers[0] and p["alive"] for p in view["frame"]["players"]),
            "invalid live C4 carrier")


def checked_event_positions(view, events, start):
    for event in events:
        if any(k in event for k in ("x", "y", "z", "position_tick")):
            require(finite(event.get("x")) and finite(event.get("y"))
                    and ("z" not in event or finite(event["z"])), "invalid event position")
            require(event['type'] in ('utility_start', 'utility_end') or finite(event.get('z')),
                    'incomplete event position')
            tick = event.get("position_tick")
            # Damage coordinates belong to our victim at the native callback.
            # A separately observed attacker has its own, possibly earlier date.
            enemy_position = event["knowledge"] == "calculated_visual_event" and event['type'] != 'player_hurt'
            basis = event.get("observation", {}).get("tick") if enemy_position else event["tick"]
            require(type(tick) is int and start <= tick == basis <= event["tick"] <= view["tick"],
                    "event position date differs")


def checked_round_sequence(previous, current):
    if previous is None:
        return
    require(previous["round"] == current["round"], "round changed inside packet")
    require(previous.get("freeze_end_tick") == current.get("freeze_end_tick"), "round instance changed inside packet")
    if previous["phase"] == "ended":
        require(current["phase"] == "ended" and current["end_tick"] == previous["end_tick"]
                and current["winner"] == previous["winner"], "terminal round reopened or outcome changed")
    if previous["phase"] == "planted":
        require(current["phase"] in ("planted", "ended", "unknown"), "planted round reverted")

