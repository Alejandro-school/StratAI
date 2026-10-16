"""Bounded model-assisted grounding review; no tactical classifier or truth oracle."""
from copy import deepcopy
import re

import jsonschema

from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.replay_contextual_reading import (
    ContextualConsultation, ContextualReadingSession, PROMPT as DRAFT_PROMPT,
)
from ai_coach.interpretation.replay_reading_context import digest
from ai_coach.interpretation.replay_factual_reading import alias
from ai_coach.interpretation.replay_factual_validation import NATIVE_EQUIPMENT
from ai_coach.support.anchors import require

VERSION = 'stratai.replay_grounded_reading@1'
PROMPT = DRAFT_PROMPT.replace('or a planting plan.', 'or a confirmed planting plan.') + """
Your finish is a DRAFT, not the final accepted reading. A separate grounding task
will check every factual premise, counterevidence and update condition against
the actual records. State hypotheses conditionally; do not invent factual premises.
A dead-player snapshot gives a state at its date, NOT a death time, cause, killer,
or starting location. Use the dated death event for its actual recorded fields.
The event may be simulated team communication and may not include cause or killer.
Public killfeed events can supply victim, killer and weapon without visual/audio
contact. Check those records before declaring those fields unknown.
Historical position/place facts describe THEIR dates, not the current cut.
Carrying C4 does not contradict waiting. Zone labels do not establish safety,
plantable ground, destination or a route. A future sound/utility/voice signal alone
would not confirm planting, a defensive reaction, safety or coordinated intent.
Do not call information absent when it is delivered or merely not consulted.
Unknown data belongs in limits, not counterevidence; use an empty list if none.
Unrecorded decisions or receipts are not established, not proved absent.
Possible planting intentions may be inferred conditionally, never confirmed by C4.
"""
AUDIT_PROMPT = """Audit the Spanish CS2 draft against the supplied evidence.
This is factual grounding, not a tactical ranking or proof of hidden intent.
Return JSON using the schema. Review ONLY the single target_card. decision covers
missing_information (including claims that data is absent); H targets cover ALL
of hypothesis, evidence, counterevidence, limits AND would_change in that card.
Review the actual TEXT FIELDS in the schema, not a checklist of implicit tactical
attributes. Do not invent an additional safety, certainty, reaction or route claim.
Current distribution is supported by current positions; a claim of actual motion,
continuous waiting or a route needs dated evidence. Do not require a separate
positioning-action field for a description of where players are situated.
supported: every factual premise matches the record's subject, type, date and
epistemic scope, while inferences remain explicitly conditional. A hypothesis
need not be proved, but its alleged facts must be grounded. unsupported: any
contradiction, invented factual premise, invalid counterevidence or unjustified
claim of confirmation. unknown: you cannot establish grounding. Cite actual
delivered IDs and explain the concrete issue. Do not approve merely because an
ID exists or a factual assertion is merely hedged/consistent with a possibility.
Compare with CURRENT context AND dated events, not only the draft's chosen IDs.
Public killfeed supplies its recorded actor, victim and weapon without requiring
visual/audio contact. Do not erase that evidence or demand extra perception.
Dead state is not death timing/cause/location of combat. Death-event fields can
establish only what they contain, preserving simulated communication status.
Past positions are not current motion. Discrete endpoints do not prove a route,
continuous movement or waiting. C4 ownership does not contradict waiting. A zone
name does not establish plantable ground, safety, intent or destination. Future
noise, utility or a pause cannot by itself confirm a plant, reaction or intent.
No contacts registered is not no enemies; unconsulted is not unavailable. Assess
each facet separately; a missing enemy intention does not erase known own state.
Do not add hidden enemy information or recommend actions. You use the same model;
this is an assisted check, never independent semantic or tactical validation.
Separate OBSERVED FACTS from CONDITIONAL INTERPRETATION. Never demand a parser
field for thoughts, evaluation, coordination or intent in order to permit a
conditional hypothesis about them. conditional_inference means a possible mental
state/intent or explanation compatible with the known facts, with relevant
context cited; it is NOT a fact, proof or tactical endorsement. For example, a
known C4 carrier COULD be considering planting or waiting for coordination even
without recorded voice. This remains an inference, not a missing cognitive field.
Compatibility plus relevant context is enough for this conditional classification;
ranking its likelihood or tactical usefulness is outside this grounding task.
Always reject a premise CONTRADICTING known facts, including a different current
C4 carrier, even when wrapped in 'possible'. Never invent a factual premise to
make an inference fit. The same positions may support several hidden intentions.
Unknown/missing data is a limitation, NOT counterevidence. Counterevidence IDs must
actually oppose the explanation; use not_asserted when that list is empty.
A future update condition need not have happened: assess its claimed implication.
More observations can reinforce/weaken a reading without proving an intention.
Reject claims that a sound, utility or a pause by itself confirms a plant/reaction.
For each text field select its EXACT quote offered by the schema, then check the
facts asserted in THAT field against dated records. not_asserted is offered only
for an empty counterevidence list; it has empty quote/evidence. Do not excuse an invented factual premise by
calling the overall hypothesis plausible. A conditional intention can be open
while 'safe area', 'planting zone', 'continuous movement', 'from this location',
'confirmed by utility', or 'no evidence available' remains unsupported. Inspect
ALL fields and the counterevidence IDs, not only the hypothesis sentence. If any
factual premise in a field fails, mark that field unsupported and explain it.
"""
AUDIT_PROMPT += '\nGive each field quote, relevant evidence and a concise concrete match or mismatch BEFORE choosing its verdict. Do not repeat the draft in the reason. If the described dated facts match, use supported; compatible conditional intent uses conditional_inference. A matching fact must not receive unsupported. A contradicted carrier must receive unsupported.'
CORE = {'own_player', 'own_life', 'place_context', 'objective', 'round_context'}
DIMENSIONS = ('hypothesis','limits','would_change','counterevidence')
INFERENCE_DIMENSIONS = {'hypothesis','limits','would_change'}


