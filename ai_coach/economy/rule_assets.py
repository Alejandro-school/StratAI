"""Build and verify historical economic references from exact pinned source bytes."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts import validate_economy_candidate as base
from ai_coach.economy.candidate_io import (
    CONTRACTS,
    _write_new,
    sha,
    validate_output_directory,
)
from ai_coach.economy.rule_asset_text import (
    economic_defaults,
    require,
    steam_info,
    vdata,
)

REGISTRY = Path(__file__).with_name("rule_source_registry.json")
SCHEMAS = {
    "economy_rule_source_registry.schema.json",
    "economy_rule_package.schema.json",
    "economy_rule_assessment.schema.json",
}
GATES = {
    "comparison_allowed": False,
    "training_allowed": False,
    "product_supported": False,
}
VERSION = "1.0.0-candidate.1"
PACKAGE = "economy_rule_package.json"
SOURCE_PATHS = {
    "version": "game/csgo/steam.inf",
    "weapons": "game/csgo/pak01_dir/scripts/weapons.vdata",
    "competitive": "game/csgo/cfg/gamemode_competitive.cfg",
}


def schema_validator(name):
    require(name in SCHEMAS, "unknown economic rule schema")
    schema = json.loads((CONTRACTS / name).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_schema(data, name):
    error = next(schema_validator(name).iter_errors(data), None)
    require(
        error is None, f"invalid {name}: {list(error.absolute_path) if error else ''}"
    )


def load_sources(directory: Path, registry_path: Path = REGISTRY):
    registry, raw = base._read(registry_path)
    check_schema(registry, "economy_rule_source_registry.schema.json")
    documents, references = {}, {}
    for entry in registry["sources"]:
        key = entry["revision"], entry["kind"]
        require(key not in documents, "duplicate economic source")
        require(
            entry["repository_path"] == SOURCE_PATHS[entry["kind"]],
            "unexpected source path",
        )
        require(
            entry["file"] == f"{entry['revision']}-{entry['kind']}.txt",
            "source filename differs",
        )
        expected = f"https://raw.githubusercontent.com/SteamTracking/GameTracking-CS2/{entry['revision']}/{entry['repository_path']}"
        require(entry["url"] == expected, "source URL/revision differs")
        path = base._safe_file(directory.absolute(), entry["file"])
        require(
            path.stat().st_size == entry["bytes"] <= 4_000_000,
            "economic source size differs",
        )
        payload = path.read_bytes()
        require(sha(payload) == entry["sha256"], "economic source hash differs")
        documents[key] = payload.decode("utf-8-sig")
        references[key] = entry
    revisions = sorted({key[0] for key in documents})
    require(
        bool(revisions)
        and all((r, k) in documents for r in revisions for k in SOURCE_PATHS),
        "incomplete economic source set",
    )
    return registry, sha(raw), documents, references


def item_facts(text):
    result = []
    for name, block in vdata(text).items():
        if not name.isdigit():
            continue
        require(
            0 <= int(name) <= 65535 and isinstance(block, dict),
            "invalid item definition block",
        )
        label = block.get("m_szName")
        require(isinstance(label, str) and label, "item name unavailable")
        facts = {}
        for key, prop in (("price", "m_nPrice"), ("kill_award", "m_nKillAward")):
            number = block.get(prop)
            require(
                number is None or type(number) is int and 0 <= number <= 2_147_483_647,
                "invalid explicit economic asset value",
            )
            facts[key] = {
                "value": number,
                "status": "reference" if number is not None else "unavailable",
                "pointer": f"/{name}/{prop}",
            }
        result.append({"item_definition_index": int(name), "name": label, **facts})
    require(bool(result), "no numeric item definitions in source")
    require(
        len({r["item_definition_index"] for r in result}) == len(result),
        "aliased numeric item definitions",
    )
    return sorted(result, key=lambda row: row["item_definition_index"])


def build_package(registry, registry_hash, documents):
    releases = []
    for revision in sorted({key[0] for key in documents}):
        releases.append(
            {
                "revision": revision,
                "version": steam_info(documents[revision, "version"]),
                "items": item_facts(documents[revision, "weapons"]),
                "defaults": economic_defaults(documents[revision, "competitive"]),
            }
        )
    return {
        "schema_id": "stratai.economy_rule_package@1",
        "contract_version": VERSION,
        "registry_sha256": registry_hash,
        "source_references": registry["sources"],
        "status": "historical_reference_only",
        "demo_build_binding": "unverified",
        "server_configuration": "not_proved_by_defaults",
        **GATES,
        "releases": releases,
    }


def validate_package(package, registry, registry_hash, documents):
    """Re-read referenced fields directly, without calling the package constructor."""
    check_schema(package, "economy_rule_package.schema.json")
    require(
        package["registry_sha256"] == registry_hash
        and package["source_references"] == registry["sources"],
        "package source binding differs",
    )
    revisions = sorted({key[0] for key in documents})
    require(
        [row["revision"] for row in package["releases"]] == revisions,
        "package release coverage differs",
    )
    values_checked = 0
    for release in package["releases"]:
        revision = release["revision"]
        require(
            release["version"] == steam_info(documents[revision, "version"]),
            "package identity differs from source",
        )
        tree = vdata(documents[revision, "weapons"])
        ids = sorted(int(key) for key in tree if key.isdigit())
        require(
            [r["item_definition_index"] for r in release["items"]] == ids
            and len(set(ids)) == len(ids),
            "item coverage differs",
        )
        for row in release["items"]:
            node = tree[str(row["item_definition_index"])]
            require(
                row["name"] == node.get("m_szName"), "item name differs from source"
            )
            for field, native in (
                ("price", "m_nPrice"),
                ("kill_award", "m_nKillAward"),
            ):
                expected = node.get(native)
                fact = row[field]
                require(
                    fact["value"] == expected
                    and fact["status"]
                    == ("reference" if native in node else "unavailable")
                    and fact["pointer"] == f"/{row['item_definition_index']}/{native}",
                    "economic fact differs from source",
                )
                values_checked += native in node
        defaults = economic_defaults(documents[revision, "competitive"])
        require(release["defaults"] == defaults, "defaults differ from source")
        values_checked += len(defaults)
    return {
        "ok": True,
        "releases": len(revisions),
        "source_values_rechecked": values_checked,
    }


def load_package(path: Path, source_directory: Path, registry_path: Path = REGISTRY):
    registry, registry_hash, documents, _ = load_sources(
        source_directory, registry_path
    )
    package, raw = base._read(base._safe_file(path.parent.absolute(), path.name))
    validate_package(package, registry, registry_hash, documents)
    return package, sha(raw)


def materialize(source_directory: Path, output: Path, registry_path: Path = REGISTRY):
    output = validate_output_directory(output)
    registry, registry_hash, documents, _ = load_sources(
        source_directory, registry_path
    )
    package = build_package(registry, registry_hash, documents)
    validation = validate_package(package, registry, registry_hash, documents)
    return {**validation, "artifact": _write_new(output / PACKAGE, package)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sources", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    print(json.dumps(materialize(args.sources, args.output), sort_keys=True))


if __name__ == "__main__":
    main()
