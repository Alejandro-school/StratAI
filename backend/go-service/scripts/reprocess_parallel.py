#!/usr/bin/env python3
"""Reprocesa las demos locales sin perder sus metadatos ni agregados."""

import argparse
import hashlib
import hmac
import json
import os
import secrets
import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

import requests

SCRIPTS_DIR = Path(__file__).resolve().parent
if str(SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPTS_DIR))

from canonical_export_validator import (
    BLOCK7_DOMAIN_FIELDS,
    REQUIRED_BLOCK7_DOMAINS,
    VALIDATOR_VERSION,
    validate_match_export,
)

BACKEND_DIR = Path(__file__).resolve().parents[2]
DEMOS_DIR = BACKEND_DIR / "data" / "demos"
EXPORTS_DIR = BACKEND_DIR / "data" / "exports"
USERS_DIR = BACKEND_DIR / "data" / "users"
AGGREGATION_SCRIPT = BACKEND_DIR / "scripts" / "migrate_to_user_aggregates.py"
GO_SERVICE_URL = "http://127.0.0.1:8080"
DEFAULT_WORKERS = 1
DEFAULT_TIMEOUT_SECONDS = 1500
DEFAULT_RETRIES = 2
EXPORT_FORMAT_VERSION = "3.8.0"
PARSER_SCHEMA_VERSION = "v16"
QUALITY_SCHEMA_VERSION = 12
ROOT_MANIFEST_PATH = Path("manifest.json")
CANONICAL_MATCH_PATH = Path("canonical/core/match.json")
CANONICAL_QUALITY_PATH = Path("canonical/diagnostics/quality_report.json")
CANONICAL_PLAYER_STATS_PATH = Path("canonical/derived/player_stats.json")
OPERATIONAL_PATHS = (
    ROOT_MANIFEST_PATH,
    CANONICAL_MATCH_PATH,
    CANONICAL_QUALITY_PATH,
    CANONICAL_PLAYER_STATS_PATH,
)
REQUIRED_QUALITY_CHECKS = {
    "replay_round_sequence",
    "economy_round_sequence",
    "tracking_round_sequence",
    "combat_round_sequence",
    "grenade_round_sequence",
    "accuracy_ranges",
    "crosshair_vector_consistency",
    "economy_round_end_reconciliation",
    "economy_round_end_native_reconciliation",
    "visibility_model",
    "player_state_contract",
    "player_state_motion_signal",
    "player_state_velocity_coverage",
    "player_state_active_weapon_coverage",
    "objective_event_contract",
    "objective_round_reconciliation",
    "objective_terminal_reconciliation",
    "objective_lifecycle",
    "objective_carrier_consistency",
    "objective_replay_projection",
    "utility_event_contract",
    "utility_throw_reconciliation",
    "utility_lifecycle",
    "utility_flash_attribution",
    "utility_damage_reconciliation",
    "utility_temporal_spatial_consistency",
    "utility_determinism",
    "utility_observation_coverage",
	"combat_contract",
	"combat_callback_accounting",
	"combat_player_stats_projection",
	"combat_replay_projection",
	"combat_native_deltas",
	"combat_determinism",
	"combat_observation_coverage",
	"engagement_event_contract",
	"engagement_atomic_provenance",
	"engagement_participant_reconciliation",
	"engagement_role_consistency",
	"engagement_temporal_consistency",
	"engagement_causal_availability",
	"engagement_trade_reconciliation",
	"engagement_stats_reconciliation",
	"engagement_determinism",
	"engagement_observation_coverage",
	"economy_team_identity",
	"economy_native_calculated_reconciliation",
	"economy_money_transition",
	"economy_purchase_provenance",
	"economy_price_table_version",
	"stats_scoreboard_reconciliation",
	"stats_utility_reconciliation",
	"clutch_attempt_reconciliation",
	"warmup_contamination",
	"metadata_provenance",
	"metadata_checksum_lineage",
	"economy_determinism",
	"stats_determinism",
	"economy_observation_coverage",
}
REQUIRED_ZERO_QUALITY_METRICS = {
    "utility_replay_projection_mismatches",
	"combat_contract_violations",
	"combat_callback_accounting_violations",
	"combat_player_stats_mismatches",
	"combat_replay_projection_mismatches",
	"combat_native_delta_mismatches",
	"combat_determinism_violations",
	"engagement_event_contract_violations",
	"engagement_atomic_provenance_violations",
	"engagement_participant_reconciliation_mismatches",
	"engagement_role_consistency_violations",
	"engagement_temporal_consistency_violations",
	"engagement_causal_availability_violations",
	"engagement_trade_reconciliation_mismatches",
	"engagement_stats_reconciliation_mismatches",
	"engagement_determinism_violations",
	"economy_team_identity_violations",
	"economy_native_calculated_reconciliation_violations",
	"economy_money_transition_violations",
	"economy_purchase_provenance_violations",
	"economy_price_table_version_violations",
	"stats_scoreboard_reconciliation_mismatches",
	"stats_utility_reconciliation_mismatches",
	"clutch_attempt_reconciliation_mismatches",
	"warmup_contamination_violations",
	"metadata_provenance_violations",
	"metadata_checksum_lineage_violations",
	"economy_determinism_violations",
	"stats_determinism_violations",
	"block7_artifact_integrity_violations",
	"block7_causal_availability_violations",
	"block7_future_leakage_violations",
	"block7_schema_compatibility_violations",
	"block7_determinism_violations",
	"block7_corpus_quality_violations",
}


