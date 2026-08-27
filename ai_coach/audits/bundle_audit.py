"""Audita bundles canónicos sin modificarlos ni descomprimirlos en disco.

La herramienta recorre un bundle cada vez. Los JSONL, incluido el gzip táctico,
se leen línea a línea. Los JSON contenedores tienen un límite de bytes explícito.
La salida no incluye una hora de ejecución, por lo que dos ejecuciones sobre los
mismos manifests producen el mismo informe.
"""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import sys
from collections import Counter
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, BinaryIO, Iterable, Iterator, Mapping, Sequence


SCHEMA_ID = "stratai.ai_coach_bundle_audit@1"
DEFAULT_MAX_JSON_BYTES = 64 * 1024 * 1024
DEFAULT_MAX_LINE_BYTES = 16 * 1024 * 1024
DEFAULT_TACTICAL_SAMPLE_ROWS = 4096
MAX_ENUM_VALUES = 40
MAX_EXAMPLES = 2


@dataclass(frozen=True)
class ArtifactSpec:
    name: str
    relative_pattern: str
    encoding: str
    collection: tuple[str, ...] = ()


ARTIFACT_SPECS: tuple[ArtifactSpec, ...] = (
    ArtifactSpec("root_manifest", "manifest.json", "json"),
    ArtifactSpec("canonical_manifest", "canonical/manifest.json", "json"),
    ArtifactSpec("match", "canonical/core/match.json", "json"),
    ArtifactSpec("participants", "canonical/core/participants.json", "json", ("players",)),
    ArtifactSpec("rounds", "canonical/core/rounds.json", "json", ("rounds",)),
    ArtifactSpec("combat_events", "canonical/events/combat_events.jsonl", "jsonl"),
    ArtifactSpec("utility_events", "canonical/events/utility_events.jsonl", "jsonl"),
    ArtifactSpec("objective_events", "canonical/events/objective_events.jsonl", "jsonl"),
    ArtifactSpec("player_states", "canonical/states/player_states/round_*.jsonl", "jsonl_glob"),
    ArtifactSpec("tactical_sampling", "canonical/states/tactical/sampling.json", "json"),
    ArtifactSpec("tactical_observations", "canonical/states/tactical/observed.jsonl.gz", "gzip_jsonl"),
    ArtifactSpec("tactical_oracle", "canonical/states/tactical/oracle.jsonl", "jsonl"),
    ArtifactSpec("tactical_gaps", "canonical/states/tactical/gaps.jsonl", "jsonl"),
    ArtifactSpec("decisions", "canonical/causal/decisions.jsonl", "jsonl"),
    ArtifactSpec("decision_features", "canonical/causal/decision_features.jsonl", "jsonl"),
    ArtifactSpec("oracle_context", "canonical/causal/oracle_context.jsonl", "jsonl"),
    ArtifactSpec("outcomes", "canonical/causal/outcomes.jsonl", "jsonl"),
    ArtifactSpec("quality_masks", "canonical/causal/quality_masks.jsonl", "jsonl"),
    ArtifactSpec("engagements", "canonical/derived/engagements.json", "json", ("engagements",)),
    ArtifactSpec("trade_candidates", "canonical/derived/trades.json", "json", ("trade_candidates",)),
    ArtifactSpec("trade_completions", "canonical/derived/trades.json", "json", ("trade_completions",)),
    ArtifactSpec("economy_players", "canonical/derived/economy_players.json", "json", ("players",)),
    ArtifactSpec("economy_rounds", "canonical/derived/economy_rounds.json", "json", ("rounds",)),
    ArtifactSpec("clutch_events", "canonical/derived/clutch_events.json", "json", ("clutch_events",)),
    ArtifactSpec("player_stats", "canonical/derived/player_stats.json", "json", ("players",)),
    ArtifactSpec("quality_report", "canonical/diagnostics/quality_report.json", "json", ("report",)),
    ArtifactSpec("replay_index", "canonical/presentation/replay/index.json", "json"),
)


NESTED_OBJECT_ARRAY_SUFFIXES = {
    "affected_players",
    "clutch_events",
    "connections",
    "exchanges",
    "horizons",
    "participants",
    "rewards",
    "teams",
    "transactions",
}


class AuditError(RuntimeError):
    """Error de datos con contexto suficiente para encontrar el bundle corrupto."""