def check_grounding(draft, checks, targets=None):
    """Verify exact target/quote coverage; the verdicts remain model judgments."""
    if targets is None:
        targets = ['decision'] + [f'H{i+1}' for i in range(len(draft['interpretation']['hypotheses']))]
    require(sorted(c['target'] for c in checks) == sorted(targets), 'grounding audit coverage differs')
    for check in checks:
        target = check['target']
        if target == 'decision':
            text = draft['interpretation']['missing_information']
            expected_fields={'missing_information'}
        else:
            row = draft['interpretation']['hypotheses'][int(target[1:])-1]
            text = '\n'.join(row[key] for key in ('hypothesis','limits','would_change'))
            text += '\n' + ' '.join(row['evidence'] + row['counterevidence'])
            expected_fields=set(DIMENSIONS)
        require(set(check['dimensions'])==expected_fields,'grounding field coverage differs')
        for assessment in check['dimensions'].values():
            if assessment['verdict'] == 'not_asserted':
                require(not assessment['quote'] and not assessment['evidence'], 'grounding empty claim differs')
            else:
                require(assessment['quote'].strip() and assessment['quote'] in text,
                        'grounding quote does not match target')
                require(assessment['evidence'], 'grounding assertion has no references')
        for dimension,assessment in check['dimensions'].items():
            if assessment['verdict']=='conditional_inference':
                require(target!='decision' and dimension in INFERENCE_DIMENSIONS
                        and assessment['quote']==row[dimension], 'conditional inference scope differs')
    return all(a['verdict'] in ('supported','not_asserted','conditional_inference')
               for c in checks for a in c['dimensions'].values())