@dataclass(frozen=True)
class ProcessResult:
    status: str
    match_id: str
    message: str


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", type=int, default=DEFAULT_WORKERS)
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_SECONDS)
    parser.add_argument("--retries", type=int, default=DEFAULT_RETRIES)
    parser.add_argument(
        "--expected-count",
        type=int,
        help=(
            "Exige que la selección final contenga exactamente este número de "
            "demos antes de crear el backup o procesar."
        ),
    )
    parser.add_argument(
        "--backup-dir",
        type=Path,
        help="Directorio externo a exports/ donde conservar el backup verificado.",
    )
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument(
        "--limit", type=int, help="Reprocesa como máximo este número de demos."
    )
    selection.add_argument(
        "--match-id",
        action="append",
        default=[],
        help="Reprocesa exactamente este match ID. Puede repetirse.",
    )
    parser.add_argument(
        "--quality-only",
        action="store_true",
        help="Reprocesa solo demos cuyo export actual ya contiene el informe de calidad canónico.",
    )
    parser.add_argument(
        "--dry-run", action="store_true", help="Lista las demos sin procesarlas."
    )
    parser.add_argument(
        "--skip-aggregate-rebuild",
        action="store_true",
        help="No regenera backend/data/users tras el reprocesado.",
    )
    return parser.parse_args()


def extract_match_id(demo_path: Path) -> str | None:
    prefix = "match_"
    return (
        demo_path.stem.removeprefix(prefix)
        if demo_path.stem.startswith(prefix)
        else None
    )


def get_export_dir(match_id: str) -> Path:
    return EXPORTS_DIR / f"match_{match_id}"


def load_json(path: Path) -> dict:
    try:
        with path.open(encoding="utf-8") as file:
            data = json.load(file)
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def get_existing_match(match_id: str) -> dict:
    return load_json(get_export_dir(match_id) / CANONICAL_MATCH_PATH)


def get_demo_files(quality_only: bool = False) -> list[Path]:
    demo_files = sorted(DEMOS_DIR.glob("*.dem")) if DEMOS_DIR.exists() else []
    if not quality_only:
        return demo_files
    return [
        demo_path
        for demo_path in demo_files
        if (match_id := extract_match_id(demo_path))
        and (get_export_dir(match_id) / CANONICAL_QUALITY_PATH).is_file()
    ]


def select_demo_files(demo_files: list[Path], match_ids: list[str]) -> list[Path]:
    if not match_ids:
        return demo_files

    requested = set(match_ids)
    selected = [
        demo_path
        for demo_path in demo_files
        if extract_match_id(demo_path) in requested
    ]
    found = {extract_match_id(demo_path) for demo_path in selected}
    missing = sorted(requested - found)
    if missing:
        raise ValueError(f"No existen demos para estos match IDs: {', '.join(missing)}")
    return selected


def require_expected_count(
    demo_files: list[Path], expected_count: int | None
) -> list[Path]:
    if expected_count is not None and len(demo_files) != expected_count:
        raise ValueError(
            "La selección contiene "
            f"{len(demo_files)} demos; se esperaban exactamente {expected_count}."
        )
    return demo_files


def get_demo_checksum(demo_path: Path) -> str:
    checksum = hashlib.sha256()
    with demo_path.open("rb") as demo_file:
        for chunk in iter(lambda: demo_file.read(1024 * 1024), b""):
            checksum.update(chunk)
    return checksum.hexdigest()


