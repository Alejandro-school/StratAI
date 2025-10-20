"""
Script para re-procesar todas las demos existentes con los datos actualizados.
Esto actualizará:
- La fecha correcta del partido (del archivo .dem)
- El campo Team de cada jugador (CT/T)
"""

import os
import glob
import requests
import time
from pathlib import Path

# Configuración
DEMOS_DIR = Path(__file__).parent.parent / "data" / "demos"
GO_SERVICE_URL = "http://localhost:8080"
REDIS_ENDPOINT = f"{GO_SERVICE_URL}/parse-demo"

def get_all_demos():
    """Obtiene todas las demos .dem del directorio"""
    demo_files = glob.glob(str(DEMOS_DIR / "*.dem"))
    return demo_files

def extract_steam_id_from_sharecode(filename):
    """
    Intenta extraer el Steam ID del nombre del archivo.
    Formato esperado: match_XXXXX.dem o similar
    
    Nota: Este script necesita el Steam ID del usuario.
    Deberás proporcionarlo manualmente o extraerlo de otra fuente.
    """
    # Por ahora, retornamos None y se usará el que esté en Redis
    return None

def reprocess_demo(demo_path, steam_id=None):
    """Re-procesa una demo individual"""
    filename = os.path.basename(demo_path)
    
    print(f"  📦 Procesando: {filename}")
    
    try:
        # Abrir el archivo .dem
        with open(demo_path, 'rb') as f:
            files = {'file': (filename, f, 'application/octet-stream')}
            
            # Parámetros (steam_id es opcional si ya está en el sistema)
            data = {}
            if steam_id:
                data['steam_id'] = steam_id
            
            # Enviar al endpoint de Go
            response = requests.post(
                REDIS_ENDPOINT,
                files=files,
                data=data,
                timeout=120  # 2 minutos de timeout
            )
            
            if response.status_code == 200:
                result = response.json()
                print(f"  ✅ Éxito: {result.get('message', 'Demo procesada')}")
                return True
            else:
                print(f"  ❌ Error {response.status_code}: {response.text}")
                return False
                
    except requests.exceptions.Timeout:
        print(f"  ⏱️  Timeout procesando {filename}")
        return False
    except Exception as e:
        print(f"  ❌ Error: {str(e)}")
        return False

def main():
    print("🔄 Script de Re-procesamiento de Demos")
    print("=" * 50)
    
    # Verificar que el directorio existe
    if not DEMOS_DIR.exists():
        print(f"❌ Error: No se encuentra el directorio {DEMOS_DIR}")
        return
    
    # Obtener todas las demos
    demos = get_all_demos()
    
    if not demos:
        print("❌ No se encontraron demos en el directorio")
        return
    
    print(f"📊 Se encontraron {len(demos)} demos")
    print()
    
    # Preguntar por el Steam ID (opcional)
    steam_id = input("🔑 Ingresa tu Steam ID (o presiona Enter para usar el guardado): ").strip()
    if not steam_id:
        steam_id = None
        print("ℹ️  Se usará el Steam ID guardado en el sistema")
    
    print()
    print("🚀 Iniciando re-procesamiento...")
    print()
    
    # Contadores
    success = 0
    failed = 0
    
    # Procesar cada demo
    for i, demo_path in enumerate(demos, 1):
        print(f"[{i}/{len(demos)}]")
        
        if reprocess_demo(demo_path, steam_id):
            success += 1
        else:
            failed += 1
        
        # Pausa entre demos para no sobrecargar
        if i < len(demos):
            time.sleep(0.5)
        
        print()
    
    # Resumen final
    print("=" * 50)
    print("📈 Resumen del Re-procesamiento")
    print(f"  ✅ Exitosas: {success}")
    print(f"  ❌ Fallidas: {failed}")
    print(f"  📊 Total: {len(demos)}")
    print()
    
    if success > 0:
        print("🎉 ¡Re-procesamiento completado!")
        print("💡 Los datos actualizados ahora incluyen:")
        print("   - Fecha real del partido")
        print("   - Campo Team asignado correctamente")
    else:
        print("⚠️  No se pudo re-procesar ninguna demo")
        print("   Verifica que el servicio Go esté corriendo en", GO_SERVICE_URL)

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠️  Proceso interrumpido por el usuario")
    except Exception as e:
        print(f"\n\n❌ Error fatal: {str(e)}")

