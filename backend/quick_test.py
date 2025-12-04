"""Reprocesar una demo para verificar valores del scoreboard"""
import requests, json, os, time

time.sleep(2)
demo_path = os.path.abspath("data/demos/match_6DhZ28LiQKRMfj4tDG8CKUGnK.dem")
match_id = "6DhZ28LiQKRMfj4tDG8CKUGnK"

print(f"Reprocesando {match_id}...")
response = requests.post("http://localhost:8080/process-demo", 
                        json={"demo_path": demo_path, "steam_id": "", "match_id": match_id}, 
                        timeout=60)

if response.status_code == 200:
    result = response.json()
    for player in result["data"]["players"]:
        if "Kerchak" in player["name"] or "Nocries" in player["name"]:
            print(f"{player['name']}: Assists={player['assists']}, ADR={player['adr']:.2f}")
