"""Lossless, readable dictionary encoding for local replay model messages.

Identical JSON containers and repeated ordered field names are shared. Facts,
IDs, precision, ordering and the causal archive remain unchanged.
"""
from collections import Counter, defaultdict
from copy import deepcopy
import hashlib
import json
from os.path import commonprefix

from ai_coach.support.anchors import require

SHARED_VERSION = "stratai.replay_model_context@1"
VERSION = "stratai.replay_model_context@2"
TEMPLATE_VERSION = "stratai.replay_model_context@3"
SHARED_INSTRUCTIONS = """The user message may use stratai.replay_model_context@1.
Read payload as the original context. An object {"$ref":"D0"} means the
complete value in definitions.D0; expand references recursively. References
share identical values, never merge facts, observers or possible recipients.
{"$literal":{...}} escapes a literal source object: keep its keys and expand
only its values. All fact IDs, timestamps, ages and uncertainty remain exact.
Dictionary IDs D0 etc. are not evidence IDs; cite only original fact IDs.
Each consultation message has its own dictionary, with the same rules.
"""
INSTRUCTIONS = SHARED_INSTRUCTIONS.replace(SHARED_VERSION, VERSION) + """
columns.C0 is an ordered list of original object field names.
{"$row":"C0","values":[...]} is one complete object: match every value
to the field in the same column position. Expand nested references and rows.
Rows retain every event, ID and exact value; they do not aggregate events.
{"$row_literal":{...}} escapes source keys: keep its keys and expand values.
"""
TEMPLATE_INSTRUCTIONS = INSTRUCTIONS + """
The user message may instead use stratai.replay_model_context@3.
First expand body with the preceding dictionary/column rules. It contains
templates and payload. A payload object {"$template":"T0","args":[...]}
instantiates templates.T0: replace each {"$arg":0} with that scalar argument.
If the placeholder also has "$prefix", prepend that exact string to the string
argument. {"$prefix_list":"text","suffixes":[...]} is the original ordered
list of strings: prepend the prefix to every suffix. "$template_literal"
escapes a source object: retain its keys and expand only its values.
The restored payload is the complete original context. No fact, date, observer
or recipient is aggregated. T0/D0/C0 are transport IDs, never evidence IDs.
Each consultation message follows the same rules and may use @2 or plain JSON.
"""


def encoded(value):
    return json.dumps(value, ensure_ascii=False, allow_nan=False, separators=(",", ":")).encode("utf-8")


def checksum(value):
    return hashlib.sha256(encoded(value)).hexdigest()


def pack_shared(value):
    """Use a dictionary only if it saves bytes; verify exact reconstruction."""
    original = encoded(value)
    counts = Counter()

    def count(node):
        if isinstance(node, (dict, list)):
            raw = encoded(node)
            if len(raw) >= 96:
                counts[raw] += 1
            for child in node.values() if isinstance(node, dict) else node:
                count(child)

    count(value)
    definitions, identities = {}, {}

    def literal(node):
        if isinstance(node, dict):
            result = {key: visit(child) for key, child in node.items()}
            return {"$literal": result} if "$ref" in node or "$literal" in node else result
        return [visit(child) for child in node]

    def visit(node):
        if not isinstance(node, (dict, list)):
            return node
        raw = encoded(node)
        if counts[raw] < 2:
            return literal(node)
        if raw not in identities:
            content = literal(node)
            identity = "D" + str(len(definitions))
            definitions[identity] = content
            identities[raw] = identity
        return {"$ref": identities[raw]}

    payload = visit(value)
    result = {"wire_schema": SHARED_VERSION, "source_sha256": hashlib.sha256(original).hexdigest(),
              "definitions": definitions, "payload": payload}
    # The discriminator must not confuse literal source data with wire data.
    if len(encoded(result)) >= len(original) and not (isinstance(value, dict) and "wire_schema" in value):
        return deepcopy(value)
    require(encoded(unpack(result)) == original, "context packing changed source bytes")
    return result


