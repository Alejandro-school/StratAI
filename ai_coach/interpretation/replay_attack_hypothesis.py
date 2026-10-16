"""Opt-in, expert-framed attack hypothesis over the unchanged protected facts.

Point 2 and a group of 2-3 are approved; no metric radius is approved. Geometry
is reproducible, while proximity remains an explicitly unevaluated model
judgment. This instrument does not label a region as a deterministic attack.
"""
from copy import deepcopy
import math
from itertools import combinations

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation.replay_factual_reading import StrictConsultation, number
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession, FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION
from ai_coach.support.anchors import require

VERSION = 'stratai.replay_attack_hypothesis@1'
BOUNDED_VERSION = 'stratai.replay_attack_hypothesis@2'
FOCUSED_VERSION = 'stratai.replay_attack_hypothesis@3'
HYPOTHESIS = 'possible_attack_b_advanced_apps'
PROMPT = """Select references for a Spanish contextual reading; never write prose.
situation/evolution/uncertainty retain the protected factual selection rules.
interpretation permits the existing conditional principles, or the single
expert-framed hypothesis possible_attack_b_advanced_apps with evidence containing
exactly 2 or 3 distinct eligible current own_player IDs. All supporting place,
round and objective facts are attached and checked by the application.
Select that hypothesis only if those players form a group near the approved
advanced B Apps point 2. Being somewhere in Apartments is insufficient. Compare
their supplied current positions and distances with the review landmarks.
Proximity is your explicit spatial estimate, not a fixed-radius fact or a route
distance. If they are farther back, dispersed, or the distinction is unclear,
abstain. Eligibility alone is not evidence of proximity or attack.
This is a possible attack, never certain intent, planting, effective support
from Mid, or a recommendation. Keep staging, withdrawal and another destination
open. Carrying C4 alone does not select a destination. Do not force a hypothesis.
Use only delivered IDs. Historical, dead or enemy tracks cannot count towards
this current own group; native hidden sound map points are review-only.
Earlier delivered start facts can be cited; other samples need a causal query.
"""
BOUNDED_PROMPT = PROMPT + """
For this bounded version, interpretation is either an empty array, up to eight
conditional-principle rows, OR exactly one attack-hypothesis row. Never repeat
the hypothesis or mix it with principle rows. For that row, choose exactly one
valid_reference_combinations entry, retaining its ID order. These combinations
only guarantee distinct current identities; they do NOT establish proximity.
Example shape: {"hypothesis_id":"possible_attack_b_advanced_apps","evidence":["id1","id2"]}.
Replace those example IDs with one delivered valid combination only when the
spatial criterion is supported. Otherwise leave interpretation empty.
Finish requires indices:[]; consult requires nonempty indices and reading:null.
"""
FOCUSED_PROMPT = BOUNDED_PROMPT + """
The application already includes ALL required_reading_facts in the reading.
Do not copy those references into situation, evolution or uncertainty. Those
arrays select only optional additional facts; empty arrays omit no protected
facts. Situation permits current facts, evolution historical/event facts, and
uncertainty only unknown/estimated/simulated facts. Select at most eight optional
references per factual section. Use the saved output to assess interpretation.
A possible-attack hypothesis does not require eliminating staging, withdrawal
or another destination. They remain open in its wording. Assess the supplied
current spatial group near point 2, without requiring confirmed intent, C4 in
the group, enemy contact, planting or verified map-build compatibility. Retain
the map-build limitation. Abstain if spatial grouping is unclear or unsupported.
Evaluate each valid 2-player or 3-player combination separately. A third player
farther away does not invalidate a nearby pair. Do not require every eligible
player to be grouped. Do not invent an implicit numeric proximity threshold.
The final spatial-review message duplicates exact current player/place facts
and their supplied distances in plain JSON. Use those linked current facts;
do not substitute a zone or position from an earlier sample. The full lossless
context remains available. The review contains no tactical assessment or label.
"""


