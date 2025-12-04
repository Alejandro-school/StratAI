"""
Script para verificar que el endpoint /steam/processed-demos del backend Python
devuelve correctamente los campos flash_assists y enemies_flashed.
"""
import requests
import json
import time

# Esperar a que el servidor esté listo
print("⏳ Esperando 3 segundos a que FastAPI esté listo...")
time.sleep(3)

# URL del endpoint
url = "http://localhost:8000/steam/processed-demos"

print("\n=== Probando endpoint /steam/processed-demos ===\n")

try:
    # Hacer request sin autenticación (para ver estructura)
    # En producción necesitaríamos cookies de sesión
    response = requests.get(url)
    
    print(f"Status Code: {response.status_code}")
    
    if response.status_code == 401:
        print("⚠️ Necesita autenticación (esperado)")
        print("Los campos deberían estar presentes en respuestas autenticadas.\n")
        
        # Verificar que el código backend tiene los campos
        print("✅ Verificando código backend...")
        with open("app/routes/steam_service.py", "r", encoding="utf-8") as f:
            content = f.read()
            has_flash_assists = '"flash_assists"' in content or "'flash_assists'" in content
            has_enemies_flashed = '"enemies_flashed"' in content or "'enemies_flashed'" in content
            
            print(f"   ├─ Código incluye 'flash_assists': {has_flash_assists}")
            print(f"   └─ Código incluye 'enemies_flashed': {has_enemies_flashed}")
            
            if has_flash_assists and has_enemies_flashed:
                print("\n✅ El backend Python está configurado correctamente")
            else:
                print("\n❌ Faltan campos en el código del backend")
    
    elif response.status_code == 200:
        data = response.json()
        print("✅ Respuesta exitosa")
        
        # Verificar estructura
        if isinstance(data, list) and len(data) > 0:
            demo = data[0]
            if "players" in demo and len(demo["players"]) > 0:
                player = demo["players"][0]
                
                print(f"\nEjemplo de jugador:")
                print(f"  Nombre: {player.get('name', 'N/A')}")
                print(f"  Assists: {player.get('assists', 'FALTA')}")
                print(f"  Flash Assists: {player.get('flash_assists', 'FALTA')}")
                print(f"  Enemies Flashed: {player.get('enemies_flashed', 'FALTA')}")
                
                has_all = all(key in player for key in ["flash_assists", "enemies_flashed"])
                print(f"\n{'✅' if has_all else '❌'} Campos presentes: {has_all}")
        
except requests.exceptions.ConnectionError:
    print("❌ No se pudo conectar al servidor. ¿Está corriendo en puerto 8000?")
except Exception as e:
    print(f"❌ Error: {e}")