def _is_identifier_path(path: str) -> bool:
    leaf = path.rsplit(".", 1)[-1].removesuffix("[]").lower()
    return (
        leaf == "display_name"
        or leaf.endswith("_id")
        or leaf.endswith("_ids")
        or leaf.endswith("_ref")
        or leaf.endswith("_refs")
        or leaf in {"match_id", "steam_id", "player_id", "decision_id", "event_id", "state_id"}
    )


def _json_scalar(value: Any) -> str | int | float | bool | None:
    if value is None or isinstance(value, (str, int, float, bool)):
        return value
    return None


@dataclass
class FieldMetric:
    present: int = 0
    nulls: int = 0
    type_counts: Counter[str] = field(default_factory=Counter)
    value_counts: Counter[str] = field(default_factory=Counter)
    value_counts_truncated: bool = False
    minimum: float | None = None
    maximum: float | None = None
    true_count: int = 0
    false_count: int = 0
    examples: list[dict[str, Any]] = field(default_factory=list)
    bundles: set[str] = field(default_factory=set)

    def observe(self, value: Any, bundle_id: str, path: str) -> None:
        self.present += 1
        self.bundles.add(bundle_id)
        if value is None:
            self.nulls += 1
            self.type_counts["null"] += 1
            return
        type_name = "bool" if isinstance(value, bool) else type(value).__name__
        self.type_counts[type_name] += 1
        if isinstance(value, bool):
            if value:
                self.true_count += 1
            else:
                self.false_count += 1
        elif isinstance(value, (int, float)) and math.isfinite(float(value)):
            numeric = float(value)
            self.minimum = numeric if self.minimum is None else min(self.minimum, numeric)
            self.maximum = numeric if self.maximum is None else max(self.maximum, numeric)

        scalar = _json_scalar(value)
        if not _is_identifier_path(path) and scalar is not None:
            if len(self.value_counts) < MAX_ENUM_VALUES or str(scalar) in self.value_counts:
                self.value_counts[str(scalar)] += 1
            else:
                self.value_counts_truncated = True
            if len(self.examples) < MAX_EXAMPLES:
                self.examples.append({"bundle": bundle_id, "value": scalar})


@dataclass
class ArtifactMetric:
    bundles_present: int = 0
    files: int = 0
    records: int = 0
    records_scanned: int = 0
    bytes: int = 0


@dataclass
class AuditState:
    expected_bundles: int
    fields: dict[str, FieldMetric] = field(default_factory=dict)
    parent_counts: Counter[str] = field(default_factory=Counter)
    artifacts: dict[str, ArtifactMetric] = field(default_factory=dict)
    manifest_artifact_presence: Counter[str] = field(default_factory=Counter)
    manifest_artifact_records: Counter[str] = field(default_factory=Counter)
    format_versions: Counter[str] = field(default_factory=Counter)
    parser_versions: Counter[str] = field(default_factory=Counter)
    bundle_ids: list[str] = field(default_factory=list)
    canonical_manifest_hashes: list[tuple[str, str]] = field(default_factory=list)

    def field(self, path: str) -> FieldMetric:
        return self.fields.setdefault(path, FieldMetric())


def _load_json_bounded(path: Path, max_bytes: int) -> Any:
    try:
        size = path.stat().st_size
    except OSError as exc:
        raise AuditError(f"no se puede medir {path}: {exc}") from exc
    if size > max_bytes:
        raise AuditError(f"JSON supera el límite de {max_bytes} bytes: {path} ({size} bytes)")
    try:
        with path.open("r", encoding="utf-8") as handle:
            return json.load(handle)
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise AuditError(f"JSON corrupto o ilegible: {path}: {exc}") from exc


def _prefix_sample_indices(total_rows: int, sample_rows: int) -> set[int] | None:
    if sample_rows <= 0 or total_rows <= sample_rows:
        return None
    return set(range(sample_rows))


