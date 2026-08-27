package handlers

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
	"cs2-demo-service/pkg/objective"
	"cs2-demo-service/pkg/playerstate"
	"cs2-demo-service/pkg/utility"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
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

	// Stable player equipment within the current round.
	playerEquipment map[uint64]replayPlayerEquipment

	// Output: all rounds
	Rounds []models.ReplayRound

	// Last sample tick
	lastSampleTick int

	// Round phase tracking
	roundPhase         string // "freezetime", "playing", "postround", "none"
	freezeTimeEndTick  int    // Tick when freeze time ends
	roundEndTick       int    // Tick when RoundEnd event fired
	eventSequence      uint64
	projectileSequence int64
}

type replayPlayerEquipment struct {
	weapons      []string
	activeWeapon string
	hasDefuseKit bool
	hasHelmet    bool
	hasC4        bool
}

// NewReplayHandler creates a new replay handler
func NewReplayHandler(ctx *models.DemoContext) *ReplayHandler {
	return &ReplayHandler{
		ctx:               ctx,
		activeProjectiles: make(map[int64]*models.ReplayProjectile),
		activeSmokes:      make(map[int64]*models.ReplayActiveEffect),
		activeInfernos:    make(map[int64]*models.ReplayActiveEffect),
		playerEquipment:   make(map[uint64]replayPlayerEquipment),
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
		gameState := ctx.Parser.GameState()
		if gameState.IsWarmupPeriod() {
			return
		}
		// TotalRoundsPlayed is owned by the parser and is already settled for this
		// event. Do not depend on another handler updating shared context first.
		roundNumber := replayRoundNumber(gameState.TotalRoundsPlayed())
		ensureObjectiveTracker(ctx).BeginRound(roundNumber, gameState.IngameTick())
		handler.currentRound = &models.ReplayRoundData{
			Round:     roundNumber,
			StartTick: gameState.IngameTick(),
			Frames:    []models.ReplayFrame{},
			Events:    []models.ReplayEvent{},
		}

		// Reset active effects
		handler.activeSmokes = make(map[int64]*models.ReplayActiveEffect)
		handler.activeInfernos = make(map[int64]*models.ReplayActiveEffect)
		handler.activeProjectiles = make(map[int64]*models.ReplayProjectile)
		handler.resetPlayerEquipment()

		// We're now in freeze time
		handler.roundPhase = "freezetime"
	})

	// Freeze Time End - gameplay starts here. Keeping RoundStart as the replay
	// boundary makes every round open on idle buy time even though we do not
	// sample that phase.
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		if handler.currentRound == nil {
			return
		}
		handler.freezeTimeEndTick = ctx.Parser.GameState().IngameTick()
		handler.currentRound.StartTick = handler.freezeTimeEndTick
		handler.roundPhase = "playing"
		ensureObjectiveTracker(ctx).NativeSnapshot(nativeObjectiveObservation(
			ctx,
			handler.currentRound.Round,
			handler.freezeTimeEndTick,
		))
	})

	// Round End - mark end but continue sampling for post-round period
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		if handler.currentRound == nil {
			return
		}

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

		// RoundEnd only declares the winner. Kills and other state changes may
		// still happen during post-round, so the replay boundary is the official
		// end emitted by the demo parser.
		handler.finishCurrentRound(ctx.Parser.GameState().IngameTick())
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

		bombState := handler.snapshotBombState(gameState, currentTick)

		// Build frame
		frame := models.ReplayFrame{
			Tick:          currentTick,
			TimeRemaining: calculateRoundTimeRemaining(ctx),
			Players:       handler.collectPlayerStates(gameState),
			Projectiles:   handler.collectProjectiles(),
			ActiveEffects: handler.collectActiveEffects(gameState),
			Bomb:          bombState,
		}

		handler.currentRound.Frames = append(handler.currentRound.Frames, frame)
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
			ID:         handler.nextReplayProjectileID(),
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

		tick := ctx.Parser.GameState().IngameTick()
		handler.activeSmokes[playerstate.EquipmentID(e.Grenade)] = &models.ReplayActiveEffect{
			Type: "smoke", X: e.Position.X, Y: e.Position.Y,
			Radius: SmokeRadius, StartTick: tick,
			TimeRemainingStatus: string(utility.AvailabilityUnavailable),
			TimeRemainingSource: utility.SourceUnavailable,
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.SmokeExpired) {
		delete(handler.activeSmokes, playerstate.EquipmentID(e.Grenade))
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
		fires := inferno.Fires().List()
		firePoints := make([]float64, 0, len(fires)*2)
		for _, fire := range fires {
			firePoints = append(firePoints, fire.X, fire.Y)
		}
		if len(hull) > 0 {
			centerX /= float64(len(hull))
			centerY /= float64(len(hull))
		}

		tick := ctx.Parser.GameState().IngameTick()
		infernoKey, _ := replayInfernoKey(inferno)
		handler.activeInfernos[infernoKey] = &models.ReplayActiveEffect{
			Type: "inferno", X: centerX, Y: centerY, StartTick: tick,
			TimeRemainingStatus: string(utility.AvailabilityUnavailable),
			TimeRemainingSource: utility.SourceUnavailable,
			Hull:                hullFlat, Points: firePoints,
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.InfernoExpired) {
		if infernoKey, ok := replayInfernoKey(e.Inferno); ok {
			delete(handler.activeInfernos, infernoKey)
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.HeExplode) {
		if handler.currentRound == nil {
			return
		}
		handler.removeDetonatedProjectile("he", e.Position.X, e.Position.Y)
	})

	return handler
}

