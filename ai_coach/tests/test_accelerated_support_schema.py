"""Differential checks against the existing JSON Schema implementation."""

from copy import deepcopy
from decimal import Decimal
import json

from jsonschema import Draft202012Validator, ValidationError
import pytest

from ai_coach.contracts.accelerated_support_schema import (
    SCHEMAS,
    compiled_validator,
)
from ai_coach.support import anchors as a, c4, objective, physical
from ai_coach.interpretation import temporal as t
from ai_coach.tests.test_temporal_description import inputs, records


def examples():
    source, sources = inputs(1)
    return {
        a.DATA: source,
        physical.DATA: sources["physical_source"],
        objective.DATA: sources["objective_source"],
        c4.DATA: sources["c4_source"],
        t.SCHEMA: records(source, sources, 1)[0],
    }


def locations(value, path=()):
    yield path, value
    if isinstance(value, dict):
        for key, item in value.items():
            yield from locations(item, (*path, key))
    elif isinstance(value, list):
        for index, item in enumerate(value):
            yield from locations(item, (*path, index))


def replaced(original, path, value):
    if not path:
        return value
    result = deepcopy(original)
    parent = result
    for part in path[:-1]:
        parent = parent[part]
    parent[path[-1]] = value
    return result


@pytest.mark.parametrize("name", sorted(SCHEMAS))
def test_contract_acceptance_matches_reference_at_every_fixture_field(name):
    original = examples()[name]
    fast = a.schema_validator(name)
    reference = Draft202012Validator(fast.schema)
    assert fast.is_valid(original) and reference.is_valid(original)
    before = deepcopy(original)
    for path, value in locations(original):
        variants = [None, True, False, -1, 0, 1.5, "", "invalid", [], {}]
        if isinstance(value, dict):
            variants.append({**value, "unexpected": True})
            variants.extend(
                {k: v for k, v in value.items() if k != key} for key in value
            )
        elif isinstance(value, list):
            variants.extend([value + value, tuple(value)])
        elif isinstance(value, str):
            variants.extend([value + "\n", value.upper(), value + "invalid"])
        elif type(value) in (int, float):
            variants.extend(
                [float(value), Decimal(str(value)), 2**100, float("nan"), float("inf")]
            )
        for variant in variants:
            candidate = replaced(original, path, variant)
            expected = reference.is_valid(candidate)
            assert fast.is_valid(candidate) == expected, (name, path, repr(variant))
    assert original == before


def test_invalid_errors_keep_reference_type_and_path():
    fast = a.schema_validator(t.SCHEMA)
    row = examples()[t.SCHEMA]
    row["actors"][0]["actor_alias"] = "invalid"
    errors = list(fast.iter_errors(row))
    assert errors and all(isinstance(e, ValidationError) for e in errors)
    assert list(errors[0].absolute_path) == ["actors", 0, "actor_alias"]
    with pytest.raises(ValidationError):
        fast.validate(row)


def test_compilation_cache_tracks_schema_content_and_never_caches_instances():
    first = compiled_validator(
        json.dumps({"type": "object", "required": ["a"]}).encode()
    )
    second = compiled_validator(
        json.dumps({"type": "object", "required": ["b"]}).encode()
    )
    instance = {"a": 0}
    assert first.is_valid(instance) and not second.is_valid(instance)
    instance.clear()
    assert not first.is_valid(instance)
    first.schema["required"].clear()
    assert not first.is_valid(instance)


@pytest.mark.parametrize(
    "value", [(1,), {1: 2}, Decimal("1"), float("nan"), float("inf")]
)
def test_non_json_python_values_preserve_reference_semantics(value):
    for kind in ("number", "integer", "array", "object", "null"):
        schema = {"type": kind}
        fast = compiled_validator(json.dumps(schema).encode())
        assert fast.is_valid(value) == Draft202012Validator(schema).is_valid(value)


@pytest.mark.parametrize("value", ["\ud800", "\udfff"])
def test_escaped_surrogates_use_reference_semantics(value):
    for schema in ({"type": "string"}, {"type": "string", "pattern": "^[a-z]+$"}):
        fast = compiled_validator(json.dumps(schema).encode())
        assert fast.is_valid(value) == Draft202012Validator(schema).is_valid(value)
        if fast.is_valid(value):
            fast.validate(value)
        else:
            with pytest.raises(ValidationError):
                fast.validate(value)
