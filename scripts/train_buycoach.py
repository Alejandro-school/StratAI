# pip install pandas lightgbm joblib
import json, glob
from pathlib import Path

import pandas as pd
import lightgbm as lgb
from joblib import dump

# ---------------------------------------------------------------------
DATA_GLOB = "backend/data/exports/*/economy.json"   # ajusta si cambia tu ruta
USE_NADES = ["Smoke Grenade", "Flashbang", "HE Grenade"]
# ---------------------------------------------------------------------


def add_team_stats(df: pd.DataFrame) -> pd.DataFrame:
    """Añade media, mínimo y máximo de dinero del equipo por ronda."""
    grp = df.groupby(["round_number", "team"])
    df["team_avg_cash"] = grp["initial_money"].transform("mean")
    df["team_min_cash"] = grp["initial_money"].transform("min")
    df["team_max_cash"] = grp["initial_money"].transform("max")
    return df


def load_economy_json(path: str) -> pd.DataFrame:
    """Convierte un economy.json en una fila por jugador / ronda."""
    data = json.load(open(path, encoding="utf-8"))
    rows = []
    for rec in data["economy_history"]:
        purchases = [p["name"] for p in rec.get("purchases", [])]

        row = {
            # claves “originales”
            "match_id":       data["match_id"],
            "round_number":   rec["round_number"],
            "team":           rec["team"],           # 'T' / 'CT'
            "initial_money":  rec["initial_money"],
            "loss_bonus":     rec["loss_bonus"],
            "spent_in_buy":   rec["spent_in_buy"],
            "kill_reward":    rec["kill_reward"],
            "reward_plant":   rec["Reward_for_Plant"],
            "final_money":    rec["final_money"],
            # features derivadas
            "nades_bought":   len(purchases),
            "has_kevlar":     int(any("Kevlar" in p for p in purchases)),
        }
        for nade in USE_NADES:                        # 0/1 por cada granada
            row[f"has_{nade.split()[0].lower()}"] = int(nade in purchases)

        # etiqueta (target) muy simple: concatenación ordenada de compras
        row["buy_label"] = ",".join(sorted(purchases))
        rows.append(row)
    return pd.DataFrame(rows)


# ---------- cargar todas las partidas --------------------------------
paths = glob.glob(DATA_GLOB)
if not paths:
    raise SystemExit("⚠️  No se encontraron economy.json; revisa DATA_GLOB")

df = pd.concat([load_economy_json(p) for p in paths], ignore_index=True)
df = add_team_stats(df)                               # ← agrega economía de equipo

# ---------- preparar X e y ------------------------------------------
feat_cols = [c for c in df.columns if c not in ["buy_label", "match_id"]]
X = pd.get_dummies(df[feat_cols], columns=["team"])
y = df["buy_label"].astype("category")

# ---------- entrenar LightGBM ---------------------------------------
train_data = lgb.Dataset(X, label=y.cat.codes)
params = dict(objective="multiclass",
              num_class=y.nunique(),
              learning_rate=0.1)
model = lgb.train(params, train_data, num_boost_round=300)

# ---------- guardar modelo + metadatos -------------------------------
MODEL_DIR = Path(__file__).resolve().parents[1] / "backend" / "app" / "models"
MODEL_DIR.mkdir(exist_ok=True)
dump(
    {"model": model,
     "feat_cols": list(X.columns),      # orden exacto de columnas
     "classes": y.cat.categories.tolist()},
    MODEL_DIR / "buycoach.pkl",
)
print("✅  Modelo entrenado y guardado en", MODEL_DIR / "buycoach.pkl")
