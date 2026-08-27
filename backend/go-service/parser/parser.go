package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"cs2-demo-service/analyzers"
	"cs2-demo-service/handlers"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
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
	p.RegisterEventHandler(func(e events.ParserWarn) {
		ctx.ParserWarnings = append(ctx.ParserWarnings, fmt.Sprintf("type=%d: %s", e.Type, e.Message))
	})

	mapManager := maps.NewMapManager(resolveMapsDir())
	lastMapAttempt := ""

	loadMap := func(mapName string) {
		if mapName == "" {
			return
		}
		ctx.MapName = mapName
		if !mapManager.IsLoaded() && lastMapAttempt != mapName {
			lastMapAttempt = mapName
			if err := mapManager.LoadMap(mapName); err != nil {
				ctx.MapLoadError = err.Error()
			} else {
				ctx.MapLoadError = ""
			}
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
	handlers.RegisterPlayerObservationHandler(ctx)
	handlers.RegisterTimelineHandlers(ctx) // NEW: Timeline & GameState sampling
	handlers.RegisterChatHandlers(ctx)     // NEW: Chat tracking
	handlers.RegisterPlayerHandlers(ctx)   // Includes: Movement, Weapon State, Spotting, Zones (Phase 1)
	handlers.RegisterCombatHandlers(ctx)
	utilityHandler := handlers.RegisterGrenadeHandlers(ctx)
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
	analyzers.RegisterReactionAnalyzer(ctx)

	// Parsear hasta el final
	err = p.ParseToEnd()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}
	utilityHandler.Finalize()
	replayHandler.Finalize()
	ctx.ParseCompleted = true

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

func resolveMapsDir() string {
	if configured := os.Getenv("CS2_MAPS_DIR"); configured != "" {
		return configured
	}

	candidates := []string{
		filepath.Join("..", "data", "maps"),
		filepath.Join("backend", "data", "maps"),
		filepath.Join("data", "maps"),
	}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(dir, "..", "data", "maps"),
			filepath.Join(dir, "data", "maps"),
		)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			absolute, err := filepath.Abs(candidate)
			if err == nil {
				return absolute
			}
			return candidate
		}
	}
	return candidates[0]
}