def internal_service_secret() -> str:
    secret = os.environ.get("INTERNAL_SERVICE_SECRET", "")
    if secret:
        return secret
    if os.environ.get("APP_ENV") == "production":
        return ""
    session_secret = os.environ.get("SESSION_SECRET_KEY", "")
    if not session_secret:
        return ""
    return hashlib.sha256(f"{session_secret}:internal-service".encode()).hexdigest()


def build_service_headers(method: str, path: str, body: bytes) -> dict[str, str]:
    secret = internal_service_secret()
    if len(secret) < 32:
        raise ValueError("INTERNAL_SERVICE_SECRET no esta configurado")
    timestamp = str(int(time.time()))
    nonce = secrets.token_hex(16)
    body_hash = hashlib.sha256(body).hexdigest()
    canonical = "\n".join(("v1", method.upper(), path, timestamp, nonce, body_hash))
    signature = hmac.new(secret.encode(), canonical.encode(), hashlib.sha256).hexdigest()
    return {
        "Content-Type": "application/json",
        "X-Service-Version": "v1",
        "X-Service-Timestamp": timestamp,
        "X-Service-Nonce": nonce,
        "X-Service-Signature": signature,
    }


def verify_demo_sidecar(demo_path: Path) -> tuple[bool, str]:
    sidecar = demo_path.with_suffix(demo_path.suffix + ".sha256")
    if not sidecar.is_file():
        return False, f"falta sidecar SHA-256: {sidecar.name}"
    try:
        expected = sidecar.read_text(encoding="ascii").strip().lower()
        actual = get_demo_checksum(demo_path)
    except (OSError, UnicodeError) as error:
        return False, f"no se pudo verificar {demo_path.name}: {error}"
    if len(expected) != 64 or any(character not in "0123456789abcdef" for character in expected):
        return False, f"sidecar SHA-256 invalido: {sidecar.name}"
    if actual != expected:
        return False, f"checksum de demo no coincide: {demo_path.name}"
    return True, actual


def tree_hashes(root: Path) -> dict[str, str]:
    return {
        path.relative_to(root).as_posix(): get_demo_checksum(path)
        for path in sorted(candidate for candidate in root.rglob("*") if candidate.is_file())
    }


def staging_is_clean() -> tuple[bool, str]:
    temp_dir = EXPORTS_DIR / ".tmp"
    if temp_dir.exists() and any(temp_dir.iterdir()):
        return False, f"staging no esta limpio: {temp_dir}"
    rollback_paths = sorted(EXPORTS_DIR.glob("match_*.rollback.*"))
    if rollback_paths:
        return False, f"existen backups de rollback pendientes: {rollback_paths[0].name}"
    return True, "staging limpio"


def create_verified_backup(demo_files: list[Path], backup_dir: Path | None) -> Path:
    if backup_dir is None:
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        backup_dir = BACKEND_DIR / "data" / "export_backups" / f"block7_reprocess_{timestamp}"
    backup_dir = backup_dir.resolve()
    exports_resolved = EXPORTS_DIR.resolve()
    if backup_dir == exports_resolved or exports_resolved in backup_dir.parents:
        raise ValueError("el backup debe estar fuera de backend/data/exports")
    backup_dir.mkdir(parents=True, exist_ok=False)
    manifest: dict[str, object] = {
        "schema_id": "stratai.reprocess_backup@1",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "exports": {},
        "demo_checksums": {},
    }
    for demo_path in demo_files:
        match_id = extract_match_id(demo_path)
        if not match_id:
            continue
        manifest["demo_checksums"][match_id] = get_demo_checksum(demo_path)
        source = get_export_dir(match_id)
        if not source.is_dir():
            continue
        source_hashes = tree_hashes(source)
        destination = backup_dir / source.name
        shutil.copytree(source, destination, copy_function=shutil.copy2)
        copied_hashes = tree_hashes(destination)
        if copied_hashes != source_hashes:
            raise RuntimeError(f"el backup SHA-256 no coincide para {match_id}")
        manifest["exports"][match_id] = source_hashes
    manifest_path = backup_dir / "backup_manifest.json"
    manifest_path.write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    return backup_dir


