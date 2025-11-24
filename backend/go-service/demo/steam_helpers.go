package demo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// SteamPlayerData representa la respuesta de Steam API
type SteamPlayerData struct {
	SteamID    string `json:"steamid"`
	AvatarFull string `json:"avatarfull"`
	PersonName string `json:"personaname"`
}

// SteamAPIResponse estructura de respuesta de Steam API
type SteamAPIResponse struct {
	Response struct {
		Players []SteamPlayerData `json:"players"`
	} `json:"response"`
}

// GetSteamAvatars obtiene avatares de múltiples jugadores en paralelo
// Usa goroutines para llamar a Steam API con timeout de 3s por jugador
// Retorna un map[steamID]avatarURL con los avatares obtenidos exitosamente
func GetSteamAvatars(steamIDs []string) map[string]string {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  STEAM_API_KEY no configurada, avatares desactivados")
		return make(map[string]string)
	}

	avatars := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	startTime := time.Now()
	log.Printf("🔄 Obteniendo avatares para %d jugadores...", len(steamIDs))

	// Procesar en paralelo con timeout individual
	for _, steamID := range steamIDs {
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()

			url := fmt.Sprintf(
				"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s",
				apiKey, sid,
			)

			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				log.Printf("⚠️  Error obteniendo avatar para %s: %v", sid, err)
				return
			}
			defer resp.Body.Close()

			var result SteamAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				log.Printf("⚠️  Error decodificando respuesta para %s: %v", sid, err)
				return
			}

			if len(result.Response.Players) > 0 {
				mu.Lock()
				avatars[sid] = result.Response.Players[0].AvatarFull
				mu.Unlock()
				log.Printf("✅ Avatar obtenido para %s", result.Response.Players[0].PersonName)
			}
		}(steamID)
	}

	// Esperar con timeout global de 5s
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(startTime)
		log.Printf("✅ Avatares obtenidos: %d/%d en %v", len(avatars), len(steamIDs), elapsed)
	case <-time.After(5 * time.Second):
		log.Printf("⏱️  Timeout global alcanzado, obtenidos: %d/%d", len(avatars), len(steamIDs))
	}

	return avatars
}
