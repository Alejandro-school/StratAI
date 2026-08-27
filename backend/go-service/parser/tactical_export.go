package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

const (
	tacticalSourceArtifactReplayData = "replay_data"

	tacticalGeometryNotRequired   = "not_required"
	tacticalGeometryLoaded        = "loaded"
	tacticalGeometryNotLoaded     = "not_loaded"
	tacticalGeometryNotApplicable = "not_applicable"
	tacticalGeometryNotEvaluated  = "not_evaluated"

	tacticalGapMissingFrame      = "missing_replay_frame_for_target_window"
	tacticalGapNoJoinablePlayers = "no_joinable_players"
	tacticalGapNoObserver        = "no_eligible_observer"
)

// TacticalGeometry is the minimal geometry contract needed to publish enemy
// observations. A maps.VisibilityChecker satisfies it without coupling this
// exporter to callout or ray-cast APIs.
type TacticalGeometry interface {
	IsLoaded() bool
	IsVisible(start, end r3.Vector) bool
}

type tacticalPlayerSample struct {
	steamID  uint64
	team     string
	position *r3.Vector
	physical models.TacticalPhysicalState
	oracle   models.TacticalOracleState
	key      string
}

type tacticalFrameSample struct {
	tick    int
	players []tacticalPlayerSample
	key     string
}

// BuildTacticalExport builds a causally safe, round-anchored 16 Hz view from
// real ReplayData. It never carries an old replay frame into a later sampling
// window: a window without a source frame is represented by an explicit gap.
func BuildTacticalExport(replay *models.ReplayData, geometry TacticalGeometry) (models.TacticalExport, error) {
	if replay == nil {
		return models.TacticalExport{}, fmt.Errorf("build tactical export: replay data is nil")
	}

	matchID := strings.TrimSpace(replay.Metadata.MatchID)
	if matchID == "" {
		return models.TacticalExport{}, fmt.Errorf("build tactical export: replay metadata match_id is empty")
	}

	tickRate := replay.Metadata.TickRate
	if math.IsNaN(tickRate) || math.IsInf(tickRate, 0) || tickRate < models.TacticalTargetHz {
		return models.TacticalExport{}, fmt.Errorf(
			"build tactical export: tick_rate must be finite and at least %d, got %v",
			models.TacticalTargetHz,
			tickRate,
		)
	}

	periodTicks := tickRate / float64(models.TacticalTargetHz)
	export := models.TacticalExport{
		SchemaID:          models.TacticalExportSchemaID,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		JoinKeys: models.TacticalExportJoinKeys{
			MatchID: matchID,
		},
		Sampling: models.TacticalSampling{
			TargetHz:    models.TacticalTargetHz,
			TickRate:    tickRate,
			PeriodTicks: periodTicks,
			Strategy:    models.TacticalSamplingStrategy,
		},
		PhysicalRows: make([]models.TacticalPhysicalRow, 0),
		OracleRows:   make([]models.TacticalOracleRow, 0),
		Gaps:         make([]models.TacticalSamplingGap, 0),
	}

	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		if rounds[i].StartTick != rounds[j].StartTick {
			return rounds[i].StartTick < rounds[j].StartTick
		}
		return rounds[i].EndTick < rounds[j].EndTick
	})

	geometryLoaded := geometry != nil && geometry.IsLoaded()
	seenRounds := make(map[int]struct{}, len(rounds))
	for _, round := range rounds {
		if round.Round <= 0 {
			return models.TacticalExport{}, fmt.Errorf("build tactical export: round number must be positive, got %d", round.Round)
		}
		if _, exists := seenRounds[round.Round]; exists {
			return models.TacticalExport{}, fmt.Errorf("build tactical export: duplicate round number %d", round.Round)
		}
		seenRounds[round.Round] = struct{}{}

		if round.EndTick < round.StartTick {
			return models.TacticalExport{}, fmt.Errorf(
				"build tactical export: round %d end tick %d precedes start tick %d",
				round.Round,
				round.EndTick,
				round.StartTick,
			)
		}

		targetTicks := tacticalTargetTicks(round.StartTick, round.EndTick, periodTicks)
		frames, err := canonicalTacticalFrames(round, tickRate)
		if err != nil {
			return models.TacticalExport{}, fmt.Errorf("build tactical export: round %d: %w", round.Round, err)
		}

		roundID := fmt.Sprintf("%s:round:%03d", matchID, round.Round)
		frameIndex := 0
		previousTarget := round.StartTick
		for targetIndex, targetTick := range targetTicks {
			var selected *tacticalFrameSample
			for frameIndex < len(frames) && frames[frameIndex].tick <= targetTick {
				candidate := &frames[frameIndex]
				if targetIndex == 0 || candidate.tick > previousTarget {
					selected = candidate
				}
				frameIndex++
			}

			if selected == nil {
				export.Gaps = append(export.Gaps, tacticalSamplingGap(
					replay,
					matchID,
					roundID,
					round.Round,
					targetTick,
					nil,
					tacticalGapMissingFrame,
					tacticalGeometryNotEvaluated,
				))
				previousTarget = targetTick
				continue
			}

			if len(selected.players) == 0 {
				sourceTick := selected.tick
				export.Gaps = append(export.Gaps, tacticalSamplingGap(
					replay,
					matchID,
					roundID,
					round.Round,
					targetTick,
					&sourceTick,
					tacticalGapNoJoinablePlayers,
					tacticalGeometryNotEvaluated,
				))
				previousTarget = targetTick
				continue
			}

			physicalBefore := len(export.PhysicalRows)
			appendTacticalFrameRows(
				&export,
				replay,
				matchID,
				roundID,
				round.Round,
				targetTick,
				*selected,
				geometry,
				geometryLoaded,
			)
			if len(export.PhysicalRows) == physicalBefore {
				sourceTick := selected.tick
				geometryStatus := tacticalGeometryNotLoaded
				if geometryLoaded {
					geometryStatus = tacticalGeometryLoaded
				}
				export.Gaps = append(export.Gaps, tacticalSamplingGap(
					replay,
					matchID,
					roundID,
					round.Round,
					targetTick,
					&sourceTick,
					tacticalGapNoObserver,
					geometryStatus,
				))
			}
			previousTarget = targetTick
		}
	}

	return export, nil
}

