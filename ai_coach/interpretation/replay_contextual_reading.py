"""Opt-in contextual drafts with protected facts and explicit semantic limits."""
from copy import deepcopy
import re
import unicodedata

import jsonschema

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation.replay_attack_region import membership
from ai_coach.interpretation.replay_factual_reading import StrictConsultation
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession, digest
from ai_coach.support.anchors import require

VERSION = 'stratai.replay_contextual_reading@1'
PROMPT = """Explain this CS2 situation in Spanish using the current cut AND the
actual dated history. Output JSON only. Factual sections select references;
the application reconstructs all required facts, including unique C4 and dates.
Select at most eight optional additional references per factual section;
do not copy required_reading_facts back into the selection.
Interpretation is ONE decision: status=hypotheses with one or more contextual
hypotheses, or status=insufficient with no hypotheses and a specific nonempty
missing_information. Never declare support then omit the supported hypothesis.
Each hypothesis has hypothesis, plausibility (possible/preferred/undetermined),
evidence, counterevidence, limits and would_change. Cite delivered fact IDs only.
Use concise natural hypotheses, not a fixed tactical label catalogue. Several
compatible explanations can coexist; do not force two, an attack or a destination.
Distinguish what happened from inferred intent. Explain contrary evidence and
what future information would change the reading. Missing information is local.
Spatial measurements are approximate evidence, never a gate or automatic intent.
Initial own history contains the actual start boundary; all other samples remain
available through sample(index), with the full sample-time catalogue. Consult
appropriate samples before making claims about recent progress or waiting.
History contains discrete actual samples, not a continuous trajectory. Two
endpoints cannot prove recent progress, waiting throughout the gap or the route.
Similar current positions can have different histories; identical available
evidence can leave several hidden realities open. Quiet does not establish fake.
Do not restate quantities, resources, clocks or enemy counts in contextual prose;
these belong to protected referenced facts. No numeric probabilities or confidence
scores. C4 and delivered participant aliases are identifiers. Qualify inference:
these are unverified drafts for review, not factual or tactical validation.
Early utility can be routine anti-rush; utility plus reload alone does not prove
a defensive reaction. Possible simulated communication is not actual reception.
Sound map-only native points are not model evidence. Do not infer exact hidden
enemy locations, identities or counts, effective Mid support or a planting plan.
No recommendations, new action scoring, training labels or retrospective hindsight.
Consult earlier samples only with supplied query/byte limits. Finish has indices:[];
consult_history has nonempty indices and reading:null and returns only dated own
positions, zones, objective and round state. Prefer it for movement or waiting.
consult returns full historical facets when needed. Request few relevant indices,
not every sample. Already supplied or attempted samples are not offered again.
An explicit not_delivered budget refusal supplies NO facts or new citations.
The final available call must finish with hypotheses OR explicit insufficiency,
never a forced tactical answer. Never invent an unavailable sample.
"""
_HISTORY_KEYS = {'own_player', 'own_life', 'place_context', 'objective', 'round_context'}
_QUANTITIES = re.compile(r'\b(?:cero|uno|dos|tres|cuatro|cinco|seis|siete|ocho|nueve|diez|once|doce|'
                         r'trece|catorce|quince|dieciseis|diecisiete|dieciocho|diecinueve|veinte|'
                         r'treinta|cuarenta|cincuenta|sesenta|setenta|ochenta|noventa|cien|ciento|mil|'
                         r'zero|one|two|three|four|five|six|seven|eight|nine|ten)\b', re.IGNORECASE)


class ContextualReadingSession(ReplayReadingSession):
    def __init__(self, packet):
        super().__init__(packet)
        require(self._version in ('stratai.replay_context_input@6', 'stratai.replay_context_input@7',
                                  'stratai.replay_context_input@8', 'stratai.replay_context_input@9'),
                'contextual reading requires protected input')
        boundary = self._archive[:1] if len(self._archive) > 1 else []
        self._view['own_context_history'] = [deepcopy(f) for sample in boundary
                                            for f in sample if f['key'] in _HISTORY_KEYS]
        self._view['history_coverage'] = {
            'sample_times_seconds': deepcopy(self._view['sample_times_seconds']),
            'continuity': 'discrete_samples_only',
            'initial_history_indices': [0] if boundary else [],
            'additional_history': 'all_actual_samples_available_via_causal_sample_index',
            'route_and_intent': 'not_established_by_sample_endpoints',
            'other_historical_facets': 'causal_consultation_with_original_limits',
        }
        places = {f['subject']: f for f in self._view['situation'] if f['key'] == 'place_context'}
        lives = {f['subject']:f['value']['alive'] for f in self._view['situation'] if f['key'] == 'own_life'}
        self._view['spatial_measurements'] = [
            {'player_fact_id': f['id'], 'place_fact_id': places[f['subject']]['id'],
             'membership': membership(f['value'], places[f['subject']]['value'], map_name=packet['map_name'])}
            for f in self._view['situation'] if f['key'] == 'own_player'
            and lives.get(f['subject'],f['value'].get('alive')) is True
            and f['subject'] in places
        ]
        self._view['contextual_reading_contract'] = {
            'version': VERSION, 'tactical_label_catalogue': 'none',
            'spatial_measurements': 'evidence_only_not_hypothesis_eligibility',
            'output_status': 'unverified_contextual_draft_for_review',
            'history_operations': {'consult_history':'own position/place/objective/round only',
                                   'consult':'all historical facets'},
            'consultation_progress':'no repeated supplied/attempted scope; explicit budget refusal; final call finishes',
        }

    def consultation(self):
        return ContextualConsultation(self)

    def receipt(self):
        return {**super().receipt(), 'reading_contract': VERSION}


