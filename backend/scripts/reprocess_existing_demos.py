"""
Script para reprocesar todas las demos existentes y almacenarlas en Redis.
Este script llama al Go service para procesar cada demo.
"""

import os
import glob
import requests
import time
from pathlib import Path

# Configuración
DEMOS_DIR = Path(__file__).parent.parent / "data" / "demos"
GO_SERVICE_URL = "http://localhost:8080/process-demo"

# Steam ID de Kerchak (usuario principal)
STEAM_ID = "76561198116485358"

def get_all_demos():
    """Obtiene todas las demos .dem del directorio"""
    demo_files = glob.glob(str(DEMOS_DIR / "*.dem"))
    return sorted(demo_files)

def extract_match_id(filename):
    """Extrae el match ID del nombre del archivo (match_XXXXX.dem)"""
    basename = os.path.basename(filename)
    if basename.startswith("match_") and basename.endswith(".dem"):
        return basename[6:-4]  # Quitar 'match_' y '.dem'
    return basename[:-4]

def reprocess_demo(demo_path):
    """Re-procesa una demo individual"""
    filename = os.path.basename(demo_path)
    match_id = extract_match_id(filename)
    
    print(f"  📦 Archivo: {filename}")
    print(f"  🔑 Match ID: {match_id}")
    
    try:
        payload = {
            "demo_path": str(demo_path),
            "steam_id": STEAM_ID,
            "match_id": match_id
        }
        
        response = requests.post(
            GO_SERVICE_URL,
            json=payload,
            timeout=120
        )
        
        if response.status_code == 200:
            result = response.json()
            if result.get("status") == "success":
                data = result.get("data", {})
                print(f"  ✅ Procesada: {data.get('map_name', 'N/A')} | "
                      f"Score: {data.get('team_score', 0)}-{data.get('opponent_score', 0)} | "
                      f"Jugadores: {len(data.get('players', []))}")
                return True
            else:
                print(f"  ❌ Error en respuesta: {result}")
                return False
        else:
            print(f"  ❌ HTTP {response.status_code}: {response.text[:200]}")
            return False
            
    except requests.exceptions.Timeout:
        print(f"  ⏱️  Timeout (la demo puede ser muy grande)")
        return False
    except Exception as e:
        print(f"  ❌ Error: {str(e)}")
        return False

def main():
    print("🔄 Reprocesamiento de Demos Existentes")
    print("=" * 60)
    print(f"📂 Directorio: {DEMOS_DIR}")
    print(f"🔑 Steam ID: {STEAM_ID}")
    print(f"🌐 Go Service: {GO_SERVICE_URL}")
    print()
    
    # Verificar directorio
    if not DEMOS_DIR.exists():
        print(f"❌ Error: No existe el directorio {DEMOS_DIR}")
        return
    
    # Obtener demos
    demos = get_all_demos()
    
    if not demos:
        print("❌ No se encontraron demos (.dem) en el directorio")
        return
    
    print(f"📊 Encontradas {len(demos)} demos")
    print()
    print("🚀 Iniciando procesamiento...")
    print()
    
    # Contadores
    success = 0
    failed = 0
    
    # Procesar cada demo
    for i, demo_path in enumerate(demos, 1):
        print(f"[{i}/{len(demos)}]")
        
        if reprocess_demo(demo_path):
            success += 1
        else:
            failed += 1
        
        # Pausa entre demos
        if i < len(demos):
            time.sleep(1)
        
        print()
    
    # Resumen
    print("=" * 60)
    print("📈 RESUMEN")
    print(f"  ✅ Exitosas: {success}")
    print(f"  ❌ Fallidas: {failed}")
    print(f"  📊 Total: {len(demos)}")
    print()
    
    if success > 0:
        print("🎉 ¡Demos reprocesadas correctamente!")
        print("💡 Los datos ahora están disponibles en Redis.")
        print("   Revisa http://localhost:3000/history-games")
    else:
        print("⚠️  No se pudo procesar ninguna demo")
        print("   Verifica que el servicio Go esté corriendo")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠️  Interrumpido por el usuario")
    except Exception as e:
        print(f"\n\n❌ Error fatal: {str(e)}")
        import traceback
        traceback.print_exc()
