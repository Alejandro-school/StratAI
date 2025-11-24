#!/usr/bin/env python3
"""
Script para analizar eventos de reaction_analysis por ronda
"""
import json
import sys
from collections import defaultdict

def analyze_reaction_events(match_id):
    file_path = f"data/exports/{match_id}/combat_analytics.json"
    
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except FileNotFoundError:
        print(f"❌ No se encontró el archivo: {file_path}")
        return
    
    print(f"\n{'='*70}")
    print(f"📊 ANÁLISIS DE REACTION_ANALYSIS - Match: {match_id}")
    print(f"{'='*70}\n")
    
    for player in data['players']:
        player_name = player['name']
        steam_id = player['steam_id']
        reaction_data = player.get('reaction_analysis', {})
        events = reaction_data.get('events', [])
        
        print(f"\n🎮 Jugador: {player_name} (SteamID: {steam_id})")
        print(f"   Total eventos: {len(events)}")
        
        if not events:
            print(f"   ⚠️  NO HAY EVENTOS REGISTRADOS")
            continue
        
        # Contar eventos por ronda
        events_by_round = defaultdict(int)
        for event in events:
            events_by_round[event['round']] += 1
        
        # Mostrar distribución por ronda
        rounds = sorted(events_by_round.keys())
        print(f"   Rondas con eventos: {len(rounds)}")
        print(f"   Distribución por ronda:")
        
        for i in range(1, max(rounds) + 1):
            count = events_by_round.get(i, 0)
            if count == 0:
                print(f"      Ronda {i:2d}: {'❌ SIN EVENTOS':>20}")
            else:
                print(f"      Ronda {i:2d}: {f'✓ {count} eventos':>20}")
        
        # Estadísticas
        avg_reaction = reaction_data.get('avg_reaction_time_ms', 0)
        fastest = reaction_data.get('fastest_ms', 0)
        slowest = reaction_data.get('slowest_ms', 0)
        
        print(f"\n   Estadísticas:")
        print(f"      Promedio: {avg_reaction:.1f}ms")
        print(f"      Más rápido: {fastest}ms")
        print(f"      Más lento: {slowest}ms")
    
    print(f"\n{'='*70}\n")

if __name__ == "__main__":
    # Usar el match más reciente por defecto
    match_id = "3778738183655653726"
    
    if len(sys.argv) > 1:
        match_id = sys.argv[1]
    
    analyze_reaction_events(match_id)
