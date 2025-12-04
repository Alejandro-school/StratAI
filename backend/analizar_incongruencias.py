import json
import os
from collections import defaultdict

# Cargar todos los archivos JSON
export_dir = os.path.join('data', 'exports', 'de_nuke_2-13_2025-11-27_11-31')

print("ANALISIS DE INCONGRUENCIAS EN DATOS EXPORTADOS")
print("=" * 70)

# Cargar archivos
with open(os.path.join(export_dir, 'match_info.json'), 'r', encoding='utf-8') as f:
    match_info = json.load(f)

with open(os.path.join(export_dir, 'players_summary.json'), 'r', encoding='utf-8') as f:
    players_data = json.load(f)

with open(os.path.join(export_dir, 'grenades.json'), 'r', encoding='utf-8') as f:
    grenades_data = json.load(f)

with open(os.path.join(export_dir, 'combat_analytics.json'), 'r', encoding='utf-8') as f:
    combat_data = json.load(f)

with open(os.path.join(export_dir, 'economy.json'), 'r', encoding='utf-8') as f:
    economy_data = json.load(f)

players = players_data['players']
grenades = grenades_data['grenades']
total_rounds = match_info['total_rounds']

print(f"\n📊 Match Info:")
print(f"   Map: {match_info['map_name']}")
print(f"   Score: {match_info['team_score']}-{match_info['opponent_score']}")
print(f"   Total Rounds: {total_rounds}")
print(f"   Players: {len(players)}")

incongruencias = []

# ==================== ANÁLISIS 1: Weapon Stats vs Global Stats ====================
print("\n\n🔫 [1] WEAPON STATS - Verificando suma de kills por arma vs kills totales")
print("-" * 70)
for player in players:
    weapon_kills = sum(ws['kills'] for ws in player.get('weapon_stats', {}).values())
    total_kills = player['kills']
    
    if weapon_kills != total_kills:
        msg = f"❌ {player['name']}: weapon_kills ({weapon_kills}) ≠ total_kills ({total_kills})"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {weapon_kills} kills (correcto)")

# ==================== ANÁLISIS 2: Shots Fired vs Weapon Stats ====================
print("\n\n🎯 [2] SHOTS FIRED - Verificando disparos totales vs suma por arma")
print("-" * 70)
for player in players:
    weapon_shots = sum(ws['shots_fired'] for ws in player.get('weapon_stats', {}).values())
    total_shots = player['shots_fired']
    
    if weapon_shots != total_shots:
        msg = f"❌ {player['name']}: weapon_shots ({weapon_shots}) ≠ total_shots ({total_shots})"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {weapon_shots} shots (correcto)")

# ==================== ANÁLISIS 3: Shots Hit vs Weapon Stats ====================
print("\n\n🎯 [3] SHOTS HIT - Verificando impactos totales vs suma por arma")
print("-" * 70)
for player in players:
    weapon_hits = sum(ws['shots_hit'] for ws in player.get('weapon_stats', {}).values())
    total_hits = player['shots_hit']
    
    if weapon_hits != total_hits:
        msg = f"❌ {player['name']}: weapon_hits ({weapon_hits}) ≠ total_hits ({total_hits})"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {weapon_hits} hits (correcto)")

# ==================== ANÁLISIS 4: Accuracy Calculation ====================
print("\n\n📐 [4] ACCURACY - Verificando cálculo de precisión")
print("-" * 70)
for player in players:
    shots_fired = player['shots_fired']
    shots_hit = player['shots_hit']
    accuracy = player['accuracy']
    
    expected_accuracy = (shots_hit / shots_fired * 100) if shots_fired > 0 else 0
    diff = abs(expected_accuracy - accuracy)
    
    if diff > 0.01:  # Tolerancia para errores de redondeo
        msg = f"❌ {player['name']}: accuracy ({accuracy:.2f}%) ≠ calculado ({expected_accuracy:.2f}%)"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {accuracy:.2f}% (correcto)")

