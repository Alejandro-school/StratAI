#!/usr/bin/env python3
"""Validate the secret-free golden demo inventory and deterministic selection.

This validator proves only manifest integrity and, when roots are supplied,
local source-file identity. It never reports Gate 1 as passed because it does
not reprocess demos or exercise the causal and determinism gates.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
from collections import Counter, defaultdict
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import Any

SCHEMA_ID = "stratai.golden_demo_corpus@2"
CORPUS_ID = "golden-demos-v2"
CORPUS_VERSION = 2
VALIDATOR_VERSION = "stratai.golden_demo_corpus_validator@2"
SEMANTIC_RUNNER_VERSION = "stratai.golden_demo_semantic_runner@1"
LEGACY_MANUAL_RUNNER_VERSION = "stratai.manual_gate1_smoke@1"
PARSER_SCHEMA_VERSION = "v16"
EXPORT_FORMAT_VERSION = "3.8.0"
CANONICAL_VALIDATOR_VERSION = "stratai.canonical_validator@2"
ALGORITHM_ID = "fixed_map_quota_sha256_ascending@1"
DEMO_MAGIC = b"PBDEMS2\x00"
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
PRIVATE_KEY_PATTERN = re.compile(
    r"(?:^|_)(?:path|directory|filename|file_name|match_id|player_id|steam_id|"
    r"faceit_id|url|token|secret)(?:$|_)",
    re.IGNORECASE,
)
PRIVATE_VALUE_PATTERN = re.compile(r"(?:^[A-Za-z]:[\\/]|^\\\\|^/|\.\.[\\/])")

SOURCE_POOLS = ("stratai_runtime", "faceit_offline")
ROOT_ENV_VARS = {
    "stratai_runtime": "STRATAI_GOLDEN_DEMO_ROOT",
    "faceit_offline": "FACEIT_GOLDEN_DEMO_ROOT",
}
MAP_ORDER = (
    "de_mirage",
    "de_dust2",
    "de_nuke",
    "de_ancient",
    "de_anubis",
    "de_inferno",
    "de_cache",
    "de_vertigo",
)
MAP_QUOTAS = {
    "de_mirage": 15,
    "de_dust2": 6,
    "de_nuke": 4,
    "de_ancient": 4,
    "de_anubis": 4,
    "de_inferno": 4,
    "de_cache": 2,
    "de_vertigo": 1,
}
EXPECTED_CONTRACTS = {
    "parser_schema_version": PARSER_SCHEMA_VERSION,
    "export_format_version": EXPORT_FORMAT_VERSION,
    "canonical_validator_version": CANONICAL_VALIDATOR_VERSION,
    "golden_demo_validator_version": VALIDATOR_VERSION,
    "semantic_runner_version": SEMANTIC_RUNNER_VERSION,
}
EXPECTED_ALGORITHM = {
    "algorithm_id": ALGORITHM_ID,
    "candidate_rule": "regular_non_symlink_dot_dem_with_pbdems2_header",
    "dedupe_key": "sha256",
    "order_key": "sha256_ascending_within_map",
    "map_order": list(MAP_ORDER),
    "map_quotas": MAP_QUOTAS,
}
PARTIAL_GATE_DECLARATION = {
    "status": "not_proven",
    "evidence": "partial_semantic_evidence_insufficient",
    "semantic_reprocessing_required": True,
}
BLOCKED_GATE_DECLARATION = {
    "status": "not_proven",
    "evidence": "corpus_replacement_required",
    "semantic_reprocessing_required": True,
}
MANUAL_REVIEW_GATE_DECLARATION = {
    "status": "not_proven",
    "evidence": "manual_decision_review_pending",
    "semantic_reprocessing_required": False,
}
EXPECTED_SEMANTIC_KEYS = frozenset(
    {
        "status",
        "evaluated_count",
        "passed_count",
        "quarantined_count",
        "cases",
    }
)
EXPECTED_SEMANTIC_CASE_KEYS = frozenset(
    {
        "source_alias",
        "status",
        "checks",
        "failure_domain",
        "reason",
        "notes",
        "runner_version",
    }
)
ALLOWED_SEMANTIC_CHECKS = frozenset(
    {
        "canonical_export",
        "canonical_validator",
        "gomaxprocs_1_4_tree_hash_equality",
        "source_identity",
    }
)
REQUIRED_PASS_CHECKS = frozenset(
    {"canonical_export", "canonical_validator", "source_identity"}
)
ALLOWED_RUNNER_VERSIONS = frozenset(
    {SEMANTIC_RUNNER_VERSION, LEGACY_MANUAL_RUNNER_VERSION}
)
ALLOWED_FAILURE_DOMAINS = frozenset(
    {
        "block6_quality",
        "canonical_contract",
        "combat_quality",
        "demo_parse",
        "determinism",
        "engagement_quality",
        "objective_round_reconciliation",
        "roster_identity",
        "tactical_export",
        "utility_quality",
    }
)
ALLOWED_FAILURE_REASONS = frozenset(
    {
        "block6_quality_gate_rejected",
        "canonical_validator_rejected_bundle",
        "combat_quality_gate_rejected",
        "determinism_mismatch",
        "engagement_quality_gate_rejected",
        "incomplete_final_round",
        "objective_quality_gate_rejected",
        "parser_rejected_demo",
        "unreconciled_playing_participant",
        "tactical_export_rejected",
        "utility_quality_gate_rejected",
    }
)
ALLOWED_SEMANTIC_NOTES = frozenset(
    {
        "decision_family_peek_hold_or_reposition_observed",
        "decision_family_spacing_or_trade_connection_observed",
        "map_geometry_loaded",
        "map_geometry_unavailable_tradeability_abstained",
        "objective_round_mismatch_detected",
    }
)
EXPECTED_ROOT_KEYS = frozenset(
    {
        "schema_id",
        "corpus_id",
        "version",
        "training_allowed",
        "candidate_count",
        "selected_count",
        "candidate_pool_sha256",
        "selection_sha256",
        "contracts",
        "selection_algorithm",
        "gate_1",
        "semantic_validation",
        "sources",
    }
)
EXPECTED_SOURCE_KEYS = frozenset(
    {
        "source_alias",
        "sha256",
        "bytes",
        "map",
        "source_pool",
        "provenance",
        "selection",
    }
)
EXPECTED_PROVENANCE_KEYS = frozenset(
    {
        "inventory_method",
        "checksum_evidence",
        "ownership_evidence",
        "bundle_linkage",
    }
)
EXPECTED_SELECTION_KEYS = frozenset(
    {"selected", "rank_within_map", "reason"}
)
ALLOWED_PROVENANCE = frozenset(
    {
        (
            "stratai_runtime",
            "sha256_file_audit@1",
            "persisted_sidecar_verified",
            "unrecorded",
            "checksum_linked_legacy_bundle",
        ),
        (
            "faceit_offline",
            "sha256_file_audit@1",
            "persisted_sidecar_verified",
            "faceit_download_owner@1",
            "none",
        ),
        (
            "faceit_offline",
            "sha256_file_audit@1",
            "audit_recomputed",
            "unrecorded",
            "none",
        ),
    }
)
LIMITATIONS = (
    "This validator verifies recorded semantic evidence but does not reprocess demos.",
    (
        "Full-corpus causal invariance, observed/oracle separation, 16 Hz "
        "continuity, and determinism remain unproven."
    ),
    "A source_inventory_valid result is not Gate 1 approval.",
)


class DuplicateKeyError(ValueError):
    """Raised when JSON contains a duplicate object key."""


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise DuplicateKeyError(f"clave JSON duplicada: {key}")
        result[key] = value
    return result


def _canonical_json(value: object) -> bytes:
    return json.dumps(
        value,
        ensure_ascii=True,
        sort_keys=True,
        separators=(",", ":"),
    ).encode("utf-8")


def candidate_pool_digest(sources: Sequence[Mapping[str, object]]) -> str:
    digest = hashlib.sha256()
    for source in sources:
        digest.update(
            _canonical_json(
                {
                    "source_alias": source.get("source_alias"),
                    "sha256": source.get("sha256"),
                    "bytes": source.get("bytes"),
                    "map": source.get("map"),
                    "source_pool": source.get("source_pool"),
                    "provenance": source.get("provenance"),
                }
            )
        )
        digest.update(b"\n")
    return digest.hexdigest()


def selection_digest(sources: Sequence[Mapping[str, object]]) -> str:
    digest = hashlib.sha256()
    digest.update(_canonical_json(EXPECTED_ALGORITHM))
    digest.update(b"\n")
    for source in sources:
        selection = source.get("selection")
        if not isinstance(selection, Mapping) or selection.get("selected") is not True:
            continue
        digest.update(
            _canonical_json(
                {
                    "source_alias": source.get("source_alias"),
                    "sha256": source.get("sha256"),
                    "map": source.get("map"),
                    "source_pool": source.get("source_pool"),
                    "rank_within_map": selection.get("rank_within_map"),
                }
            )
        )
        digest.update(b"\n")
    return digest.hexdigest()


def _exact_keys(
    value: Mapping[str, object],
    expected: frozenset[str],
    location: str,
    errors: list[str],
) -> None:
    actual = set(value)
    missing = sorted(expected - actual)
    unknown = sorted(actual - expected)
    if missing:
        errors.append(f"{location}: faltan campos {missing}")
    if unknown:
        errors.append(f"{location}: campos no permitidos {unknown}")


def _private_reference_errors(value: object, location: str = "manifest") -> list[str]:
    errors: list[str] = []
    if isinstance(value, Mapping):
        for key, item in value.items():
            key_location = f"{location}.{key}"
            if PRIVATE_KEY_PATTERN.search(str(key)):
                errors.append(f"{key_location}: referencia privada no permitida")
            errors.extend(_private_reference_errors(item, key_location))
    elif isinstance(value, list):
        for index, item in enumerate(value):
            errors.extend(_private_reference_errors(item, f"{location}[{index}]"))
    elif isinstance(value, str) and (
        "\\" in value or "/" in value or PRIVATE_VALUE_PATTERN.search(value)
    ):
        errors.append(f"{location}: ruta privada no permitida")
    return errors


def load_manifest(path: Path) -> tuple[dict[str, object] | None, list[str]]:
    try:
        value = json.loads(
            Path(path).read_text(encoding="utf-8"), object_pairs_hook=_unique_object
        )
    except (OSError, UnicodeError, json.JSONDecodeError, DuplicateKeyError) as error:
        return None, [f"manifest.json invalido ({error})"]
    if not isinstance(value, dict):
        return None, ["manifest.json debe contener un objeto"]
    return value, []


def _validate_versions(manifest: Mapping[str, object], errors: list[str]) -> None:
    contracts = manifest.get("contracts")
    if not isinstance(contracts, Mapping):
        errors.append("manifest.contracts debe ser objeto")
        return
    _exact_keys(
        contracts,
        frozenset(EXPECTED_CONTRACTS),
        "manifest.contracts",
        errors,
    )
    for key, expected in EXPECTED_CONTRACTS.items():
        if contracts.get(key) != expected:
            errors.append(
                f"manifest.contracts.{key}: version incompatible; debe ser {expected}"
            )


def _validate_algorithm(manifest: Mapping[str, object], errors: list[str]) -> None:
    algorithm = manifest.get("selection_algorithm")
    if algorithm != EXPECTED_ALGORITHM:
        errors.append(
            "manifest.selection_algorithm: algoritmo o cuotas incompatibles con "
            f"{ALGORITHM_ID}"
        )


def _validate_sources(
    manifest: Mapping[str, object], errors: list[str]
) -> list[Mapping[str, object]]:
    raw_sources = manifest.get("sources")
    if not isinstance(raw_sources, list):
        errors.append("manifest.sources debe ser una lista")
        return []
    sources: list[Mapping[str, object]] = []
    aliases: list[object] = []
    checksums: list[object] = []
    grouped: dict[str, list[Mapping[str, object]]] = defaultdict(list)
    for index, raw_source in enumerate(raw_sources):
        location = f"manifest.sources[{index}]"
        if not isinstance(raw_source, Mapping):
            errors.append(f"{location}: debe ser objeto")
            continue
        source = raw_source
        sources.append(source)
        _exact_keys(source, EXPECTED_SOURCE_KEYS, location, errors)
        alias = source.get("source_alias")
        checksum = source.get("sha256")
        aliases.append(alias)
        checksums.append(checksum)
        if not isinstance(checksum, str) or not SHA256_PATTERN.fullmatch(checksum):
            errors.append(f"{location}.sha256: SHA-256 lowercase invalido")
        expected_alias = (
            f"demo-{checksum[:20]}" if isinstance(checksum, str) else None
        )
        if (
            not isinstance(alias, str)
            or not ALIAS_PATTERN.fullmatch(alias)
            or alias != expected_alias
        ):
            errors.append(
                f"{location}.source_alias: debe ser el alias opaco derivado del SHA-256"
            )
        size = source.get("bytes")
        if isinstance(size, bool) or not isinstance(size, int) or size <= len(DEMO_MAGIC):
            errors.append(f"{location}.bytes: tamano positivo invalido")
        map_name = source.get("map")
        if map_name not in MAP_ORDER:
            errors.append(f"{location}.map: mapa no permitido")
        else:
            grouped[str(map_name)].append(source)
        source_pool = source.get("source_pool")
        if source_pool not in SOURCE_POOLS:
            errors.append(f"{location}.source_pool: pool no permitido")
        provenance = source.get("provenance")
        if not isinstance(provenance, Mapping):
            errors.append(f"{location}.provenance: debe ser objeto")
        else:
            _exact_keys(
                provenance, EXPECTED_PROVENANCE_KEYS, f"{location}.provenance", errors
            )
            provenance_tuple = (
                source_pool,
                provenance.get("inventory_method"),
                provenance.get("checksum_evidence"),
                provenance.get("ownership_evidence"),
                provenance.get("bundle_linkage"),
            )
            if provenance_tuple not in ALLOWED_PROVENANCE:
                errors.append(f"{location}.provenance: combinacion no permitida")
        selection = source.get("selection")
        if not isinstance(selection, Mapping):
            errors.append(f"{location}.selection: debe ser objeto")
        else:
            _exact_keys(
                selection, EXPECTED_SELECTION_KEYS, f"{location}.selection", errors
            )
            rank = selection.get("rank_within_map")
            if isinstance(rank, bool) or not isinstance(rank, int) or rank <= 0:
                errors.append(
                    f"{location}.selection.rank_within_map: rango positivo obligatorio"
                )
            if not isinstance(selection.get("selected"), bool):
                errors.append(f"{location}.selection.selected: booleano obligatorio")
    duplicate_aliases = sorted(
        str(value) for value, count in Counter(aliases).items() if count > 1
    )
    if duplicate_aliases:
        errors.append("manifest.sources: source_alias duplicado")
    duplicate_checksums = sorted(
        str(value) for value, count in Counter(checksums).items() if count > 1
    )
    if duplicate_checksums:
        errors.append("manifest.sources: sha256 duplicado")

    expected_global_order: list[Mapping[str, object]] = []
    for map_name in MAP_ORDER:
        map_sources = sorted(grouped[map_name], key=lambda item: str(item.get("sha256")))
        expected_global_order.extend(map_sources)
        quota = MAP_QUOTAS[map_name]
        if len(map_sources) < quota:
            errors.append(
                f"manifest.sources: {map_name} tiene {len(map_sources)} candidatos y cuota {quota}"
            )
        for rank, source in enumerate(map_sources, 1):
            selection = source.get("selection")
            if not isinstance(selection, Mapping):
                continue
            expected_selected = rank <= quota
            expected_reason = (
                "selected_within_map_quota"
                if expected_selected
                else "outside_map_quota"
            )
            if selection.get("rank_within_map") != rank:
                errors.append(
                    f"{source.get('source_alias')}: rank_within_map no coincide con SHA ascendente"
                )
            if selection.get("selected") is not expected_selected:
                errors.append(
                    f"{source.get('source_alias')}: seleccion no coincide con la cuota"
                )
            if selection.get("reason") != expected_reason:
                errors.append(
                    f"{source.get('source_alias')}: motivo de seleccion invalido"
                )
    if [source.get("source_alias") for source in sources] != [
        source.get("source_alias") for source in expected_global_order
    ]:
        errors.append(
            "manifest.sources: orden global debe ser map_order y SHA-256 ascendente"
        )
    return sources


def _validate_semantic_evidence(
    manifest: Mapping[str, object],
    sources: Sequence[Mapping[str, object]],
    errors: list[str],
) -> None:
    evidence = manifest.get("semantic_validation")
    if not isinstance(evidence, Mapping):
        errors.append("manifest.semantic_validation debe ser objeto")
        return
    _exact_keys(evidence, EXPECTED_SEMANTIC_KEYS, "manifest.semantic_validation", errors)
    cases = evidence.get("cases")
    if not isinstance(cases, list):
        errors.append("manifest.semantic_validation.cases debe ser lista")
        return
    selected_aliases = [
        str(source.get("source_alias"))
        for source in sources
        if isinstance(source.get("selection"), Mapping)
        and source["selection"].get("selected") is True
    ]
    selected_alias_set = set(selected_aliases)
    seen_aliases: set[str] = set()
    observed_aliases: list[str] = []
    passed = 0
    quarantined = 0
    determinism_checks = 0
    for index, case in enumerate(cases):
        location = f"manifest.semantic_validation.cases[{index}]"
        if not isinstance(case, Mapping):
            errors.append(f"{location}: debe ser objeto")
            continue
        _exact_keys(case, EXPECTED_SEMANTIC_CASE_KEYS, location, errors)
        alias = case.get("source_alias")
        if not isinstance(alias, str) or alias not in selected_alias_set:
            errors.append(f"{location}.source_alias: no referencia una fuente seleccionada")
        elif alias in seen_aliases:
            errors.append(f"{location}.source_alias: evidencia duplicada")
        else:
            seen_aliases.add(alias)
            observed_aliases.append(alias)
        runner_version = case.get("runner_version")
        if runner_version not in ALLOWED_RUNNER_VERSIONS:
            errors.append(f"{location}.runner_version: version no permitida")
        status = case.get("status")
        if status == "passed":
            passed += 1
            if case.get("failure_domain") is not None or case.get("reason") is not None:
                errors.append(f"{location}: un caso passed no puede declarar fallo")
        elif status == "quarantined":
            quarantined += 1
            if case.get("failure_domain") not in ALLOWED_FAILURE_DOMAINS:
                errors.append(f"{location}: failure_domain no permitido")
            if case.get("reason") not in ALLOWED_FAILURE_REASONS:
                errors.append(f"{location}: reason no permitido")
        else:
            errors.append(f"{location}.status: debe ser passed o quarantined")
        checks = case.get("checks")
        if (
            not isinstance(checks, list)
            or not checks
            or not all(isinstance(check, str) and check in ALLOWED_SEMANTIC_CHECKS for check in checks)
            or checks != sorted(set(checks))
        ):
            errors.append(f"{location}.checks: checks invalidos o no deterministas")
        elif "gomaxprocs_1_4_tree_hash_equality" in checks:
            determinism_checks += 1
        if (
            status == "passed"
            and runner_version == SEMANTIC_RUNNER_VERSION
            and isinstance(checks, list)
            and not REQUIRED_PASS_CHECKS <= set(checks)
        ):
            errors.append(f"{location}: passed requiere identidad, export y validador")
        notes = case.get("notes")
        if (
            not isinstance(notes, list)
            or not all(note in ALLOWED_SEMANTIC_NOTES for note in notes)
            or notes != sorted(set(notes))
        ):
            errors.append(f"{location}.notes: notas invalidas o no deterministas")
    expected_alias_order = [alias for alias in selected_aliases if alias in seen_aliases]
    if observed_aliases != expected_alias_order:
        errors.append(
            "manifest.semantic_validation.cases: orden debe seguir las fuentes seleccionadas"
        )
    if not cases:
        expected_status = "not_evaluated"
    elif len(seen_aliases) < len(selected_aliases):
        expected_status = "partial"
    elif quarantined:
        expected_status = "blocked"
    else:
        expected_status = "passed"
    if evidence.get("status") != expected_status:
        errors.append(
            f"manifest.semantic_validation.status debe ser {expected_status}"
        )
    if expected_status in {"blocked", "passed"}:
        if any(
            case.get("runner_version") != SEMANTIC_RUNNER_VERSION
            for case in cases
            if isinstance(case, Mapping)
        ):
            errors.append(
                "manifest.semantic_validation: la evaluacion completa requiere el runner vigente"
            )
        if determinism_checks != 1:
            errors.append(
                "manifest.semantic_validation: la evaluacion completa requiere exactamente una prueba GOMAXPROCS"
            )
    expected_counts = {
        "evaluated_count": len(cases),
        "passed_count": passed,
        "quarantined_count": quarantined,
    }
    for field, expected in expected_counts.items():
        if evidence.get(field) != expected:
            errors.append(f"manifest.semantic_validation.{field} debe ser {expected}")


def expected_gate_declaration(semantic_status: object) -> dict[str, object]:
    if semantic_status == "blocked":
        return dict(BLOCKED_GATE_DECLARATION)
    if semantic_status == "passed":
        return dict(MANUAL_REVIEW_GATE_DECLARATION)
    return dict(PARTIAL_GATE_DECLARATION)


def validate_manifest_metadata(
    manifest: Mapping[str, object],
) -> tuple[list[str], list[Mapping[str, object]]]:
    errors = _private_reference_errors(manifest)
    _exact_keys(manifest, EXPECTED_ROOT_KEYS, "manifest", errors)
    if manifest.get("schema_id") != SCHEMA_ID:
        errors.append(f"manifest.schema_id debe ser {SCHEMA_ID}")
    if manifest.get("corpus_id") != CORPUS_ID:
        errors.append(f"manifest.corpus_id debe ser {CORPUS_ID}")
    if manifest.get("version") != CORPUS_VERSION:
        errors.append(f"manifest.version debe ser {CORPUS_VERSION}")
    if manifest.get("training_allowed") is not False:
        errors.append("manifest.training_allowed debe ser false")
    _validate_versions(manifest, errors)
    _validate_algorithm(manifest, errors)
    sources = _validate_sources(manifest, errors)
    _validate_semantic_evidence(manifest, sources, errors)
    semantic = manifest.get("semantic_validation")
    semantic_status = semantic.get("status") if isinstance(semantic, Mapping) else None
    expected_gate = expected_gate_declaration(semantic_status)
    if manifest.get("gate_1") != expected_gate:
        errors.append(
            "manifest.gate_1 no coincide con el estado de validacion semantica"
        )
    candidate_count = manifest.get("candidate_count")
    selected_count = manifest.get("selected_count")
    actual_selected = sum(
        1
        for source in sources
        if isinstance(source.get("selection"), Mapping)
        and source["selection"].get("selected") is True
    )
    if candidate_count != len(sources):
        errors.append("manifest.candidate_count no coincide con sources")
    if selected_count != actual_selected:
        errors.append("manifest.selected_count no coincide con sources")
    if isinstance(selected_count, bool) or not isinstance(selected_count, int):
        errors.append("manifest.selected_count debe ser entero")
    elif not 30 <= selected_count <= 50:
        errors.append("manifest.selected_count debe estar entre 30 y 50")
    if selected_count != sum(MAP_QUOTAS.values()):
        errors.append("manifest.selected_count no coincide con las cuotas")
    if manifest.get("candidate_pool_sha256") != candidate_pool_digest(sources):
        errors.append("manifest.candidate_pool_sha256 no coincide")
    if manifest.get("selection_sha256") != selection_digest(sources):
        errors.append("manifest.selection_sha256 no coincide")
    return errors, sources


def _inspect_demo(
    path: Path, known_maps: Sequence[str]
) -> tuple[str | None, int, str | None, str | None]:
    digest = hashlib.sha256()
    found_maps: set[str] = set()
    encoded_maps = {name: name.encode("ascii") for name in known_maps}
    overlap = max(len(value) for value in encoded_maps.values()) - 1
    tail = b""
    size = 0
    magic: bytes | None = None
    try:
        with path.open("rb") as handle:
            while chunk := handle.read(1024 * 1024):
                if magic is None:
                    magic = chunk[: len(DEMO_MAGIC)]
                size += len(chunk)
                digest.update(chunk)
                searchable = tail + chunk
                for name, encoded in encoded_maps.items():
                    if encoded in searchable:
                        found_maps.add(name)
                tail = searchable[-overlap:] if overlap else b""
    except OSError as error:
        return None, 0, None, type(error).__name__
    if magic != DEMO_MAGIC:
        return digest.hexdigest(), size, None, "invalid_demo_magic"
    if len(found_maps) != 1:
        return digest.hexdigest(), size, None, "map_unavailable_or_ambiguous"
    return digest.hexdigest(), size, next(iter(found_maps)), None


def resolve_local_sources(
    sources: Sequence[Mapping[str, object]], roots: Mapping[str, Path]
) -> tuple[dict[str, Path], list[str]]:
    """Resolve opaque aliases to local paths without persisting those paths."""

    errors: list[str] = []
    normalized_roots: dict[str, Path] = {}
    required_pools = {str(source.get("source_pool")) for source in sources}
    for pool in sorted(required_pools):
        root = roots.get(pool)
        if root is None:
            errors.append(
                f"root de {pool} obligatorio por CLI o {ROOT_ENV_VARS.get(pool, 'env')}"
            )
            continue
        candidate_root = Path(root)
        resolved = candidate_root.resolve()
        if candidate_root.is_symlink() or not resolved.is_dir():
            errors.append(f"root de {pool} no es un directorio regular")
            continue
        normalized_roots[pool] = resolved
    root_values = list(normalized_roots.values())
    for index, left in enumerate(root_values):
        for right in root_values[index + 1 :]:
            if left == right or left in right.parents or right in left.parents:
                errors.append("los roots de fuentes deben ser distintos y disjuntos")
    if errors:
        return {}, errors

    expected_sizes: dict[str, set[int]] = defaultdict(set)
    for source in sources:
        size = source.get("bytes")
        pool = source.get("source_pool")
        if isinstance(size, int) and not isinstance(size, bool) and pool in SOURCE_POOLS:
            expected_sizes[str(pool)].add(size)
    inspected: dict[
        str,
        dict[int, list[tuple[str, str | None, str | None, Path]]],
    ] = {
        pool: defaultdict(list) for pool in normalized_roots
    }
    for pool, root in normalized_roots.items():
        for path in sorted(root.rglob("*")):
            if (
                not path.is_file()
                or path.is_symlink()
                or path.suffix.lower() != ".dem"
            ):
                continue
            try:
                size = path.stat().st_size
            except OSError:
                continue
            if size not in expected_sizes[pool]:
                continue
            checksum, observed_size, map_name, inspection_error = _inspect_demo(
                path, MAP_ORDER
            )
            if checksum is None:
                continue
            inspected[pool][observed_size].append(
                (checksum, map_name, inspection_error, path.resolve())
            )

    resolved_sources: dict[str, Path] = {}
    for source in sources:
        alias = str(source.get("source_alias"))
        pool = str(source.get("source_pool"))
        expected_size = source.get("bytes")
        expected_checksum = source.get("sha256")
        candidates = inspected.get(pool, {}).get(expected_size, [])
        exact = [item for item in candidates if item[0] == expected_checksum]
        if not candidates:
            errors.append(f"{alias}: bytes no coincide o fuente local ausente")
            continue
        if not exact:
            errors.append(f"{alias}: sha256 no coincide con la fuente local")
            continue
        if len(exact) != 1:
            errors.append(f"{alias}: fuente local duplicada o ambigua")
            continue
        _, observed_map, inspection_error, resolved_path = exact[0]
        if inspection_error is not None:
            errors.append(f"{alias}: {inspection_error}")
        elif observed_map != source.get("map"):
            errors.append(f"{alias}: mapa no coincide con la fuente local")
        else:
            resolved_sources[alias] = resolved_path
    if errors:
        return {}, errors
    return resolved_sources, []


def validate_local_sources(
    sources: Sequence[Mapping[str, object]], roots: Mapping[str, Path]
) -> list[str]:
    _, errors = resolve_local_sources(sources, roots)
    return errors


def validate_golden_demo_corpus(
    manifest_path: Path,
    *,
    roots: Mapping[str, Path] | None = None,
    metadata_only: bool = False,
) -> dict[str, object]:
    manifest, errors = load_manifest(Path(manifest_path))
    sources: list[Mapping[str, object]] = []
    if manifest is not None:
        metadata_errors, sources = validate_manifest_metadata(manifest)
        errors.extend(metadata_errors)
    mode = "metadata_only" if metadata_only else "source_inventory"
    if not metadata_only and not errors:
        errors.extend(validate_local_sources(sources, roots or {}))
    if errors:
        status = "invalid"
    elif metadata_only:
        status = "metadata_valid"
    else:
        status = "source_inventory_valid"
    selected_count = 0
    for source in sources:
        selection = source.get("selection")
        if isinstance(selection, Mapping) and selection.get("selected") is True:
            selected_count += 1
    semantic = manifest.get("semantic_validation") if manifest is not None else None
    semantic_status = (
        semantic.get("status") if isinstance(semantic, Mapping) else "unknown"
    )
    evaluated_count = (
        semantic.get("evaluated_count", 0) if isinstance(semantic, Mapping) else 0
    )
    quarantined_count = (
        semantic.get("quarantined_count", 0) if isinstance(semantic, Mapping) else 0
    )
    return {
        "status": status,
        "validation_mode": mode,
        "gate_1_proven": False,
        "semantic_validation_status": semantic_status,
        "semantic_evaluated_count": evaluated_count,
        "semantic_quarantined_count": quarantined_count,
        "candidate_count": len(sources),
        "selected_count": selected_count,
        "errors": errors,
        "limitations": list(LIMITATIONS),
    }


def default_manifest_path() -> Path:
    return (
        Path(__file__).resolve().parent.parent
        / "testdata"
        / "golden-demos"
        / "v2"
        / "manifest.json"
    )


def _root_argument(cli_value: Path | None, pool: str) -> Path | None:
    if cli_value is not None:
        return cli_value
    value = os.environ.get(ROOT_ENV_VARS[pool])
    return Path(value) if value else None


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", nargs="?", type=Path, default=default_manifest_path())
    parser.add_argument(
        "--metadata-only",
        action="store_true",
        help="validate only secret-free metadata; never claims Gate 1",
    )
    parser.add_argument(
        "--stratai-root",
        type=Path,
        help=f"local .dem root (or ${ROOT_ENV_VARS['stratai_runtime']})",
    )
    parser.add_argument(
        "--faceit-root",
        type=Path,
        help=f"local .dem root (or ${ROOT_ENV_VARS['faceit_offline']})",
    )
    arguments = parser.parse_args()
    roots = {
        "stratai_runtime": _root_argument(
            arguments.stratai_root, "stratai_runtime"
        ),
        "faceit_offline": _root_argument(arguments.faceit_root, "faceit_offline"),
    }
    result = validate_golden_demo_corpus(
        arguments.manifest.resolve(),
        roots={key: value for key, value in roots.items() if value is not None},
        metadata_only=arguments.metadata_only,
    )
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))
    return 0 if result["status"] != "invalid" else 1


if __name__ == "__main__":
    raise SystemExit(main())