func (h *ReplayHandler) replayEventID(round, tick int, eventType string, actorID uint64) string {
	h.eventSequence++
	return fmt.Sprintf("r%d-t%d-%s-%d-%d", round, tick, eventType, actorID, h.eventSequence)
}

func (h *ReplayHandler) nextReplayProjectileID() int64 {
	h.projectileSequence++
	return h.projectileSequence
}

// Finalize persists a valid last round when a truncated or older demo omits
// RoundEndOfficial. ParseToEnd must call it exactly once.
func (h *ReplayHandler) Finalize() {
	if h.currentRound != nil {
		endTick := h.roundEndTick
		if count := len(h.currentRound.Frames); count > 0 && h.currentRound.Frames[count-1].Tick > endTick {
			endTick = h.currentRound.Frames[count-1].Tick
		}
		if endTick >= h.currentRound.StartTick {
			h.finishCurrentRound(endTick)
		}
	}
	h.reprojectFinalCombat()
}

func (h *ReplayHandler) reprojectFinalCombat() {
	if h.ctx == nil || h.ctx.Combat == nil {
		return
	}
	ledger := h.ctx.Combat.Snapshot()
	for index := range h.Rounds {
		round := &h.Rounds[index]
		round.Events = projectReplayCombatMarkers(round.Events, ledger, round.Round)
		round.CombatShots = projectReplayCombatShots(ledger, round.Round)
		round.Frames = projectReplayCombatFrames(round.Frames, ledger, round.Round)
	}
}

func (h *ReplayHandler) finishCurrentRound(endTick int) {
	if h.currentRound == nil {
		return
	}
	h.currentRound.EndTick = endTick
	var combatEvents []combat.Event
	if h.ctx != nil && h.ctx.Combat != nil {
		combatEvents = h.ctx.Combat.Snapshot()
	}
	h.currentRound.Events = projectReplayCombatMarkers(
		h.currentRound.Events,
		combatEvents,
		h.currentRound.Round,
	)
	h.currentRound.Frames = projectReplayCombatFrames(
		h.currentRound.Frames,
		combatEvents,
		h.currentRound.Round,
	)
	var objectiveEvents []objective.Event
	if h.ctx != nil && h.ctx.Objectives != nil {
		objectiveEvents = h.ctx.Objectives.Events()
	}
	h.currentRound.Events = projectReplayObjectiveMarkers(
		h.currentRound.Events,
		objectiveEvents,
		h.currentRound.Round,
	)
	var utilityThrows []utility.Throw
	if h.ctx != nil && h.ctx.Utilities != nil {
		utilityThrows = h.ctx.Utilities.Snapshot()
	}
	h.currentRound.Events = projectReplayUtilityMarkers(
		h.currentRound.Events,
		utilityThrows,
		h.currentRound.Round,
	)
	tickRate := 0.0
	if h.ctx != nil && h.ctx.Parser != nil {
		tickRate = h.ctx.Parser.TickRate()
	}
	h.currentRound.Frames = projectReplayUtilityFrames(
		h.currentRound.Frames,
		utilityThrows,
		h.currentRound.Round,
		tickRate,
	)
	h.Rounds = append(h.Rounds, models.ReplayRound{
		Round:       h.currentRound.Round,
		StartTick:   h.currentRound.StartTick,
		EndTick:     h.currentRound.EndTick,
		Winner:      h.currentRound.Winner,
		Frames:      h.currentRound.Frames,
		Events:      h.currentRound.Events,
		CombatShots: projectReplayCombatShots(combatEvents, h.currentRound.Round),
	})
	h.currentRound = nil
	h.roundPhase = "none"
}

