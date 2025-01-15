# sharecode_decoder.py

ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

def decode_sharecode(sharecode: str):
    # 1) Quitar prefijo y guiones
    if not sharecode.startswith("CSGO-"):
        raise ValueError("❌ El ShareCode debe comenzar con 'CSGO-'.")
    cleaned = sharecode.replace("CSGO-", "").replace("-", "")

    # 2) Interpretar como número en base 64 (con el alfabeto de CSGO)
    value = 0
    # Puedes iterar de adelante hacia atrás o al revés, pero ojo al extraer bits
    # Aquí lo hago de atrás hacia delante
    for c in reversed(cleaned):
        # Encuentra la posición del carácter en el alfabeto
        index = ALPHABET.index(c)
        value = value * 64 + index

    # 3) Extraer cada campo con bitshifting
    #    - token = últimos 16 bits
    #    - outcome_id = siguientes 32 bits
    #    - match_id = siguientes 48 bits
    token = value & 0xFFFF               # 16 bits
    value >>= 16

    outcome_id = value & 0xFFFFFFFF      # 32 bits
    value >>= 32

    match_id = value & 0xFFFFFFFFFFFF    # 48 bits

    return {
        "match_id": match_id,
        "outcome_id": outcome_id,
        "token": token
    }