def grounding_schema(packet):
    reference = {'enum':sorted(f['id'] for key in ('current','events','cited_history') for f in packet[key])}
    card = packet['target_card']
    def assessment(verdicts, allowed_quotes, evidence):
        return {'type':'object','additionalProperties':False,
            'required':['quote','evidence','reason','verdict'], 'properties':{
                'quote':{'enum':allowed_quotes}, 'evidence':evidence,
                'reason':{'type':'string','minLength':1,'maxLength':350},
                'verdict':{'enum':verdicts}}}
    fields=DIMENSIONS if 'hypothesis' in card else ('missing_information',)
    properties={}
    for field in fields:
        if field=='counterevidence' and not card[field]:
            properties[field]=assessment(['not_asserted'],[''],{'enum':[[]]})
            continue
        quote=' '.join(card[field]) if field=='counterevidence' else card[field]
        verdicts=['supported','unsupported','unknown']
        if field in INFERENCE_DIMENSIONS:verdicts.append('conditional_inference')
        properties[field]=assessment(verdicts,[quote],
            {'type':'array','minItems':1,'maxItems':16,'uniqueItems':True,'items':reference})
    check = {'type':'object','additionalProperties':False,'required':['target','dimensions'],
        'properties':{'target':{'enum':packet['targets']},
            'dimensions':{'type':'object','additionalProperties':False,'required':list(fields),'properties':properties}}}
    return {'type':'object','additionalProperties':False,
        'required':['action','indices','reading'], 'properties':{
            'action':{'enum':['audit']},'indices':{'enum':[[]]},
            'reading':{'type':'object','additionalProperties':False,'required':['checks'],
                'properties':{'checks':{'type':'array','minItems':len(packet['targets']),
                                       'maxItems':len(packet['targets']),'items':check}}}}}


class GroundedReadingSession(ContextualReadingSession):
    def __init__(self, packet):
        super().__init__(packet)
        self._view['contextual_reading_contract']['grounding'] = VERSION
        # Rule identifiers must not collide with participant aliases P01...P10.
        self._view['bank'] = deepcopy(self._view['bank'])
        for rule in self._view['bank']['principles']:
            rule['id'] = 'RULE_' + rule['id']

    def consultation(self):
        return GroundedConsultation(self)

    def receipt(self):
        return {**super().receipt(), 'grounding_contract':VERSION}