def restore_shared(definitions, payload):
    require(isinstance(definitions, dict), "invalid context definitions")
    cache, active, used = {}, set(), set()

    def restore(node):
        if isinstance(node, list):
            return [restore(child) for child in node]
        if not isinstance(node, dict):
            return node
        if "$ref" in node:
            require(set(node) == {"$ref"} and isinstance(node["$ref"], str), "invalid context reference")
            identity = node["$ref"]
            require(identity in definitions and identity not in active, "missing or cyclic context reference")
            used.add(identity)
            if identity not in cache:
                active.add(identity)
                cache[identity] = restore(definitions[identity])
                active.remove(identity)
            return deepcopy(cache[identity])
        if "$literal" in node:
            require(set(node) == {"$literal"} and isinstance(node["$literal"], dict), "invalid context literal")
            node = node["$literal"]
        return {key: restore(child) for key, child in node.items()}

    result = restore(payload)
    require(used == set(definitions), "unused context definitions")
    return result


def pack(value, *, version=VERSION):
    """Share repeated column names as well as identical complete values."""
    require(version in (SHARED_VERSION, VERSION, TEMPLATE_VERSION), "unknown model context encoding")
    if version == TEMPLATE_VERSION:
        return _pack_templates(value)
    if version == SHARED_VERSION:
        return pack_shared(value)
    original = encoded(value)
    shapes = Counter()

    def count(node):
        if isinstance(node, dict):
            shapes[tuple(node)] += 1
            for child in node.values():
                count(child)
        elif isinstance(node, list):
            for child in node:
                count(child)

    count(value)
    eligible = {keys for keys, times in shapes.items() if times >= 3 and len(encoded(list(keys))) >= 64}
    columns, identities = {}, {}

    def visit(node):
        if isinstance(node, list):
            return [visit(child) for child in node]
        if not isinstance(node, dict):
            return node
        if "$row" in node or "$row_literal" in node:
            return {"$row_literal": {key: visit(child) for key, child in node.items()}}
        keys = tuple(node)
        if keys in eligible:
            if keys not in identities:
                identity = "C" + str(len(columns))
                identities[keys] = identity
                columns[identity] = list(keys)
            return {"$row": identities[keys], "values": [visit(child) for child in node.values()]}
        return {key: visit(child) for key, child in node.items()}

    tabulated = visit(value)
    shared = pack_shared(tabulated)
    if isinstance(shared, dict) and shared.get("wire_schema") == SHARED_VERSION:
        definitions, payload = shared["definitions"], shared["payload"]
    else:
        definitions, payload = {}, shared
    result = {"wire_schema": VERSION, "source_sha256": hashlib.sha256(original).hexdigest(),
              "columns": columns, "definitions": definitions, "payload": payload}
    if len(encoded(result)) >= len(original) and not (isinstance(value, dict) and "wire_schema" in value):
        return deepcopy(value)
    require(encoded(unpack(result)) == original, "context packing changed source bytes")
    return result