func projectReplayObjectiveMarkers(
	events []models.ReplayEvent,
	objectiveEvents []objective.Event,
	round int,
) []models.ReplayEvent {
	projected := make([]models.ReplayEvent, 0, len(events)+3)
	for _, event := range events {
		if isReplayObjectiveMarker(event.Type) {
			continue
		}
		projected = append(projected, event)
	}
	for _, event := range objectiveEvents {
		if event.Round != round {
			continue
		}
		marker, ok := replayObjectiveMarker(event)
		if ok {
			projected = append(projected, marker)
		}
	}
	slices.SortFunc(projected, compareReplayEvents)
	return projected
}

func projectReplayUtilityMarkers(
	events []models.ReplayEvent,
	throws []utility.Throw,
	round int,
) []models.ReplayEvent {
	projected := make([]models.ReplayEvent, 0, len(events)+len(throws))
	for _, event := range events {
		if event.Type != "utility_detonate" {
			projected = append(projected, event)
		}
	}
	for _, entry := range throws {
		if entry.Round != round {
			continue
		}
		moment, ok := utilityEffectMoment(entry.Lifecycle)
		if !ok {
			continue
		}
		marker := models.ReplayEvent{
			ID: "utility:" + entry.ID, Tick: moment.Tick, Type: "utility_detonate",
			GrenadeType: string(entry.Type), UtilityType: string(entry.Type),
			PlayerID: entry.Actor.ID, ActorID: entry.Actor.ID,
			PositionStatus: string(moment.PositionStatus), PositionSource: utility.SourceUnavailable,
			CorrelationStatus: string(entry.Lifecycle.Correlation.Status),
			CorrelationSource: entry.Lifecycle.Correlation.Source,
			SourceThrowID:     entry.ID,
		}
		if moment.PositionStatus == utility.AvailabilityObserved {
			marker.X, marker.Y, marker.Z = moment.Position.X, moment.Position.Y, moment.Position.Z
			marker.PositionSource = moment.Source
		}
		marker.AffectedPlayerIDs = replayAffectedPlayerIDs(entry.Flashes)
		for _, effect := range entry.Damage {
			marker.Damage += max(0, effect.HealthDamage)
		}
		marker.DurationMS, marker.DurationStatus, marker.DurationSource = replayUtilityDuration(entry)
		projected = append(projected, marker)
	}
	slices.SortFunc(projected, compareReplayEvents)
	return projected
}

func replayAffectedPlayerIDs(effects []utility.FlashEffect) []string {
	seen := make(map[uint64]struct{}, len(effects))
	for _, effect := range effects {
		if effect.Victim.Status == utility.AvailabilityObserved && effect.Victim.ID != 0 {
			seen[effect.Victim.ID] = struct{}{}
		}
	}
	ids := make([]uint64, 0, len(seen))
	for steamID := range seen {
		ids = append(ids, steamID)
	}
	slices.Sort(ids)
	result := make([]string, 0, len(ids))
	for _, steamID := range ids {
		result = append(result, strconv.FormatUint(steamID, 10))
	}
	return result
}

func replayUtilityDuration(entry utility.Throw) (int, string, string) {
	if entry.Lifecycle.Duration.Status == utility.AvailabilityObserved {
		return int(math.Round(entry.Lifecycle.Duration.Value * 1000)),
			string(utility.AvailabilityObserved), entry.Lifecycle.Duration.Source
	}
	maximum := 0.0
	source := utility.SourceUnavailable
	for _, effect := range entry.Flashes {
		if effect.Duration.Status == utility.AvailabilityObserved && effect.Duration.Value >= maximum {
			maximum, source = effect.Duration.Value, effect.Duration.Source
		}
	}
	if source != utility.SourceUnavailable {
		return int(math.Round(maximum * 1000)), string(utility.AvailabilityObserved), source
	}
	return 0, string(utility.AvailabilityUnavailable), utility.SourceUnavailable
}