def _binary_lines(
    handle: BinaryIO,
    path: Path,
    max_line_bytes: int,
    *,
    selected_indices: set[int] | None = None,
    expected_rows: int | None = None,
) -> Iterator[Mapping[str, Any]]:
    line_number = 0
    row_index = 0
    last_selected = max(selected_indices) if selected_indices else None
    while True:
        if last_selected is not None and row_index > last_selected:
            break
        raw = handle.readline(max_line_bytes + 1)
        if not raw:
            break
        line_number += 1
        if len(raw) > max_line_bytes and not raw.endswith(b"\n"):
            raise AuditError(f"línea {line_number} supera {max_line_bytes} bytes: {path}")
        if not raw.strip():
            continue
        selected = selected_indices is None or row_index in selected_indices
        row_index += 1
        if not selected:
            continue
        try:
            value = json.loads(raw)
        except (UnicodeError, json.JSONDecodeError) as exc:
            raise AuditError(f"JSONL corrupto en {path}, línea {line_number}: {exc}") from exc
        if not isinstance(value, Mapping):
            raise AuditError(f"se esperaba un objeto en {path}, línea {line_number}")
        yield value
    if selected_indices is None and expected_rows is not None and row_index != expected_rows:
        raise AuditError(
            f"conteo táctico distinto del manifest en {path}: "
            f"esperado={expected_rows}, real={row_index}"
        )


def _iter_jsonl(
    path: Path,
    gzip_encoded: bool,
    max_line_bytes: int,
    *,
    sample_rows: int = 0,
    expected_rows: int | None = None,
) -> Iterator[Mapping[str, Any]]:
    selected_indices = (
        _prefix_sample_indices(expected_rows, sample_rows)
        if expected_rows is not None
        else None
    )
    try:
        if gzip_encoded:
            with gzip.open(path, "rb") as handle:
                yield from _binary_lines(
                    handle,
                    path,
                    max_line_bytes,
                    selected_indices=selected_indices,
                    expected_rows=expected_rows,
                )
        else:
            with path.open("rb") as handle:
                yield from _binary_lines(
                    handle,
                    path,
                    max_line_bytes,
                    selected_indices=selected_indices,
                    expected_rows=expected_rows,
                )
    except AuditError:
        raise
    except (OSError, EOFError, gzip.BadGzipFile) as exc:
        raise AuditError(f"stream corrupto o ilegible: {path}: {exc}") from exc


def _resolve_collection(payload: Any, collection: Sequence[str], path: Path) -> Iterable[Mapping[str, Any]]:
    value = payload
    for key in collection:
        if not isinstance(value, Mapping) or key not in value:
            raise AuditError(f"falta la colección {'.'.join(collection)} en {path}")
        value = value[key]
    if not collection:
        if not isinstance(value, Mapping):
            raise AuditError(f"se esperaba un objeto JSON en {path}")
        return (value,)
    if isinstance(value, Mapping):
        return (value,)
    if not isinstance(value, list) or any(not isinstance(item, Mapping) for item in value):
        raise AuditError(f"la colección {'.'.join(collection)} no es una lista de objetos en {path}")
    return value


def _scan_mapping(
    state: AuditState,
    value: Mapping[str, Any],
    prefix: str,
    bundle_id: str,
    depth: int = 0,
) -> None:
    state.parent_counts[prefix] += 1
    for key in sorted(value):
        child = value[key]
        path = f"{prefix}.{key}"
        state.field(path).observe(child, bundle_id, path)
        if isinstance(child, Mapping) and depth < 8:
            _scan_mapping(state, child, path, bundle_id, depth + 1)
        elif isinstance(child, list):
            state.field(f"{path}.__length").observe(len(child), bundle_id, f"{path}.__length")
            if child and all(isinstance(item, Mapping) for item in child):
                if key in NESTED_OBJECT_ARRAY_SUFFIXES and depth < 6:
                    for item in child:
                        _scan_mapping(state, item, f"{path}[]", bundle_id, depth + 1)
            elif child and all(_json_scalar(item) is not None for item in child):
                scalar_path = f"{path}[]"
                state.parent_counts[scalar_path] += len(child)
                for item in child:
                    state.field(scalar_path).observe(item, bundle_id, scalar_path)


