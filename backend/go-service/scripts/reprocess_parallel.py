#!/usr/bin/env python3
"""
Script para reprocesar TODAS las demos con CONCURRENCIA.
Mantiene los matchIDs existentes para no perder asociación con usuarios.

USO:
  cd backend/go-service/scripts
  python reprocess_parallel.py

REQUISITOS:
  pip install requests colorama

NOTA: Requiere que el servicio Go esté corriendo en localhost:8080
"""

import os
import json
import requests
import time
import concurrent.futures
from pathlib import Path
from threading import Lock
from colorama import init, Fore, Style

# Inicializar colorama para Windows
init(autoreset=True)

# Configuración
GO_SERVICE_URL = "http://localhost:8080"
DEMOS_DIR = Path(__file__).parent.parent.parent / "data" / "demos"
EXPORTS_DIR = Path(__file__).parent.parent.parent / "data" / "exports"

# Concurrencia - 6 workers para 5800X3D (deja 2 cores libres)
MAX_WORKERS = 6

# Estado global con lock para thread safety
stats_lock = Lock()
stats = {
    "processed": 0,
    "failed": 0,
    "skipped": 0,
    "total": 0
}


def print_header():
    """Imprime el encabezado del script"""
    print(Fore.CYAN + "=" * 60)
    print(Fore.CYAN + "  Reprocesamiento PARALELO de Demos")
    print(Fore.CYAN + f"  Workers: {MAX_WORKERS} | CPU: AMD 5800X3D")
    print(Fore.CYAN + "=" * 60)
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
    print(Fore.CYAN + "   cd backend/go-service && go run main.go")
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
    name = demo_filename.stem
    if name.startswith("match_"):
        return name
    return None


def get_existing_metadata(match_id):
    """Obtiene metadata existente (fecha, steam_id) para mantener asociaciones"""
    metadata_path = EXPORTS_DIR / match_id / "metadata.json"
    if metadata_path.exists():
        try:
            with open(metadata_path, 'r', encoding='utf-8') as f:
                return json.load(f)
        except Exception:
            pass
    return {}


def process_single_demo(demo_info):
    """
    Procesa una sola demo. Diseñado para ejecutarse en paralelo.
    Mantiene el matchID existente para preservar asociación con usuario.
    """
    idx, demo_path, total = demo_info
    demo_name = demo_path.name
    
    # Extraer match_id del nombre del archivo
    match_id = extract_match_id(demo_path)
    if not match_id:
        with stats_lock:
            stats["skipped"] += 1
        return f"⚠️  [{idx}/{total}] {demo_name} - Nombre inválido, saltado"
    
    # Obtener metadata existente para mantener fecha
    existing_metadata = get_existing_metadata(match_id)
    match_date = existing_metadata.get('date', '')
    
    # Preparar request - IMPORTANTE: usamos el mismo match_id
    request_body = {
        "demo_path": str(demo_path.absolute()),
        "match_id": match_id,  # Mantiene el ID original
        "match_date": match_date,
        "match_duration": 0
    }
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{GO_SERVICE_URL}/process-demo",
            json=request_body,
            timeout=300  # 5 minutos por demo
        )
        response.raise_for_status()
        elapsed = time.time() - start_time
        
        with stats_lock:
            stats["processed"] += 1
        
        return f"✅ [{idx}/{total}] {match_id} - Procesada en {elapsed:.1f}s"
        
    except requests.exceptions.RequestException as e:
        with stats_lock:
            stats["failed"] += 1
        return f"❌ [{idx}/{total}] {match_id} - Error: {str(e)[:50]}"


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
    stats["total"] = total
    
    print(Fore.CYAN + f"📁 Demos encontradas: {total}")
    print(Fore.CYAN + f"🔄 Procesando con {MAX_WORKERS} workers en paralelo...")
    print(Fore.YELLOW + "⚠️  Esto puede tardar varios minutos. No cierres esta ventana.")
    print()
    
    # Preparar lista de trabajos
    jobs = [(idx, demo_path, total) for idx, demo_path in enumerate(demo_files, 1)]
    
    # Procesar en paralelo
    start_time = time.time()
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        # Enviar todos los trabajos
        futures = {executor.submit(process_single_demo, job): job for job in jobs}
        
        # Procesar resultados a medida que completan
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            print(result)
    
    elapsed_total = time.time() - start_time
    
    # Resumen final
    print()
    print(Fore.CYAN + "=" * 60)
    print(Fore.CYAN + "                    RESUMEN FINAL")
    print(Fore.CYAN + "=" * 60)
    print(Fore.GREEN + f"✅ Procesadas exitosamente: {stats['processed']} / {total}")
    print(Fore.WHITE + f"⏭️  Saltadas: {stats['skipped']}")
    print(Fore.RED + f"❌ Fallidas: {stats['failed']}")
    print()
    print(Fore.CYAN + f"⏱️  Tiempo total: {elapsed_total:.1f}s ({elapsed_total/60:.1f} min)")
    print(Fore.CYAN + f"📊 Promedio por demo: {elapsed_total/total:.1f}s")
    print()
    
    if stats["processed"] > 0:
        print(Fore.GREEN + "🎉 ¡Reprocesamiento completado!")
        print(Fore.YELLOW + "")
        print(Fore.YELLOW + "⚠️  IMPORTANTE: Los matchIDs se mantuvieron.")
        print(Fore.YELLOW + "   Las demos siguen asociadas al usuario original.")
    
    return 0


if __name__ == "__main__":
    try:
        exit(main())
    except KeyboardInterrupt:
        print("\n" + Fore.YELLOW + "⚠️  Proceso interrumpido por el usuario")
        print(Fore.YELLOW + f"   Procesadas hasta ahora: {stats['processed']}")
        exit(1)
