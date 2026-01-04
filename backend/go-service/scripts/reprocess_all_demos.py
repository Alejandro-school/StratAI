#!/usr/bin/env python3
"""
Script para reprocesar todas las demos existentes con el servicio Go actualizado.
Genera players_summary.json para todas las demos sin necesidad de re-descargarlas.
"""

import os
import json
import requests
import time
from pathlib import Path
from colorama import init, Fore, Style

# Inicializar colorama para Windows
init(autoreset=True)

# Configuración
GO_SERVICE_URL = "http://localhost:8080"
DEMOS_DIR = Path("../data/demos")
EXPORTS_DIR = Path("../data/exports")

def print_header():
    """Imprime el encabezado del script"""
    print(Fore.CYAN + "=" * 50)
    print(Fore.CYAN + "  Reprocesamiento de Demos Existentes")
    print(Fore.CYAN + "=" * 50)
    print()

def check_go_service():
    """Verifica que el servicio Go esté corriendo"""
    print(Fore.YELLOW + "🔍 Verificando servicio Go...")
    try:
        response = requests.get(f"{GO_SERVICE_URL}/health", timeout=3)
        if response.status_code == 200:
            print(Fore.GREEN + "✅ Servicio Go está activo")
            return True
    except requests.exceptions.RequestException:
        pass
    
    print(Fore.RED + f"❌ Error: El servicio Go no está corriendo en {GO_SERVICE_URL}")
    print(Fore.YELLOW + "   Por favor, inicia el servicio Go primero:")
    print(Fore.CYAN + "   cd go-service && go run main.go")
    return False

def get_demo_files():
    """Obtiene la lista de archivos de demos"""
    if not DEMOS_DIR.exists():
        print(Fore.RED + f"❌ Error: No se encontró el directorio de demos: {DEMOS_DIR}")
        return []
    
    demo_files = sorted(DEMOS_DIR.glob("*.dem"))
    return demo_files

def extract_match_id(demo_filename):
    """Extrae el match_id del nombre del archivo de demo"""
    # Formato esperado: match_XXXXX.dem
    name = demo_filename.stem  # Obtiene el nombre sin extensión
    if name.startswith("match_"):
        return name
    return None

def get_match_date(match_id):
    """Obtiene la fecha del match desde metadata.json si existe"""
    metadata_path = EXPORTS_DIR / match_id / "metadata.json"
    if metadata_path.exists():
        try:
            with open(metadata_path, 'r', encoding='utf-8') as f:
                metadata = json.load(f)
                return metadata.get('date', '')
        except Exception:
            pass
    return ''

def process_demo(demo_path, match_id, match_date):
    """Procesa una demo usando el servicio Go"""
    request_body = {
        "demo_path": str(demo_path.absolute()),
        "match_id": match_id,
        "match_date": match_date,
        "match_duration": 0
    }
    
    try:
        response = requests.post(
            f"{GO_SERVICE_URL}/process-demo",
            json=request_body,
            timeout=120
        )
        response.raise_for_status()
        return True, response.json()
    except requests.exceptions.RequestException as e:
        return False, str(e)

def main():
    """Función principal"""
    print_header()
    
    # Verificar servicio Go
    if not check_go_service():
        return 1
    
    print()
    
    # Obtener demos
    demo_files = get_demo_files()
    if not demo_files:
        print(Fore.YELLOW + "⚠️  No se encontraron archivos .dem")
        return 0
    
    total = len(demo_files)
    print(Fore.CYAN + f"📁 Demos encontradas: {total}")
    print()
    
    # Contadores
    processed = 0
    failed = 0
    skipped = 0
    
    # Procesar cada demo
    for idx, demo_path in enumerate(demo_files, 1):
        demo_name = demo_path.name
        
        # Extraer match_id
        match_id = extract_match_id(demo_path)
        if not match_id:
            print(Fore.YELLOW + f"⚠️  [{demo_name}] Formato de nombre no válido, saltando...")
            skipped += 1
            continue
        
        # Verificar si ya existe players_summary.json
        export_dir = EXPORTS_DIR / match_id
        players_summary_path = export_dir / "players_summary.json"
        
        if players_summary_path.exists():
            print(Fore.WHITE + f"⏭️  [{idx}/{total}] {match_id} - YA TIENE players_summary.json")
            skipped += 1
            continue
        
        # Determinar status
        status = "REPROCESANDO" if export_dir.exists() else "NUEVO"
        print(Fore.CYAN + f"🔄 [{idx}/{total}] {match_id} - {status}")
        
        # Obtener fecha del match
        match_date = get_match_date(match_id)
        
        # Procesar demo
        success, result = process_demo(demo_path, match_id, match_date)
        
        if success:
            # Verificar que se generó players_summary.json
            time.sleep(0.5)  # Dar tiempo para que se escriba el archivo
            if players_summary_path.exists():
                print(Fore.GREEN + "   ✅ Procesada correctamente - players_summary.json generado")
                processed += 1
            else:
                print(Fore.YELLOW + "   ⚠️  Procesada pero no se generó players_summary.json")
                processed += 1
        else:
            print(Fore.RED + f"   ❌ Error procesando: {result}")
            failed += 1
        
        print()
    
    # Resumen final
    print(Fore.CYAN + "=" * 50)
    print(Fore.CYAN + "           RESUMEN FINAL")
    print(Fore.CYAN + "=" * 50)
    print(Fore.GREEN + f"✅ Procesadas exitosamente: {processed} / {total}")
    print(Fore.WHITE + f"⏭️  Saltadas (ya tenían datos): {skipped}")
    print(Fore.RED + f"❌ Fallidas: {failed}")
    print()
    
    if processed > 0:
        print(Fore.GREEN + "🎉 ¡Reprocesamiento completado!")
        print(Fore.CYAN + "   Ahora todas las demos deberían tener players_summary.json")
        print(Fore.CYAN + "   con estadísticas avanzadas (HLTV Rating, KAST, Entry, etc.)")
    
    return 0

if __name__ == "__main__":
    try:
        exit(main())
    except KeyboardInterrupt:
        print("\n" + Fore.YELLOW + "⚠️  Proceso interrumpido por el usuario")
        exit(1)
