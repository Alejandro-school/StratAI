"""Reprocesar todas las demos"""
import requests, os, time

demos_dir = "data/demos"
demos = [f for f in os.listdir(demos_dir) if f.endswith(".dem")]

print(f"📊 Encontradas {len(demos)} demos para reprocesar\n")

for i, demo_file in enumerate(demos, 1):
    demo_path = os.path.abspath(os.path.join(demos_dir, demo_file))
    match_id = demo_file.replace("match_", "").replace(".dem", "")
    
    print(f"[{i}/{len(demos)}] Procesando {match_id[:20]}...", end=" ", flush=True)
    
    try:
        response = requests.post(
            "http://localhost:8080/process-demo",
            json={"demo_path": demo_path, "steam_id": "76561198116485358", "match_id": match_id},
            timeout=180
        )
        
        if response.status_code == 200:
            data = response.json()
            if data.get("status") == "success":
                result = data.get("data", {})
                print(f"✅ {result.get('result', '?')} {result.get('team_score', 0)}-{result.get('opponent_score', 0)}")
            else:
                print(f"⚠️ {data.get('message', 'Unknown error')}")
        else:
            print(f"❌ HTTP {response.status_code}")
    except requests.Timeout:
        print(f"❌ Timeout")
    except Exception as e:
        print(f"❌ {str(e)[:50]}")
    
    time.sleep(0.5)

print("\n✅ Reprocesamiento completado!")
