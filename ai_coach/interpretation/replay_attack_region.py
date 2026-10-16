"""Expert-reviewed local region; deterministic membership, optional hypothesis.

The contour is a reviewed radar approximation. Computation does not upgrade
its physical precision or the unverified map build to observed game truth.
Historical attack sessions and their spatial model estimates remain unchanged.
"""
from copy import deepcopy
import hashlib
from itertools import combinations
import json
import math
from pathlib import Path

from ai_coach.interpretation.place_volumes import EPSILON, load_reference, resolve
from ai_coach.interpretation.replay_attack_hypothesis import FocusedAttackHypothesisSession, FocusedAttackConsultation
from ai_coach.interpretation.replay_factual_reading import number

VERSION = 'stratai.replay_attack_hypothesis@4'
ASSET = Path(__file__).with_name('maps') / 'de_mirage_attack_region_v1.json'
_REGION_BYTES = ASSET.read_bytes()
_REGION = json.loads(_REGION_BYTES)
REGION_SHA256 = hashlib.sha256(_REGION_BYTES).hexdigest()
PROMPT = """Select references for a Spanish contextual reading; never write prose.
The application includes ALL required_reading_facts automatically. Situation,
evolution and uncertainty select at most eight optional additional references
per section; empty arrays omit no protected facts. Retain the factual date and
section rules. Use only delivered IDs. Earlier samples need a causal query.
Interpretation is empty, up to eight conditional principles, OR one optional
possible_attack_b_advanced_apps hypothesis using exactly one delivered
valid_reference_combinations entry in its supplied order. Never repeat or mix.
The expert approved the full final B Apps room from the partition passage by
point 2 through the elbow by point 3 to the B exit, with a group of 2-3 own T.
The application computes membership against that versioned approximate contour
and the original 3D Apartments volume. Do not estimate proximity from distances,
invent a radius, count outside/boundary/unknown players, or require every T to
belong to the group. A third outside player cannot veto an inside pair.
Each valid combination establishes only current group membership in that local
reference. It can support a POSSIBLE attack B; assess that hypothesis without
requiring C4 in the group, enemy contact or confirmed planting intent. Staging,
withdrawal and another destination remain alternatives. Never claim confirmed
intent, planting, effective Mid support, a defensive reaction or a recommendation.
Retain approximate geometry and unverified map-build limitations. Membership
is calculated evidence, not an automatic tactical answer. Abstention is allowed.
The third message repeats current own facts and their derived memberships,
not hidden enemy sound map points or a selected tactical interpretation.
Finish requires indices:[]; consult requires nonempty indices and reading:null.
"""


def membership(position, place, *, map_name):
    """Intersect the reviewed XY trace with the pinned original 3D place hull.

    Source XY precision and the existing numerical epsilon protect boundaries;
    neither is a new tactical radius or a claim about radar trace accuracy.
    """
    result = {'region_id': _REGION['id'], 'reference_sha256': REGION_SHA256,
              'parent_reference_id': _REGION['parent_place_reference_id'],
              'relation': 'unknown', 'status': 'estimated',
              'geometry_status': _REGION['geometry_status'], 'build_compatibility': 'unverified'}
    if map_name != _REGION['map_name'] or place.get('reference_id') != _REGION['parent_place_reference_id']:
        return {**result, 'reason': 'reference_unavailable_or_incompatible'}
    precision = place.get('xy_precision')
    if (type(precision) not in (int, float) or not math.isfinite(precision) or precision < 0
            or not all(type(position.get(k)) in (int,float) and math.isfinite(position[k]) for k in ('x','y','z'))):
        return {**result, 'reason': 'position_or_precision_unavailable'}
    actual = resolve(position, load_reference(), map_name=map_name, xy_precision=precision)
    if any(place.get(k) != actual[k] for k in ('status','zone','candidates','reference_id','build_compatibility','reason')):
        return {**result, 'reason': 'place_disagrees_with_geometry'}
    if actual['status'] in ('boundary', 'ambiguous'):
        return {**result, 'relation': 'boundary' if actual['status']=='boundary' else 'unknown',
                'reason': 'parent_place_not_unique_interior'}
    if actual['status'] != 'mapped' or actual['zone'] != _REGION['parent_zone']:
        return {**result, 'relation': 'outside', 'reason': 'outside_parent_3d_place'}
    x, y = position['x'], position['y']
    polygon = _REGION['polygon_xy']
    tolerance = precision + EPSILON
    inside = False
    for (ax,ay), (bx,by) in zip(polygon, polygon[1:] + polygon[:1]):
        # This reviewed L has only axis-aligned edges; the expanded edge box
        # is exactly the source XY uncertainty square, including corner cases.
        if min(ax,bx)-tolerance <= x <= max(ax,bx)+tolerance and min(ay,by)-tolerance <= y <= max(ay,by)+tolerance:
            return {**result, 'relation': 'boundary', 'reason': 'reviewed_contour_boundary'}
        if (ay > y) != (by > y) and x < ax + (y-ay)*(bx-ax)/(by-ay):
            inside = not inside
    return {**result, 'relation': 'inside' if inside else 'outside', 'reason': 'reviewed_xy_and_parent_3d_intersection'}