func tacticalTargetTicks(startTick, endTick int, periodTicks float64) []int {
	ticks := make([]int, 0, int(math.Ceil(float64(endTick-startTick)/periodTicks))+1)
	lastTick := startTick - 1
	for slot := 0; ; slot++ {
		targetTick := startTick + int(math.Round(float64(slot)*periodTicks))
		if targetTick > endTick {
			break
		}
		if targetTick > lastTick {
			ticks = append(ticks, targetTick)
			lastTick = targetTick
		}
	}
	return ticks
}

func canonicalTacticalFrames(round models.ReplayRound, tickRate float64) ([]tacticalFrameSample, error) {
	byTick := make(map[int]tacticalFrameSample, len(round.Frames))
	for _, frame := range round.Frames {
		if frame.Tick < round.StartTick || frame.Tick > round.EndTick {
			continue
		}

		players, err := canonicalTacticalPlayers(frame.Players)
		if err != nil {
			return nil, fmt.Errorf("frame tick %d: %w", frame.Tick, err)
		}
		keyBytes, err := json.Marshal(playersForTacticalKey(players))
		if err != nil {
			return nil, fmt.Errorf("frame tick %d: canonicalize players: %w", frame.Tick, err)
		}
		candidate := tacticalFrameSample{
			tick:    frame.Tick,
			players: players,
			key:     string(keyBytes),
		}

		current, exists := byTick[frame.Tick]
		if !exists || len(candidate.players) > len(current.players) ||
			(len(candidate.players) == len(current.players) && candidate.key < current.key) {
			byTick[frame.Tick] = candidate
		}
	}

	frames := make([]tacticalFrameSample, 0, len(byTick))
	for _, frame := range byTick {
		frames = append(frames, frame)
	}
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].tick < frames[j].tick
	})
	deriveTacticalVelocities(frames, tickRate)
	return frames, nil
}

