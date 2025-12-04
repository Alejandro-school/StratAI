import requests
import json
from datetime import datetime

demo_file = 'data/demos/match_2ycniSRrujsiUfXJf2WqQ4zAC.dem'

with open(demo_file, 'rb') as f:
    resp = requests.post(
        'http://localhost:8080/process-demo',
        data=f,
        headers={'X-Match-ID': '2ycniSRrujsiUfXJf2WqQ4zAC'}
    )

print(f'Status: {resp.status_code}')

if resp.status_code == 200:
    data = resp.json()
    print(f"✅ Kills: {len(data.get('kills', []))}")
    print(f"✅ Rounds: {len(data.get('rounds', []))}")
    print(f"✅ Players: {len(data.get('players', {}))}")
    print(f"✅ Map: {data.get('map_name', 'N/A')}")
    print(f"✅ Score: {data.get('ct_score', 0)} - {data.get('t_score', 0)}")
    
    # Exportar JSON
    map_name = data.get('map_name', 'unknown')
    ct_score = data.get('ct_score', 0)
    t_score = data.get('t_score', 0)
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M")
    
    export_dir = f"data/exports/{map_name}_{ct_score}-{t_score}_{timestamp}"
    import os
    os.makedirs(export_dir, exist_ok=True)
    
    output_file = f"{export_dir}/match_data.json"
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
    
    print(f"📁 Exportado a: {output_file}")
else:
    print(f"❌ Error: {resp.text}")
