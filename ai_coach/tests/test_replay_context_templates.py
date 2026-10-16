"""A compact message cannot drop evidence, invent fields or change JSON types."""
from copy import deepcopy
import json

import pytest

from ai_coach.interpretation import replay_model_context as wire


def source():
    return {'observations': [
        {'id': f'sound-observation:{i}', 'kind': 'footstep', 'source_side': 'CT',
         'observation_tick': i, 'availability_tick': i,
         'region': {'status': 'compatible_region_estimate', 'label': 'BombsiteB'},
         'status': 'estimated_possible_reception', 'exact_position': 'unknown',
         'recipients': [{'player_id': str(p), 'reception': 'possible',
                         'availability_tick': i} for p in (1, 6, 7, 9, 10)],
         'source': {'kind': 'go_calculated_sound', 'reference': f'sound-observation:{i}',
                    'emission_reference': f'sound-emission:{i + 100}'}}
        for i in range(64)],
        'ids': [f'sound-observation:{i}' for i in range(64)],
        'numbers': [-0.0, 1, 1.0, True, False, None, -1765.505126953125],
        'uncertainty': 'current_position_unknown'}


def find(node, key):
    if isinstance(node, dict):
        if key in node:
            return node
        children = node.values()
    elif isinstance(node, list):
        children = node
    else:
        return None
    return next((found for child in children if (found := find(child, key)) is not None), None)


def test_templates_share_structure_but_preserve_every_dated_record_and_type():
    original = source()
    before = wire.encoded(original)
    packed = wire.pack(original, version=wire.TEMPLATE_VERSION)
    assert packed['wire_schema'] == wire.TEMPLATE_VERSION
    assert len(wire.encoded(packed)) < len(wire.encoded(wire.pack(original)))
    restored = wire.unpack(json.loads(wire.encoded(packed)))
    assert wire.encoded(restored) == before
    assert wire.encoded(original) == before
    restored['observations'][0]['recipients'].clear()
    assert restored['observations'][1]['recipients']
    assert original['observations'][0]['recipients']


def test_template_control_names_are_literal_when_the_source_uses_them():
    original = source()
    original.update({'$arg': True, '$template': 'literal', 'args': [1, None],
                     '$prefix_list': 'literal', 'suffixes': ['x'],
                     '$template_literal': {'$arg': 0, '$prefix': 'literal'},
                     'wire_schema': 'literal source field'})
    packed = wire.pack(original, version=wire.TEMPLATE_VERSION)
    assert packed['wire_schema'] == wire.TEMPLATE_VERSION
    assert wire.encoded(wire.unpack(packed)) == wire.encoded(original)


@pytest.mark.parametrize('original', [None, False, 0, -0.0, 'text', [], {},
                                    {'wire_schema': 'literal', 'unknown': None}])
def test_small_template_packets_keep_plain_values_or_historical_encoding(original):
    assert wire.encoded(wire.unpack(wire.pack(original, version=wire.TEMPLATE_VERSION))) == wire.encoded(original)


@pytest.mark.parametrize('mutation', ['unknown_template', 'missing_arg', 'extra_arg', 'object_arg',
                                    'bool_slot', 'bad_prefix', 'cycle', 'unused_template',
                                    'bad_suffix', 'body_fields', 'source_hash', 'outer_fields'])
def test_tampered_template_messages_are_rejected(mutation):
    packed = wire.pack(source(), version=wire.TEMPLATE_VERSION)
    body = wire.unpack(packed['body'])
    row = find(body['payload'], '$template')
    template = body['templates'][row['$template']]
    slot = find(template, '$arg')
    if mutation == 'unknown_template':
        row['$template'] = 'absent'
    elif mutation == 'missing_arg':
        row['args'].pop()
    elif mutation == 'extra_arg':
        row['args'].append('future')
    elif mutation == 'object_arg':
        row['args'][0] = {'future': True}
    elif mutation == 'bool_slot':
        slot['$arg'] = True
    elif mutation == 'bad_prefix':
        slot['$prefix'] = 123
    elif mutation == 'cycle':
        body['templates'][row['$template']] = deepcopy(row)
    elif mutation == 'unused_template':
        body['templates']['unused'] = {'future': 'unconsulted'}
    elif mutation == 'bad_suffix':
        find(body['payload'], '$prefix_list')['suffixes'][0] = False
    elif mutation == 'body_fields':
        body['unconsulted'] = 'future'
    elif mutation == 'source_hash':
        packed['source_sha256'] = '0' * 64
    elif mutation == 'outer_fields':
        packed['future'] = 'extra'
    packed['body'] = wire.pack(body)
    with pytest.raises(ValueError):
        wire.unpack(packed)


def test_template_encoding_rejects_nonfinite_numbers():
    with pytest.raises(ValueError):
        wire.pack({'tick': float('nan')}, version=wire.TEMPLATE_VERSION)
