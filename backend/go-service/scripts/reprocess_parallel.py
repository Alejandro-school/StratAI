#!/usr/bin/env python3
"""Reprocesa las demos locales sin perder sus metadatos ni agregados."""

import argparse
import concurrent.futures
import json
import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path

import requests

BACKEND_DIR = Path(__file__).resolve().parents[2]
DEMOS_DIR = BACKEND_DIR / "data" / "demos"
EXPORTS_DIR = BACKEND_DIR / "data" / "exports"
USERS_DIR = BACKEND_DIR / "data" / "users"
AGGREGATION_SCRIPT = BACKEND_DIR / "scripts" / "migrate_to_user_aggregates.py"
GO_SERVICE_URL = "http://localhost:8080"
DEFAULT_WORKERS = 2
DEFAULT_TIMEOUT_SECONDS = 600


@dataclass(frozen=True)
class ProcessResult:
    status: str
    match_id: str
    message: str


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", type=int, default=DEFAULT_WORKERS)
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_SECONDS)
    parser.add_argument("--dry-run", action="store_true", help="Lista las demos sin procesarlas.")
    parser.add_argument(
        "--skip-aggregate-rebuild",
        action="store_true",
        help="No regenera backend/data/users tras el reprocesado.",
    )
    return parser.parse_args()


def extract_match_id(demo_path: Path) -> str | None:
    prefix = "match_"
    return demo_path.stem.removeprefix(prefix) if demo_path.stem.startswith(prefix) else None


def get_export_dir(match_id: str) -> Path:
    return EXPORTS_DIR / f"match_{match_id}"


def load_json(path: Path) -> dict:
    try:
        with path.open(encoding="utf-8") as file:
            data = json.load(file)
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def get_existing_metadata(match_id: str) -> dict:
    return load_json(get_export_dir(match_id) / "metadata.json")


def get_demo_files() -> list[Path]:
    return sorted(DEMOS_DIR.glob("*.dem")) if DEMOS_DIR.exists() else []


def check_go_service() -> bool:
    try:
        return requests.get(f"{GO_SERVICE_URL}/health", timeout=3).ok
    except requests.RequestException:
        return False


def process_demo(demo_path: Path, timeout: int) -> ProcessResult:
    match_id = extract_match_id(demo_path)
    if not match_id:
        return ProcessResult("skipped", demo_path.name, "nombre de demo no válido")

    metadata = get_existing_metadata(match_id)
    started_at = time.monotonic()
    try:
        response = requests.post(
            f"{GO_SERVICE_URL}/process-demo",
            json={
                "demo_path": str(demo_path.resolve()),
                "match_id": match_id,
                "match_date": metadata.get("date", ""),
                "match_duration": metadata.get("duration_seconds", 0),
            },
            timeout=timeout,
        )
        response.raise_for_status()
    except requests.RequestException as error:
        return ProcessResult("failed", match_id, str(error))

    export_dir = get_export_dir(match_id)
    new_metadata = load_json(export_dir / "metadata.json")
    if not new_metadata:
        return ProcessResult("failed", match_id, "Go respondió correctamente, pero no generó metadata.json")
    if metadata.get("date") and new_metadata.get("date") != metadata["date"]:
        return ProcessResult("failed", match_id, "la fecha original no se preservó")
    if not (export_dir / "players_summary.json").exists():
        return ProcessResult("failed", match_id, "Go no generó players_summary.json")

    elapsed = time.monotonic() - started_at
    return ProcessResult("processed", match_id, f"{elapsed:.1f}s")


def get_affected_steam_ids(match_ids: list[str]) -> set[str]:
    steam_ids: set[str] = set()
    for match_id in match_ids:
        players = load_json(get_export_dir(match_id) / "players_summary.json").get("players", [])
        for player in players:
            steam_id = str(player.get("steam_id", "")) if isinstance(player, dict) else ""
            if steam_id.isdigit():
                steam_ids.add(steam_id)
    return steam_ids


def rebuild_aggregates(steam_ids: set[str]) -> bool:
    for steam_id in steam_ids:
        shutil.rmtree(USERS_DIR / steam_id, ignore_errors=True)
    if not steam_ids:
        return True
    result = subprocess.run([sys.executable, str(AGGREGATION_SCRIPT)], cwd=BACKEND_DIR, check=False)
    return result.returncode == 0


def print_result(index: int, total: int, result: ProcessResult) -> None:
    labels = {"processed": "OK", "skipped": "SKIP", "failed": "ERROR"}
    print(f"[{index}/{total}] {labels[result.status]} {result.match_id}: {result.message}")


def main() -> int:
    args = parse_arguments()
    if args.workers < 1 or args.timeout < 1:
        print("--workers y --timeout deben ser mayores que cero.")
        return 2

    demo_files = get_demo_files()
    if not demo_files:
        print(f"No se encontraron demos en {DEMOS_DIR}")
        return 0
    if args.dry_run:
        print(f"Se reprocesarían {len(demo_files)} demos con {args.workers} workers:")
        for demo_path in demo_files:
            print(f"- {demo_path.name}")
        return 0
    if not check_go_service():
        print("El servicio Go no responde en http://localhost:8080/health.")
        print("Inícialo desde backend/go-service con: go run main.go")
        return 1

    results: list[ProcessResult] = []
    total = len(demo_files)
    started_at = time.monotonic()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = {
            executor.submit(process_demo, demo_path, args.timeout): index
            for index, demo_path in enumerate(demo_files, start=1)
        }
        for future in concurrent.futures.as_completed(futures):
            index = futures[future]
            result = future.result()
            results.append(result)
            print_result(index, total, result)

    processed_ids = [result.match_id for result in results if result.status == "processed"]
    failed = sum(result.status == "failed" for result in results)
    skipped = sum(result.status == "skipped" for result in results)
    print(f"\nResumen: {len(processed_ids)} procesadas, {skipped} saltadas, {failed} fallidas en {time.monotonic() - started_at:.1f}s.")

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
