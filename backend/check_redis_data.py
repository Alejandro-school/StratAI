#!/usr/bin/env python3
"""
Script para verificar qué datos hay en Redis para el usuario Kerchak
"""

import redis
import json
import sys

def main():
    # Conectar a Redis
    r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)
    
    # Steam ID de Kerchak
    steam_id = "76561198116485358"
    
    print(f"Verificando datos en Redis para steam_id: {steam_id}")
    print("=" * 60)
    
    # Obtener las demos procesadas
    key = f"processed_demos:{steam_id}"
    demos = r.lrange(key, 0, -1)
    
    if not demos:
        print("ERROR: No hay demos en Redis para este usuario")
        return
    
    print(f"Encontradas {len(demos)} demos en Redis")
    print()
    
    # Analizar cada demo
    for i, demo_json in enumerate(demos):
        try:
            demo = json.loads(demo_json)
            print(f"Demo {i+1}:")
            print(f"  - Match ID: {demo.get('match_id', 'N/A')}")
            print(f"  - Map: {demo.get('map_name', 'N/A')}")
            print(f"  - Players: {len(demo.get('players', []))}")
            
            # Verificar event_logs
            event_logs = demo.get('event_logs', [])
            print(f"  - event_logs type: {type(event_logs)}")
            print(f"  - event_logs length: {len(event_logs) if isinstance(event_logs, list) else 'not list'}")
            
            if isinstance(event_logs, list) and len(event_logs) > 0:
                print(f"  - Primer evento: {event_logs[0]}")
                
                # Contar eventos de kill
                kill_events = [e for e in event_logs if e.get('event_type') == 'Kill']
                print(f"  - Eventos de Kill: {len(kill_events)}")
                
                if len(kill_events) > 0:
                    print(f"  - Primer Kill: {kill_events[0]}")
            else:
                print("  - ERROR: No hay event_logs o están vacíos")
            
            print()
            
        except Exception as e:
            print(f"ERROR procesando demo {i+1}: {e}")
            print()

if __name__ == "__main__":
    main()