def _artifact_paths(bundle: Path, spec: ArtifactSpec) -> list[Path]:
    if "*" in spec.relative_pattern:
        return sorted(bundle.glob(spec.relative_pattern))
    path = bundle / spec.relative_pattern
    return [path] if path.is_file() else []


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _inspect_manifest(state: AuditState, bundle: Path, bundle_id: str, max_json_bytes: int) -> None:
    root = _load_json_bounded(bundle / "manifest.json", max_json_bytes)
    canonical_path = bundle / "canonical" / "manifest.json"
    canonical = _load_json_bounded(canonical_path, max_json_bytes)
    if not isinstance(root, Mapping) or not isinstance(canonical, Mapping):
        raise AuditError(f"manifest no es un objeto: {bundle}")
    format_version = str(root.get("format_version") or root.get("export_format_version") or canonical.get("format_version") or "unknown")
    parser_version = str(root.get("parser_version") or canonical.get("parser_version") or "unknown")
    lineage = canonical.get("lineage")
    if isinstance(lineage, Mapping):
        format_version = str(lineage.get("export_format_version") or lineage.get("export_version") or format_version)
        parser_version = str(lineage.get("parser_version") or parser_version)
    state.format_versions[format_version] += 1
    state.parser_versions[parser_version] += 1
    artifacts = canonical.get("artifacts")
    if not isinstance(artifacts, list):
        raise AuditError(f"canonical/manifest.json no contiene artifacts[]: {bundle}")
    seen_types: set[str] = set()
    for artifact in artifacts:
        if not isinstance(artifact, Mapping):
            raise AuditError(f"descriptor de artefacto inválido: {bundle}")
        artifact_type = str(artifact.get("artifact_type") or artifact.get("type") or "unknown")
        if artifact_type not in seen_types:
            state.manifest_artifact_presence[artifact_type] += 1
            seen_types.add(artifact_type)
        count = artifact.get("record_count")
        if isinstance(count, int):
            state.manifest_artifact_records[artifact_type] += count
    state.canonical_manifest_hashes.append((bundle_id, _sha256(canonical_path)))


def audit_bundle(
    state: AuditState,
    bundle: Path,
    max_json_bytes: int,
    max_line_bytes: int,
    tactical_sample_rows: int,
) -> None:
    bundle_id = bundle.name.removeprefix("match_")
    state.bundle_ids.append(bundle_id)
    _inspect_manifest(state, bundle, bundle_id, max_json_bytes)
    sampling = _load_json_bounded(
        bundle / "canonical" / "states" / "tactical" / "sampling.json",
        max_json_bytes,
    )
    if not isinstance(sampling, Mapping):
        raise AuditError(f"sampling táctico inválido: {bundle}")
    tactical_counts = {
        "tactical_observations": int(sampling.get("physical_row_count") or 0),
        "tactical_oracle": int(sampling.get("oracle_row_count") or 0),
    }
    for spec in ARTIFACT_SPECS:
        paths = _artifact_paths(bundle, spec)
        metric = state.artifacts.setdefault(spec.name, ArtifactMetric())
        if not paths:
            continue
        metric.bundles_present += 1
        metric.files += len(paths)
        seen_records = 0
        for path in paths:
            metric.bytes += path.stat().st_size
            if spec.encoding == "json":
                payload = _load_json_bounded(path, max_json_bytes)
                records = _resolve_collection(payload, spec.collection, path)
            elif spec.encoding in {"jsonl", "jsonl_glob", "gzip_jsonl"}:
                expected_rows = tactical_counts.get(spec.name)
                records = _iter_jsonl(
                    path,
                    spec.encoding == "gzip_jsonl",
                    max_line_bytes,
                    sample_rows=tactical_sample_rows if expected_rows is not None else 0,
                    expected_rows=expected_rows,
                )
            else:
                raise AuditError(f"encoding de auditoría desconocido: {spec.encoding}")
            for record in records:
                seen_records += 1
                _scan_mapping(state, record, spec.name, bundle_id)
        metric.records += tactical_counts.get(spec.name, seen_records)
        metric.records_scanned += seen_records


def _percentage(numerator: int, denominator: int) -> float:
    return round(100.0 * numerator / denominator, 6) if denominator else 0.0


def _clean_number(value: float | None) -> int | float | None:
    if value is None:
        return None
    return int(value) if value.is_integer() else value


def _field_payload(path: str, metric: FieldMetric, state: AuditState) -> dict[str, Any]:
    parent = path.rsplit(".", 1)[0]
    denominator = state.parent_counts[parent]
    top_values = sorted(metric.value_counts.items(), key=lambda item: (-item[1], item[0]))[:12]
    return {
        "path": path,
        "parent_observations": denominator,
        "present": metric.present,
        "presence_pct": _percentage(metric.present, denominator),
        "nulls": metric.nulls,
        "null_pct_of_present": _percentage(metric.nulls, metric.present),
        "bundles_present": len(metric.bundles),
        "bundle_presence_pct": _percentage(len(metric.bundles), state.expected_bundles),
        "types": dict(sorted(metric.type_counts.items())),
        "minimum": _clean_number(metric.minimum),
        "maximum": _clean_number(metric.maximum),
        "true_count": metric.true_count,
        "false_count": metric.false_count,
        "top_values": [{"value": value, "count": count} for value, count in top_values],
        "value_counts_truncated": metric.value_counts_truncated,
        "examples": metric.examples,
    }