func canonicalTacticalPlayers(players []models.ReplayPlayerState) ([]tacticalPlayerSample, error) {
	bySteamID := make(map[uint64]tacticalPlayerSample, len(players))
	for _, player := range players {
		if player.SteamID == 0 {
			continue
		}

		sample, err := newTacticalPlayerSample(player)
		if err != nil {
			return nil, fmt.Errorf("player steam:%d: %w", player.SteamID, err)
		}
		current, exists := bySteamID[player.SteamID]
		if !exists || sample.key < current.key {
			bySteamID[player.SteamID] = sample
		}
	}

	result := make([]tacticalPlayerSample, 0, len(bySteamID))
	for _, player := range bySteamID {
		result = append(result, player)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].steamID < result[j].steamID
	})
	return result, nil
}

func deriveTacticalVelocities(frames []tacticalFrameSample, tickRate float64) {
	type previousPosition struct {
		tick     int
		position r3.Vector
	}
	previous := make(map[uint64]previousPosition)
	for frameIndex := range frames {
		frame := &frames[frameIndex]
		for playerIndex := range frame.players {
			player := &frame.players[playerIndex]
			if player.position == nil {
				continue
			}
			if prior, exists := previous[player.steamID]; exists && frame.tick > prior.tick {
				seconds := float64(frame.tick-prior.tick) / tickRate
				if seconds > 0 {
					velocity := models.TacticalVector{
						X: (player.position.X - prior.position.X) / seconds,
						Y: (player.position.Y - prior.position.Y) / seconds,
						Z: (player.position.Z - prior.position.Z) / seconds,
					}
					horizontal := math.Hypot(velocity.X, velocity.Y)
					player.physical.Velocity = &velocity
					player.physical.HorizontalVelocity = &horizontal
					player.physical.FieldAvailability["velocity_world_units_per_second"] = "derived"
					player.physical.FieldAvailability["horizontal_velocity_world_units_per_second"] = "derived"
				}
			}
			player.oracle.Physical = player.physical
			previous[player.steamID] = previousPosition{tick: frame.tick, position: *player.position}
		}
	}
}

func newTacticalPlayerSample(player models.ReplayPlayerState) (tacticalPlayerSample, error) {
	team := canonicalTacticalTeam(player.Team)
	teamPointer := optionalTacticalString(team)
	position, geometryPosition := tacticalPosition(player)
	yaw := optionalFiniteFloat(float64(player.Yaw))
	pitch := optionalFiniteFloat(float64(player.Pitch))
	activeWeapon := optionalTacticalString(strings.TrimSpace(player.Weapon))
	flashDuration := optionalFiniteFloat(player.FlashDuration)
	var isBlind *bool
	if flashDuration != nil {
		isBlind = tacticalBoolPointer(*flashDuration > 0)
	}

	physical := models.TacticalPhysicalState{
		Team:           teamPointer,
		Position:       position,
		Yaw:            yaw,
		Pitch:          pitch,
		Health:         tacticalIntPointer(player.Health),
		Armor:          tacticalIntPointer(player.Armor),
		Alive:          tacticalBoolPointer(player.Alive),
		ActiveWeapon:   activeWeapon,
		Grenades:       canonicalTacticalGrenades(player.Weapons),
		HasC4:          tacticalBoolPointer(player.HasC4),
		HasDefuseKit:   tacticalBoolPointer(player.HasDefuseKit),
		AmmoInMagazine: nil,
		AmmoReserve:    nil,
		IsDucking:      tacticalBoolPointer(player.IsDucking),
		IsWalking:      tacticalBoolPointer(player.IsWalking),
		IsScoped:       tacticalBoolPointer(player.IsScoped),
		// ReplayPlayerState currently has no producer for IsReloading. Keep the
		// field null until a callback-backed observation is wired.
		IsReloading:          nil,
		IsBlind:              isBlind,
		FlashDurationSeconds: flashDuration,
		Money:                tacticalIntPointer(player.Money),
		IsDefusing:           tacticalBoolPointer(player.IsDefusing),
		FieldAvailability: tacticalBaseAvailability(
			teamPointer, position, yaw, pitch, activeWeapon, flashDuration,
		),
	}
	oracle := models.TacticalOracleState{
		Physical:             physical,
		Health:               tacticalIntPointer(player.Health),
		Armor:                tacticalIntPointer(player.Armor),
		Weapons:              canonicalTacticalWeapons(player.Weapons),
		HasDefuseKit:         tacticalBoolPointer(player.HasDefuseKit),
		HasHelmet:            tacticalBoolPointer(player.HasHelmet),
		HasC4:                tacticalBoolPointer(player.HasC4),
		FlashDurationSeconds: flashDuration,
		Money:                tacticalIntPointer(player.Money),
	}

	keyBytes, err := json.Marshal(struct {
		SteamID uint64                     `json:"steam_id"`
		State   models.TacticalOracleState `json:"state"`
	}{
		SteamID: player.SteamID,
		State:   oracle,
	})
	if err != nil {
		return tacticalPlayerSample{}, fmt.Errorf("canonicalize state: %w", err)
	}

	return tacticalPlayerSample{
		steamID:  player.SteamID,
		team:     team,
		position: geometryPosition,
		physical: physical,
		oracle:   oracle,
		key:      string(keyBytes),
	}, nil
}