def validate_export(match_id: str, demo_checksum: str) -> str | None:
    export_dir = get_export_dir(match_id)
    missing = [
        path.as_posix()
        for path in OPERATIONAL_PATHS
        if not (export_dir / path).is_file()
    ]
    if missing:
        return f"faltan artefactos requeridos: {', '.join(missing)}"

    manifest = load_json(export_dir / ROOT_MANIFEST_PATH)
    if manifest.get("match_id") != match_id:
        return "manifest.json contiene un match_id inesperado"
    if manifest.get("checksum") != demo_checksum:
        return "manifest.json no corresponde al checksum de la demo"
    if manifest.get("parser_schema_version") != PARSER_SCHEMA_VERSION:
        return f"manifest.json no usa el schema esperado {PARSER_SCHEMA_VERSION}"
    if manifest.get("export_format_version") != EXPORT_FORMAT_VERSION:
        return f"manifest.json no usa el formato esperado {EXPORT_FORMAT_VERSION}"

    quality_document = load_json(export_dir / CANONICAL_QUALITY_PATH)
    if quality_document.get("match_id") != match_id:
        return (
            "canonical/diagnostics/quality_report.json contiene un match_id inesperado"
        )
    quality = quality_document.get("report")
    if not isinstance(quality, dict):
        return "canonical/diagnostics/quality_report.json no contiene report"
    if quality.get("schema_version") != QUALITY_SCHEMA_VERSION:
        return f"canonical/diagnostics/quality_report.json no usa el schema esperado {QUALITY_SCHEMA_VERSION}"
    if quality.get("parse_completed") is not True:
        return "canonical/diagnostics/quality_report.json indica que el parseo no se completó"
    if quality.get("usable_for_training") is not True or quality.get("status") not in {
        "pass",
        "warning",
    }:
        return "canonical/diagnostics/quality_report.json marca el export como no utilizable"
    invalid_metrics = sorted(
        metric for metric in REQUIRED_ZERO_QUALITY_METRICS if quality.get(metric) != 0
    )
    if invalid_metrics:
        return (
            "canonical/diagnostics/quality_report.json no supera metricas hard: "
            + ", ".join(invalid_metrics)
        )

    checks = quality.get("checks")
    if not isinstance(checks, list) or not all(
        isinstance(check, dict) for check in checks
    ):
        return "canonical/diagnostics/quality_report.json no contiene una lista de checks válida"
    check_names = {str(check.get("name", "")) for check in checks}
    missing_checks = sorted(REQUIRED_QUALITY_CHECKS - check_names)
    if missing_checks:
        return f"canonical/diagnostics/quality_report.json no contiene checks requeridos: {', '.join(missing_checks)}"
    allowed_check_statuses = {"pass", "warning", "not_available"}
    if any(check.get("status") not in allowed_check_statuses for check in checks):
        return "canonical/diagnostics/quality_report.json contiene checks fallidos o con estado desconocido"
    domains = quality.get("domains")
    if not isinstance(domains, list) or not all(
        isinstance(domain, dict) for domain in domains
    ):
        return "canonical/diagnostics/quality_report.json no contiene domains validos"
    domain_names = {domain.get("name") for domain in domains}
    if domain_names != REQUIRED_BLOCK7_DOMAINS or len(domains) != len(
        REQUIRED_BLOCK7_DOMAINS
    ):
        return "canonical/diagnostics/quality_report.json no contiene los 20 domains Bloque 7"
    for domain in domains:
        if BLOCK7_DOMAIN_FIELDS - set(domain):
            return f"quality domain {domain.get('name')} no contiene todos los campos"
        if domain.get("status") == "fail" or (
            domain.get("severity") == "hard" and domain.get("status") != "pass"
        ):
            return f"quality domain {domain.get('name')} no supera el gate hard"

    match = load_json(export_dir / CANONICAL_MATCH_PATH)
    if match.get("match_id") != match_id:
        return "canonical/core/match.json contiene un match_id inesperado"
    if match.get("round_count") != quality.get("expected_rounds"):
        return (
            "canonical/core/match.json y quality_report.json discrepan en round_count"
        )
    player_stats = load_json(export_dir / CANONICAL_PLAYER_STATS_PATH)
    if player_stats.get("match_id") != match_id or not isinstance(
        player_stats.get("players"), list
    ):
        return (
            "canonical/derived/player_stats.json no cumple el contrato esperado"
        )
    canonical_errors = validate_match_export(export_dir, match_id, demo_checksum)
    if canonical_errors:
        return f"bundle canónico inválido: {'; '.join(canonical_errors)}"
    return None


def duration_ms_to_seconds(value: object) -> int:
    try:
        return max(0, int(float(value) / 1000))
    except (TypeError, ValueError, OverflowError):
        return 0