class ContextualConsultation(StrictConsultation):
    contextual_selection = True

    def __init__(self, session):
        super().__init__(session)
        self._consulted.update(f['id'] for f in session._view['own_context_history'])
        self._query_attempts = []
        self._context_budget_refusals = []
        self._remaining_model_calls = 8

    def available_indices(self, scope):
        attempted = {(r['scope'],r['index']) for r in self._query_attempts}
        return [index for index, facts in enumerate(self._session._archive)
                if (scope,index) not in attempted and any(f['id'] not in self._consulted for f in facts
                    if scope == 'full' or f['key'] in _HISTORY_KEYS)]

    def query(self, index, scope):
        require(scope in ('history','full') and type(index) is int
                and index in self.available_indices(scope), 'query makes no causal disclosure progress')
        require(max(len(self._queries),len(self._query_attempts)) < comparison.QUERY_LIMIT, 'query budget exhausted')
        self._query_attempts.append({'index':index,'scope':scope})
        if scope == 'full':
            return super().sample(index)
        facts = deepcopy([f for f in self._session._archive[index] if f['key'] in _HISTORY_KEYS])
        size = len(comparison.encoded(facts))
        require(self._bytes + size <= comparison.QUERY_BYTES, 'response byte budget exhausted')
        self._bytes += size
        self._queries.append({'index':index,'scope':'history','bytes':size,'response_hash':digest(facts)})
        self._consulted.update(f['id'] for f in facts)
        return facts

    def runtime_state(self):
        return {'query_attempts':deepcopy(self._query_attempts),
                'context_budget_refusals':deepcopy(self._context_budget_refusals)}

    def current_overview(self):
        """Duplicate factual text and type/ID index; never a tactical summary."""
        initial = self.initial()
        catalogue = {}
        for fact in initial['situation']:
            catalogue.setdefault(fact['key'], {}).setdefault(fact['status'], []).append(fact['id'])
        return {'version':'stratai.contextual_factual_overview@1',
            'purpose':'Índice factual; contexto íntegro en el mensaje anterior.',
            'facts':[{'id':f['id'],'text':self._claim(f)['text']} for f in initial['situation'] if f['key'] in _HISTORY_KEYS],
            'catalogue':{key:{status:' '.join(ids) for status,ids in states.items()}
                         for key,states in catalogue.items()},
            'spatial_excerpt':[{'player_fact_id':m['player_fact_id'],'place_fact_id':m['place_fact_id'],
                               'relation':m['membership']['relation'],'status':m['membership']['status']}
                              for m in initial['spatial_measurements']],
            'limits':'Región aproximada; señales estimadas no confirman acciones. Conservar fechas y ámbitos.',
            'output':'Responde en español, sin cifras ni índices libres; utiliza referencias.'}

    def envelope_schema(self):
        reference = {'enum': sorted(self._consulted)}
        references = {'type': 'array', 'maxItems': 32, 'uniqueItems': True, 'items': reference}
        text = {'type': 'string', 'minLength': 1, 'maxLength': 600}
        hypothesis = {'type': 'object', 'additionalProperties': False,
            'required': ['hypothesis', 'plausibility', 'evidence', 'counterevidence', 'limits', 'would_change'],
            'properties': {'hypothesis': text, 'plausibility': {'enum': ['possible','preferred','undetermined']},
                'evidence': {**references, 'minItems': 1}, 'counterevidence': references,
                'limits': text, 'would_change': text}}
        def decision(status, minimum, maximum):
            return {'type':'object', 'additionalProperties':False,
                'required':['status','hypotheses','missing_information'], 'properties':{
                    'status':{'const':status}, 'missing_information':text,
                    'hypotheses':{'type':'array', 'minItems':minimum, 'maxItems':maximum, 'items':hypothesis}}}
        properties = {}
        for section in ('situation','evolution','uncertainty'):
            required = set(self.initial()['required_reading_facts'][section])
            optional = []
            for identity in sorted(self._consulted - required):
                try:
                    self._fact(identity, section)
                except ValueError:
                    continue
                optional.append(identity)
            properties[section] = ({'type':'array', 'maxItems':8, 'uniqueItems':True,
                'items':{'enum':optional}} if optional else {'enum':[[]]})
        reading = {'type':'object', 'additionalProperties':False, 'required':list(comparison.SECTIONS),
                   'properties':properties}
        reading['properties']['interpretation'] = {'anyOf':[
            decision('hypotheses', 1, 6), decision('insufficient', 0, 0)]}
        def envelope(action, indices, result):
            return {'type':'object', 'additionalProperties':False, 'required':['action','indices','reading'],
                    'properties':{'action':{'enum':[action]}, 'indices':indices, 'reading':result}}
        choices = [envelope('finish', {'enum':[[]]}, reading)]
        remaining = comparison.QUERY_LIMIT-max(len(self._queries),len(self._query_attempts))
        if self._remaining_model_calls > 1 and remaining > 0:
            for action, scope in (('consult','full'),('consult_history','history')):
                indices = self.available_indices(scope)
                if indices:
                    choices.append(envelope(action, {'type':'array','minItems':1,'uniqueItems':True,
                        'maxItems':remaining,'items':{'enum':indices}}, {'type':'null'}))
        return {'anyOf':choices}

    def _contextual_identifiers(self):
        return {'C4'} | {f['subject'] for f in self._facts.values()
                        if f['key'] == 'own_player' and f['id'] in self._consulted}

    def _checked_text(self, value):
        require(value.strip() and all(c.isprintable() or c in '\n\t' for c in value), 'invalid contextual text')
        normalized = unicodedata.normalize('NFKC', value)
        for identifier in self._contextual_identifiers():
            normalized = re.sub(r'\b'+re.escape(identifier)+r'\b', '', normalized)
        normalized = ''.join(c for c in unicodedata.normalize('NFKD', normalized) if not unicodedata.combining(c))
        require(not any(c.isnumeric() for c in normalized) and not _QUANTITIES.search(normalized),
                'unprotected quantity in contextual text')

    def render(self, selection):
        jsonschema.validate({'action':'finish','indices':[],'reading':selection}, self.envelope_schema())
        decision = selection['interpretation']
        self._checked_text(decision['missing_information'])
        plain = {**selection, 'interpretation': []}
        reading, verification = super().render(plain)
        shared = {k:deepcopy(decision[k]) for k in ('status','missing_information')}
        for row in decision['hypotheses']:
            for key in ('hypothesis','limits','would_change'):
                self._checked_text(row[key])
            ids = list(dict.fromkeys(row['evidence'] + row['counterevidence']))
            facts = [deepcopy(self._fact(identity, 'interpretation')) for identity in ids]
            reading['interpretation'].append({
                'kind':'inferred', 'semantic_status':'unverified_model_inference',
                'text':f"Borrador contextual pendiente de revisión ({row['plausibility']}): {row['hypothesis']} "
                       f"Límites: {row['limits']} Cambiaría la lectura: {row['would_change']} "
                       f"Información pendiente: {decision['missing_information']}",
                'evidence':ids, 'principles':[], 'facts':facts,
                'contextual_hypothesis':deepcopy(row), 'contextual_decision':deepcopy(shared)})
        if decision['status'] == 'insufficient':
            reading['interpretation'] = [{'kind':'limit', 'semantic_status':'unverified_model_inference',
                'text':'Insuficiencia contextual declarada por el modelo, pendiente de revisión: '+decision['missing_information'],
                'evidence':[], 'principles':[], 'facts':[], 'contextual_decision':shared}]
        verification.update(representation=VERSION, claims=sum(map(len, reading.values())),
            factual_integrity_scope='situation_evolution_uncertainty_only',
            contextual_factual_integrity='not_mechanically_established',
            contextual_status=decision['status'], contextual_review='required',
            contextual_query_runtime=self.runtime_state(),
            contextual_quantity_guard='digits_and_common_number_words_not_semantic_validation')
        return reading, verification

    def verify(self, reading):
        require(isinstance(reading, dict) and set(reading) == set(comparison.SECTIONS), 'reading sections differ')
        rows = reading['interpretation']
        require(isinstance(rows, list) and rows and all(isinstance(r, dict) for r in rows), 'missing contextual decision')
        shared = rows[0]['contextual_decision']
        require(all(row['contextual_decision'] == shared for row in rows), 'inconsistent contextual decision')
        required = self.initial()['required_reading_facts']
        selected = {s:[c['evidence'][0] for c in reading[s] if c['evidence'][0] not in required[s]]
                    for s in comparison.SECTIONS if s != 'interpretation'}
        selected['interpretation'] = {**shared, 'hypotheses': [r['contextual_hypothesis'] for r in rows]
                                      if shared['status'] == 'hypotheses' else []}
        expected, verification = self.render(selected)
        require(reading == expected, 'rendered contextual facts or text differ')
        return verification
