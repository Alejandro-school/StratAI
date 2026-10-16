"""Read verified v2 observations and write economic opportunities only in TEMP."""

from __future__ import annotations

import argparse
import hashlib
import json
import stat
import tempfile
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts import validate_economy_candidate as source_io
from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import opportunities as domain

CONTRACTS = Path(__file__).resolve().parents[1] / "contracts"
SCHEMAS = {
    "economy_opportunities.schema.json",
    "economy_opportunities_manifest.schema.json",
    "economy_opportunity_protocol.schema.json",
}


def schema_validator(filename: str) -> Draft202012Validator:
    source_io._require(filename in SCHEMAS, "unknown opportunity schema")
    schema = json.loads((CONTRACTS / filename).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_schema(value: dict, name: str) -> None:
    error = next(schema_validator(name).iter_errors(value), None)
    if error:
        raise RuntimeContractError(
            f"{name}: invalid value at {list(error.absolute_path)}"
        )


def sha(payload: bytes) -> str:
    return hashlib.sha256(payload).hexdigest()


def validate_output_directory(path: Path) -> Path:
    """Preflight before loading inputs or starting any demo parser."""
    path = path.absolute()
    temp = Path(tempfile.gettempdir()).resolve()
    source_io._require(
        path.is_dir() and path != temp and path.resolve().is_relative_to(temp),
        "opportunity output must be a dedicated directory under TEMP",
    )
    for entry in [*path.parents, path]:
        info = entry.lstat()
        source_io._require(
            not stat.S_ISLNK(info.st_mode)
            and not (
                getattr(info, "st_file_attributes", 0)
                & stat.FILE_ATTRIBUTE_REPARSE_POINT
            ),
            "opportunity output uses link/reparse point",
        )
    source_io._require(
        not any(path.iterdir()), "opportunity output must be empty; no overwrites"
    )
    return path


def load_bindings(catalog_path: Path | None = None) -> tuple[dict, str, dict, str]:
    protocol, raw = source_io._read(CONTRACTS / "economy_opportunity_protocol.json")
    check_schema(protocol, "economy_opportunity_protocol.schema.json")
    catalog, catalog_raw = source_io._read(
        catalog_path or CONTRACTS / "capability_catalog.json"
    )
    from ai_coach.contracts.validate_contract import validate_contract

    validate_contract(catalog)
    source_io._require(
        catalog["version"] == "1.7.0", "economic window requires catalog 1.7.0"
    )
    expected = "strictly after T0 and strictly before observed freeze end; boundary ticks ambiguous; execution labels only"
    decision = next(
        d for d in catalog["decisions"] if d["id"] == protocol["decision_id"]
    )
    source_io._require(
        decision["action_label_window"] == expected
        and decision.get("specific_contract") == "economy_opportunity_protocol.json",
        "catalog/economic window contract mismatch",
    )
    for id_ in protocol["future_action_ids"]:
        action = next(a for a in catalog["actions"] if a["id"] == id_)
        source_io._require(
            action["required_physical_parameters"]
            == protocol["future_action_requirements"],
            "economic action parameter contract mismatch",
        )
    return protocol, sha(raw), catalog, sha(catalog_raw)


def load_source(path: Path) -> tuple[dict, str, str]:
    path = source_io._safe_file(path.parent.absolute(), path.name)
    manifest, raw = source_io._read(path)
    source_io._require(
        manifest.get("contract_version") == "1.0.0-candidate.2",
        "opportunities require economic candidate v2",
    )
    source_io.validate_candidate(path)
    artifact_path = source_io._safe_file(path.parent, manifest["artifact"]["path"])
    data, payload = source_io._read(artifact_path)
    # Recheck bytes read after validation to avoid consuming a changed artifact.
    source_io._require(
        sha(payload) == manifest["artifact"]["sha256"]
        and len(payload) == manifest["artifact"]["bytes"],
        "source changed during opportunity load",
    )
    source_io._require(path.read_bytes() == raw, "source manifest changed during load")
    return data, sha(raw), sha(payload)


def _write_new(path: Path, data: dict) -> dict:
    payload = (
        json.dumps(data, ensure_ascii=False, separators=(",", ":"), allow_nan=False)
        + "\n"
    ).encode()
    with path.open("xb") as handle:
        handle.write(payload)
    return {"path": path.name, "bytes": len(payload), "sha256": sha(payload)}


def materialize(
    source: Path, output: Path, *, catalog_path: Path | None = None
) -> dict:
    output = validate_output_directory(output)
    protocol, protocol_hash, catalog, catalog_hash = load_bindings(catalog_path)
    data, manifest_hash, data_hash = load_source(source)
    result = domain.build_opportunities(
        data,
        protocol,
        manifest_hash,
        data_hash,
        protocol_hash,
        catalog["version"],
        catalog_hash,
    )
    from ai_coach.economy.validate_opportunities import validate_data

    validate_data(result, data, protocol)
    reference = _write_new(output / domain.DATA_FILE, result)
    manifest = {
        k: v
        for k, v in result.items()
        if k
        not in (
            "schema_id",
            "map_id",
            "tick_rate",
            "capability_id",
            "decision_id",
            "opportunities",
        )
    }
    manifest.update(
        schema_id=domain.MANIFEST_SCHEMA,
        artifact=reference,
        summary=domain.summary(result),
    )
    check_schema(manifest, "economy_opportunities_manifest.schema.json")
    _write_new(output / domain.MANIFEST_FILE, manifest)
    return {
        "ok": True,
        "summary": manifest["summary"],
        "manifest": str(output / domain.MANIFEST_FILE),
        "training_allowed": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--catalog", type=Path)
    args = parser.parse_args()
    try:
        report = materialize(args.source, args.output, catalog_path=args.catalog)
    except (OSError, ValueError, RuntimeContractError) as exc:
        parser.exit(1, f"economic opportunities rejected: {exc}\n")
    print(json.dumps(report, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
