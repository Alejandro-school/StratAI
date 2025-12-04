import os
import requests
import json

# Ruta absoluta a la demo
demo_path = os.path.abspath(os.path.join('data', 'demos', 'match_2ycniSRrujsiUfXJf2WqQ4zAC.dem'))

print(f"Procesando: {demo_path}")
print(f"Existe: {os.path.exists(demo_path)}")

# Hacer request al servicio Go
try:
    r = requests.post('http://localhost:8080/process-demo', 
                     json={
                         'demo_path': demo_path, 
                         'steam_id': '76561198116485358', 
                         'match_id': '2ycniSRrujsiUfXJf2WqQ4zAC'
                     }, 
                     timeout=60)
    
    result = r.json()
    print(f"\n✅ Status: {result['status']}")
    
    if result['status'] == 'success':
        data = result.get('data', {})
        print(f"🗺️  Map: {data.get('map_name', 'N/A')}")
        print(f"📊 Score: {data.get('team_score', 0)}-{data.get('opponent_score', 0)}")
        print(f"👥 Players: {len(data.get('players', []))}")
        
        if data.get('players'):
            print("\n📈 Verificando nuevas métricas:")
            for i, player in enumerate(data['players'][:3], 1):
                print(f"\n  {i}. {player.get('name', 'Unknown')}")
                print(f"     Assists: {player.get('assists', 0)}")
                print(f"     Flash Assists: {player.get('flash_assists', 0)}")
                print(f"     Enemies Flashed: {player.get('enemiesFlashed', 0)}")
                
except requests.exceptions.ConnectionError:
    print("❌ Error: No se pudo conectar al servicio Go en http://localhost:8080")
    print("   Asegúrate de que el servicio esté corriendo")
except Exception as e:
    print(f"❌ Error: {e}")
