"""
Script para verificar que los nuevos campos flash_assists y enemies_flashed
están presentes en los datos devueltos por el backend.
"""
import json
import os

# Ruta al export más reciente
exports_dir = "data/exports"
latest_export = "de_nuke_13-2_2025-11-27_11-37"
players_path = os.path.join(exports_dir, latest_export, "players_summary.json")

print("=== Verificación de Nuevos Campos ===\n")
print(f"Leyendo: {players_path}\n")

with open(players_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

players = data.get("players", [])
print(f"Total jugadores: {len(players)}\n")

# Verificar cada jugador
for i, player in enumerate(players[:3], 1):  # Solo primeros 3 para brevedad
    name = player.get("name", "Unknown")
    assists = player.get("assists", 0)
    flash_assists = player.get("flash_assists", 0)
    enemies_flashed = player.get("enemies_flashed", 0)
    
    print(f"Jugador {i}: {name}")
    print(f"  ├─ Assists: {assists}")
    print(f"  ├─ Flash Assists: {flash_assists}")
    print(f"  └─ Enemies Flashed: {enemies_flashed}")
    print()

# Verificar que los campos existen en el JSON
sample_player = players[0]
has_flash_assists = "flash_assists" in sample_player
has_enemies_flashed = "enemies_flashed" in sample_player

print("\n=== Resultado ===")
print(f"✅ Campo 'flash_assists' presente: {has_flash_assists}")
print(f"✅ Campo 'enemies_flashed' presente: {has_enemies_flashed}")

if has_flash_assists and has_enemies_flashed:
    print("\n✅ TODOS LOS CAMPOS ESTÁN PRESENTES EN EL JSON")
else:
    print("\n❌ FALTAN CAMPOS EN EL JSON")
