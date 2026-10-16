"""Compiled validation for the five measured temporal/support JSON contracts.

The reference validator still checks schemas and supplies validation errors.
Only schema compilation is cached, by content; instances are always checked.
Non-JSON Python values take the reference path to avoid coercion by the binding.
"""

from copy import deepcopy
from functools import lru_cache
import json
import math

from jsonschema import Draft202012Validator
from jsonschema_rs import Draft202012Validator as CompiledValidator


SCHEMAS = frozenset(
    (
        "support_observations.json",
        "support_physical_observations.json",
        "support_objective_observations.json",
        "support_c4_observations.json",
        "temporal_description.json",
    )
)


def _plain_json(value):
    """Rust accepts tuples/Decimal and treats nonfinite numbers differently."""
    kind = type(value)
    if kind in (str, int, bool, type(None)):
        return True
    if kind is float:
        return math.isfinite(value)
    if kind is list:
        return all(_plain_json(item) for item in value)
    if kind is dict:
        return all(
            type(key) is str and _plain_json(item) for key, item in value.items()
        )
    return False


class SupportValidator:
    def __init__(self, schema):
        Draft202012Validator.check_schema(schema)
        self._reference = Draft202012Validator(schema)
        self._compiled = CompiledValidator(schema, validate_formats=False, offline=True)

    @property
    def schema(self):
        return deepcopy(self._reference.schema)

    def _accepted(self, instance):
        if not _plain_json(instance):
            return False
        try:
            return self._compiled.is_valid(instance)
        except UnicodeEncodeError:
            # Python's JSON decoder can retain escaped lone surrogates; the
            # native binding requires UTF-8. Keep reference semantics for them.
            return False

    def validate(self, instance):
        if not self._accepted(instance):
            self._reference.validate(instance)

    def iter_errors(self, instance):
        if not self._accepted(instance):
            yield from self._reference.iter_errors(instance)

    def is_valid(self, instance):
        return self._accepted(instance) or self._reference.is_valid(instance)


@lru_cache(maxsize=16)
def compiled_validator(raw_schema):
    return SupportValidator(json.loads(raw_schema))
