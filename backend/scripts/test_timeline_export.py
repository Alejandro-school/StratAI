import requests
import os

# Demo a probar
demo_path = r"E:\Carpeta compartida\Proyecto IA\Proyecto IA\backend\data\demos\match_Y3mtCUnDHKbBcUyjpaZhutHrL.dem"
match_id = "Y3mtCUnDHKbBcUyjpaZhutHrL"

print(f"📦 Enviando demo {match_id} para procesar...")

with open(demo_path, 'rb') as f:
    demo_data = f.read()

response = requests.post(
    'http://localhost:8080/process-demo',
    data=demo_data,
    headers={'X-Match-ID': match_id}
)

if response.status_code == 200:
    print("✅ Demo procesada correctamente")
    print(f"📊 Response: {response.json()}")
else:
    print(f"❌ Error: {response.status_code}")
    print(response.text)

print("\n🔍 Verificando archivos exportados...")
export_dir = r"E:\Carpeta compartida\Proyecto IA\Proyecto IA\backend\data\exports"
match_dir = os.path.join(export_dir, f"match_{match_id}")

if os.path.exists(match_dir):
    print(f"✅ Directorio match encontrado: {match_dir}")
    
    # Verificar metadata
    metadata_path = os.path.join(match_dir, "metadata.json")
    if os.path.exists(metadata_path):
        print(f"✅ metadata.json existe")
        import json
        with open(metadata_path) as f:
            metadata = json.load(f)
            print(f"   Mapa: {metadata['map']}")
            print(f"   Rondas: {metadata['total_rounds']}")
            print(f"   Score: {metadata['ct_score']}-{metadata['t_score']}")
    
    # Verificar timeline
    timeline_dir = os.path.join(match_dir, "timeline")
    if os.path.exists(timeline_dir):
        timeline_files = os.listdir(timeline_dir)
        print(f"✅ Timeline exportada: {len(timeline_files)} archivos")
        
        # Leer primera ronda
        if timeline_files:
            first_round = os.path.join(timeline_dir, sorted(timeline_files)[0])
            with open(first_round) as f:
                round_data = json.load(f)
                print(f"   Ronda 1: {len(round_data['events'])} eventos")
                
                # Contar tipos de eventos
                event_types = {}
                for event in round_data['events']:
                    event_type = event['type']
                    event_types[event_type] = event_types.get(event_type, 0) + 1
                
                print(f"   Tipos: {event_types}")
    
    # Verificar categorías
    categories_dir = os.path.join(match_dir, "categories")
    if os.path.exists(categories_dir):
        category_files = os.listdir(categories_dir)
        print(f"✅ Categorías exportadas: {len(category_files)} archivos")
        for cat_file in category_files:
            cat_path = os.path.join(categories_dir, cat_file)
            with open(cat_path) as f:
                cat_data = json.load(f)
                print(f"   {cat_file}: {len(cat_data)} eventos")
else:
    print(f"❌ No se encontró directorio: {match_dir}")