def get_go_service_health() -> tuple[bool, str]:
    try:
        response = requests.get(f"{GO_SERVICE_URL}/health", timeout=3)
        payload = response.json()
    except (requests.RequestException, ValueError) as error:
        return False, f"health no disponible: {error}"
    if not isinstance(payload, dict):
        return False, "health incompatible: la respuesta JSON no es un objeto"
    expected = {
        "status": "ok",
        "redis": "available",
        "parser_schema_version": PARSER_SCHEMA_VERSION,
        "export_format_version": EXPORT_FORMAT_VERSION,
        "validator_version": VALIDATOR_VERSION,
    }
    mismatches = {
        key: (payload.get(key), value)
        for key, value in expected.items()
        if payload.get(key) != value
    }
    if not response.ok or mismatches:
        return False, f"health incompatible: {mismatches or response.status_code}"
    return True, "servicio Go y Redis disponibles"


def check_go_service() -> bool:
    return get_go_service_health()[0]


def process_demo(demo_path: Path, timeout: int, retries: int = DEFAULT_RETRIES) -> ProcessResult:
    match_id = extract_match_id(demo_path)
    if not match_id:
        return ProcessResult("skipped", demo_path.name, "nombre de demo no válido")

    existing_match = get_existing_match(match_id)
    try:
        demo_checksum = get_demo_checksum(demo_path)
    except OSError as error:
        return ProcessResult(
            "failed", match_id, f"no se pudo calcular el checksum de la demo: {error}"
        )
    started_at = time.monotonic()
    payload = {
        "demo_path": str(demo_path.resolve()),
        "match_id": match_id,
        "match_date": existing_match.get("played_at", ""),
        "match_duration": duration_ms_to_seconds(existing_match.get("duration_ms", 0)),
        "checksum": demo_checksum,
        "parser_schema_version": PARSER_SCHEMA_VERSION,
        "source": "demo",
        "source_endpoint": f"demo:{match_id}",
        "source_version": PARSER_SCHEMA_VERSION,
        "force": True,
    }
    body = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    if len(internal_service_secret()) < 32:
        return ProcessResult("failed", match_id, "INTERNAL_SERVICE_SECRET no esta configurado")
    response = None
    request_error: requests.RequestException | None = None
    for attempt in range(retries + 1):
        try:
            headers = build_service_headers("POST", "/process-demo", body)
            response = requests.post(
                f"{GO_SERVICE_URL}/process-demo",
                data=body,
                headers=headers,
                timeout=timeout,
            )
            response.raise_for_status()
            request_error = None
            break
        except requests.RequestException as error:
            request_error = error
            status_code = error.response.status_code if error.response is not None else None
            if status_code is not None and 400 <= status_code < 500:
                break
            if attempt < retries:
                time.sleep(0.25 * (2**attempt))
    if request_error is not None:
        detail = (
            request_error.response.text.strip()
            if request_error.response is not None
            else ""
        )
        message = f"{request_error}: {detail}" if detail else str(request_error)
        return ProcessResult("failed", match_id, message)

    validation_error = validate_export(match_id, demo_checksum)
    if validation_error:
        return ProcessResult("failed", match_id, validation_error)

    export_dir = get_export_dir(match_id)
    new_match = load_json(export_dir / CANONICAL_MATCH_PATH)
    if (
        existing_match.get("played_at")
        and new_match.get("played_at") != existing_match["played_at"]
    ):
        return ProcessResult("failed", match_id, "la fecha original no se preservó")

    elapsed = time.monotonic() - started_at
    return ProcessResult("processed", match_id, f"{elapsed:.1f}s")


def get_affected_steam_ids(match_ids: list[str]) -> set[str]:
    steam_ids: set[str] = set()
    for match_id in match_ids:
        players = load_json(get_export_dir(match_id) / CANONICAL_PLAYER_STATS_PATH).get(
            "players", []
        )
        for player in players:
            player_id = (
                str(player.get("player_id", "")) if isinstance(player, dict) else ""
            )
            steam_id = (
                player_id.removeprefix("steam:")
                if player_id.startswith("steam:")
                else ""
            )
            if steam_id.isdigit():
                steam_ids.add(steam_id)
    return steam_ids


def rebuild_aggregates(steam_ids: set[str]) -> bool:
    for steam_id in steam_ids:
        shutil.rmtree(USERS_DIR / steam_id, ignore_errors=True)
    if not steam_ids:
        return True
    result = subprocess.run(
        [sys.executable, str(AGGREGATION_SCRIPT)], cwd=BACKEND_DIR, check=False
    )
    return result.returncode == 0


