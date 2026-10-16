"""Read the bounded text formats used by pinned economic reference assets."""

from __future__ import annotations

import json
import re

from ai_coach.contracts.runtime_contracts import RuntimeContractError

TOKEN = re.compile(
    r'\s+|<!--.*?-->|//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\])*"|[{}\[\]=,:]|[^\s{}\[\]=,:"/]+',
    re.DOTALL,
)
ECONOMIC_NAME = re.compile(
    r"cash_[a-z_]+|ammo_grenade_limit_[a-z_]+|mp_(?:buy[a-z_]*|freezetime|maxmoney|startmoney|maxrounds|overtime_[a-z_]+|weapons_allow_[a-z_]+|afterroundmoney|loser_bonus_[a-z_]+)"
)


def require(condition, message):
    if not condition:
        raise RuntimeContractError(message)


def vdata(text: str) -> dict:
    """Parse explicit KV3 values; do not invent engine defaults or inheritance."""
    tokens = []
    position = 0
    for match in TOKEN.finditer(text):
        require(match.start() == position, "unsupported economic KV3 token")
        position = match.end()
        token = match.group()
        if token.isspace() or token.startswith(("<!--", "//", "/*")):
            continue
        tokens.append(token)
    require(position == len(text), "trailing unsupported economic KV3 data")
    cursor = 0

    def take():
        nonlocal cursor
        require(cursor < len(tokens), "truncated economic KV3")
        token = tokens[cursor]
        cursor += 1
        return token

    def value(depth=0):
        nonlocal cursor
        require(depth < 64, "economic KV3 nesting limit")
        token = take()
        if token == "{":
            result = {}
            while cursor < len(tokens) and tokens[cursor] != "}":
                key = take()
                key = json.loads(key) if key.startswith('"') else key
                require(
                    re.fullmatch(r"[A-Za-z0-9_]+", key) is not None,
                    "invalid economic KV3 key",
                )
                require(
                    key not in result and take() == "=",
                    "duplicate key or missing economic KV3 assignment",
                )
                result[key] = value(depth + 1)
                if cursor < len(tokens) and tokens[cursor] == ",":
                    cursor += 1
            require(take() == "}", "unclosed economic KV3 object")
            return result
        if token == "[":
            result = []
            while cursor < len(tokens) and tokens[cursor] != "]":
                result.append(value(depth + 1))
                if cursor < len(tokens) and tokens[cursor] == ",":
                    cursor += 1
            require(take() == "]", "unclosed economic KV3 array")
            return result
        require(token not in "{}[]=,:", "unexpected economic KV3 punctuation")
        if cursor < len(tokens) and tokens[cursor] == ":":
            cursor += 1
            return (token, value(depth + 1))
        if token.startswith('"'):
            return json.loads(token)
        if token in ("true", "false", "null"):
            return json.loads(token)
        if re.fullmatch(r"-?[0-9]+", token):
            return int(token)
        if re.fullmatch(r"-?[0-9]+\.[0-9]+(?:[eE][+-]?[0-9]+)?", token):
            return float(token)
        return token

    result = value()
    require(
        cursor == len(tokens) and isinstance(result, dict), "invalid economic KV3 root"
    )
    return result


def steam_info(text: str) -> dict:
    result = {}
    for line in text.splitlines():
        if not line.strip():
            continue
        parts = line.split("=", 1)
        require(
            len(parts) == 2 and parts[0] not in result,
            "invalid or duplicate steam.inf field",
        )
        result[parts[0]] = parts[1].strip()
    require(
        result.get("ProductName") == "cs2" and result.get("appID") == "730",
        "wrong game reference",
    )
    for name in ("ClientVersion", "ServerVersion", "SourceRevision"):
        require(
            re.fullmatch(r"[1-9][0-9]*", result.get(name, "")) is not None,
            "missing source build identity",
        )
    require(
        re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+", result.get("PatchVersion", ""))
        is not None,
        "invalid patch label",
    )
    return result


def economic_defaults(text: str) -> list[dict]:
    result, seen = [], set()
    for line_number, line in enumerate(text.splitlines(), 1):
        line = line.split("//", 1)[0].strip()
        if not line:
            continue
        parts = line.split()
        if not ECONOMIC_NAME.fullmatch(parts[0]):
            continue
        require(
            len(parts) == 2
            and re.fullmatch(r"-?[0-9]+(?:\.[0-9]+)?", parts[1]) is not None,
            "unsupported economic config assignment",
        )
        require(parts[0] not in seen, "duplicate economic default")
        seen.add(parts[0])
        result.append({"name": parts[0], "value": parts[1], "line": line_number})
    return result