func projectReplayUtilityFrames(
	frames []models.ReplayFrame,
	throws []utility.Throw,
	round int,
	tickRate float64,
) []models.ReplayFrame {
	for frameIndex := range frames {
		frame := &frames[frameIndex]
		for effectIndex := range frame.ActiveEffects {
			effect := &frame.ActiveEffects[effectIndex]
			entry, found := replayEffectThrow(*effect, throws, round)
			if !found || tickRate <= 0 || entry.Lifecycle.Expiration.Status != utility.AvailabilityObserved {
				effect.TimeRemaining = 0
				effect.TimeRemainingStatus = string(utility.AvailabilityUnavailable)
				effect.TimeRemainingSource = utility.SourceUnavailable
				continue
			}
			remainingTicks := max(0, entry.Lifecycle.Expiration.Tick-frame.Tick)
			effect.TimeRemaining = float64(remainingTicks) / tickRate
			effect.TimeRemainingStatus = string(utility.AvailabilityObserved)
			effect.TimeRemainingSource = utility.SourceCallbackTicks
		}
	}
	return frames
}

func replayEffectThrow(
	effect models.ReplayActiveEffect,
	throws []utility.Throw,
	round int,
) (utility.Throw, bool) {
	best := utility.Throw{}
	found := false
	bestDistance := math.Inf(1)
	for _, entry := range throws {
		if entry.Round != round || entry.Lifecycle.EffectStart.Status != utility.AvailabilityObserved ||
			entry.Lifecycle.EffectStart.Tick != effect.StartTick || !replayEffectTypeMatches(effect.Type, entry.Type) {
			continue
		}
		distance := math.Inf(1)
		if entry.Lifecycle.EffectStart.PositionStatus == utility.AvailabilityObserved {
			distance = math.Hypot(
				entry.Lifecycle.EffectStart.Position.X-effect.X,
				entry.Lifecycle.EffectStart.Position.Y-effect.Y,
			)
		}
		if !found || distance < bestDistance ||
			(distance == bestDistance && entry.Sequence < best.Sequence) {
			best, bestDistance, found = entry, distance, true
		}
	}
	return best, found
}

func replayEffectTypeMatches(effectType string, utilityType utility.Type) bool {
	switch effectType {
	case "smoke":
		return utilityType == utility.TypeSmoke
	case "inferno":
		return utilityType == utility.TypeMolotov || utilityType == utility.TypeIncendiary ||
			utilityType == utility.TypeUnknown
	default:
		return false
	}
}

func replayInfernoKey(inferno *common.Inferno) (int64, bool) {
	if inferno == nil {
		return 0, false
	}
	return int64(inferno.UniqueID()), true
}

func replayObjectiveMarker(event objective.Event) (models.ReplayEvent, bool) {
	eventType := ""
	durationMS := 0
	switch event.Type {
	case objective.EventPlant:
		eventType = "bomb_plant"
	case objective.EventDefuse:
		eventType = "bomb_defuse"
	case objective.EventExplode:
		eventType = "bomb_explode"
		durationMS = 1200
	default:
		return models.ReplayEvent{}, false
	}

	marker := models.ReplayEvent{
		ID:         event.ID,
		Tick:       event.Tick,
		Type:       eventType,
		Site:       event.Site,
		PlayerID:   event.Actor.SteamID,
		DurationMS: durationMS,
	}
	if event.Position.Available() {
		marker.X = event.Position.X
		marker.Y = event.Position.Y
		marker.Z = event.Position.Z
	}
	return marker, true
}

func isReplayObjectiveMarker(eventType string) bool {
	switch eventType {
	case "bomb_plant", "bomb_defuse", "bomb_explode":
		return true
	default:
		return false
	}
}

func compareReplayEvents(left, right models.ReplayEvent) int {
	if order := cmp.Compare(left.Tick, right.Tick); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Type, right.Type); order != 0 {
		return order
	}
	return cmp.Compare(left.ID, right.ID)
}

// ========================================
// HELPER METHODS
// ========================================