def unpack(value):
    """Restore both historical dictionaries and current column dictionaries."""
    if not isinstance(value, dict) or "wire_schema" not in value:
        encoded(value)
        return deepcopy(value)
    version = value["wire_schema"]
    if version == TEMPLATE_VERSION:
        require(set(value) == {'wire_schema', 'source_sha256', 'body'}, "template envelope fields differ")
        result = _restore_templates(unpack(value['body']))
        require(checksum(result) == value['source_sha256'], "context source hash differs")
        return result
    fields = {"wire_schema", "source_sha256", "definitions", "payload"}
    require(version in (SHARED_VERSION, VERSION), "unknown model context wire contract")
    require(set(value) == fields | ({"columns"} if version == VERSION else set()), "model context fields differ")
    result = restore_shared(value["definitions"], value["payload"])
    if version == VERSION:
        columns, used = value["columns"], set()
        require(isinstance(columns, dict), "invalid context columns")

        def restore(node):
            if isinstance(node, list):
                return [restore(child) for child in node]
            if not isinstance(node, dict):
                return node
            if "$row" in node:
                require(set(node) == {"$row", "values"} and isinstance(node["$row"], str)
                        and node["$row"] in columns and isinstance(node["values"], list), "invalid context row")
                identity = node["$row"]
                keys = columns[identity]
                require(isinstance(keys, list) and all(isinstance(key, str) for key in keys)
                        and len(keys) == len(set(keys)) and len(keys) == len(node["values"]), "invalid context row width")
                used.add(identity)
                return {key: restore(child) for key, child in zip(keys, node["values"], strict=True)}
            if "$row_literal" in node:
                require(set(node) == {"$row_literal"} and isinstance(node["$row_literal"], dict), "invalid row literal")
                node = node["$row_literal"]
            return {key: restore(child) for key, child in node.items()}

        result = restore(result)
        require(used == set(columns), "unused context columns")
    require(checksum(result) == value["source_sha256"], "context source hash differs")
    return result


_TEMPLATE_CONTROLS = {'$arg', '$template', '$prefix_list', '$template_literal'}


def _escape_templates(node):
    if isinstance(node, dict):
        literal = {key: _escape_templates(child) for key, child in node.items()}
        return {'$template_literal': literal} if _TEMPLATE_CONTROLS.intersection(node) else literal
    if isinstance(node, list):
        return [_escape_templates(child) for child in node]
    return node


def _template_shape(node):
    if isinstance(node, dict):
        return ('object', tuple((key, _template_shape(child)) for key, child in node.items()))
    if isinstance(node, list):
        return ('array', tuple(_template_shape(child) for child in node))
    return ('scalar',)


def _template_leaves(node):
    if isinstance(node, (dict, list)):
        for child in node.values() if isinstance(node, dict) else node:
            yield from _template_leaves(child)
    else:
        yield node


def _pack_templates(value):
    original = encoded(value)
    escaped = _escape_templates(value)
    groups = defaultdict(list)

    def collect(node):
        if isinstance(node, dict):
            if len(encoded(node)) >= 256:
                groups[_template_shape(node)].append(node)
            for child in node.values():
                collect(child)
        elif isinstance(node, list):
            for child in node:
                collect(child)

    collect(escaped)
    candidates = {}
    for shape, nodes in groups.items():
        if len(nodes) < 3:
            continue
        records = [list(_template_leaves(node)) for node in nodes]
        constants, variables, slots = {}, {}, {}
        for index, column in enumerate(zip(*records, strict=True)):
            values = tuple(encoded(child) for child in column)
            if len(set(values)) == 1:
                constants[index] = column[0]
                continue
            prefix = commonprefix(column) if all(isinstance(child, str) for child in column) else ''
            prefix = prefix if len(prefix) >= 4 else ''
            suffixes = tuple(encoded(child[len(prefix):] if prefix else child) for child in column)
            if suffixes not in slots:
                slots[suffixes] = len(slots)
            variables[index] = (slots[suffixes], prefix)
        leaf_indices = iter(range(len(records[0])))

        def blueprint(node):
            if isinstance(node, dict):
                return {key: blueprint(child) for key, child in node.items()}
            if isinstance(node, list):
                return [blueprint(child) for child in node]
            index = next(leaf_indices)
            if index in constants:
                return constants[index]
            slot, prefix = variables[index]
            return {'$arg': slot, '$prefix': prefix} if prefix else {'$arg': slot}

        template = blueprint(nodes[0])
        representatives = {slot: (index, prefix) for index, (slot, prefix) in variables.items()}
        bindings = [representatives[slot] for slot in range(len(slots))]
        cost = len(encoded(template)) + sum(len(encoded({'$template': 'T000', 'args': [
            record[index][len(prefix):] if prefix else record[index] for index, prefix in bindings
        ]})) for record in records)
        if cost < sum(len(encoded(node)) for node in nodes):
            candidates[shape] = (template, bindings)

    templates, identities = {}, {}

    def visit(node):
        if isinstance(node, dict):
            shape = _template_shape(node)
            if shape in candidates:
                if shape not in identities:
                    identity = 'T' + str(len(templates))
                    identities[shape] = identity
                    templates[identity] = candidates[shape][0]
                leaves = list(_template_leaves(node))
                return {'$template': identities[shape], 'args': [
                    leaves[index][len(prefix):] if prefix else leaves[index]
                    for index, prefix in candidates[shape][1]]}
            return {key: visit(child) for key, child in node.items()}
        if isinstance(node, list):
            if len(node) >= 3 and all(isinstance(child, str) for child in node):
                prefix = commonprefix(node)
                candidate = {'$prefix_list': prefix, 'suffixes': [child[len(prefix):] for child in node]}
                if len(prefix) >= 4 and len(encoded(candidate)) < len(encoded(node)):
                    return candidate
            return [visit(child) for child in node]
        return node

    payload = visit(escaped)
    result = {'wire_schema': TEMPLATE_VERSION, 'source_sha256': hashlib.sha256(original).hexdigest(),
              'body': pack({'templates': templates, 'payload': payload})}
    previous = pack(value)
    if len(encoded(result)) >= len(encoded(previous)):
        return previous
    require(encoded(unpack(result)) == original, "context templates changed source bytes")
    return result