class GroundedConsultation(ContextualConsultation):
    grounded_selection = True

    def __init__(self, session):
        super().__init__(session)
        self.grounding_stage = 'draft'
        self._draft = None
        self._audit = None
        self._audit_hashes = []
        self._draft_schema = None

    def current_overview(self):
        initial = self.initial()
        current = initial['situation']
        objective = next((f for f in current if f['key']=='objective'),None)
        round_fact = next((f for f in current if f['key']=='round_context'),None)
        def dated(fact, value):
            return {'fact_id':fact['id'],'seconds':fact['seconds_in_window'],
                    'status':fact['status'],'value':deepcopy(value)}
        places = {f['subject']:f for f in current if f['key']=='place_context'}
        lives = {f['subject']:f for f in current if f['key']=='own_life'}
        players=[];flags={}
        for fact in current:
            if fact['key']!='own_player':continue
            who=fact['subject'];state=deepcopy(fact['value'])
            if 'has_c4' in state:
                flags[who]={'fact_id':fact['id'],'has_c4':state.pop('has_c4')}
            row={'participant':who,'snapshot':dated(fact,state)}
            if who in lives:
                life=lives[who]
                row['life']=dated(life,{k:life['value'][k] for k in ('alive','health') if k in life['value']})
            if who in places:
                place=places[who]
                row['place']=dated(place,{k:place['value'][k] for k in
                    ('status','zone','build_compatibility','reason') if k in place['value']})
            players.append(row)
        bomb = (objective or {}).get('value') or {}
        c4 = {'state':bomb.get('state'),'carrier':alias(bomb['carrier_id']) if bomb.get('carrier_id') else None,
              'player_flags':flags,**{k:bomb[k] for k in ('objective_phase','is_planted_now','was_planted_this_round','site','position_status') if k in bomb}}
        if bomb.get('position_status')=='observed':
            c4['position']={k:bomb[k] for k in ('x','y','z') if k in bomb}
        if objective:c4=dated(objective,c4)
        round_state = (round_fact or {}).get('value') or {}
        round_summary={k:round_state[k] for k in ('round','phase','winner','decision_status','clock_status',
            'pause_state','round_remaining_seconds','bomb_remaining_seconds','eligibility') if k in round_state}
        return {'version':'stratai.contextual_factual_overview@2',
            'purpose':'Estado del corte actual. Contexto íntegro y consultas históricas conservados en la conversación.',
            'cut_seconds':initial['sample_times_seconds'][-1],
            'round':dated(round_fact,round_summary) if round_fact else None,
            'c4':c4,'players':players,
            'event_index':[{k:deepcopy(f[k]) for k in ('id','key','subject','status','seconds_in_window')}
                for f in initial['events'] if f['key']=='death' or f['key'].startswith('bomb_')],
            'public_killfeed':[{'id':f['id'],'text':self._claim(f)['text']}
                               for f in initial['events'] if f['key']=='kill'],
            'spatial_excerpt':[{'player_fact_id':m['player_fact_id'],'place_fact_id':m['place_fact_id'],
                               'relation':m['membership']['relation'],'status':m['membership']['status']}
                              for m in initial['spatial_measurements']],
            'limits':'Datos fechados y zonas aproximadas; ninguna pertenencia, estado de C4 o señal acredita intención, seguridad o ruta. Puntos rivales no suministrados siguen desconocidos. RULE_ identifica reglas, P identifica participantes.',
            'output':'Español, hipótesis condicionales. Las cifras se muestran mediante hechos protegidos; no las repitas en prosa.'}

    def _contextual_identifiers(self):
        identifiers = super()._contextual_identifiers()
        for fact in self._facts.values():
            if fact['id'] not in self._consulted:
                continue
            if fact['key']=='kill' and fact['status']=='observed' and fact['value'].get('knowledge')=='public_killfeed':
                identifiers.update(alias(fact['value'][key]) for key in ('killer_id','victim_id') if fact['value'].get(key))
                if fact['value'].get('weapon') in NATIVE_EQUIPMENT:
                    identifiers.add(fact['value']['weapon'])
            if fact['key']=='own_inventory':
                identifiers.update(name for name in fact['value'].get('weapons',[]) if name in NATIVE_EQUIPMENT)
        return identifiers

    def refresh_current_boundary(self, messages):
        from ai_coach.interpretation.context_comparison_ollama import compact
        boundary={'role':'user','content':compact(self.current_overview())}
        # Move the existing duplicate, never drop the full context or add copies.
        messages[:]=[m for m in messages if m != boundary]+[boundary]

    def envelope_schema(self):
        if self.grounding_stage == 'checked':
            return deepcopy(self._draft_schema)
        if self.grounding_stage != 'audit':
            # Both drafting and checking consume the original max_calls budget.
            remaining = self._remaining_model_calls
            try:
                self._remaining_model_calls = max(1,remaining-1)
                schema = super().envelope_schema()
                decisions = schema['anyOf'][0]['properties']['reading']['properties']['interpretation']['anyOf']
                maximum = min(6,remaining-2)
                if maximum > 0:
                    decisions[0]['properties']['hypotheses']['maxItems'] = maximum
                else:
                    del decisions[0]
                # Prevent ASCII quantities/reference indices during sampling;
                # the existing post-check still rejects words/Unicode numbers.
                names=[re.escape(i).replace(r'\-','-').replace(r'\ ',' ')
                       for i in sorted(self._contextual_identifiers()) if any(c in '0123456789' for c in i)]
                # Exclude JSON quotes/backslashes and bound native repetition.
                # Some providers do not compose maxLength with pattern; an
                # unbounded negated class can consume the closing JSON quote.
                pattern="^([A-Za-zÀ-ž .,;:!?()'/_%+=<>—–-]|"+'|'.join(names)+'){1,600}$'
                def protect(node):
                    if isinstance(node,dict):
                        if node.get('type')=='string' and node.get('maxLength')==600:
                            node['pattern']=pattern
                        for child in node.values():protect(child)
                    elif isinstance(node,list):
                        for child in node:protect(child)
                protect(schema)
                return schema
            finally:
                self._remaining_model_calls = remaining
        return grounding_schema(self.audit_packet())

    def _targets(self):
        return [f'H{i+1}' for i in range(len(self._draft['interpretation']['hypotheses']))] + ['decision']

    def submit(self, selection):
        require(self.grounding_stage == 'draft', 'draft already submitted')
        require(self._remaining_model_calls >= len(selection['interpretation']['hypotheses'])+2,
                'no remaining grounding audit calls')
        # Validate against the draft schema and original quantity/factual guards.
        # The resulting draft is not returned or counted as an accepted reading.
        ContextualConsultation.render(self, selection)
        self._draft_schema = deepcopy(self.envelope_schema())
        self._draft = deepcopy(selection)
        self._audit = {'checks':[]}
        self.grounding_stage = 'audit'

    def audit_packet(self):
        initial = self.initial()
        ids = {i for h in self._draft['interpretation']['hypotheses']
               for i in h['evidence'] + h['counterevidence']}
        current = {f['id'] for f in initial['situation'] + initial['events']}
        target = self._targets()[len(self._audit['checks'])]
        card = ({'missing_information':self._draft['interpretation']['missing_information']} if target=='decision'
                else deepcopy(self._draft['interpretation']['hypotheses'][int(target[1:])-1]))
        return {'version':VERSION, 'target_card':card,
            'current':deepcopy(initial['situation']), 'events':deepcopy(initial['events']),
            'cited_history':[deepcopy(self._fact(i,'interpretation')) for i in sorted(self._consulted-current)
                             if i in ids or self._facts[i]['key'] in CORE],
            'current_factual_overview':self.current_overview(),
            'history_status':'unconsulted_samples_are_not_absent_data',
            'sample_times_seconds':deepcopy(initial['sample_times_seconds']),
            'spatial_measurements':deepcopy(initial['spatial_measurements']),
            'targets':[target]}

    def audit_messages(self):
        return [{'role':'system','content':AUDIT_PROMPT+'\n'+wire.TEMPLATE_INSTRUCTIONS},
                {'role':'user','content':wire_json(self.audit_packet())}]

    def complete_audit(self, answer):
        require(self.grounding_stage == 'audit', 'unexpected grounding audit')
        jsonschema.validate(answer,self.envelope_schema())
        checks = answer['reading']['checks']
        target = self._targets()[len(self._audit['checks'])]
        packet_hash = digest(self.audit_packet())
        # Retain even failed judgments and exact input hashes before blocking.
        self._audit['checks'].extend(deepcopy(checks))
        self._audit_hashes.append(packet_hash)
        require(check_grounding(self._draft,checks,[target]), 'grounding audit rejected factual premises')
        if len(self._audit['checks']) < len(self._targets()):
            return None
        self.grounding_stage = 'checked'
        return self.render(self._draft)

    def render(self, selection):
        require(self.grounding_stage == 'checked' and selection == self._draft, 'grounding audit required')
        reading, verification = super().render(selection)
        verification.update(grounding_contract=VERSION,
            grounding_policy='evidence_before_verdict_v3',
            grounding_review='same_model_assisted_not_independent',
            grounding_audit=deepcopy(self._audit), grounding_packet_hashes=deepcopy(self._audit_hashes))
        return reading, verification

    def runtime_state(self):
        return {**super().runtime_state(),'grounding_stage':self.grounding_stage,
                'draft':deepcopy(self._draft),'audit':deepcopy(self._audit),
                'audit_packet_hashes':deepcopy(self._audit_hashes)}


def wire_json(value):
    from ai_coach.interpretation.context_comparison_ollama import compact
    return compact(wire.pack(value,version=wire.TEMPLATE_VERSION))