class AttackHypothesisSession(ReplayReadingSession):
    def __init__(self, packet):
        super().__init__(packet)
        require(self._version in (FACTUAL_VERSION, INFORMATION_VERSION, UTILITY_VERSION, SOUND_VERSION),
                'hypothesis requires protected replay input')
        current = self._archive[-1]
        self._attack_support = [f['id'] for f in current if f['key'] in ('round_context', 'objective')]
        context = next(f for f in current if f['key'] == 'round_context')
        objective = next(f for f in current if f['key'] == 'objective')
        now = (packet['cut_tick']-packet['start_tick']) / packet['tick_rate']
        gate = (context['seconds_in_window'] == now and context['value']['phase'] == 'preplant'
                and context['value']['decision_status'] == 'active_round'
                and context['value']['pause_state'] != 'paused'
                and objective['seconds_in_window'] == now and objective['value'] is not None
                and objective['value']['state'] in ('carried', 'dropped'))
        places = {f['subject']: f for f in current if f['key'] == 'place_context'}
        rows = []
        for fact in current:
            value, place = fact['value'], places.get(fact['subject'])
            if (not gate or fact['key'] != 'own_player' or fact['status'] != 'observed'
                    or fact['seconds_in_window'] != now or value.get('alive') is not True
                    or value.get('team') != 'T' or not place or place['seconds_in_window'] != now
                    or place['value']['status'] != 'mapped' or place['value']['zone'] != 'Apartments'
                    or not all(type(value.get(k)) in (int, float) and math.isfinite(value[k]) for k in ('x', 'y', 'z'))):
                continue
            rows.append({'player_fact_id': fact['id'], 'place_fact_id': place['id'],
                         'distance_xy_units': math.hypot(value['x']+1500, value['y']-750)})
        self._view['hypothesis_contract'] = {
            'version': VERSION, 'hypothesis_id': HYPOTHESIS,
            'expert_criterion': '2-3 own T grouped near point 2 can support a possible attack B; not all Apartments',
            'advanced_reference': {'point': 2, 'position_xy': [-1500, 750]},
            'other_review_landmarks': [{'point': 1, 'position_xy': [-1900, 750]},
                                       {'point': 3, 'position_xy': [-1150, 600]}],
            'spatial_classification': 'model_estimate_no_metric_threshold',
            'geometry_limits': 'XY game units, not route distance; local volume build compatibility remains unverified',
            'eligible_players': rows, 'required_support': self._attack_support,
            'alternatives': ['staging', 'withdrawal', 'another_destination'],
            'not_established': ['plant_intent', 'effective_mid_support', 'defensive_reaction', 'recommendation']}

    def receipt(self):
        return {**super().receipt(), 'reading_contract': VERSION}

    def consultation(self):
        return AttackConsultation(self)