func (h *ReplayHandler) collectPlayerStates(gs dem.GameState) []models.ReplayPlayerState {
	players := []models.ReplayPlayerState{}
	objectiveState := ensureObjectiveTracker(h.ctx).Snapshot()
	participants := append([]*common.Player(nil), gs.Participants().Playing()...)
	slices.SortFunc(participants, compareReplayParticipants)

	for _, p := range participants {
		if p == nil {
			continue
		}

		equipment := h.resolvePlayerEquipment(p)
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
			Weapon:        equipment.activeWeapon,
			Weapons:       equipment.weapons,
			HasDefuseKit:  equipment.hasDefuseKit,
			HasHelmet:     equipment.hasHelmet,
			HasC4:         replayPlayerHasC4(p.SteamID64, p.IsAlive(), objectiveState),
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

func compareReplayParticipants(left, right *common.Player) int {
	if left == nil {
		if right == nil {
			return 0
		}
		return 1
	}
	if right == nil {
		return -1
	}
	if order := cmp.Compare(left.SteamID64, right.SteamID64); order != 0 {
		return order
	}
	if order := cmp.Compare(left.UserID, right.UserID); order != 0 {
		return order
	}
	if order := cmp.Compare(left.EntityID, right.EntityID); order != 0 {
		return order
	}
	return cmp.Compare(left.Name, right.Name)
}

func (h *ReplayHandler) resolvePlayerEquipment(player *common.Player) replayPlayerEquipment {
	cached := h.playerEquipment[player.SteamID64]
	weapons := getPlayerWeapons(player)
	activeWeapon := ""
	if active := player.ActiveWeapon(); active != nil {
		activeWeapon = active.String()
	}

	cached = mergePlayerEquipment(
		cached,
		player.IsAlive(),
		weapons,
		activeWeapon,
		player.HasDefuseKit(),
		player.HasHelmet(),
		hasC4(player),
	)

	h.playerEquipment[player.SteamID64] = cached
	cached.weapons = append([]string(nil), cached.weapons...)
	return cached
}

func mergePlayerEquipment(
	cached replayPlayerEquipment,
	alive bool,
	weapons []string,
	activeWeapon string,
	hasDefuseKit bool,
	hasHelmet bool,
	hasC4 bool,
) replayPlayerEquipment {
	if alive {
		if len(weapons) > 0 {
			cached.weapons = append([]string(nil), weapons...)
		}
		if activeWeapon != "" {
			cached.activeWeapon = activeWeapon
		}
		cached.hasHelmet = hasHelmet
	}
	cached.hasDefuseKit = alive && hasDefuseKit
	cached.hasC4 = alive && hasC4

	if len(cached.weapons) == 0 && len(weapons) > 0 {
		cached.weapons = append([]string(nil), weapons...)
	}
	return cached
}

func (h *ReplayHandler) resetPlayerEquipment() {
	h.playerEquipment = make(map[uint64]replayPlayerEquipment)
}

func (h *ReplayHandler) snapshotBombState(gs dem.GameState, tick int) *models.ReplayBombState {
	if h.currentRound == nil || gs == nil {
		return nil
	}
	tracker := ensureObjectiveTracker(h.ctx)
	snapshot := tracker.NativeSnapshot(nativeObjectiveObservation(
		h.ctx,
		h.currentRound.Round,
		tick,
	))
	return replayBombState(snapshot)
}

func replayBombState(snapshot objective.Snapshot) *models.ReplayBombState {
	return &models.ReplayBombState{
		State:               snapshot.State,
		ObjectivePhase:      snapshot.Phase,
		IsPlantedNow:        snapshot.IsPlantedNow,
		WasPlantedThisRound: snapshot.WasPlantedThisRound,
		X:                   snapshot.Position.X,
		Y:                   snapshot.Position.Y,
		CarrierID:           replayPlayerIDPointer(snapshot.Carrier.SteamID),
		Site:                snapshot.Site,
		PlantTick:           snapshot.PlantTick,
		DefuserID:           replayPlayerIDPointer(snapshot.Defuser.SteamID),
		PositionStatus:      snapshot.Position.Status,
		PositionSource:      snapshot.Position.Source,
	}
}

func replayPlayerIDPointer(steamID uint64) *uint64 {
	if steamID == 0 {
		return nil
	}
	return &steamID
}

func replayPlayerHasC4(steamID uint64, alive bool, snapshot objective.Snapshot) bool {
	return alive && steamID != 0 && snapshot.Carrier.SteamID == steamID
}

func (h *ReplayHandler) removeDetonatedProjectile(grenadeType string, x, y float64) {
	var closestID int64
	closestDistance := math.Inf(1)
	closestSequence := int64(0)
	found := false
	for id, projectile := range h.activeProjectiles {
		if projectile == nil || projectile.Type != grenadeType {
			continue
		}
		distance := math.Hypot(projectile.X-x, projectile.Y-y)
		if distance < closestDistance || (distance == closestDistance && (!found || projectile.ID < closestSequence)) {
			closestID = id
			closestDistance = distance
			closestSequence = projectile.ID
			found = true
		}
	}
	if found && closestDistance <= 256 {
		delete(h.activeProjectiles, closestID)
	}
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
	sortReplayProjectiles(projectiles)

	return projectiles
}

func sortReplayProjectiles(projectiles []models.ReplayProjectile) {
	slices.SortFunc(projectiles, func(left, right models.ReplayProjectile) int {
		return cmp.Compare(left.ID, right.ID)
	})
}

func (h *ReplayHandler) collectActiveEffects(gs dem.GameState) []models.ReplayActiveEffect {
	effects := []models.ReplayActiveEffect{}

	// Collect smokes
	for _, smoke := range h.activeSmokes {
		snapshot := *smoke
		snapshot.TimeRemaining = 0
		snapshot.TimeRemainingStatus = string(utility.AvailabilityUnavailable)
		snapshot.TimeRemainingSource = utility.SourceUnavailable
		effects = append(effects, snapshot)
	}

	// Update inferno hulls from game state
	for _, inferno := range gs.Infernos() {
		infernoKey, valid := replayInfernoKey(inferno)
		if !valid {
			continue
		}
		if active, ok := h.activeInfernos[infernoKey]; ok {
			hull := inferno.Fires().ConvexHull2D()
			hullFlat := make([]float64, 0, len(hull)*2)
			for _, p := range hull {
				hullFlat = append(hullFlat, p.X, p.Y)
			}
			active.Hull = hullFlat
			fires := inferno.Fires().List()
			firePoints := make([]float64, 0, len(fires)*2)
			for _, fire := range fires {
				firePoints = append(firePoints, fire.X, fire.Y)
			}
			active.Points = firePoints
		}
	}

	// Collect infernos
	for _, inferno := range h.activeInfernos {
		snapshot := *inferno
		snapshot.TimeRemaining = 0
		snapshot.TimeRemainingStatus = string(utility.AvailabilityUnavailable)
		snapshot.TimeRemainingSource = utility.SourceUnavailable
		effects = append(effects, snapshot)
	}
	slices.SortFunc(effects, compareReplayActiveEffects)

	return effects
}

func compareReplayActiveEffects(left, right models.ReplayActiveEffect) int {
	if order := cmp.Compare(left.Type, right.Type); order != 0 {
		return order
	}
	if order := cmp.Compare(left.X, right.X); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Y, right.Y); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Radius, right.Radius); order != 0 {
		return order
	}
	if order := cmp.Compare(left.TimeRemaining, right.TimeRemaining); order != 0 {
		return order
	}
	if order := cmp.Compare(left.TimeRemainingStatus, right.TimeRemainingStatus); order != 0 {
		return order
	}
	if order := cmp.Compare(left.TimeRemainingSource, right.TimeRemainingSource); order != 0 {
		return order
	}
	if order := cmp.Compare(left.StartTick, right.StartTick); order != 0 {
		return order
	}
	if order := slices.Compare(left.Hull, right.Hull); order != 0 {
		return order
	}
	return slices.Compare(left.Points, right.Points)
}

// GetReplayData builds the final replay data structure
func (h *ReplayHandler) GetReplayData(matchID string) models.ReplayData {
	mapName := h.ctx.MatchData.MapName
	tickRate := h.ctx.Parser.TickRate()

	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 5,
			MatchID:       matchID,
			MapName:       mapName,
			TickRate:      tickRate,
			SampleRate:    1000 / ReplaySampleRateHz, // Convert Hz to ms
			MapConfig:     models.GetMapConfig(mapName),
		},
		Rounds: h.Rounds,
	}
}

func replayRoundNumber(totalRoundsPlayed int) int {
	if totalRoundsPlayed < 0 {
		return 1
	}
	return totalRoundsPlayed + 1
}

func normalizeGrenadeType(t string) string {
	switch t {
	case "Smoke Grenade", "SmokeGrenade":
		return "smoke"
	case "Flashbang":
		return "flashbang"
	case "HE Grenade", "HEGrenade":
		return "he"
	case "Molotov":
		return "molotov"
	case "Incendiary Grenade", "IncendiaryGrenade":
		return "incendiary"
	case "Decoy Grenade", "DecoyGrenade":
		return "decoy"
	default:
		return t
	}
}