def print_result(index: int, total: int, result: ProcessResult) -> None:
    labels = {"processed": "OK", "skipped": "SKIP", "failed": "ERROR"}
    print(
        f"[{index}/{total}] {labels[result.status]} {result.match_id}: {result.message}"
    )


def main() -> int:
    args = parse_arguments()
    if (
        args.workers < 1
        or args.timeout < 1
        or args.retries < 0
        or (args.limit is not None and args.limit < 1)
        or (args.expected_count is not None and args.expected_count < 1)
    ):
        print(
            "--workers, --timeout, --limit y --expected-count deben ser positivos; "
            "--retries no negativo."
        )
        return 2
    if args.workers != 1:
        print("Bloque 7 requiere --workers 1 para detener el lote en el primer fallo.")
        return 2
    if not args.match_id:
        print("Bloque 7 requiere al menos un --match-id explicito.")
        return 2
    if not args.skip_aggregate_rebuild:
        print("Bloque 7 requiere --skip-aggregate-rebuild; no se reconstruyen agregados desde subconjuntos.")
        return 2

    demo_files = get_demo_files(args.quality_only)
    try:
        demo_files = select_demo_files(demo_files, args.match_id)
    except ValueError as error:
        print(error)
        return 2
    if args.limit is not None:
        demo_files = demo_files[: args.limit]
    try:
        demo_files = require_expected_count(demo_files, args.expected_count)
    except ValueError as error:
        print(error)
        return 2
    if not demo_files:
        print(f"No se encontraron demos en {DEMOS_DIR}")
        return 0
    verified_checksums: dict[str, str] = {}
    for demo_path in demo_files:
        valid, detail = verify_demo_sidecar(demo_path)
        if not valid:
            print(f"ERROR preflight: {detail}")
            return 1
        verified_checksums[demo_path.name] = detail
    staging_clean, staging_detail = staging_is_clean()
    if not staging_clean:
        print(f"ERROR preflight: {staging_detail}")
        return 1
    auth_ready = len(internal_service_secret()) >= 32
    if args.dry_run:
        service_ok, service_detail = get_go_service_health()
        print(
            f"Dry-run preflight: workers=1 timeout={args.timeout} retries={args.retries}; "
            f"lote_esperado={args.expected_count or 'sin_gate'}; {staging_detail}; "
            f"servicio={service_detail}; firma_interna={'ok' if auth_ready else 'missing'}"
        )
        print(
            f"Backup previsto: {args.backup_dir or 'backend/data/export_backups/block7_reprocess_<UTC>'}"
        )
        print(f"Se reprocesarían {len(demo_files)} demos con {args.workers} workers:")
        for demo_path in demo_files:
            print(f"- {demo_path.name} sha256={verified_checksums[demo_path.name]}")
        return 0 if service_ok and auth_ready else 1
    if not auth_ready:
        print("ERROR preflight: INTERNAL_SERVICE_SECRET no esta configurado")
        return 1
    if not check_go_service():
        print("El servicio Go no responde en http://localhost:8080/health.")
        print("Inícialo desde backend/go-service con: go run main.go")
        return 1

    try:
        backup_dir = create_verified_backup(demo_files, args.backup_dir)
    except (OSError, RuntimeError, ValueError) as error:
        print(f"ERROR preflight: no se pudo crear/verificar el backup: {error}")
        return 1
    print(f"Backup SHA-256 verificado: {backup_dir}")

    results: list[ProcessResult] = []
    total = len(demo_files)
    started_at = time.monotonic()
    for index, demo_path in enumerate(demo_files, start=1):
        result = process_demo(demo_path, args.timeout, args.retries)
        results.append(result)
        print_result(index, total, result)
        if result.status == "failed":
            print(f"Lote detenido; backup conservado en {backup_dir}")
            break

    processed_ids = [
        result.match_id for result in results if result.status == "processed"
    ]
    failed = sum(result.status == "failed" for result in results)
    skipped = sum(result.status == "skipped" for result in results)
    print(
        f"\nResumen: {len(processed_ids)} procesadas, {skipped} saltadas, {failed} fallidas en {time.monotonic() - started_at:.1f}s."
    )

    if processed_ids and not args.skip_aggregate_rebuild:
        steam_ids = get_affected_steam_ids(processed_ids)
        print(f"Regenerando agregados de {len(steam_ids)} jugadores afectados...")
        if not rebuild_aggregates(steam_ids):
            print("El reprocesado terminó, pero falló la regeneración de agregados.")
            return 1
    return 1 if failed else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("Proceso interrumpido por el usuario.")
        raise SystemExit(130)
