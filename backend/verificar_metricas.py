import json

# Cargar el export más reciente
with open('data/exports/de_nuke_2-13_2025-11-27_11-31/players_summary.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

print("\n" + "=" * 80)
print("VERIFICACION DE NUEVAS METRICAS - de_nuke 2-13")
print("=" * 80)
print(f"\n{'Jugador':<20} | {'Kills':<6} | {'Assists':<8} | {'Flash Assists':<14} | {'Enemies Flashed':<16}")
print("-" * 80)

for player in data['players']:
    print(f"{player['name']:<20} | {player['kills']:<6} | {player['assists']:<8} | {player['flash_assists']:<14} | {player['enemies_flashed']:<16}")

print("\n" + "=" * 80)
print("RESUMEN:")
print("=" * 80)

total_assists = sum(p['assists'] for p in data['players'])
total_flash_assists = sum(p['flash_assists'] for p in data['players'])
total_enemies_flashed = sum(p['enemies_flashed'] for p in data['players'])

print(f"Total Assists: {total_assists}")
print(f"Total Flash Assists: {total_flash_assists}")
print(f"Total Enemies Flashed: {total_enemies_flashed}")

# Verificar que no hay ceros donde debería haber datos
players_with_assists = sum(1 for p in data['players'] if p['assists'] > 0)
players_with_enemies_flashed = sum(1 for p in data['players'] if p['enemies_flashed'] > 0)

print(f"\nJugadores con asistencias: {players_with_assists}/10")
print(f"Jugadores con enemigos cegados: {players_with_enemies_flashed}/10")

if total_assists > 0 and total_enemies_flashed > 0:
    print("\n✅ ¡LAS NUEVAS METRICAS ESTAN FUNCIONANDO CORRECTAMENTE!")
else:
    print("\n❌ Parece que aún hay métricas en cero")
