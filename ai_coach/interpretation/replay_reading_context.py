"""Opt-in development instrument from a causal team replay projection.

No raw enemy snapshots, future rounds, human comments or historical readings
are accepted. The frozen CS3 comparison remains a separate experiment.
"""
from copy import deepcopy
import hashlib
import json
import math

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.support.anchors import require
from ai_coach.interpretation.replay_place_context import checked_place
from ai_coach.interpretation.place_volumes import load_reference
from ai_coach.interpretation.replay_round_context import checked_round_context
from ai_coach.interpretation.replay_event_context import checked_events, checked_utility
from ai_coach.interpretation import replay_factual_validation as factual
from ai_coach.interpretation.replay_information import checked_information
from ai_coach.interpretation.replay_utility_information import checked_utility_information
from ai_coach.interpretation.replay_sound_information import checked_sound_information
from ai_coach.interpretation.place_volumes import resolve

VERSION = "stratai.replay_context_input@1"
PROJECTION = "stratai.team_replay_projection@3"
C4_VERSION = "stratai.replay_context_input@2"
C4_PROJECTION = "stratai.team_replay_projection@4"
ZONES_VERSION = "stratai.replay_context_input@3"
ZONES_PROJECTION = "stratai.team_replay_projection@5"
ROUND_VERSION = "stratai.replay_context_input@4"
ROUND_PROJECTION = "stratai.team_replay_projection@6"
EVENTS_VERSION = "stratai.replay_context_input@5"
EVENTS_PROJECTION = "stratai.team_replay_projection@7"
FACTUAL_VERSION = "stratai.replay_context_input@6"
FACTUAL_PROJECTION = "stratai.team_replay_projection@8"
INFORMATION_VERSION = "stratai.replay_context_input@7"
INFORMATION_PROJECTION = "stratai.team_replay_projection@9"
UTILITY_VERSION = "stratai.replay_context_input@8"
UTILITY_PROJECTION = "stratai.team_replay_projection@10"
SOUND_VERSION = "stratai.replay_context_input@9"
SOUND_PROJECTION = "stratai.team_replay_projection@11"


def digest(value):
    return hashlib.sha256(comparison.encoded(value)).hexdigest()