# ==================== ANÁLISIS 5: KD Ratio ====================
print("\n\n⚔️  [5] KD RATIO - Verificando cálculo de K/D")
print("-" * 70)
for player in players:
    kills = player['kills']
    deaths = player['deaths']
    kd_ratio = player['kd_ratio']
    
    expected_kd = kills / deaths if deaths > 0 else kills
    diff = abs(expected_kd - kd_ratio)
    
    if diff > 0.01:
        msg = f"❌ {player['name']}: kd_ratio ({kd_ratio:.2f}) ≠ calculado ({expected_kd:.2f})"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {kd_ratio:.2f} (correcto)")

# ==================== ANÁLISIS 6: Headshot Percentage ====================
print("\n\n🎯 [6] HEADSHOT % - Verificando porcentaje de headshots")
print("-" * 70)
for player in players:
    kills = player['kills']
    hs_pct = player['hs_percentage']
    
    # Contar headshots desde weapon_stats
    total_headshots = sum(ws['headshots'] for ws in player.get('weapon_stats', {}).values())
    
    expected_hs_pct = (total_headshots / kills * 100) if kills > 0 else 0
    diff = abs(expected_hs_pct - hs_pct)
    
    if diff > 0.01:
        msg = f"❌ {player['name']}: hs% ({hs_pct:.1f}%) ≠ calculado ({expected_hs_pct:.1f}%, {total_headshots} HS)"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {hs_pct:.1f}% ({total_headshots} HS, correcto)")

# ==================== ANÁLISIS 7: Weapon Accuracy per Weapon ====================
print("\n\n🔫 [7] WEAPON ACCURACY - Verificando precisión por arma")
print("-" * 70)
for player in players:
    for weapon_name, ws in player.get('weapon_stats', {}).items():
        shots = ws['shots_fired']
        hits = ws['shots_hit']
        accuracy = ws['accuracy']
        
        expected_acc = (hits / shots * 100) if shots > 0 else 0
        diff = abs(expected_acc - accuracy)
        
        if diff > 0.01:
            msg = f"❌ {player['name']} - {weapon_name}: acc ({accuracy:.1f}%) ≠ calculado ({expected_acc:.1f}%)"
            incongruencias.append(msg)
            print(msg)

print(f"\n✅ Armas verificadas para todos los jugadores")

# ==================== ANÁLISIS 8: Granadas - Verificar conteos ====================
print("\n\n💣 [8] GRENADES - Verificando conteos de granadas lanzadas")
print("-" * 70)

# Contar granadas por jugador desde players_summary weapon_stats
player_grenades_from_stats = defaultdict(lambda: defaultdict(int))
for player in players:
    steam_id = player['steam_id']
    for weapon_name, ws in player.get('weapon_stats', {}).items():
        if 'Grenade' in weapon_name or weapon_name in ['Flashbang', 'Molotov', 'Smoke Grenade', 'Decoy Grenade']:
            player_grenades_from_stats[steam_id][weapon_name] = ws['shots_fired']

# Contar granadas desde grenades.json
player_grenades_from_file = defaultdict(lambda: defaultdict(int))
for nade in grenades:
    steam_id = nade['thrower_steam_id']
    nade_type = nade['grenade_type']
    player_grenades_from_file[steam_id][nade_type] += 1

print("Comparando conteos de granadas entre weapon_stats y grenades.json:")
for player in players:
    steam_id = player['steam_id']
    print(f"\n{player['name']}:")
    
    # Mapear tipos
    type_mapping = {
        'Flash': 'Flashbang',
        'HE': 'HE Grenade',
        'Smoke': 'Smoke Grenade',
        'Molotov': 'Molotov',
        'Incendiary': 'Incendiary Grenade',
        'Decoy': 'Decoy Grenade'
    }
    
    for nade_type_file, weapon_name_stats in type_mapping.items():
        count_file = player_grenades_from_file[steam_id][nade_type_file]
        count_stats = player_grenades_from_stats[steam_id].get(weapon_name_stats, 0)
        
        if count_file != count_stats and count_file > 0:
            msg = f"  ❌ {nade_type_file}: grenades.json ({count_file}) ≠ weapon_stats ({count_stats})"
            incongruencias.append(msg)
            print(msg)