class AttackConsultation(StrictConsultation):
    hypothesis_selection = True
    hypothesis_adapter_version = '1.4.0-local.1'

    def interpretation_schema(self, conditional):
        ids = [r['player_fact_id'] for r in self.initial()['hypothesis_contract']['eligible_players']]
        if len(ids) < 2:
            return conditional
        return {'anyOf': [conditional, {
            'type': 'object', 'additionalProperties': False, 'required': ['hypothesis_id', 'evidence'],
            'properties': {'hypothesis_id': {'enum': [HYPOTHESIS]},
                           'evidence': {'type': 'array', 'minItems': 2, 'maxItems': 3, 'uniqueItems': True,
                                        'items': {'enum': ids}}}}]}

    def _hypothesis(self, row):
        require(set(row) == {'hypothesis_id', 'evidence'} and row['hypothesis_id'] == HYPOTHESIS,
                'unsupported hypothesis or free prose')
        ids = row['evidence']
        require(isinstance(ids, list) and 2 <= len(ids) <= 3 and all(isinstance(i, str) for i in ids)
                and len(set(ids)) == len(ids), 'hypothesis requires 2-3 distinct current players')
        eligible = {r['player_fact_id']: r for r in self.initial()['hypothesis_contract']['eligible_players']}
        require(all(i in eligible for i in ids), 'ineligible current attack evidence')
        players = [self._fact(i, 'situation') for i in ids]
        require(len({f['subject'] for f in players}) == len(ids), 'duplicate participant')
        support = ids + [eligible[i]['place_fact_id'] for i in ids] + self._session._attack_support
        facts = [self._fact(i, 'situation') for i in support]
        when = number(players[0]['seconds_in_window'])
        names = ', '.join(f['subject'] for f in players)
        return {'kind': 'inferred',
                'text': f'A los {when} s de la ventana. Posible ataque a B: {len(ids)} T ({names}) '
                        'en Apartments, con agrupación próxima al punto 2 estimada por el modelo. '
                        'La proximidad es una lectura espacial pendiente de revisión, sin radio táctico fijado. '
                        'También pueden estar preparando la jugada, replegarse o cambiar de destino. '
                        'No confirma intención de plantar ni apoyo efectivo desde Medio. '
                        'La referencia local del mapa mantiene compatibilidad de build no verificada.',
                'evidence': support, 'principles': [], 'facts': deepcopy(facts),
                'hypothesis': {'id': HYPOTHESIS, 'player_fact_ids': list(ids)}}

    def render(self, selection):
        require(isinstance(selection, dict) and set(selection) == set(comparison.SECTIONS), 'reading sections differ')
        rows = selection['interpretation']
        require(isinstance(rows, list) and len(rows) <= 256 and all(isinstance(r, dict) for r in rows),
                'invalid interpretation selection')
        hypotheses = [r for r in rows if 'hypothesis_id' in r]
        require(len(hypotheses) <= 1, 'duplicate attack hypothesis')
        plain = deepcopy(selection)
        plain['interpretation'] = [r for r in rows if 'hypothesis_id' not in r]
        reading, verification = super().render(plain)
        conditional = iter(reading['interpretation'])
        reading['interpretation'] = [self._hypothesis(r) if 'hypothesis_id' in r else next(conditional) for r in rows]
        verification.update(representation=VERSION, claims=sum(map(len, reading.values())),
                            spatial_hypothesis='model_estimate_not_independently_evaluated')
        return reading, verification

    def verify(self, reading):
        require(isinstance(reading, dict) and set(reading) == set(comparison.SECTIONS), 'reading sections differ')
        plain = deepcopy(reading)
        require(isinstance(plain['interpretation'], list), 'claims must be arrays')
        plain['interpretation'] = [c for c in plain['interpretation'] if isinstance(c, dict) and 'hypothesis' not in c]
        super().verify(plain)
        selection = {key: [c['evidence'][0] for c in plain[key]] for key in comparison.SECTIONS if key != 'interpretation'}
        selection['interpretation'] = []
        for claim in reading['interpretation']:
            require(isinstance(claim, dict), 'invalid rendered claim')
            if 'hypothesis' in claim:
                h = claim['hypothesis']
                require(isinstance(h, dict) and set(h) == {'id', 'player_fact_ids'}, 'invalid rendered hypothesis')
                selection['interpretation'].append({'hypothesis_id': h['id'], 'evidence': h['player_fact_ids']})
            else:
                selection['interpretation'].append({'principle_id': claim['principles'][0], 'evidence': claim['evidence']})
        expected, verification = self.render(selection)
        require(reading == expected, 'rendered hypothesis or text differs from evidence')
        return verification


class BoundedAttackHypothesisSession(AttackHypothesisSession):
    """Same game criterion, with a grammar that cannot enumerate hypotheses."""
    def __init__(self, packet):
        super().__init__(packet)
        contract = self._view['hypothesis_contract']
        ids = sorted(row['player_fact_id'] for row in contract['eligible_players'])
        contract.update(version=BOUNDED_VERSION,
                        valid_reference_combinations=[list(group) for size in (2, 3) for group in combinations(ids, size)],
                        reference_combination_scope='distinct eligible identities only; not evidence of proximity',
                        interpretation_mode='zero-to-eight conditional principles OR one optional attack hypothesis')

    def receipt(self):
        return {**super().receipt(), 'reading_contract': BOUNDED_VERSION}

    def consultation(self):
        return BoundedAttackConsultation(self)