class RegionAttackHypothesisSession(FocusedAttackHypothesisSession):
    def __init__(self, packet):
        super().__init__(packet)
        contract = self._view['hypothesis_contract']
        original = contract['spatial_review']['eligible_players']
        players = [{**row, 'membership': membership(row['current_player_fact']['value'],
                    row['current_place_fact']['value'], map_name=packet.get('map_name'))} for row in original]
        eligible = [{k:deepcopy(row[k]) for k in ('player_fact_id','place_fact_id','distance_xy_units','membership')}
                    for row in players if row['membership']['relation']=='inside']
        ids = sorted(row['player_fact_id'] for row in eligible)
        groups = [list(group) for size in (2,3) for group in combinations(ids,size)]
        contract.update(version=VERSION,
            expert_criterion='2-3 own T within the full reviewed final B Apps room can support possible attack B',
            spatial_classification='deterministic_membership_in_reviewed_approximate_region',
            geometry_limits='reviewed radar approximation intersected with original 3D Apartments; exact build unverified',
            region={**deepcopy(_REGION), 'reference_sha256':REGION_SHA256},
            eligible_players=eligible, valid_reference_combinations=groups,
            reference_combination_scope='distinct current inside identities; not a tactical label',
            spatial_review={'version':'stratai.attack_spatial_review@2',
                'origin':'current own facts plus deterministic membership in reviewed approximate region; not a tactical label',
                'region':{**deepcopy(_REGION), 'reference_sha256':REGION_SHA256},
                'geometry_limits':'source XY uncertainty protects edges; radar geometry and exact map build remain qualified',
                'players':players, 'valid_reference_combinations':deepcopy(groups)})

    def receipt(self):
        return {**super().receipt(), 'reading_contract':VERSION}

    def consultation(self):
        return RegionAttackConsultation(self)


class RegionAttackConsultation(FocusedAttackConsultation):
    hypothesis_adapter_version = '1.7.0-local.1'

    def _hypothesis(self, row):
        claim = super()._hypothesis(row)
        players = [self._fact(identity,'situation') for identity in row['evidence']]
        names = ', '.join(f['subject'] for f in players)
        claim['text'] = (
            f"A los {number(players[0]['seconds_in_window'])} s de la ventana. Posible ataque a B: "
            f'{len(players)} T ({names}) en el tramo avanzado acordado de B Apps, '
            'desde el paso junto al punto 2 hasta la salida B, incluido el codo junto a 3. '
            'Pertenencia calculada sobre una región local aproximada y el volumen 3D de Apartments. '
            'También pueden estar preparando la jugada, replegarse o cambiar de destino. '
            'No confirma intención de plantar ni apoyo efectivo desde Medio. '
            'La referencia local del mapa mantiene compatibilidad de build no verificada.')
        eligible = {r['player_fact_id']:r for r in self.initial()['hypothesis_contract']['eligible_players']}
        claim['region_support'] = [{'player_fact_id':identity, 'place_fact_id':eligible[identity]['place_fact_id'],
                                   **deepcopy(eligible[identity]['membership'])} for identity in row['evidence']]
        return claim

    def render(self, selection):
        reading, verification = super().render(selection)
        verification.update(representation=VERSION, spatial_hypothesis='reviewed_region_estimate_not_independently_evaluated')
        return reading, verification
