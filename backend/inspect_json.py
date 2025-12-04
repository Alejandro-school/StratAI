import json

with open('data/exports/de_inferno_3-13_2025-11-29_21-27/match_data.json', encoding='utf-8') as f:
    data = json.load(f)

print('=== KILL EXAMPLE ===')
print(json.dumps(data['kills'][0], indent=2))

print('\n=== PLAYER EXAMPLE ===')
player_id = list(data['players'].keys())[0]
player = data['players'][player_id]
print(f"Player ID: {player_id}")
print(json.dumps(player, indent=2)[:500])

print(f'\n=== TOTALS ===')
print(f"Kills: {len(data['kills'])}")
print(f"Damage events: {len(data.get('damage', []))}")
print(f"Flashes: {len(data.get('flashes', []))}")
print(f"Economy entries: {len(data.get('economy', []))}")
print(f"Bomb events: {len(data.get('bomb_events', []))}")

print(f'\n=== MOVEMENT DATA ===')
for player_id, player in data['players'].items():
    movement_count = len(player.get('movement', []))
    print(f"{player['name']}: {movement_count} snapshots")