class BoundedAttackConsultation(AttackConsultation):
    hypothesis_adapter_version = '1.5.0-local.1'

    def interpretation_schema(self, conditional):
        # The bounded envelope builds alternatives of whole arrays, so an
        # attack row cannot repeat or become mixed with conditional rows.
        return conditional

    def envelope_schema(self, reading, through):
        reading = deepcopy(reading)
        conditional = reading['properties']['interpretation']['items']
        alternatives = [{'type': 'array', 'maxItems': 8, 'items': conditional}]
        groups = self.initial()['hypothesis_contract']['valid_reference_combinations']
        if groups:
            alternatives.append({'type': 'array', 'minItems': 1, 'maxItems': 1, 'items': {
                'type': 'object', 'additionalProperties': False, 'required': ['hypothesis_id', 'evidence'],
                'properties': {'hypothesis_id': {'enum': [HYPOTHESIS]}, 'evidence': {'enum': groups}}}})
        reading['properties']['interpretation'] = {'anyOf': alternatives}
        def envelope(action, indices, result):
            return {'type': 'object', 'additionalProperties': False, 'required': ['action', 'indices', 'reading'],
                    'properties': {'action': {'enum': [action]}, 'indices': indices, 'reading': result}}
        return {'anyOf': [envelope('finish', {'enum': [[]]}, reading),
                          envelope('consult', {'type': 'array', 'minItems': 1, 'maxItems': comparison.QUERY_LIMIT,
                                               'items': {'type': 'integer', 'minimum': 0, 'maximum': through}},
                                   {'type': 'null'})]}

    def render(self, selection):
        require(isinstance(selection, dict) and set(selection) == set(comparison.SECTIONS), 'reading sections differ')
        rows = selection['interpretation']
        require(isinstance(rows, list) and len(rows) <= 8 and all(isinstance(r, dict) for r in rows),
                'invalid bounded interpretation')
        if any('hypothesis_id' in row for row in rows):
            require(len(rows) == 1, 'one attack hypothesis without mixed or repeated rows')
            require(rows[0].get('evidence') in self.initial()['hypothesis_contract']['valid_reference_combinations'],
                    'not a canonical distinct-player combination')
        reading, verification = super().render(selection)
        verification['representation'] = BOUNDED_VERSION
        return reading, verification


class FocusedAttackHypothesisSession(BoundedAttackHypothesisSession):
    """Same full factual input and game criterion; no mandatory-reference copying."""
    def __init__(self, packet):
        super().__init__(packet)
        contract = self._view['hypothesis_contract']
        facts = {f['id']: f for f in self._view['situation']}
        contract.update(
            version=FOCUSED_VERSION,
            factual_selection='up to eight optional additional references per section; all required facts rendered automatically',
            spatial_review={
                'version': 'stratai.attack_spatial_review@1',
                'origin': 'exact duplicate of delivered current facts, not new evidence or a tactical label',
                'advanced_reference': deepcopy(contract['advanced_reference']),
                'geometry_limits': contract['geometry_limits'],
                'valid_reference_combinations': deepcopy(contract['valid_reference_combinations']),
                'eligible_players': [{**row,
                    'current_player_fact': deepcopy(facts[row['player_fact_id']]),
                    'current_place_fact': deepcopy(facts[row['place_fact_id']])}
                    for row in contract['eligible_players']]})

    def receipt(self):
        return {**super().receipt(), 'reading_contract': FOCUSED_VERSION}

    def consultation(self):
        return FocusedAttackConsultation(self)


class FocusedAttackConsultation(BoundedAttackConsultation):
    hypothesis_adapter_version = '1.6.0-local.1'

    def spatial_review(self):
        return self.initial()['hypothesis_contract']['spatial_review']

    def envelope_schema(self, reading, through):
        reading = deepcopy(reading)
        required = self.initial()['required_reading_facts']
        for section in ('situation', 'evolution', 'uncertainty'):
            optional = []
            for identity in sorted(self._consulted - set(required[section])):
                # Apply the same section/date/status checks as the renderer,
                # before generation rather than only rejecting the final answer.
                try:
                    self._fact(identity, section)
                except ValueError:
                    continue
                optional.append(identity)
            reading['properties'][section] = (
                {'type': 'array', 'maxItems': 8, 'items': {'enum': optional}}
                if optional else {'enum': [[]]})
        return super().envelope_schema(reading, through)

    def render(self, selection):
        require(isinstance(selection, dict), 'reading sections differ')
        required = self.initial()['required_reading_facts']
        for section in ('situation', 'evolution', 'uncertainty'):
            ids = selection.get(section)
            require(isinstance(ids, list) and all(isinstance(i, str) for i in ids), 'invalid optional references')
            # Verification reconstructs the rendered selection including the
            # automatic facts; only genuinely additional references cost slots.
            require(sum(i not in required[section] for i in ids) <= 8, 'too many optional references')
        reading, verification = super().render(selection)
        verification['representation'] = FOCUSED_VERSION
        return reading, verification
