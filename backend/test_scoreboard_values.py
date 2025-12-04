"""
Script para reprocesar UNA demo y verificar los nuevos valores de ADR y Assists del scoreboard.
"""
import requests
import json
import os
import time

# Esperar a que el servicio esté listo
print("⏳ Esperando 2 segundos a que el servicio Go esté listo...")
time.sleep(2)

# Elegir la demo de nuke
demo_path = os.path.abspath("data/demos/match_6DhZ28LiQKRMfj4tDG8CKUGnK.dem")
match_id = "6DhZ28LiQKRMfj4tDG8CKUGnK"

print(f"\n=== Reprocesando Demo: {match_id} ===")
print(f"Ruta: {demo_path}")

if not os.path.exists(demo_path):
    print(f"❌ No se encontró el archivo: {demo_path}")
    exit(1)

# Llamar al servicio Go
url = "http://localhost:8080/process-demo"
payload = {
    "demo_path": demo_path,
    "steam_id": "",
    "match_id": match_id
}

print(f"\n📡 Enviando request a {url}...")
try:
    response = requests.post(url, json=payload, timeout=60)
    
    if response.status_code == 200:
        result = response.json()
        print("✅ Demo procesada exitosamente!")
        
        # Buscar a Kerchak en los jugadores
        if "data" in result and "players" in result["data"]:
            for player in result["data"]["players"]:
                if "Kerchak" in player.get("name", ""):
                    print(f"\n🎯 Estadísticas de {player['name']}:")
                    print(f"   Kills: {player.get('kills', 0)}")
                    print(f"   Deaths: {player.get('deaths', 0)}")
                    print(f"   Assists: {player.get('assists', 0)} (del SCOREBOARD)")
                    print(f"   ADR: {player.get('adr', 0):.2f} (del SCOREBOARD)")
                    print(f"   Flash Assists: {player.get('flash_assists', 0)}")
                    print(f"   Enemies Flashed: {player.get('enemies_flashed', 0)}")
                    break
        
        # Verificar que se exportó correctamente
        exports_dir = "data/exports"
        print(f"\n📁 Buscando export en {exports_dir}...")
        
        for folder in os.listdir(exports_dir):
            if match_id in folder or "nuke" in folder.lower():
                players_path = os.path.join(exports_dir, folder, "players_summary.json")
                if os.path.exists(players_path):
                    with open(players_path, 'r', encoding='utf-8') as f:
                        data = json.load(f)
                        for player in data.get("players", []):
                            if "Kerchak" in player.get("name", ""):
                                print(f"\n✅ Verificado en {folder}/players_summary.json:")
                                print(f"   Assists: {player.get('assists', 0)}")
                                print(f"   ADR: {player.get('adr', 0):.2f}")
                                break
                    break
    else:
        print(f"❌ Error: {response.status_code}")
        print(response.text[:500])
        
except requests.exceptions.Timeout:
    print("⏱️ Timeout esperando respuesta (normal para demos grandes)")
except Exception as e:
    print(f"❌ Error: {e}")