func appendTacticalFrameRows(
	export *models.TacticalExport,
	replay *models.ReplayData,
	matchID string,
	roundID string,
	roundNumber int,
	targetTick int,
	frame tacticalFrameSample,
	geometry TacticalGeometry,
	geometryLoaded bool,
) {
	availabilityTick := frame.tick
	for _, subject := range frame.players {
		subjectID := tacticalPlayerID(subject.steamID)
		export.OracleRows = append(export.OracleRows, models.TacticalOracleRow{
			SchemaID:          models.TacticalOracleSchemaID,
			MatchID:           matchID,
			IdentitySemantics: models.TacticalIdentityJoinOnly,
			JoinKeys: models.TacticalJoinKeys{
				MatchID:   matchID,
				RoundID:   roundID,
				SubjectID: &subjectID,
			},
			RoundNumber:      roundNumber,
			Tick:             targetTick,
			AvailabilityTick: tacticalIntPointer(availabilityTick),
			Status:           models.TacticalStatusObserved,
			CausalRole:       models.TacticalCausalRoleOracle,
			VisibilityScope:  models.TacticalVisibilityOracle,
			Source:           models.TacticalSourceReplayPlayerState,
			Provenance: tacticalProvenance(
				replay,
				roundNumber,
				&availabilityTick,
				tacticalGeometryNotApplicable,
				nil,
			),
			State: subject.oracle,
		})
	}

	for _, observer := range frame.players {
		if observer.team == "" {
			continue
		}
		observerID := tacticalPlayerID(observer.steamID)
		for _, subject := range frame.players {
			visibilityScope, geometryStatus, lineOfSight, visible := tacticalVisibility(
				observer,
				subject,
				geometry,
				geometryLoaded,
			)
			if !visible {
				continue
			}

			subjectID := tacticalPlayerID(subject.steamID)
			export.PhysicalRows = append(export.PhysicalRows, models.TacticalPhysicalRow{
				SchemaID:          models.TacticalPhysicalSchemaID,
				MatchID:           matchID,
				IdentitySemantics: models.TacticalIdentityJoinOnly,
				JoinKeys: models.TacticalJoinKeys{
					MatchID:    matchID,
					RoundID:    roundID,
					ObserverID: &observerID,
					SubjectID:  &subjectID,
				},
				RoundNumber:      roundNumber,
				Tick:             targetTick,
				AvailabilityTick: tacticalIntPointer(availabilityTick),
				Status:           models.TacticalStatusObserved,
				CausalRole:       models.TacticalCausalRoleModelInput,
				VisibilityScope:  visibilityScope,
				Source:           models.TacticalSourceReplayPlayerState,
				Provenance: tacticalProvenance(
					replay,
					roundNumber,
					&availabilityTick,
					geometryStatus,
					lineOfSight,
				),
				State: tacticalPhysicalStateForScope(subject.physical, visibilityScope),
			})
		}
	}
}

