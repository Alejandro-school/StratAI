from collections import defaultdict
from pathlib import Path

import pandas as pd
from joblib import load

# ---------------------------------------------------------------------
USE_NADES = ["Smoke Grenade", "Flashbang", "HE Grenade"]
# ---------------------------------------------------------------------

# ---------- carga del modelo ----------
BUNDLE    = load(Path(__file__).resolve().parents[1] / "models" / "buycoach.pkl")
model     = BUNDLE["model"]
feat_cols = BUNDLE["feat_cols"]
classes   = BUNDLE["classes"]


def add_team_stats(df: pd.DataFrame) -> pd.DataFrame:
    grp = df.groupby(["round_number", "team"])
    df["team_avg_cash"] = grp["initial_money"].transform("mean")
    df["team_min_cash"] = grp["initial_money"].transform("min")
    df["team_max_cash"] = grp["initial_money"].transform("max")
    return df


def features_from_record(rec: dict) -> dict:
    """Convierte UNA entrada de economy_history en features."""
    purchases = [p["name"] for p in rec.get("purchases", [])]
    row = {
        "round_number":  rec["round_number"],
        "team":          rec["team"],
        "initial_money": rec["initial_money"],
        "loss_bonus":    rec["loss_bonus"],
        "spent_in_buy":  rec["spent_in_buy"],
        "kill_reward":   rec["kill_reward"],
        "reward_plant":  rec["Reward_for_Plant"],
        "final_money":   rec["final_money"],
        "nades_bought":  len(purchases),
        "has_kevlar":    int(any("Kevlar" in p for p in purchases)),
    }
    for nade in USE_NADES:
        row[f"has_{nade.split()[0].lower()}"] = int(nade in purchases)
    return row


def get_buy_advice(economy_json: dict) -> dict:
    """
    Devuelve una lista de recomendaciones: una por jugador/ronda.
    Estructura de salida:
    {
        "advices": [
            {round, player, team, recommendation, confidence},
            ...
        ]
    }
    """
    records = economy_json["economy_history"]
    rows = [features_from_record(r) for r in records]

    rows_df = add_team_stats(pd.DataFrame(rows))
    # ---------- CORREGIDO ----------
    df_hot = pd.get_dummies(rows_df, columns=["team"])      # crea team_CT / team_T
    X = df_hot.reindex(columns=feat_cols, fill_value=0)     # alinea con el modelo
    # --------------------------------
    probs = model.predict(X)
    idxs  = probs.argmax(axis=1)

    advices = []
    for rec, ix, pr in zip(records, idxs, probs):
        advices.append({
            "round":          rec["round_number"],
            "player":         rec["name"],
            "team":           rec["team"],
            "recommendation": classes[ix],
            "confidence":     round(float(pr[ix]) * 100, 1),
        })

    return {"advices": advices}
