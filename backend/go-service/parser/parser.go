package parser

import (
	"fmt"
	"os"

	"cs2-demo-service/analyzers"
	"cs2-demo-service/handlers"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// ParseDemo es la función principal que procesa una demo completa
// Devuelve el contexto completo para poder exportar timeline
func ParseDemo(demoPath string) (*models.DemoContext, error) {
	// Abrir archivo demo
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open demo file: %w", err)
	}
	defer f.Close()

	// Crear parser
	p := dem.NewParser(f)
	defer p.Close()

	// Parse header to get map name
	_, err = p.ParseHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Crear contexto
	ctx := models.NewDemoContext(p)

	// Initialize Map Manager
	// Assuming maps are stored in backend/data/maps
	// We need to construct the absolute path or relative to execution
	// For now, hardcoded relative path
	mapManager := maps.NewMapManager("../data/maps")

	// Attempt to load the map
	mapName := p.Header().MapName
	if mapName != "" {
		_ = mapManager.LoadMap(mapName)
	} else {
		fmt.Println("Map name not found in header. Waiting for RoundStart event.")
	}

	ctx.MapManager = mapManager

	// Registrar todos los handlers
	handlers.RegisterTimelineHandlers(ctx) // NEW: Timeline & GameState sampling
	handlers.RegisterChatHandlers(ctx)     // NEW: Chat tracking
	handlers.RegisterPlayerHandlers(ctx)   // Includes: Movement, Weapon State, Spotting, Zones (Phase 1)
	handlers.RegisterCombatHandlers(ctx)
	handlers.RegisterGrenadeHandlers(ctx)
	handlers.RegisterRoundHandlers(ctx) // Includes: Zone reset (Phase 1)
	handlers.RegisterEconomyHandlers(ctx)
	handlers.RegisterBombHandlers(ctx)    // Includes: Defuse kit tracking (Phase 1)
	handlers.RegisterTrackingHandler(ctx) // NEW: AI Tracking (2Hz sampling)

	// NEW: Advanced Player Stats (Phase 1 AI)
	statsHandler := handlers.RegisterPlayerStatsHandler(ctx)

	// Register analyzers
	analyzers.RegisterSprayAnalyzer(ctx)
	analyzers.RegisterMechanicsAnalyzer(ctx) // NEW: Counter-Strafe & Mechanics
	// TODO: Reaction time analyzer deshabilitado - IsSpottedBy() no es confiable
	analyzers.RegisterReactionAnalyzer(ctx)
	analyzers.RegisterCrosshairAnalyzer(ctx)

	// Ensure map is loaded (sometimes header map name is empty in CS2)
	p.RegisterEventHandler(func(e events.RoundStart) {
		if mapManager.IsLoaded() {
			return
		}
		if p.Header().MapName != "" {
			_ = mapManager.LoadMap(p.Header().MapName)
		}
	})

	// Parsear hasta el final
	err = p.ParseToEnd()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Final Step: Collect aggregated player stats with combat metrics
	ctx.AI_PlayersSummary = statsHandler.GetStatsWithContext(ctx)

	// Construir output final
	matchData := BuildMatchData(ctx)
	ctx.MatchData = matchData

	return ctx, nil
}