func tacticalPhysicalStateForScope(
	state models.TacticalPhysicalState,
	scope string,
) models.TacticalPhysicalState {
	if scope == models.TacticalVisibilitySelf {
		state.FieldAvailability = copyTacticalAvailability(state.FieldAvailability)
		state.Grenades = append(make([]string, 0, len(state.Grenades)), state.Grenades...)
		return state
	}
	result := models.TacticalPhysicalState{
		Team:              state.Team,
		Position:          state.Position,
		Alive:             state.Alive,
		FieldAvailability: tacticalUnavailableAvailability(),
	}
	for _, field := range []string{"team", "position", "alive"} {
		result.FieldAvailability[field] = state.FieldAvailability[field]
	}
	if scope == models.TacticalVisibilityTeam {
		result.Velocity = state.Velocity
		result.HorizontalVelocity = state.HorizontalVelocity
		result.Health = state.Health
		result.Armor = state.Armor
		result.HasC4 = state.HasC4
		for _, field := range []string{
			"velocity_world_units_per_second",
			"horizontal_velocity_world_units_per_second",
			"health", "armor", "has_c4",
		} {
			result.FieldAvailability[field] = state.FieldAvailability[field]
		}
		return result
	}
	result.Yaw = state.Yaw
	result.Pitch = state.Pitch
	result.ActiveWeapon = state.ActiveWeapon
	result.IsDucking = state.IsDucking
	result.IsWalking = state.IsWalking
	result.IsScoped = state.IsScoped
	result.IsDefusing = state.IsDefusing
	for _, field := range []string{
		"yaw", "pitch", "active_weapon", "is_ducking", "is_walking",
		"is_scoped", "is_defusing",
	} {
		result.FieldAvailability[field] = state.FieldAvailability[field]
	}
	return result
}

func tacticalVisibility(
	observer tacticalPlayerSample,
	subject tacticalPlayerSample,
	geometry TacticalGeometry,
	geometryLoaded bool,
) (scope string, geometryStatus string, lineOfSight *bool, visible bool) {
	if observer.steamID == subject.steamID {
		return models.TacticalVisibilitySelf, tacticalGeometryNotRequired, nil, true
	}
	if subject.team != "" && observer.team == subject.team {
		return models.TacticalVisibilityTeam, tacticalGeometryNotRequired, nil, true
	}
	if subject.team == "" || !geometryLoaded || observer.position == nil || subject.position == nil {
		return "", tacticalGeometryNotLoaded, nil, false
	}

	hasLineOfSight := geometry.IsVisible(*observer.position, *subject.position)
	if !hasLineOfSight {
		return "", tacticalGeometryLoaded, tacticalBoolPointer(false), false
	}
	return models.TacticalVisibilityEnemyLOS, tacticalGeometryLoaded, tacticalBoolPointer(true), true
}

func tacticalSamplingGap(
	replay *models.ReplayData,
	matchID string,
	roundID string,
	roundNumber int,
	targetTick int,
	sourceFrameTick *int,
	reason string,
	geometryStatus string,
) models.TacticalSamplingGap {
	return models.TacticalSamplingGap{
		SchemaID:          models.TacticalSamplingGapSchemaID,
		MatchID:           matchID,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		JoinKeys: models.TacticalJoinKeys{
			MatchID: matchID,
			RoundID: roundID,
		},
		RoundNumber:      roundNumber,
		Tick:             targetTick,
		AvailabilityTick: nil,
		Status:           models.TacticalStatusUnavailable,
		CausalRole:       models.TacticalCausalRoleGap,
		VisibilityScope:  models.TacticalVisibilityGap,
		Source:           models.TacticalSourceReplaySampling,
		Reason:           reason,
		Provenance: tacticalProvenance(
			replay,
			roundNumber,
			sourceFrameTick,
			geometryStatus,
			nil,
		),
	}
}

func tacticalProvenance(
	replay *models.ReplayData,
	roundNumber int,
	frameTick *int,
	geometryStatus string,
	lineOfSight *bool,
) models.TacticalProvenance {
	return models.TacticalProvenance{
		SourceArtifact:      tacticalSourceArtifactReplayData,
		SourceSchemaVersion: optionalPositiveInt(replay.Metadata.SchemaVersion),
		SourceRound:         roundNumber,
		SourceFrameTick:     copyTacticalIntPointer(frameTick),
		GeometryStatus:      geometryStatus,
		LineOfSight:         copyTacticalBoolPointer(lineOfSight),
	}
}

func tacticalPosition(player models.ReplayPlayerState) (*models.TacticalVector, *r3.Vector) {
	if math.IsNaN(player.Z) || math.IsInf(player.Z, 0) {
		return nil, nil
	}
	vector := models.TacticalVector{
		X: float64(player.X),
		Y: float64(player.Y),
		Z: player.Z,
	}
	geometryVector := r3.Vector{X: vector.X, Y: vector.Y, Z: vector.Z}
	return &vector, &geometryVector
}

