package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cs2-demo-service/db"
	"cs2-demo-service/demo"
	"cs2-demo-service/models"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🔄 Script de Re-procesamiento de Demos")
	fmt.Println(strings.Repeat("=", 60))

	// 1. Cargar variables de entorno
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("⚠️  No se pudo cargar .env, continuando sin él")
	}

	// 2. Inicializar Redis
	db.InitRedis()
	defer db.Rdb.Close()

	// 3. Solicitar Steam ID
	var steamID string
	fmt.Print("\n🔑 Ingresa tu Steam ID: ")
	fmt.Scanln(&steamID)

	if steamID == "" {
		log.Fatal("❌ Error: Steam ID requerido")
	}

	// 4. Obtener todas las demos del directorio
	demosDir := filepath.Join("..", "data", "demos")
	demos, err := filepath.Glob(filepath.Join(demosDir, "*.dem"))
	if err != nil {
		log.Fatalf("❌ Error buscando demos: %v", err)
	}

	if len(demos) == 0 {
		log.Fatal("❌ No se encontraron demos en " + demosDir)
	}

	fmt.Printf("\n📊 Se encontraron %d demos\n", len(demos))
	fmt.Println()

	// 5. Preguntar si desea limpiar datos anteriores
	var cleanOld string
	fmt.Print("🗑️  ¿Deseas limpiar los datos anteriores en Redis? (s/n): ")
	fmt.Scanln(&cleanOld)

	if cleanOld == "s" || cleanOld == "S" {
		key := fmt.Sprintf("processed_demos:%s", steamID)
		deleted, err := db.Rdb.Del(context.Background(), key).Result()
		if err != nil {
			log.Printf("⚠️  Error limpiando Redis: %v", err)
		} else {
			fmt.Printf("✅ Se eliminaron %d entradas anteriores\n", deleted)
		}
	}

	fmt.Println("\n🚀 Iniciando re-procesamiento...")
	fmt.Println()

	// 6. Procesar cada demo
	success := 0
	failed := 0

	for i, demoPath := range demos {
		filename := filepath.Base(demoPath)
		fmt.Printf("[%d/%d] 📦 %s\n", i+1, len(demos), filename)

		// Procesar la demo (versión básica para el frontend)
		basicResult, err := demo.ProcessDemoFileBasic(demoPath, steamID, "{}")
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			failed++
			continue
		}

		// Calcular el matchID
		matchID, err := demo.ComputeFileHash(demoPath)
		if err != nil {
			fmt.Printf("  ❌ Error calculando hash: %v\n\n", err)
			failed++
			continue
		}

		basicResult.MatchID = matchID

		// Guardar en Redis
		key := fmt.Sprintf("processed_demos:%s", steamID)
		data, err := json.Marshal(basicResult)
		if err != nil {
			fmt.Printf("  ❌ Error serializando: %v\n\n", err)
			failed++
			continue
		}

		// Verificar si ya existe para evitar duplicados
		exists := false
		entries, _ := db.Rdb.LRange(context.Background(), key, 0, -1).Result()
		for _, e := range entries {
			var existing models.BasicDemoParseResult
			if json.Unmarshal([]byte(e), &existing) == nil {
				if existing.MatchID == matchID {
					exists = true
					break
				}
			}
		}

		if !exists {
			err = db.Rdb.RPush(context.Background(), key, string(data)).Err()
			if err != nil {
				fmt.Printf("  ❌ Error guardando en Redis: %v\n\n", err)
				failed++
				continue
			}
		}

		fmt.Printf("  ✅ Procesada exitosamente\n")
		fmt.Printf("     • Mapa: %s\n", basicResult.MapName)
		fmt.Printf("     • Fecha: %s\n", basicResult.Date)
		fmt.Printf("     • Resultado: %s (%d - %d)\n",
			basicResult.Result, basicResult.TeamScore, basicResult.OpponentScore)
		fmt.Printf("     • Jugadores: %d\n\n", len(basicResult.Players))

		success++

		// Pausa para no sobrecargar
		time.Sleep(100 * time.Millisecond)
	}

	// 7. Resumen final
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📈 Resumen del Re-procesamiento")
	fmt.Printf("  ✅ Exitosas: %d\n", success)
	fmt.Printf("  ❌ Fallidas: %d\n", failed)
	fmt.Printf("  📊 Total: %d\n", len(demos))
	fmt.Println()

	if success > 0 {
		fmt.Println("🎉 ¡Re-procesamiento completado!")
		fmt.Println("💡 Los datos actualizados ahora incluyen:")
		fmt.Println("   - Fecha real del partido (del archivo .dem)")
		fmt.Println("   - Campo Team asignado correctamente (CT/T)")
		fmt.Println()
		fmt.Println("🔄 Recarga el frontend para ver los cambios")
	} else {
		fmt.Println("⚠️  No se pudo re-procesar ninguna demo")
	}
}