# ==================== ANÁLISIS 9: Verificar granadas con efectos ====================
print("\n\n💥 [9] GRENADE EFFECTS - Verificando que granadas tienen datos de efectividad")
print("-" * 70)

flash_count_with_data = 0
flash_count_total = 0
he_count_with_data = 0
he_count_total = 0
smoke_count_with_data = 0
smoke_count_total = 0
molotov_count_with_data = 0
molotov_count_total = 0

for nade in grenades:
    nade_type = nade['grenade_type']
    
    if nade_type == 'Flash':
        flash_count_total += 1
        if nade.get('enemies_flashed') is not None:
            flash_count_with_data += 1
    elif nade_type == 'HE':
        he_count_total += 1
        if nade.get('total_damage') is not None:
            he_count_with_data += 1
    elif nade_type == 'Smoke':
        smoke_count_total += 1
        if nade.get('smoke_duration_ms') is not None:
            smoke_count_with_data += 1
    elif nade_type in ['Molotov', 'Incendiary']:
        molotov_count_total += 1
        if nade.get('fire_duration_ms') is not None:
            molotov_count_with_data += 1

print(f"Flash: {flash_count_with_data}/{flash_count_total} tienen datos de enemigos cegados")
print(f"HE: {he_count_with_data}/{he_count_total} tienen datos de daño")
print(f"Smoke: {smoke_count_with_data}/{smoke_count_total} tienen datos de duración")
print(f"Molotov/Inc: {molotov_count_with_data}/{molotov_count_total} tienen datos de duración")

if flash_count_with_data == 0 and flash_count_total > 0:
    msg = "❌ NINGUNA flash tiene datos de enemies_flashed"
    incongruencias.append(msg)
    print(f"\n{msg}")

# ==================== ANÁLISIS 10: Enemies Flashed en players vs grenades ====================
print("\n\n👁️  [10] ENEMIES FLASHED - Verificando consistencia entre players y grenades")
print("-" * 70)

# Contar enemies_flashed desde players_summary
player_enemies_flashed = {p['steam_id']: p['enemies_flashed'] for p in players}

# Contar desde grenades.json
player_enemies_flashed_grenades = defaultdict(int)
for nade in grenades:
    if nade['grenade_type'] == 'Flash' and nade.get('enemies_flashed'):
        steam_id = nade['thrower_steam_id']
        player_enemies_flashed_grenades[steam_id] += nade['enemies_flashed']

for player in players:
    steam_id = player['steam_id']
    count_player = player_enemies_flashed[steam_id]
    count_grenades = player_enemies_flashed_grenades.get(steam_id, 0)
    
    if count_player != count_grenades:
        msg = f"❌ {player['name']}: enemies_flashed ({count_player}) ≠ suma grenades ({count_grenades})"
        incongruencias.append(msg)
        print(msg)
    else:
        print(f"✅ {player['name']}: {count_player} enemies flashed (correcto)")

# ==================== RESUMEN FINAL ====================
print("\n\n" + "=" * 70)
print("📋 RESUMEN DE INCONGRUENCIAS ENCONTRADAS")
print("=" * 70)

if incongruencias:
    print(f"\n❌ Se encontraron {len(incongruencias)} incongruencias:\n")
    for i, inc in enumerate(incongruencias, 1):
        print(f"{i}. {inc}")
else:
    print("\n✅ ¡NO SE ENCONTRARON INCONGRUENCIAS! Todos los datos son consistentes.")

print("\n" + "=" * 70)
