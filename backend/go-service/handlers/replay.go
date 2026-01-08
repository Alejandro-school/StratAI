package handlers

import (
	"cs2-demo-service/models"
	"math"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

const (
	// ReplaySampleRateHz is how many times per second we sample player positions
	// 16 Hz = 62.5ms between samples, enables hyper-fluid 60FPS playback
	// Frontend interpolates between these samples for smooth animation
	ReplaySampleRateHz = 16

	// ShotVisibilityTicks is how many ticks a shot line stays visible
	ShotVisibilityTicks = 32 // ~500ms at 64 tick - visible longer for better visualization

	// SmokeRadius is the standard smoke grenade radius in game units
	SmokeRadius = 144.0

	// TrajectoryHistorySize is max points to keep in projectile trajectory
	TrajectoryHistorySize = 20 // More points for smoother grenade curves
)

// ReplayHandler manages replay data collection
type ReplayHandler struct {
	ctx *models.DemoContext

	// Current round data
	currentRound *models.ReplayRoundData

	// Active projectiles (grenade ID -> projectile state)
	activeProjectiles map[int64]*models.ReplayProjectile

	// Active effects (smoke/inferno)
	activeSmokes   map[int64]*models.ReplayActiveEffect
	activeInfernos map[int64]*models.ReplayActiveEffect

	// Recent shots for visualization (cleared after N ticks)
	recentShots []shotWithTick

	// Bomb state
	bombState *models.ReplayBombState

	// Output: all rounds
	Rounds []models.ReplayRound

	// Last sample tick
	lastSampleTick int

	// Round phase tracking
	roundPhase        string // "freezetime", "playing", "postround", "none"
	freezeTimeEndTick int    // Tick when freeze time ends
	roundEndTick      int    // Tick when RoundEnd event fired
}

type shotWithTick struct {
	shot models.ReplayShot
	tick int
}

// NewReplayHandler creates a new replay handler
func NewReplayHandler(ctx *models.DemoContext) *ReplayHandler {
	return &ReplayHandler{
		ctx:               ctx,
		activeProjectiles: make(map[int64]*models.ReplayProjectile),
		activeSmokes:      make(map[int64]*models.ReplayActiveEffect),
		activeInfernos:    make(map[int64]*models.ReplayActiveEffect),
		recentShots:       []shotWithTick{},
		Rounds:            []models.ReplayRound{},
		roundPhase:        "none",
	}
}

// RegisterReplayHandlers registers all handlers needed for replay data collection
func RegisterReplayHandlers(ctx *models.DemoContext) *ReplayHandler {
	handler := NewReplayHandler(ctx)

	// ========================================
	// ROUND LIFECYCLE
	// ========================================

	// Round Start - just create the round, don't start sampling yet
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		handler.currentRound = &models.ReplayRoundData{
			Round:     ctx.ActualRoundNumber + 1, // 1-based round numbering
			StartTick: ctx.Parser.GameState().IngameTick(),
			Frames:    []models.ReplayFrame{},
			Events:    []models.ReplayEvent{},
		}

		// Reset active effects
		handler.activeSmokes = make(map[int64]*models.ReplayActiveEffect)
		handler.activeInfernos = make(map[int64]*models.ReplayActiveEffect)
		handler.activeProjectiles = make(map[int64]*models.ReplayProjectile)
		handler.recentShots = []shotWithTick{}

		// Initialize bomb state
		handler.bombState = &models.ReplayBombState{State: "carried"}

		// We're now in freeze time
		handler.roundPhase = "freezetime"
	})

	// Freeze Time End - start sampling from the last 2 seconds of freeze time
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		if handler.currentRound == nil {
			return
		}
		handler.freezeTimeEndTick = ctx.Parser.GameState().IngameTick()
		handler.roundPhase = "playing"
	})

	// Round End - mark end but continue sampling for post-round period
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		if handler.currentRound == nil {
			return
		}

		handler.currentRound.EndTick = ctx.Parser.GameState().IngameTick()
		handler.roundEndTick = ctx.Parser.GameState().IngameTick()

		switch e.Winner {
		case common.TeamCounterTerrorists:
			handler.currentRound.Winner = "CT"
		case common.TeamTerrorists:
			handler.currentRound.Winner = "T"
		}

		// Switch to post-round sampling - continue capturing frames
		handler.roundPhase = "postround"
	})

	// Round Officially Ended - now save the round
	ctx.Parser.RegisterEventHandler(func(e events.RoundEndOfficial) {
		if handler.currentRound == nil {
			return
		}

		// Save the round with all frames (including post-round)
		handler.Rounds = append(handler.Rounds, models.ReplayRound{
			Round:     handler.currentRound.Round,
			StartTick: handler.currentRound.StartTick,
			EndTick:   handler.currentRound.EndTick,
			Winner:    handler.currentRound.Winner,
			Frames:    handler.currentRound.Frames,
			Events:    handler.currentRound.Events,
		})

		handler.currentRound = nil
		handler.roundPhase = "none"
	})

	// ========================================
	// FRAME SAMPLING (Player positions, effects, etc.)
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		if handler.currentRound == nil {
			return
		}

		gameState := ctx.Parser.GameState()
		currentTick := gameState.IngameTick()
		tickRate := ctx.Parser.TickRate()

		// Calculate ticks per sample
		ticksPerSample := int(tickRate / float64(ReplaySampleRateHz))
		if ticksPerSample == 0 {
			ticksPerSample = 4 // Fallback for 16Hz at 64 tick
		}

		// Determine if we should sample based on phase
		shouldSample := false

		switch handler.roundPhase {
		case "freezetime":
			// Skip freeze time entirely - start sampling from actual gameplay
			shouldSample = false
		case "playing":
			// Normal gameplay - always sample
			shouldSample = true
		case "postround":
			// Post-round period - sample until RoundOfficiallyEnded
			shouldSample = true
		}

		if !shouldSample {
			return
		}

		// Check if it's time to sample
		if currentTick-handler.lastSampleTick < ticksPerSample {
			return
		}
		handler.lastSampleTick = currentTick

		// Build frame
		frame := models.ReplayFrame{
			Tick:          currentTick,
			TimeRemaining: calculateRoundTimeRemaining(ctx),
			Players:       handler.collectPlayerStates(gameState),
			Projectiles:   handler.collectProjectiles(),
			ActiveEffects: handler.collectActiveEffects(gameState),
			Bomb:          handler.bombState,
		}

		// Add recent shots and clean old ones
		frame.Shots = handler.collectRecentShots(currentTick)

		handler.currentRound.Frames = append(handler.currentRound.Frames, frame)
	})

	// ========================================
	// WEAPON FIRE (Shots visualization)
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.WeaponFire) {
		if handler.currentRound == nil || e.Shooter == nil {
			return
		}

		shooter := e.Shooter
		pos := shooter.Position()

		// Calculate shot endpoint using view direction
		yaw := float64(shooter.ViewDirectionX()) * math.Pi / 180.0
		pitch := float64(shooter.ViewDirectionY()) * math.Pi / 180.0

		// Project line ~1500 units in view direction
		distance := 1500.0
		toX := pos.X + math.Cos(yaw)*math.Cos(pitch)*distance
		toY := pos.Y + math.Sin(yaw)*math.Cos(pitch)*distance

		shot := models.ReplayShot{
			ShooterID: shooter.SteamID64,
			FromX:     pos.X,
			FromY:     pos.Y,
			ToX:       toX,
			ToY:       toY,
			Weapon:    e.Weapon.String(),
		}

		handler.recentShots = append(handler.recentShots, shotWithTick{
			shot: shot,
			tick: ctx.Parser.GameState().IngameTick(),
		})
	})

	// ========================================
	// GRENADE PROJECTILES
	// ========================================

	// Grenade thrown (projectile created)
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if handler.currentRound == nil {
			return
		}

		proj := e.Projectile
		if proj == nil {
			return
		}

		var throwerID uint64
		if proj.Thrower != nil {
			throwerID = proj.Thrower.SteamID64
		}

		grenadeType := "unknown"
		if proj.WeaponInstance != nil {
			grenadeType = proj.WeaponInstance.Type.String()
		}

		handler.activeProjectiles[proj.UniqueID()] = &models.ReplayProjectile{
			ID:         proj.UniqueID(),
			Type:       normalizeGrenadeType(grenadeType),
			ThrowerID:  throwerID,
			X:          proj.Position().X,
			Y:          proj.Position().Y,
			Z:          proj.Position().Z,
			Trajectory: []float64{proj.Position().X, proj.Position().Y},
		}
	})

	// Grenade destroyed (exploded/expired)
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		if e.Projectile != nil {
			delete(handler.activeProjectiles, e.Projectile.UniqueID())
		}
	})

	// ========================================
	// SMOKE EFFECTS
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.SmokeStart) {
		if handler.currentRound == nil {
			return
		}

		handler.activeSmokes[e.Grenade.UniqueID()] = &models.ReplayActiveEffect{
			Type:          "smoke",
			X:             e.Position.X,
			Y:             e.Position.Y,
			Radius:        SmokeRadius,
			TimeRemaining: 18.0, // Smoke lasts ~18 seconds
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.SmokeExpired) {
		delete(handler.activeSmokes, e.Grenade.UniqueID())
	})

	// ========================================
	// INFERNO (Molotov/Incendiary)
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.InfernoStart) {
		if handler.currentRound == nil {
			return
		}

		inferno := e.Inferno
		if inferno == nil {
			return
		}

		// Get convex hull for fire area
		hull := inferno.Fires().ConvexHull2D()
		hullFlat := make([]float64, 0, len(hull)*2)
		for _, p := range hull {
			hullFlat = append(hullFlat, p.X, p.Y)
		}

		// Calculate center
		var centerX, centerY float64
		for _, p := range hull {
			centerX += p.X
			centerY += p.Y
		}
		if len(hull) > 0 {
			centerX /= float64(len(hull))
			centerY /= float64(len(hull))
		}

		handler.activeInfernos[int64(inferno.UniqueID())] = &models.ReplayActiveEffect{
			Type:          "inferno",
			X:             centerX,
			Y:             centerY,
			TimeRemaining: 7.0, // Molotov lasts ~7 seconds
			Hull:          hullFlat,
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.InfernoExpired) {
		delete(handler.activeInfernos, int64(e.Inferno.UniqueID()))
	})

	// ========================================
	// FLASH EXPLODE (for player blind effect)
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.FlashExplode) {
		if handler.currentRound == nil {
			return
		}

		// Add event for timeline
		var throwerID uint64
		if e.Thrower != nil {
			throwerID = e.Thrower.SteamID64
		}

		handler.currentRound.Events = append(handler.currentRound.Events, models.ReplayEvent{
			Tick:        ctx.Parser.GameState().IngameTick(),
			Type:        "flash_explode",
			GrenadeType: "flashbang",
			X:           e.Position.X,
			Y:           e.Position.Y,
			PlayerID:    throwerID,
		})
	})

	// ========================================
	// KILLS
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.Kill) {
		if handler.currentRound == nil {
			return
		}

		var killerID, victimID uint64
		var killerName, victimName string
		var killerTeam, victimTeam string
		var killerX, killerY, victimX, victimY float64
		weapon := "unknown"

		if e.Killer != nil {
			killerID = e.Killer.SteamID64
			killerName = e.Killer.Name
			killerTeam = getTeamString(e.Killer.Team)
			pos := e.Killer.Position()
			killerX = pos.X
			killerY = pos.Y
		}
		if e.Victim != nil {
			victimID = e.Victim.SteamID64
			victimName = e.Victim.Name
			victimTeam = getTeamString(e.Victim.Team)
			pos := e.Victim.Position()
			victimX = pos.X
			victimY = pos.Y
		}
		if e.Weapon != nil {
			weapon = e.Weapon.String()
		}

		handler.currentRound.Events = append(handler.currentRound.Events, models.ReplayEvent{
			Tick:       ctx.Parser.GameState().IngameTick(),
			Type:       "kill",
			KillerID:   killerID,
			VictimID:   victimID,
			KillerName: killerName,
			VictimName: victimName,
			KillerTeam: killerTeam,
			VictimTeam: victimTeam,
			KillerX:    killerX,
			KillerY:    killerY,
			VictimX:    victimX,
			VictimY:    victimY,
			Weapon:     weapon,
			Headshot:   e.IsHeadshot,
			Wallbang:   e.PenetratedObjects > 0,
			NoScope:    e.NoScope,
		})

		// Mark the shot that killed as a hit
		for i := len(handler.recentShots) - 1; i >= 0; i-- {
			if handler.recentShots[i].shot.ShooterID == killerID {
				handler.recentShots[i].shot.Hit = true
				break
			}
		}
	})

	// ========================================
	// BOMB EVENTS
	// ========================================

	ctx.Parser.RegisterEventHandler(func(e events.BombDropped) {
		if handler.bombState != nil && e.Player != nil {
			handler.bombState.State = "dropped"
			handler.bombState.X = e.Player.Position().X
			handler.bombState.Y = e.Player.Position().Y
			handler.bombState.CarrierID = 0
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPickup) {
		if handler.bombState != nil && e.Player != nil {
			handler.bombState.State = "carried"
			handler.bombState.CarrierID = e.Player.SteamID64
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPlanted) {
		if handler.bombState == nil || handler.currentRound == nil {
			return
		}

		handler.bombState.State = "planted"
		handler.bombState.PlantTick = ctx.Parser.GameState().IngameTick()
		handler.bombState.Site = string(e.Site)

		if e.Player != nil {
			handler.bombState.X = e.Player.Position().X
			handler.bombState.Y = e.Player.Position().Y

			handler.currentRound.Events = append(handler.currentRound.Events, models.ReplayEvent{
				Tick:     ctx.Parser.GameState().IngameTick(),
				Type:     "bomb_plant",
				Site:     string(e.Site),
				PlayerID: e.Player.SteamID64,
				X:        e.Player.Position().X,
				Y:        e.Player.Position().Y,
			})
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefuseStart) {
		if handler.bombState != nil && e.Player != nil {
			handler.bombState.State = "defusing"
			handler.bombState.DefuserID = e.Player.SteamID64
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefuseAborted) {
		if handler.bombState != nil {
			handler.bombState.State = "planted"
			handler.bombState.DefuserID = 0
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefused) {
		if handler.bombState == nil || handler.currentRound == nil {
			return
		}

		handler.bombState.State = "defused"

		if e.Player != nil {
			handler.currentRound.Events = append(handler.currentRound.Events, models.ReplayEvent{
				Tick:     ctx.Parser.GameState().IngameTick(),
				Type:     "bomb_defuse",
				Site:     handler.bombState.Site,
				PlayerID: e.Player.SteamID64,
			})
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombExplode) {
		if handler.bombState == nil || handler.currentRound == nil {
			return
		}

		handler.bombState.State = "exploded"

		handler.currentRound.Events = append(handler.currentRound.Events, models.ReplayEvent{
			Tick: ctx.Parser.GameState().IngameTick(),
			Type: "bomb_explode",
			Site: handler.bombState.Site,
			X:    handler.bombState.X,
			Y:    handler.bombState.Y,
		})
	})

	return handler
}

// ========================================
// HELPER METHODS
// ========================================

func (h *ReplayHandler) collectPlayerStates(gs dem.GameState) []models.ReplayPlayerState {
	players := []models.ReplayPlayerState{}

	for _, p := range gs.Participants().Playing() {
		if p == nil {
			continue
		}

		state := models.ReplayPlayerState{
			SteamID:       p.SteamID64,
			Name:          p.Name,
			Team:          getTeamString(p.Team),
			X:             int(math.Round(p.Position().X)),
			Y:             int(math.Round(p.Position().Y)),
			Z:             p.Position().Z,
			Yaw:           p.ViewDirectionX(),
			Pitch:         p.ViewDirectionY(),
			Health:        p.Health(),
			Armor:         p.Armor(),
			Alive:         p.IsAlive(),
			Weapon:        getActiveWeapon(p),
			HasDefuseKit:  p.HasDefuseKit(),
			HasC4:         hasC4(p),
			FlashDuration: p.FlashDurationTimeRemaining().Seconds(),
			Money:         p.Money(),
			IsDucking:     p.IsDucking(),
			IsWalking:     p.IsWalking(),
			IsScoped:      p.IsScoped(),
			IsDefusing:    p.IsDefusing,
		}

		players = append(players, state)
	}

	return players
}

func (h *ReplayHandler) collectProjectiles() []models.ReplayProjectile {
	projectiles := []models.ReplayProjectile{}

	// Update positions from game state
	gs := h.ctx.Parser.GameState()
	for _, proj := range gs.GrenadeProjectiles() {
		if active, ok := h.activeProjectiles[proj.UniqueID()]; ok {
			// Update position
			active.X = proj.Position().X
			active.Y = proj.Position().Y
			active.Z = proj.Position().Z

			// Add to trajectory (limit size)
			active.Trajectory = append(active.Trajectory, proj.Position().X, proj.Position().Y)
			if len(active.Trajectory) > TrajectoryHistorySize*2 {
				active.Trajectory = active.Trajectory[2:]
			}
		}
	}

	// Collect all active projectiles
	for _, proj := range h.activeProjectiles {
		projectiles = append(projectiles, *proj)
	}

	return projectiles
}

func (h *ReplayHandler) collectActiveEffects(gs dem.GameState) []models.ReplayActiveEffect {
	effects := []models.ReplayActiveEffect{}

	// Collect smokes
	for _, smoke := range h.activeSmokes {
		effects = append(effects, *smoke)
	}

	// Update inferno hulls from game state
	for _, inferno := range gs.Infernos() {
		if active, ok := h.activeInfernos[int64(inferno.UniqueID())]; ok {
			hull := inferno.Fires().ConvexHull2D()
			hullFlat := make([]float64, 0, len(hull)*2)
			for _, p := range hull {
				hullFlat = append(hullFlat, p.X, p.Y)
			}
			active.Hull = hullFlat
		}
	}

	// Collect infernos
	for _, inferno := range h.activeInfernos {
		effects = append(effects, *inferno)
	}

	return effects
}

func (h *ReplayHandler) collectRecentShots(currentTick int) []models.ReplayShot {
	// Clean old shots
	validShots := []shotWithTick{}
	for _, s := range h.recentShots {
		if currentTick-s.tick <= ShotVisibilityTicks {
			validShots = append(validShots, s)
		}
	}
	h.recentShots = validShots

	// Collect for frame
	shots := []models.ReplayShot{}
	for _, s := range h.recentShots {
		shots = append(shots, s.shot)
	}

	return shots
}

// GetReplayData builds the final replay data structure
func (h *ReplayHandler) GetReplayData(matchID string) models.ReplayData {
	mapName := h.ctx.MatchData.MapName
	tickRate := h.ctx.Parser.TickRate()

	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			MatchID:    matchID,
			MapName:    mapName,
			TickRate:   tickRate,
			SampleRate: 1000 / ReplaySampleRateHz, // Convert Hz to ms
			MapConfig:  models.GetMapConfig(mapName),
		},
		Rounds: h.Rounds,
	}
}

func normalizeGrenadeType(t string) string {
	switch t {
	case "Smoke Grenade", "SmokeGrenade":
		return "smoke"
	case "Flashbang":
		return "flashbang"
	case "HE Grenade", "HEGrenade":
		return "he"
	case "Molotov", "Incendiary Grenade", "IncendiaryGrenade":
		return "molotov"
	case "Decoy Grenade", "DecoyGrenade":
		return "decoy"
	default:
		return t
	}
}
