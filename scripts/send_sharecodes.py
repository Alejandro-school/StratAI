import requests

# URL del backend
url = "http://localhost:8000/steam/process-demos"

# ShareCode
payload = {
    "sharecodes": ["CSGO-NJMCc-iL42b-xKScd-o6Y9L-CnF8D"]
}

# 🚀 La cookie de sesión obtenida tras iniciar sesión en Steam
cookies = {
    "session": "eyJzdGVhbV9pZCI6ICI3NjU2MTE5ODExNjQ4NTM1OCJ9.Z4XDCg.GXiTpSq2pjdMLvlCkhv4t999XvQ"  # 🔑 Reemplaza esto con el valor real de la cookie de sesión
}

headers = {
    "Content-Type": "application/json"
}

try:
    # Enviar la solicitud con la cookie de sesión
    response = requests.post(url, json=payload, headers=headers, cookies=cookies)

    if response.status_code == 200:
        print("✅ Demos procesadas correctamente:")
        print(response.json())
    else:
        print(f"❌ Error {response.status_code}: {response.text}")

except requests.RequestException as e:
    print(f"❌ Error en la conexión: {e}")