func canonicalTacticalTeam(team string) string {
	switch strings.ToUpper(strings.TrimSpace(team)) {
	case "CT":
		return "CT"
	case "T":
		return "T"
	default:
		return ""
	}
}

func canonicalTacticalWeapons(weapons []string) []string {
	if len(weapons) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, len(weapons))
	for _, weapon := range weapons {
		weapon = strings.TrimSpace(weapon)
		if weapon != "" {
			unique[weapon] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}
	result := make([]string, 0, len(unique))
	for weapon := range unique {
		result = append(result, weapon)
	}
	sort.Strings(result)
	return result
}

func canonicalTacticalGrenades(weapons []string) []string {
	result := make([]string, 0)
	for _, weapon := range canonicalTacticalWeapons(weapons) {
		lower := strings.ToLower(weapon)
		if strings.Contains(lower, "flash") || strings.Contains(lower, "smoke") ||
			strings.Contains(lower, "hegrenade") || strings.Contains(lower, "molotov") ||
			strings.Contains(lower, "incgrenade") || strings.Contains(lower, "decoy") {
			result = append(result, weapon)
		}
	}
	return result
}

func tacticalBaseAvailability(
	team *string,
	position *models.TacticalVector,
	yaw, pitch *float64,
	activeWeapon *string,
	flashDuration *float64,
) map[string]string {
	result := tacticalUnavailableAvailability()
	observedIf := func(field string, available bool) {
		if available {
			result[field] = "observed"
		}
	}
	observedIf("team", team != nil)
	observedIf("position", position != nil)
	observedIf("yaw", yaw != nil)
	observedIf("pitch", pitch != nil)
	for _, field := range []string{
		"health", "armor", "alive", "grenades", "has_c4", "has_defuse_kit",
		"is_ducking", "is_walking", "is_scoped", "money", "is_defusing",
	} {
		result[field] = "observed"
	}
	observedIf("active_weapon", activeWeapon != nil)
	observedIf("is_blind", flashDuration != nil)
	observedIf("flash_duration_seconds", flashDuration != nil)
	return result
}

func tacticalUnavailableAvailability() map[string]string {
	result := make(map[string]string, len(tacticalPhysicalFieldNames()))
	for _, field := range tacticalPhysicalFieldNames() {
		result[field] = "unavailable"
	}
	return result
}

func tacticalPhysicalFieldNames() []string {
	return []string{
		"team", "position", "velocity_world_units_per_second",
		"horizontal_velocity_world_units_per_second", "yaw", "pitch", "health",
		"armor", "alive", "active_weapon", "grenades", "has_c4", "has_defuse_kit",
		"ammo_in_magazine", "ammo_reserve", "is_ducking", "is_walking", "is_scoped",
		"is_reloading", "is_blind", "flash_duration_seconds", "money", "is_defusing",
	}
}

func copyTacticalAvailability(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func playersForTacticalKey(players []tacticalPlayerSample) []struct {
	SteamID uint64                     `json:"steam_id"`
	State   models.TacticalOracleState `json:"state"`
} {
	result := make([]struct {
		SteamID uint64                     `json:"steam_id"`
		State   models.TacticalOracleState `json:"state"`
	}, 0, len(players))
	for _, player := range players {
		result = append(result, struct {
			SteamID uint64                     `json:"steam_id"`
			State   models.TacticalOracleState `json:"state"`
		}{
			SteamID: player.steamID,
			State:   player.oracle,
		})
	}
	return result
}

func tacticalPlayerID(steamID uint64) string {
	return "steam:" + strconv.FormatUint(steamID, 10)
}

func optionalTacticalString(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}

func optionalFiniteFloat(value float64) *float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	result := value
	return &result
}

func optionalPositiveInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return tacticalIntPointer(value)
}

func tacticalIntPointer(value int) *int {
	result := value
	return &result
}

func tacticalBoolPointer(value bool) *bool {
	result := value
	return &result
}

func copyTacticalIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	return tacticalIntPointer(*value)
}

func copyTacticalBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	return tacticalBoolPointer(*value)
}
