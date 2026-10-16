"""Bounded model-assisted grounding review; no tactical classifier or truth oracle."""
from copy import deepcopy

import jsonschema

from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.replay_contextual_reading import (
    ContextualConsultation, ContextualReadingSession, PROMPT as DRAFT_PROMPT,
)
from ai_coach.interpretation.replay_reading_context import digest
from ai_coach.support.anchors import require

VERSION = 'stratai.replay_grounded_reading@1'
PROMPT = DRAFT_PROMPT + """
Your finish is a DRAFT, not the final accepted reading. A separate grounding task
will check every factual premise, counterevidence and update condition against
the actual records. State hypotheses conditionally; do not invent factual premises.
A dead-player snapshot gives a state at its date, NOT a death time, cause, killer,
or starting location. Use the dated death event for its actual recorded fields.
The event may be simulated team communication and may not include cause or killer.
Historical position/place facts describe THEIR dates, not the current cut.
Carrying C4 does not contradict waiting. Zone labels do not establish safety,
plantable ground, destination or a route. A future sound/utility/voice signal alone
would not confirm planting, a defensive reaction, safety or coordinated intent.
Do not call information absent when it is delivered or merely not consulted.
"""
AUDIT_PROMPT = """Audit the Spanish CS2 draft against the supplied evidence.
This is factual grounding, not a tactical ranking or proof of hidden intent.
Return JSON using the schema. Check EACH target once: decision covers the shared
missing_information (including claims that data is absent); H targets cover ALL
of hypothesis, evidence, counterevidence, limits AND would_change in that row.
supported: every factual premise matches the record's subject, type, date and
epistemic scope, while inferences remain explicitly conditional. A hypothesis
need not be proved, but its alleged facts must be grounded. unsupported: any
contradiction, invented factual premise, invalid counterevidence or unjustified
claim of confirmation. unknown: you cannot establish grounding. Cite actual
delivered IDs and explain the concrete issue. Do not approve merely because an
ID exists, prose hedges somewhere, or evidence is consistent with a possibility.
Compare with CURRENT context AND dated events, not only the draft's chosen IDs.
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
"""
CORE = {'own_player', 'place_context', 'objective', 'round_context'}


class GroundedReadingSession(ContextualReadingSession):
    def __init__(self, packet):
        super().__init__(packet)
        self._view['contextual_reading_contract']['grounding'] = VERSION

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

    def current_overview(self):
        view = super().current_overview()
        # A factual index over already delivered events, not a generated reading.
        view['event_index'] = [{k:deepcopy(f[k]) for k in ('id','key','subject','status','seconds_in_window')}
            for f in self.initial()['events'] if f['key'] == 'death' or f['key'].startswith('bomb_')]
        view['scope'] = 'Snapshot dates are not death dates. Event fields do not supply missing cause. Historical zones are dated; labels do not accredit safety or plantability.'
        return view

    def envelope_schema(self):
        if self.grounding_stage != 'audit':
            # Both drafting and checking consume the original max_calls budget.
            remaining = self._remaining_model_calls
            try:
                self._remaining_model_calls = max(1,remaining-1)
                return super().envelope_schema()
            finally:
                self._remaining_model_calls = remaining
        targets = self._targets()
        check = {'type':'object','additionalProperties':False,
            'required':['target','verdict','evidence','reason'], 'properties':{
                'target':{'enum':targets},
                'verdict':{'enum':['supported','unsupported','unknown']},
                'evidence':{'type':'array','minItems':1,'maxItems':32,'uniqueItems':True,
                            'items':{'enum':sorted(f['id'] for key in ('current','events','cited_history')
                                                   for f in self.audit_packet()[key])}},
                'reason':{'type':'string','minLength':1,'maxLength':1000}}}
        return {'type':'object','additionalProperties':False,
            'required':['action','indices','reading'], 'properties':{
                'action':{'enum':['audit']},'indices':{'enum':[[]]},
                'reading':{'type':'object','additionalProperties':False,'required':['checks'],
                    'properties':{'checks':{'type':'array','minItems':len(targets),
                                           'maxItems':len(targets),'items':check}}}}}

    def _targets(self):
        return ['decision'] + [f'H{i+1}' for i in range(len(self._draft['interpretation']['hypotheses']))]

    def submit(self, selection):
        require(self.grounding_stage == 'draft', 'draft already submitted')
        require(self._remaining_model_calls >= 2, 'no remaining grounding audit call')
        # Validate against the draft schema and original quantity/factual guards.
        # The resulting draft is not returned or counted as an accepted reading.
        ContextualConsultation.render(self, selection)
        self._draft = deepcopy(selection)
        self.grounding_stage = 'audit'

    def audit_packet(self):
        initial = self.initial()
        ids = {i for h in self._draft['interpretation']['hypotheses']
               for i in h['evidence'] + h['counterevidence']}
        current = {f['id'] for f in initial['situation'] + initial['events']}
        return {'version':VERSION, 'draft':deepcopy(self._draft),
            'current':deepcopy(initial['situation']), 'events':deepcopy(initial['events']),
            'cited_history':[deepcopy(self._fact(i,'interpretation')) for i in sorted(ids-current)],
            'current_factual_overview':self.current_overview(),
            'history_status':'unconsulted_samples_are_not_absent_data',
            'sample_times_seconds':deepcopy(initial['sample_times_seconds']),
            'spatial_measurements':deepcopy(initial['spatial_measurements']),
            'targets':self._targets()}

    def audit_messages(self):
        return [{'role':'system','content':AUDIT_PROMPT+'\n'+wire.TEMPLATE_INSTRUCTIONS},
                {'role':'user','content':wire_json(self.audit_packet())}]

    def complete_audit(self, answer):
        require(self.grounding_stage == 'audit', 'unexpected grounding audit')
        jsonschema.validate(answer,self.envelope_schema())
        checks = answer['reading']['checks']
        require(sorted(c['target'] for c in checks) == sorted(self._targets()), 'grounding audit coverage differs')
        self._audit = deepcopy(answer['reading'])
        require(all(c['verdict'] == 'supported' for c in checks), 'grounding audit rejected factual premises')
        self.grounding_stage = 'checked'
        return self.render(self._draft)

    def render(self, selection):
        require(self.grounding_stage == 'checked' and selection == self._draft, 'grounding audit required')
        reading, verification = super().render(selection)
        verification.update(grounding_contract=VERSION,
            grounding_review='same_model_assisted_not_independent',
            grounding_audit=deepcopy(self._audit), grounding_packet_hash=digest(self.audit_packet()))
        return reading, verification

    def runtime_state(self):
        return {**super().runtime_state(),'grounding_stage':self.grounding_stage,
                'draft':deepcopy(self._draft),'audit':deepcopy(self._audit)}


def wire_json(value):
    from ai_coach.interpretation.context_comparison_ollama import compact
    return compact(wire.pack(value,version=wire.TEMPLATE_VERSION))