def build_report(state: AuditState, bundles_root: Path) -> dict[str, Any]:
    fingerprint = hashlib.sha256()
    for bundle_id, manifest_hash in sorted(state.canonical_manifest_hashes):
        fingerprint.update(f"{bundle_id}\0{manifest_hash}\n".encode("utf-8"))
    return {
        "schema_id": SCHEMA_ID,
        "input": {
            "bundles_root_name": bundles_root.name,
            "bundle_count": len(state.bundle_ids),
            "bundle_ids": sorted(state.bundle_ids),
            "canonical_manifest_set_sha256": fingerprint.hexdigest(),
            "format_versions": dict(sorted(state.format_versions.items())),
            "parser_versions": dict(sorted(state.parser_versions.items())),
        },
        "artifacts": {
            name: {
                "bundles_present": metric.bundles_present,
                "bundle_presence_pct": _percentage(metric.bundles_present, state.expected_bundles),
                "files": metric.files,
                "records": metric.records,
                "records_scanned": metric.records_scanned,
                "bytes": metric.bytes,
            }
            for name, metric in sorted(state.artifacts.items())
        },
        "manifest_artifacts": {
            name: {
                "bundles_present": state.manifest_artifact_presence[name],
                "records_declared": state.manifest_artifact_records[name],
            }
            for name in sorted(state.manifest_artifact_presence)
        },
        "fields": [_field_payload(path, state.fields[path], state) for path in sorted(state.fields)],
        "method": {
            "bundle_order": "lexicographic",
            "memory_policy": "one bundle; JSONL streamed; JSON files bounded",
            "gzip_policy": "streamed in memory; no persistent decompression",
            "tactical_sampling": "deterministic prefix sample per bundle and tactical partition",
            "identifier_policy": "join-only identifiers counted but values omitted from examples/frequencies",
            "nested_array_policy": sorted(NESTED_OBJECT_ARRAY_SUFFIXES),
        },
    }


def audit_bundles(
    bundles_root: Path,
    expected_bundles: int,
    max_json_bytes: int = DEFAULT_MAX_JSON_BYTES,
    max_line_bytes: int = DEFAULT_MAX_LINE_BYTES,
    tactical_sample_rows: int = DEFAULT_TACTICAL_SAMPLE_ROWS,
    progress: bool = False,
) -> dict[str, Any]:
    if not bundles_root.is_dir():
        raise AuditError(f"no existe el directorio de bundles: {bundles_root}")
    bundles = sorted(path for path in bundles_root.iterdir() if path.is_dir() and not path.name.startswith("."))
    if len(bundles) != expected_bundles:
        raise AuditError(f"se esperaban {expected_bundles} bundles y se encontraron {len(bundles)}")
    state = AuditState(expected_bundles=expected_bundles)
    for index, bundle in enumerate(bundles, start=1):
        audit_bundle(state, bundle, max_json_bytes, max_line_bytes, tactical_sample_rows)
        if progress:
            print(f"audit_progress {index}/{len(bundles)} {bundle.name}", file=sys.stderr, flush=True)
    return build_report(state, bundles_root)


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("bundles_root", type=Path)
    parser.add_argument("--expected-bundles", type=int, default=44)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--max-json-bytes", type=int, default=DEFAULT_MAX_JSON_BYTES)
    parser.add_argument("--max-line-bytes", type=int, default=DEFAULT_MAX_LINE_BYTES)
    parser.add_argument(
        "--tactical-sample-rows",
        type=int,
        default=DEFAULT_TACTICAL_SAMPLE_ROWS,
        help="filas iniciales por bundle y por partición táctica; 0 audita todas",
    )
    parser.add_argument("--progress", action="store_true")
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv)
    try:
        report = audit_bundles(
            args.bundles_root.resolve(),
            expected_bundles=args.expected_bundles,
            max_json_bytes=args.max_json_bytes,
            max_line_bytes=args.max_line_bytes,
            tactical_sample_rows=args.tactical_sample_rows,
            progress=args.progress,
        )
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(
            json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
    except AuditError as exc:
        print(f"bundle_audit_failed: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
