import json

from backend.app.utils.performance_detail_aggregator import (
    _classify_buy,
    aggregate_performance_details,
)


def _participant(steam_id, name, shots, hits, **extra):
    return {
        "steam_id": steam_id,
        "name": name,
        "weapon": "AK-47",
        "map_area": "Middle",
        "shots_fired": shots,
        "hits": hits,
        "time_to_reaction": 420,
        "time_to_first_damage": 510,
        "initial_crosshair_error": 7.5,
        "velocity": 20,
        "engagement_type": "hold",
        **extra,
    }


def test_classify_buy_uses_real_equipment_and_spend():
    assert _classify_buy(4700, 3200) == "full_buy"
    assert _classify_buy(1800, 2300) == "force_buy"
    assert _classify_buy(2600, 1200) == "partial_buy"
    assert _classify_buy(900, 700) == "eco"


def test_aggregates_combat_and_economy_from_export_files(tmp_path):
    user_id = "76561198116485358"
    rival_id = "76561198065602953"
    combat = {
        "rounds": [{
            "round": 1,
            "duels": [
                {
                    "type": "duel",
                    "outcome": "kill",
                    "attacker": _participant(user_id, "Kerchak", 4, 2),
                    "victims": [_participant(rival_id, "Rival", 3, 1)],
                    "context": {"is_opening_kill": True, "through_smoke": True},
                },
                {
                    "type": "duel",
                    "outcome": "kill",
                    "attacker": _participant(rival_id, "Rival", 5, 2),
                    "victims": [_participant(user_id, "Kerchak", 2, 1, is_blind=True)],
                    "context": {"is_trade": True},
                },
            ],
        }],
    }
    economy = [{
        "match_id": "match-1",
        "rounds": [
            {
                "round": 1,
                "teams": {"T": {"gini_coefficient": 0.12}},
                "players": [{
                    "steam_id": user_id,
                    "team": "T",
                    "spent_in_buy": 3200,
                    "final_equipment_value": 4700,
                    "final_money": 1200,
                    "equipment_value_end": 3600,
                    "outcome": "win",
                    "survived": True,
                }],
            },
            {
                "round": 2,
                "teams": {"T": {"gini_coefficient": 0.08}},
                "players": [{
                    "steam_id": user_id,
                    "team": "T",
                    "spent_in_buy": 700,
                    "final_equipment_value": 900,
                    "final_money": 2100,
                    "equipment_value_end": 0,
                    "outcome": "win",
                    "survived": False,
                }],
            },
        ],
    }]

    canonical = tmp_path / "canonical"
    derived = canonical / "derived"
    derived.mkdir(parents=True)
    engagements = [
        {
            "round_number": round_data["round"],
            "details": duel,
        }
        for round_data in combat["rounds"]
        for duel in round_data["duels"]
    ]
    (derived / "engagements.json").write_text(
        json.dumps({"engagements": engagements}),
        encoding="utf-8",
    )
    economy_records = []
    economy_rounds = []
    for economy_match in economy:
        for round_data in economy_match["rounds"]:
            economy_rounds.append(
                {"round_number": round_data["round"], "teams": round_data["teams"]}
            )
            for player in round_data["players"]:
                economy_records.append(
                    {
                        "round_number": round_data["round"],
                        "player_id": f"steam:{player['steam_id']}",
                        "details": player,
                    }
                )
    (derived / "player_round_economy.json").write_text(
        json.dumps({"rounds": economy_rounds, "records": economy_records}),
        encoding="utf-8",
    )
    (canonical / "manifest.json").write_text("{}", encoding="utf-8")
    (tmp_path / "manifest.json").write_text(
        json.dumps(
            {
                "artifacts": [
                    {"artifact_type": "canonical_manifest", "path": "canonical/manifest.json"},
                    {"artifact_type": "engagements", "path": "canonical/derived/engagements.json"},
                    {
                        "artifact_type": "player_round_economy",
                        "path": "canonical/derived/player_round_economy.json",
                    },
                ]
            }
        ),
        encoding="utf-8",
    )
    details = aggregate_performance_details(
        [{"match_dir": str(tmp_path)}],
        user_id,
    )

    assert details["duels"]["kills_won"] == 1
    assert details["duels"]["kills_lost"] == 1
    assert details["duels"]["encounters"][0]["name"] == "Rival"
    assert details["mechanics"]["engagements"] == 2
    assert details["mechanics"]["through_smoke_pct"] == 50.0
    assert details["economy"]["rounds"] == 2
    assert details["economy"]["save_conversion_rate"] == 100.0
    assert details["economy"]["buy_types"][0]["rounds"] == 1
    assert details["sources"] == {"combat_matches": 1, "economy_matches": 1}
