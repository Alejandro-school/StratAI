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
	result, err := ParseDemoWithReplayContacts(demoPath)
	if err != nil {
		return nil, err
	}
	return result.Context, nil
}

// ParseDemoWithReplay parses a demo and returns full results including replay data
func ParseDemoWithReplay(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 0)
}

// Runtime replay may use the explicitly labelled calculated contact extension.
// Canonical/support/economy candidate paths keep their historical replay inputs.
func ParseDemoWithReplayContacts(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObserverOptions(demoPath, 0, false, false, true)
}

// New opt-in information producer; historical replay/canonical paths stay unchanged.
func ParseDemoWithReplayUtilityInformation(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithUtilityOptions(demoPath, 0, false, false, true, true)
}

// Opt-in sound extension; historical producers and receipts stay unchanged.
func ParseDemoWithReplaySoundInformation(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithSoundOptions(demoPath, 0, false, false, true, true, true)
}

// ParseDemoWithEconomyObservation uses the same parser once, with an additional
// immutable economic observation ledger. Normal exports keep their old path.
func ParseDemoWithEconomyObservation(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 1)
}

func ParseDemoWithEconomyInitialFrontiers(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 2)
}

func ParseDemoWithEconomyNativeSources(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 3)
}

func ParseDemoWithEconomyRuleContext(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 4)
}

func ParseDemoWithEconomySessionPayloads(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 5)
}

func ParseDemoWithEconomyManifest(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 6)
}

func ParseDemoWithEconomyTransactionTrace(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 7)
}

func ParseDemoWithEconomyNativeIdentity(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 8)
}

func ParseDemoWithEconomyOwnership(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 9)
}

func ParseDemoWithEconomyEntries(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 10)
}

func ParseDemoWithEconomyCashflow(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithReplay(demoPath, 11)
}

func parseDemoWithReplay(demoPath string, economyVersion int) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, economyVersion, false, false)
}

func ParseDemoWithSupportObservations(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, false)
}

func ParseDemoWithSupportPhysical(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true)
}

func ParseDemoWithSupportObjective(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true)
}

func ParseDemoWithSupportC4(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true)
}

func ParseDemoWithSupportObjectiveClock(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true)
}

func ParseDemoWithSupportAttemptOrigin(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true, true)
}

func ParseDemoWithSupportObjectivePosition(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true, true, true)
}

func ParseDemoWithSupportNativeMotion(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true, true, true, true)
}

func ParseDemoWithSupportWeaponTiming(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true, true, true, true, true)
}

func ParseDemoWithSupportVisualContact(demoPath string) (*ParseDemoResult, error) {
	return parseDemoWithObservers(demoPath, 0, true, true, true, true, true, true, true, true, true, true)
}

func parseDemoWithObservers(demoPath string, economyVersion int, support, physical bool, objective ...bool) (*ParseDemoResult, error) {
	return parseDemoWithObserverOptions(demoPath, economyVersion, support, physical, false, objective...)
}

func parseDemoWithObserverOptions(demoPath string, economyVersion int, support, physical, calculatedVision bool, objective ...bool) (*ParseDemoResult, error) {
	return parseDemoWithUtilityOptions(demoPath, economyVersion, support, physical, calculatedVision, false, objective...)
}

func parseDemoWithUtilityOptions(demoPath string, economyVersion int, support, physical, calculatedVision, utilityVision bool, objective ...bool) (*ParseDemoResult, error) {
	return parseDemoWithSoundOptions(demoPath, economyVersion, support, physical, calculatedVision, utilityVision, false, objective...)
}

