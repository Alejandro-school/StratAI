#!/usr/bin/env python3
"""
Script para limpiar Redis y reprocesar demos
"""

import redis
import json
import sys

def main():
    # Conectar a Redis
    r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)
    
    # Steam ID de Kerchak
    steam_id = "76561198116485358"
    
    print(f"Limpiando Redis para steam_id: {steam_id}")
    print("=" * 60)
    
    # Limpiar datos antiguos
    key = f"processed_demos:{steam_id}"
    deleted = r.delete(key)
    print(f"Eliminadas {deleted} entradas antiguas de Redis")
    
    print("Redis limpiado. Ahora ejecuta el script de reprocesamiento de Go.")
    print("Luego ejecuta este script nuevamente para verificar.")

if __name__ == "__main__":
    main()
