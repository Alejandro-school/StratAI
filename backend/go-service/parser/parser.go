package parser

import (
	"fmt"
	"os"

	"cs2-demo-service/analyzers"
	"cs2-demo-service/handlers"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// ParseDemoResult contains the parsing results including replay data
type ParseDemoResult struct {
	Context    *models.DemoContext
	ReplayData *models.ReplayData
}

// ParseDemo es la función principal que procesa una demo completa
// Devuelve el contexto completo para poder exportar timeline
func ParseDemo(demoPath string) (*models.DemoContext, error) {
	result, err := ParseDemoWithReplay(demoPath)
	if err != nil {
		return nil, err
	}
	return result.Context, nil
}

// ParseDemoWithReplay parses a demo and returns full results including replay data
func ParseDemoWithReplay(demoPath string) (*ParseDemoResult, error) {
	// Abrir archivo demo
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open demo file: %w", err)
	}
	defer f.Close()

	// Crear parser
	p := dem.NewParser(f)
	defer p.Close()

	// Crear contexto
	ctx := models.NewDemoContext(p)

	// Initialize Map Manager
	// Assuming maps are stored in backend/data/maps
	// We need to construct the absolute path or relative to execution
	// For now, hardcoded relative path
	mapManager := maps.NewMapManager("../data/maps")

	loadMap := func(mapName string) {
		if mapName == "" {
			return
		}
		ctx.MapName = mapName
		if !mapManager.IsLoaded() {
			_ = mapManager.LoadMap(mapName)
		}
	}
	p.RegisterNetMessageHandler(func(header *msg.CDemoFileHeader) {
		loadMap(header.GetMapName())
	})
	p.RegisterNetMessageHandler(func(serverInfo *msg.CSVCMsg_ServerInfo) {
		loadMap(serverInfo.GetMapName())
	})

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

	// NEW: 2D Replay data collection
	replayHandler := handlers.RegisterReplayHandlers(ctx)

	// NEW: Advanced Player Stats (Phase 1 AI)
	statsHandler := handlers.RegisterPlayerStatsHandler(ctx)

	// Register analyzers
	analyzers.RegisterSprayAnalyzer(ctx)
	analyzers.RegisterMechanicsAnalyzer(ctx) // NEW: Counter-Strafe & Mechanics
	// TODO: Reaction time analyzer deshabilitado - IsSpottedBy() no es confiable
	analyzers.RegisterReactionAnalyzer(ctx)
	analyzers.RegisterCrosshairAnalyzer(ctx)

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

	// Build replay data and attach to context (placeholder matchID, will be set on export)
	replayData := replayHandler.GetReplayData("")
	ctx.ReplayData = &replayData

	return &ParseDemoResult{
		Context:    ctx,
		ReplayData: &replayData,
	}, nil
}
