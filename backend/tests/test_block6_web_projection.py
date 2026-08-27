from backend.app.matches.match_web_projection import project_economy


class Block6Match:
    match_id = "block6-test"

    def participants(self):
        return ({"player_id": "steam:1", "display_name": "one"},)

    def rounds(self):
        return ({"round_number": 1, "win_reason": "target_bombed"},)

    def economy_rounds(self):
        return (
            {
                "round_number": 1,
                "team_id": "team_a",
                "side": "t",
                "loss_bonus": {"amount": 1900, "status": "calculated"},
            },
        )

    def economy_records(self):
        return (
            {
                "round_number": 1,
                "player_id": "steam:1",
                "team_id": "team_a",
                "side": "t",
                "outcome": "win",
                "survived": True,
                "money": {
                    "round_start_observed": {
                        "amount": 800,
                        "status": "observed",
                    },
                    "after_buy_observed": {
                        "amount": None,
                        "status": "not_observed",
                    },
                    "after_buy_calculated": {
                        "amount": 800,
                        "status": "calculated",
                    },
                    "round_end_observed": {
                        "amount": None,
                        "status": "not_observed",
                    },
                    "next_round_observed": {
                        "amount": None,
                        "status": "not_observed",
                    },
                },
                "inventory_start": {
                    "native_value": 1000,
                    "calculated_value": 900,
                    "items": (),
                },
                "inventory_freeze_end": {
                    "native_value": 1100,
                    "calculated_value": 1000,
                    "items": (),
                },
                "inventory_round_end": {
                    "native_value": 1200,
                    "calculated_value": 1300,
                    "items": (),
                },
                "spent_in_buy": {"amount": 0, "status": "observed"},
                "transactions": (
                    {
                        "type": "purchase",
                        "item": {
                            "observed_item": "unknown_weapon",
                            "purchased_item": "unknown_weapon",
                            "price": {
                                "amount": None,
                                "status": "unknown",
                                "table_version": "stratai.cs2_prices@1",
                            },
                            "original_owner_player_id": None,
                            "original_owner_status": "not_observed",
                        },
                    },
                ),
                "warnings": ("purchase price unavailable",),
            },
        )


def test_block6_projection_preserves_nulls_native_values_and_stable_team_id() -> None:
    payload = project_economy(Block6Match())
    round_data = payload[0]["rounds"][0]
    player = round_data["players"][0]
    team = round_data["teams"]["T"]

    assert player["team_id"] == "team_a"
    assert player["money_after_buy"] == 800
    assert player["final_money"] is None
    assert player["purchases_observed_value"] is None
    assert player["purchases_unpriced_count"] == 1
    assert player["purchases_vs_spent_delta"] is None
    assert player["equipment_value_end"] == 1200
    assert player["equipment_value_end_native"] == 1200
    assert player["equipment_value_end_calculated"] == 1300
    assert team["team_id"] == "team_a"
    assert team["loss_bonus"] == 1900
