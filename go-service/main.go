package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big" // <-- Añadido para decodificar en base 62 sin overflow
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// DecodedShareCode mantiene los datos más importantes que suelen extraerse
// del share code: matchID, outcomeID y token.
type DecodedShareCode struct {
	MatchID   uint64
	OutcomeID uint32
	Token     uint32
}

// MatchStats representa la respuesta final que devolvemos al front,
// una vez consultada la API de Steam con el match_id decodificado.
type MatchStats struct {
	MatchID   string `json:"match_id"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Assists   int    `json:"assists"`
	Headshots int    `json:"headshots"`
	Rounds    int    `json:"rounds_played"`
}

// -----------------------------------------------------
// CARGA DE VARIABLES DE ENTORNO
// -----------------------------------------------------
func loadEnv() {
	err := godotenv.Load("../backend/.env") // Ajusta la ruta a tu .env
	if err != nil {
		fmt.Println("❌ Error al cargar el archivo .env:", err)
	} else {
		fmt.Println("✅ Variables de entorno cargadas correctamente")
	}
}

// -----------------------------------------------------
// DECODIFICADOR REAL DEL SHARE CODE (usando math/big)
// -----------------------------------------------------
func decodeShareCode(shareCode string) (*DecodedShareCode, error) {
	// 1. Asegúrate de quitar el prefijo CSGO- y los guiones
	cleaned := strings.ReplaceAll(shareCode, "CSGO-", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	if len(cleaned) == 0 {
		return nil, errors.New("share code vacío o inválido")
	}

	// 2. Decodificar el cleaned string de base 62 en un número grande (big.Int)
	codeBig, err := decodeBase62(cleaned)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar base62: %v", err)
	}

	// 3. Verificar que codeBig quepa en 64 bits
	maxUint64 := new(big.Int).SetUint64(^uint64(0)) // 2^64 - 1
	if codeBig.Cmp(maxUint64) > 0 {
		fmt.Println("⚠️ El número es más grande de 64 bits, pero continuamos...")
	}

	// Pasar de big.Int a uint64 una vez que ya confirmamos que no excede 64 bits
	codeValue := codeBig.Uint64()

	// 4. Extraer token, outcomeId, matchId de los bits correspondientes:
	//    - bits [0..5]   = token (6 bits)
	//    - bits [6..37]  = outcomeId (32 bits)
	//    - bits [38..63] = matchId (26 bits)
	token := uint32(codeValue & 0x3F)                  // 0x3F = 63 decimal => bits [0..5]
	outcomeId := uint32((codeValue >> 6) & 0xFFFFFFFF) // bits [6..37]
	matchId := uint64(codeValue >> 38)                 // bits [38..63]

	decoded := &DecodedShareCode{
		MatchID:   matchId,
		OutcomeID: outcomeId,
		Token:     token,
	}
	return decoded, nil
}

// decodeBase62 convierte un string con caracteres [A-Z, a-z, 0-9] a un *big.Int,
// sin riesgo de overflow. Luego, más adelante, verificamos si cabe en 64 bits.
func decodeBase62(s string) (*big.Int, error) {
	base := big.NewInt(62)
	result := big.NewInt(0)

	for _, char := range s {
		val, err := charToValue(char)
		if err != nil {
			return nil, err
		}
		// result = result*62 + val (en enteros arbitrarios)
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(val)))
	}
	return result, nil
}

// charToValue convierte un carácter [A-Z, a-z, 0-9] a su valor base 62.
func charToValue(ch rune) (int, error) {
	switch {
	case ch >= 'A' && ch <= 'Z':
		return int(ch - 'A'), nil // 0..25
	case ch >= 'a' && ch <= 'z':
		return int(ch-'a') + 26, nil // 26..51
	case ch >= '0' && ch <= '9':
		return int(ch-'0') + 52, nil // 52..61
	}
	return 0, fmt.Errorf("carácter inválido en el share code: %c", ch)
}

// extractMatchID es la función que tu código ya llama.
// Internamente usa decodeShareCode() para sacar el matchID real.
func extractMatchID(shareCode string) (string, error) {
	decoded, err := decodeShareCode(shareCode)
	if err != nil {
		return "", fmt.Errorf("❌ Share Code inválido o error al decodificar: %v", err)
	}
	// Devolvemos el matchID como string (lo que tu getMatchDetails() espera).
	return strconv.FormatUint(decoded.MatchID, 10), nil
}

// -----------------------------------------------------
// EJEMPLO DE CONSULTA A LA API DE STEAM (GETMATCHDETAILS)
// -----------------------------------------------------
func getMatchDetails(matchID string) (*MatchStats, error) {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("❌ STEAM_API_KEY no configurada en .env")
	}

	// Ejemplo de endpoint para obtener detalles de la partida (CS2/CS:GO).
	url := fmt.Sprintf("https://api.steampowered.com/ICSGOPlayers_730/GetMatchDetails/v1/?key=%s&match_id=%s", apiKey, matchID)

	fmt.Println("🔎 Consultando detalles de la partida:", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("❌ Error al conectar con la API de Steam: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("❌ Error HTTP %d al obtener datos de Steam", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("❌ Error al procesar respuesta de la API de Steam: %v", err)
	}

	// Ten en cuenta que la estructura real del JSON de la API puede diferir.
	// Ajusta este acceso a datos según tu caso de uso.
	roundstats, ok := result["result"].(map[string]interface{})["roundstats"].(map[string]interface{})
	if !ok {
		return nil, errors.New("❌ No se encontraron roundstats en la respuesta de Steam")
	}

	stats := &MatchStats{
		MatchID:   matchID,
		Kills:     int(roundstats["kills"].(float64)),
		Deaths:    int(roundstats["deaths"].(float64)),
		Assists:   int(roundstats["assists"].(float64)),
		Headshots: int(roundstats["headshots"].(float64)),
		Rounds:    int(roundstats["roundsplayed"].(float64)),
	}

	return stats, nil
}

// -----------------------------------------------------
// HANDLER PRINCIPAL: /process-demo
// -----------------------------------------------------
func handleProcessDemo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShareCode string `json:"share_code"`
		SteamID   string `json:"steam_id"` // si deseas capturar el steam_id, está aquí
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "❌ Error al leer el cuerpo JSON", http.StatusBadRequest)
		return
	}

	// 1. Decodificar el share code para obtener el matchID.
	matchID, err := extractMatchID(req.ShareCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("❌ Error al decodificar el Share Code: %v", err),
			http.StatusInternalServerError)
		return
	}

	// 2. Obtener estadísticas reales de la partida desde la API de Steam.
	stats, err := getMatchDetails(matchID)
	if err != nil {
		http.Error(w, fmt.Sprintf("❌ Error al obtener detalles de la partida: %v", err),
			http.StatusInternalServerError)
		return
	}

	// 3. Devolver estadísticas al frontend en JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func main() {
	loadEnv()
	http.HandleFunc("/process-demo", handleProcessDemo)

	fmt.Println("✅ Servidor Go ejecutándose en el puerto 8080...")
	http.ListenAndServe(":8080", nil)
}