class ReplayReadingSession:
    def __init__(self, packet):
        self._version = packet.get("schema_id")
        require(self._version in (VERSION, C4_VERSION, ZONES_VERSION, ROUND_VERSION, EVENTS_VERSION, FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION), "unknown replay input instrument")
        sound = self._version == SOUND_VERSION
        utility = self._version in (UTILITY_VERSION, SOUND_VERSION)
        information = self._version in (INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION)
        strict = self._version in (FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION)
        actions = self._version in (EVENTS_VERSION, FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION)
        rounds = self._version in (ROUND_VERSION, EVENTS_VERSION, FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION)
        zones = self._version in (ZONES_VERSION, ROUND_VERSION, EVENTS_VERSION, FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION)
        c4 = self._version != VERSION
        projection = {VERSION:PROJECTION, C4_VERSION:C4_PROJECTION, ZONES_VERSION:ZONES_PROJECTION, ROUND_VERSION:ROUND_PROJECTION, EVENTS_VERSION:EVENTS_PROJECTION, FACTUAL_VERSION:FACTUAL_PROJECTION, INFORMATION_VERSION:INFORMATION_PROJECTION, UTILITY_VERSION:UTILITY_PROJECTION, SOUND_VERSION:SOUND_PROJECTION}[self._version]
        if zones:
            require(packet.get("map_name") == "de_mirage" and packet.get("place_reference_id") == load_reference()["reference_id"], "map reference differs")
        require(packet.get("side") == "T", "only framed T development readings")
        rate, start, cut = packet["tick_rate"], packet["start_tick"], packet["cut_tick"]
        if strict:
            require(factual.finite(rate) and type(start) is int and type(cut) is int, "invalid frontier types")
        require(rate > 0 and start <= cut, "invalid causal frontier")
        views = packet["views"]
        require(bool(views) and views[-1]["tick"] == cut, "missing final projected view")
        self._archive, events, previous = [], {}, start - 1
        previous_round = None
        previous_information = {}
        previous_utility = {}
        previous_sound = {}
        for index, view in enumerate(views):
            require(view.get("schema_id") == projection and view.get("side") == "T", "projection differs")
            if zones:
                require(view.get("place_reference_id") == packet["place_reference_id"], "view map reference differs")
            require(start <= view["tick"] <= cut and view["tick"] > previous, "future or unordered view")
            require(view["availability_tick"] <= view["tick"], "future snapshot")
            previous = view["tick"]
            seconds = (view["tick"] - start) / rate
            frame = view["frame"]
            if strict:
                factual.checked_players(view, start)
                factual.checked_objective(view)
            bomb = frame.get("bomb")
            if c4 and bomb:
                require(start <= bomb["observation_tick"] <= bomb["availability_tick"] <= view["tick"], "future objective")
                require(bool(bomb.get("source", {}).get("reference")), "objective source absent")
                has_position = "x" in bomb or "y" in bomb
                require(not has_position or (bomb.get("position_status") == "observed"
                        and bomb.get("position_tick") == view["tick"]
                        and all(isinstance(bomb.get(k), (int, float)) and math.isfinite(bomb[k]) for k in ("x","y"))), "stale objective position")
                historical = bomb.get("last_observed_position")
                if historical:
                    require(start <= historical["observation_tick"] < view["tick"]
                            and bool(historical.get("source", {}).get("reference"))
                            and all(isinstance(historical.get(k), (int,float)) and math.isfinite(historical[k]) for k in ("x","y")), "invalid historical objective position")
                carriers = [str(p["steam_id"]) for p in frame["players"] if p["team"] == "T" and p.get("has_c4")]
                expected = [str(bomb["carrier_id"])] if bomb.get("carrier_id") is not None else []
                require(carriers == expected, "conflicting objective carrier")
            facts = []
            def fact(subject, key, value, status="observed"):
                facts.append({"id":f"S{index}:{len(facts)}", "subject":subject, "key":key,
                              "value":deepcopy(value), "status":status, "seconds_in_window":seconds})
            round_context = checked_round_context(view, rate, strict=strict) if rounds else None
            if strict:
                factual.checked_round_sequence(previous_round, round_context)
                previous_round = round_context
            remaining = round_context['round_remaining_seconds'] if rounds else frame.get('time_remaining')
            fact(None, "round_remaining_seconds", remaining, "observed" if remaining is not None else "unknown")
            if rounds:
                fact(None, "round_context", round_context)
                if strict:
                    facts[-1]['seconds_in_window'] = (round_context['observation_tick'] - start) / rate
            fact(None, "objective", frame.get("bomb"), "observed" if frame.get("bomb") else "unknown")
            if strict and bomb:
                facts[-1]['seconds_in_window'] = (bomb['observation_tick'] - start) / rate
            own_alive = 0
            for player in frame["players"]:
                identity = str(player["steam_id"])
                require(identity.isdigit() and 1 <= int(identity) <= 10, "nonlocal participant identity")
                alias = f"P{int(identity):02d}"
                if player["team"] == "T":
                    require(player.get("knowledge") == "own_team_snapshot", "unprojected own player")
                    own_alive += int(player["alive"])
                    keys = ("team","alive","x","y","z","health","armor","weapon","weapons","money","has_helmet")
                    if strict:
                        keys = tuple(key for key in keys if key not in ("weapon", "weapons", "has_helmet"))
                    if c4: keys += ("has_c4",)
                    life = player.get('life_context') if strict else None
                    if life:
                        keys = tuple(key for key in keys if key not in ('alive', 'health', 'has_c4'))
                    elif strict and view['tick'] != player['observation_tick']:
                        keys = tuple(key for key in keys if key != 'has_c4')
                    fact(alias, "own_player", {key:player[key] for key in keys if key in player})
                    if strict:
                        facts[-1]['seconds_in_window'] = (player['observation_tick'] - start) / rate
                        if life:
                            fact(alias, 'own_life', factual.checked_life(player, view))
                            facts[-1]['seconds_in_window'] = (life['observation_tick'] - start) / rate
                        inventory = factual.checked_inventory(player, view, start)
                        fact(alias, "own_inventory", inventory, {"observed":"observed", "last_observed":"estimated", "unavailable":"unknown"}[inventory["status"]])
                        if inventory["observation_tick"] is not None:
                            facts[-1]["seconds_in_window"] = (inventory["observation_tick"] - start) / rate
                else:
                    require(player.get("knowledge") == "calculated_visual_contact" and player["team"] == "CT", "unprojected enemy")
                    require(player["availability_tick"] <= view["tick"] < player["observation_tick"] + 4, "stale enemy contact")
                    require(player["alive"] is True, "dead live contact")
                    fact(alias, "enemy_contact", {"team":"CT", "alive":True,
                         "position":{key:player[key] for key in ("x","y","z") if key in player},
                         "contact_age_seconds":(view["tick"]-player["observation_tick"])/rate}, "estimated")
                if zones:
                    place = checked_place(player, start_tick=start, cut_tick=view["tick"])
                    if place:
                        fact(alias, "place_context", place, "estimated" if place["status"] == "mapped" else "unknown")
                        if strict:
                            facts[-1]['seconds_in_window'] = (place['observation_tick'] - start) / rate
            if information:
                info = checked_information(view, start, previous_information)
                refs = {o['id']: o for o in info['observations']}
                fact(None, 'information_policy', {k: info[k] for k in ('communication_model', 'reception_status', 'own_team_state_scope', 'enemy_identity_scope')}, 'simulated')
                for contact in info['contacts']:
                    value = {**contact, 'age_seconds': contact['age_ticks'] / rate,
                             'observations': [refs[r] for r in contact['observation_ids']]}
                    value['last_observed_place'] = resolve(contact['position'], load_reference(), map_name='de_mirage', xy_precision=0)
                    fact(f"P{int(contact['subject_id']):02d}", 'enemy_information', value, 'estimated')
                for player in info['by_player']:
                    fact(f"P{int(player['player_id']):02d}", 'possible_player_information',
                         {'observations': [{**refs[r], 'age_seconds': (view['tick'] - refs[r]['observation_tick']) / rate}
                                           for r in player['observation_ids']]}, 'simulated')
            if utility:
                signals = checked_utility_information(view, rate, previous_utility)
                fact(None, 'utility_information_policy', {k: signals[k] for k in ('source_status', 'communication_model', 'reception_status')}, 'estimated')
                for obj in signals['objects']:
                    fact(None, 'utility_signal', {**obj, 'age_seconds': obj['age_ticks'] / rate,
                        'observed_place': resolve(obj['position'], load_reference(), map_name='de_mirage', xy_precision=0) if obj['position'] else None}, 'estimated')
                for estimate in signals['launcher_estimates']:
                    fact(None, 'utility_launcher_estimate', {**estimate, 'interval_seconds': (estimate['last_launch_tick'] - estimate['first_launch_tick']) / rate}, 'estimated')
                for segment in signals['segments']:
                    fact(None, 'utility_segment', segment, 'estimated')
                for player in signals['by_player']:
                    fact(f"P{int(player['player_id']):02d}", 'possible_utility_information', player, 'simulated')
            if sound:
                signals = checked_sound_information(view, previous_sound)
                fact(None, 'sound_information_policy', {k:signals[k] for k in ('source_status','communication_model','reception_status','geometry_model','build_compatibility','perception')}, 'estimated')
                for row in signals['observations']:
                    # Raw native coordinates, exact geometric distance and profile
                    # names/hashes are audit inputs, not perceived enemy facts.
                    value = {k:row[k] for k in ('id','observer_id','kind','source_side','side_status','region','observation_tick','availability_tick','status','recipient_basis','recipients')}
                    value['source'] = {k:row['source'][k] for k in ('kind','reference','emission_reference','catalog_id','catalog_sha256')}
                    value['age_seconds'] = (view['tick']-row['observation_tick'])/rate
                    fact(None, 'sound_signal', value, 'estimated')
                for player in signals['by_player']:
                    fact(f"P{int(player['player_id']):02d}", 'possible_sound_information', player, 'simulated')
            fact(None, "own_alive_count", own_alive)
            fact(None, "contact_status", view["vision_status"], "estimated")
            fact(None, "uncertainty", {**view["uncertainty"], "eligibility":round_context['eligibility'] if rounds else "unknown",
                 "enemy_alive_count":"unknown", "prior_round_patterns":"not_supplied"}, "unknown")
            for death in view["deaths"]:
                require(start <= death["tick"] <= view["tick"] and death["availability_tick"] <= view["tick"], "future death")
                key = (death["tick"], death["victim_id"])
                if key not in events:
                    events[key] = {"id":f"E{len(events)+1}", "subject":f"P{int(death['victim_id']):02d}",
                        "key":"death", "value":{"team":death["team"], "position":death["position"],
                        "seconds_in_window":(death["tick"]-start)/rate, "knowledge":"simulated_team_communication"},
                        "status":"simulated" if strict else "observed", "seconds_in_window":(death["tick"]-start)/rate}
            if c4:
                for event in view.get("events", []):
                    if not event["type"].startswith("bomb_"): continue
                    require(start <= event["tick"] <= event["availability_tick"] <= view["tick"], "future objective event")
                    require(event.get("knowledge") in ("own_team_objective_event", "public_objective_event"), "unprojected objective event")
                    require(bool(event.get("source", {}).get("reference")), "objective event source absent")
                    actor = event.get("player_id")
                    require(actor is None or (str(actor).isdigit() and 1 <= int(actor) <= 10), "nonlocal objective actor")
                    require(actor is None or any(str(p["steam_id"]) == str(actor) and p["team"] == "T" for p in frame["players"]), "hidden objective actor")
                    key = ("objective", event["id"])
                    if key not in events:
                        events[key] = {"id":f"E{len(events)+1}", "subject":f"P{int(actor):02d}" if actor else None,
                            "key":event["type"], "value":{k:deepcopy(event[k]) for k in
                                ("tick","availability_tick","knowledge","source","state_after","objective_phase","objective_sequence") if k in event},
                            "status":"observed", "seconds_in_window":(event["tick"]-start)/rate}
            if actions:
                action_rows=checked_events(view,start)
                if strict:
                    factual.checked_event_positions(view, action_rows, start)
                fact(None, 'own_utility_tracking', checked_utility(view,action_rows), 'simulated')
                for event in action_rows:
                    key=('action',event['id'])
                    if key in events:
                        require(events[key]['value']==event, 'event changed after availability')
                        continue
                    actor=event.get('actor_id') or event.get('killer_id')
                    events[key]={'id':f'E{len(events)+1}', 'subject':f'P{int(actor):02d}' if actor else None,
                        'key':event['type'], 'value':event,
                        'status':'estimated' if event['knowledge']=='calculated_visual_event' else 'observed',
                        'seconds_in_window':(event['tick']-start)/rate}
            self._archive.append(facts)
        times=[(v["tick"]-start)/rate for v in views]
        self._events=list(events.values())
        self._view={"version":self._version, "side":"T", "map":"de_mirage", "instrument":self._version,
            "through_index":len(views)-1, "sample_times_seconds":times,
            "situation":self._archive[-1], "events":self._events,
            "evolution":{"sample_count":len(views), "duration_seconds":times[-1],
                         "boundaries":{"start":self._archive[0]}},
            "reading_constraints":[
                "P01..P10 are the same local identities on map and in all samples.",
                "Situation describes the final cut; earlier deaths are dated evolution. Never mix player identities or time cuts.",
                "Enemy contacts are calculated estimates; absence of contact is not enemy absence. Death positions simulate team communication and do not reveal current enemy positions.",
                "Consult samples to justify evolution. Sample indices are not seconds. Do not invent zones, trajectories, tactical intent, kills, damage, or utility not supplied.",
                "Describe meaningful distribution with supplied world coordinates; named map zones are unknown in this instrument.",
                "No previous rounds, exact enemy total, eligibility, or hidden resources supplied. Explain limits locally."],
            "prior_rounds_status":"not_supplied", "native_status":"not_supplied",
            "bank":{"principles":deepcopy(list(comparison.BANK)), "examples":deepcopy(list(comparison.EXAMPLES))},
            "consultation":{"max_queries":comparison.QUERY_LIMIT, "max_response_bytes":comparison.QUERY_BYTES,
                            "operation":"sample(index)", "frontier":len(views)-1}}
        if c4:
            self._view["reading_constraints"].append(
                "C4 possession comes only from objective.carrier_id and has_c4, never cached weapons. "
                "State, position availability and objective phase are separate. A resolved round does not prove defuse or explosion. "
                "Carrying or dropping C4 does not establish tactical intent. Objective events are dated evolution; never reuse their coordinates as current.")
        if zones:
            self._view["reading_constraints"][4] = (
                "Use supplied place_context for named zones. Membership uses recovered 3D Source 2 volumes, not nearest landmarks. "
                "Apartments denotes the B apartments in this local Mirage reference; Catwalk denotes the B catwalk/short. "
                "Exact demo-to-map-build compatibility remains unverified: describe zones as local-map references. "
                "Boundary, overlap and uncovered points have no unique zone. Do not resolve them from proximity or invent a named zone. "
                "A place belongs to its dated player position; it establishes neither route, sight line, intent nor hidden enemy knowledge.")
            self._view["place_reference_id"] = packet["place_reference_id"]
        if rounds:
            self._view['native_status'] = 'round_context_supplied'
            self._view['reading_constraints'][5] = (
                'Round context separates game phase, eligibility and clocks. A held knife is not a knife-round classifier. '
                'An ended round is a closure, not an active decision to win it. Winner is known only at the dated end. '
                'Elapsed demo seconds are not a remaining game clock; null countdowns are unknown, never zero. '
                'Bomb and preplant clocks are separate. No previous rounds or exact enemy total supplied.')
        self._source_hash=digest(packet)
        if strict:
            # Essential current facts and objective succession cannot disappear
            # merely because the model selects only a few salient references.
            self._view['required_reading_facts'] = {
                'situation': [f['id'] for f in self._archive[-1]],
                'evolution': [f['id'] for f in self._archive[0] if f['key'] in ('objective', 'round_context') and f['id'] not in {r['id'] for r in self._archive[-1]}]
                             + [e['id'] for e in self._events if e['key'].startswith('bomb_')],
                'uncertainty': [f['id'] for f in self._archive[-1] if f['status'] in ('unknown', 'estimated', 'simulated')],
            }
        if actions:
            self._view['reading_constraints'].append(
                'Events have causal callback times and perspective sources. Enemy actions based on calculated contacts are estimates. '
                'A launch does not establish detonation; fire does not establish a hit or effective support. '
                'Own utility trajectories/effects simulate team tracking, not confirmed player view or actual communication. '
                'Killfeed identity does not reveal location or resources. Temporal sequences may support a possible exchange, '
                'never prove tactical intent, efficacy or audibility. Final damage aggregates and future effect durations are excluded.')
        if information:
            self._view['reading_constraints'].append(
                'Contact memory is dated: last_observed is not a current enemy position or proof of current life. '
                'Each observation records its living observer and present possible recipients under immediate simulated team communication. '
                'Possible reception never proves actual speech, hearing, attention or remembered knowledge. '
                'Death prevents new direct visual contacts but does not exclude possible shared reception. '
                'Keep the latest direct and possible shared reports separately; an unconfirmed message cannot erase a direct report. '
                'Missing recipient evidence does not prove they could not receive information from another source. '
                'Own team snapshots are a collective demo overview, not proof of each player knowing all allied resources. '
                'Enemy IDs correlate opaque demo tracks, not recognized identities; matching an ID alone does not establish a perceived rotation. '
                'Do not infer sound, utility origins or cleared zones from this information.')

        if utility:
            self._view['reading_constraints'].append(
                'Utility signals are geometric visual estimates with dated points and possible simulated recipients. '
                'Never read effect points as full visible geometry, or late projectile sightings as observable launch times. '
                'Three same-type observable HE/smoke/molotov/incendiary launches within five seconds support only the agreed '
                'estimated minimum of three launchers. Flashes are excluded. Pickups/extra supply are not excluded. '
                'The launcher side, origin and current location stay unknown; overlapping groups cannot be added. '
                'Impact near a site does not locate defenders there. No sound or efficacy is supplied.')

        if sound:
            self._view['reading_constraints'][-1] = self._view['reading_constraints'][-1].replace(
                'No sound or efficacy is supplied.', 'No sound or efficacy is supplied by this visual utility facet.')
            self._view['reading_constraints'].append(
                'Sound is a separate native-emission estimate using fixed pinned game profiles and 3D native points. '
                'All directions and heights are eligible; visual FOV, walls and smoke are not acoustic exclusions. '
                'Possible direct reception and immediate simulated possible sharing never establish actual hearing, '
                'attention, recognized identity, exact perceived position, enemy count or a currently occupied zone. '
                'Native source side and compatible region are demo-derived estimates, not player recognition. '
                'Missing/unsupported/local-only/zero-volume sources abstain; profile build applicability and masking remain unverified. '
                'Dated direct and shared memories remain separate. Multiple signals do not count enemies.')

    def consultation(self):
        if self._version in (FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION):
            from ai_coach.interpretation.replay_factual_reading import StrictConsultation
            return StrictConsultation(self)
        return comparison.Consultation(self)

    def receipt(self):
        return {"input_hash":digest(self._view), "projected_packet_hash":self._source_hash,
                "version":self._version, "training_allowed":False, "product_supported":False,
                "independent_evaluation":False}
