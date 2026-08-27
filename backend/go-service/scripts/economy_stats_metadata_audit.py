#!/usr/bin/env python3
"""Auditoría semántica reproducible del Bloque 6 sobre un export canónico."""

import argparse
import hashlib
import json
from collections import Counter
from pathlib import Path
from typing import Any

from canonical_export_validator import (
    HARD_BLOCK6_QUALITY_METRICS,
    REQUIRED_BLOCK6_QUALITY_CHECKS,
    validate_match_export,
)


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def canonical_tree_sha256(canonical_dir: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(item for item in canonical_dir.rglob("*") if item.is_file()):
        relative = path.relative_to(canonical_dir).as_posix()
        digest.update(relative.encode("utf-8"))
        digest.update(b"\0")
        digest.update(hashlib.sha256(path.read_bytes()).digest())
        digest.update(b"\n")
    return digest.hexdigest()


def audit(match_dir: Path) -> dict[str, Any]:
    canonical = match_dir / "canonical"
    economy_players = read_json(canonical / "derived/economy_players.json")
    economy_rounds = read_json(canonical / "derived/economy_rounds.json")
    player_stats = read_json(canonical / "derived/player_stats.json")
    clutch_events = read_json(canonical / "derived/clutch_events.json")
    metadata = read_json(canonical / "core/match_metadata.json")
    quality = read_json(canonical / "diagnostics/quality_report.json")["report"]

    price_statuses: Counter[str] = Counter()
    transaction_types: Counter[str] = Counter()
    money_coverage: Counter[str] = Counter()
    native_calculated_differences = 0
    economy_warnings = 0
    for player in economy_players["players"]:
        for field, value in player["money"].items():
            if isinstance(value, dict) and "status" in value:
                money_coverage[f"{field}:{value['status']}"] += 1
        if player["money"].get("native_calculated_delta") not in {None, 0}:
            native_calculated_differences += 1
        economy_warnings += len(player.get("warnings") or [])
        for inventory_name in (
            "inventory_start",
            "inventory_freeze_end",
            "inventory_round_end",
        ):
            for item in player[inventory_name].get("items") or []:
                price_statuses[str(item["price"]["status"])] += 1
        for transaction in player.get("transactions") or []:
            transaction_types[str(transaction["type"])] += 1
            price_statuses[str(transaction["item"]["price"]["status"])] += 1

    clutch_results = Counter(
        str(event["result"]) for event in clutch_events["clutch_events"]
    )
    clutch_states = Counter(
        str(event["state"]) for event in clutch_events["clutch_events"]
    )
    native_scoreboard_statuses = Counter(
        str(player["native_scoreboard_status"])
        for player in player_stats["players"]
    )
    checks = {
        str(check.get("name")): str(check.get("status"))
        for check in quality.get("checks") or []
        if isinstance(check, dict)
    }
    critical_gates = {
        name: checks.get(name, "missing")
        for name in sorted(REQUIRED_BLOCK6_QUALITY_CHECKS)
    }
    hard_metrics = {
        name: quality.get(name) for name in sorted(HARD_BLOCK6_QUALITY_METRICS)
    }
    validation_errors = validate_match_export(match_dir)
    return {
        "match_id": metadata["match_id"],
        "status": "pass" if not validation_errors else "fail",
        "validation_errors": validation_errors,
        "contracts": {
            "economy_round": economy_rounds["schema_id"],
            "economy_player": economy_players["schema_id"],
            "player_stats": player_stats["schema_id"],
            "clutch_event": clutch_events["schema_id"],
            "match_metadata": metadata["schema_id"],
        },
        "economy": {
            "round_rows": len(economy_rounds["rounds"]),
            "player_round_rows": len(economy_players["players"]),
            "transaction_types": dict(sorted(transaction_types.items())),
            "price_statuses": dict(sorted(price_statuses.items())),
            "native_calculated_differences": native_calculated_differences,
            "money_coverage": dict(sorted(money_coverage.items())),
            "warnings": economy_warnings,
        },
        "stats": {
            "players": len(player_stats["players"]),
            "native_scoreboard_statuses": dict(
                sorted(native_scoreboard_statuses.items())
            ),
            "approximate_ratings": sum(
                player.get("rating", {}).get("approximate") is True
                for player in player_stats["players"]
            ),
        },
        "clutch": {
            "attempts": len(clutch_events["clutch_events"]),
            "results": dict(sorted(clutch_results.items())),
            "states": dict(sorted(clutch_states.items())),
        },
        "metadata": {
            "played_at": metadata.get("played_at"),
            "played_at_status": metadata.get("played_at_status"),
            "source": metadata.get("source"),
            "configuration_hashes": metadata.get("configuration_hashes"),
            "transformation_versions": metadata.get("transformation_versions"),
            "warnings": metadata.get("warnings") or [],
        },
        "quality_gates": critical_gates,
        "hard_metrics": hard_metrics,
        "canonical_tree_sha256": canonical_tree_sha256(canonical),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("match_dirs", nargs="+", type=Path)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    reports = [audit(path) for path in args.match_dirs]
    if args.json:
        print(json.dumps(reports, indent=2, sort_keys=True))
    else:
        for report in reports:
            print(
                f"{report['match_id']}: {report['status']} "
                f"sha256={report['canonical_tree_sha256']} "
                f"clutches={report['clutch']['attempts']}"
            )
            for error in report["validation_errors"]:
                print(f"  ERROR: {error}")
    return 1 if any(report["status"] != "pass" for report in reports) else 0


if __name__ == "__main__":
    raise SystemExit(main())