def _restore_templates(body):
    require(isinstance(body, dict) and set(body) == {'templates', 'payload'}, "template body fields differ")
    templates = body['templates']
    require(isinstance(templates, dict) and all(isinstance(key, str) for key in templates), "invalid templates")
    used, active = set(), set()

    def restore(node, arguments=None, bindings=None):
        if isinstance(node, list):
            return [restore(child, arguments, bindings) for child in node]
        if not isinstance(node, dict):
            return node
        if '$template_literal' in node:
            require(set(node) == {'$template_literal'} and isinstance(node['$template_literal'], dict), "invalid template literal")
            return {key: restore(child, arguments, bindings) for key, child in node['$template_literal'].items()}
        if '$template' in node:
            require(set(node) == {'$template', 'args'} and isinstance(node['$template'], str), "invalid template instance")
            identity, args = node['$template'], node['args']
            require(identity in templates and identity not in active, "missing or cyclic template")
            require(isinstance(args, list) and not any(isinstance(arg, (dict, list)) for arg in args), "template arguments must be scalars")
            used.add(identity)
            active.add(identity)
            referenced = set()
            try:
                result = restore(templates[identity], args, referenced)
            finally:
                active.remove(identity)
            require(referenced == set(range(len(args))), "unused template arguments")
            return result
        if '$arg' in node:
            require(set(node) in ({'$arg'}, {'$arg', '$prefix'}) and arguments is not None, "invalid argument placeholder")
            index = node['$arg']
            require(type(index) is int and 0 <= index < len(arguments), "invalid template argument index")
            bindings.add(index)
            value = arguments[index]
            if '$prefix' in node:
                require(isinstance(node['$prefix'], str) and isinstance(value, str), "template prefix requires text")
                return node['$prefix'] + value
            return value
        if '$prefix_list' in node:
            require(set(node) == {'$prefix_list', 'suffixes'} and isinstance(node['$prefix_list'], str)
                    and isinstance(node['suffixes'], list) and all(isinstance(item, str) for item in node['suffixes']), "invalid prefix list")
            return [node['$prefix_list'] + suffix for suffix in node['suffixes']]
        return {key: restore(child, arguments, bindings) for key, child in node.items()}

    result = restore(body['payload'])
    require(used == set(templates), "unused templates")
    return result