func parseDemoWithSoundOptions(demoPath string, economyVersion int, support, physical, calculatedVision, utilityVision, soundInformation bool, objective ...bool) (*ParseDemoResult, error) {
	// Abrir archivo demo
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open demo file: %w", err)
	}
	defer f.Close()

	// Crear parser
	var p dem.Parser = dem.NewParser(f)
	if economyVersion >= 8 {
		p = newEconomyOrderedParser(p)
	}
	defer p.Close()

	// Crear contexto
	ctx := models.NewDemoContext(p)
	ctx.CaptureCalculatedVision = calculatedVision
	ctx.CaptureUtilityVision = utilityVision
	ctx.CaptureSoundInformation = soundInformation
	if support {
		ctx.SupportObservations = &models.SupportObservations{Frames: []models.SupportFrame{}}
	}
	if physical {
		ctx.SupportPhysicalObservations = &models.SupportPhysicalObservations{Frames: []models.SupportPhysicalFrame{}}
	}
	if len(objective) > 0 && objective[0] {
		ctx.SupportObjectiveObservations = &models.SupportObjectiveObservations{Frames: []models.SupportObjectiveFrame{}, Events: []models.SupportObjectiveEvent{}}
	}
	if len(objective) > 1 && objective[1] {
		ctx.SupportC4Observations = &models.SupportC4Observations{Cuts: []models.SupportC4Cut{}, States: []models.SupportC4State{}}
		ctx.SupportInventoryObservations = &models.SupportInventoryObservations{Cuts: []models.SupportInventoryCut{}, States: []models.SupportInventoryState{}}
	}
	if len(objective) > 2 && objective[2] {
		ctx.SupportObjectiveClockObservations = &models.SupportObjectiveClockObservations{Frames: []models.SupportObjectiveClockFrame{}}
	}
	if len(objective) > 3 && objective[3] {
		ctx.SupportAttemptOriginObservations = &models.SupportAttemptOriginObservations{Frames: []models.SupportAttemptFrame{}, Cuts: []models.SupportAttemptCut{}, Deliveries: []models.SupportAttemptDelivery{}}
	}
	if len(objective) > 4 && objective[4] {
		ctx.SupportObjectivePositionObservations = &models.SupportObjectivePositionObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, Births: []models.SupportObjectivePositionBirth{}, States: []models.SupportObjectivePositionState{}, Cuts: []models.SupportObjectivePositionCut{}}
	}
	if len(objective) > 5 && objective[5] {
		ctx.SupportNativeMotionObservations = &models.SupportNativeMotionObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, States: []models.SupportNativeMotionState{}, Cuts: []models.SupportNativeMotionCut{}}
	}
	if len(objective) > 6 && objective[6] {
		ctx.SupportWeaponTimingObservations = &models.SupportWeaponTimingObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, States: []models.SupportWeaponTimingState{}, Cuts: []models.SupportWeaponTimingCut{}}
	}
	if len(objective) > 7 && objective[7] {
		ctx.SupportVisualObservations = &models.SupportVisualObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, States: []models.SupportVisualState{}, Cuts: []models.SupportVisualCut{}}
	}
	p.RegisterEventHandler(func(e events.ParserWarn) {
		ctx.ParserWarnings = append(ctx.ParserWarnings, fmt.Sprintf("type=%d: %s", e.Type, e.Message))
	})

	mapManager := maps.NewMapManager(resolveMapsDir())
	if ctx.CaptureCalculatedVision {
		mapManager = maps.NewReplayVisionMapManager(resolveMapsDir())
	}
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
	if economyVersion == 1 {
		handlers.RegisterEconomyObservationHandler(ctx)
	} else if economyVersion >= 2 {
		handlers.RegisterEconomyInitialFrontierHandler(ctx)
		if economyVersion >= 3 {
			handlers.RegisterEconomySourceHandler(ctx)
		}
		if economyVersion == 4 {
			handlers.RegisterEconomyRuleContextHandler(ctx)
		}
		if economyVersion == 5 {
			handlers.RegisterEconomySessionPayloadHandler(ctx)
		}
		if economyVersion >= 6 {
			handlers.RegisterEconomyManifestHandler(ctx)
		}
		if economyVersion >= 7 {
			handlers.RegisterEconomyTransactionTraceHandler(ctx)
		}
		if economyVersion == 8 {
			handlers.RegisterEconomyNativeIdentityHandler(ctx)
		}
		if economyVersion == 9 {
			handlers.RegisterEconomyOwnershipHandler(ctx)
		}
		if economyVersion == 10 {
			handlers.RegisterEconomyEntryHandler(ctx)
		}
		if economyVersion == 11 {
			handlers.RegisterEconomyCashflowHandler(ctx)
		}
	}
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
	if calculatedVision {
		for i := range replayData.Rounds {
			round := &replayData.Rounds[i]
			samples := ctx.CalculatedVisionSamples[round.Round]
			if samples == nil {
				samples = []models.ReplayVisionSample{}
			}
			round.CalculatedVision = &models.ReplayCalculatedVision{
				SchemaID: "stratai.calculated_vision@2", Producer: "go.reaction.world_occlusion_v2",
				SampleStride: 4, Samples: samples,
			}
		}
	}
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
