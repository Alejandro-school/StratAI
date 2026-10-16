package handlers

import (
	"cs2-demo-service/analyzers"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"fmt"
	"slices"
	"strconv"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type replayUtilityVisualEffect struct {
	ID, Type, Source string
	Position         r3.Vector
	SmokeKey         int64
	SmokeEntityID    int
	InfernoKey       int64
	Inferno          *common.Inferno
}

func (h *ReplayHandler) utilityVisionLedger() *models.ReplayUtilityVision {
	if h.ctx == nil || !h.ctx.CaptureUtilityVision || h.currentRound == nil {
		return nil
	}
	if h.currentRound.UtilityVision == nil {
		h.currentRound.UtilityVision = &models.ReplayUtilityVision{SchemaID: "stratai.calculated_utility_vision@1",
			Producer: "go.replay.utility_point_occlusion_v1", Observations: []models.ReplayUtilityObservation{}, Abstentions: map[string]int{}}
	}
	return h.currentRound.UtilityVision
}

// Only current native projectiles are sampled; cached/final trajectories are
// not observation inputs. Every observer/object sample has its own timestamp.
func (h *ReplayHandler) captureReplayUtilityVisionFrame(tick int) {
	if h.utilityVisionLedger() == nil || tick%4 != 0 {
		return
	}
	projectiles := []*common.GrenadeProjectile{}
	for _, projectile := range h.ctx.Parser.GameState().GrenadeProjectiles() {
		if projectile != nil {
			projectiles = append(projectiles, projectile)
		}
	}
	slices.SortFunc(projectiles, func(a, b *common.GrenadeProjectile) int {
		if a.UniqueID() < b.UniqueID() {
			return -1
		}
		if a.UniqueID() > b.UniqueID() {
			return 1
		}
		return 0
	})
	for _, projectile := range projectiles {
		if projectile == nil {
			continue
		}
		active := h.activeProjectiles[projectile.UniqueID()]
		if active != nil {
			h.observeReplayUtilityPoint("projectile", "projectile:"+strconv.FormatInt(active.ID, 10), active.Type,
				projectile.Position(), "demoinfocs_state:GrenadeProjectiles", "native_projectile_position", "")
		}
	}
	keys := make([]string, 0, len(h.utilityVisualEffects))
	for key := range h.utilityVisualEffects {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		effect := h.utilityVisualEffects[key]
		point := effect.Position
		if effect.Type == "inferno" {
			fires := effect.Inferno.Fires().Active().List()
			if len(fires) == 0 {
				continue
			}
			point = fires[0].Vector
		}
		h.observeReplayUtilityPoint("effect", effect.ID, effect.Type, point, effect.Source,
			"effect_representative_point_not_full_geometry", effect.ID)
	}
}

func (h *ReplayHandler) observeReplayUtilityPoint(kind, objectID, utilityType string, point r3.Vector, source, positionBasis, ignoredSmokeID string) {
	ledger := h.utilityVisionLedger()
	if ledger == nil {
		return
	}
	if h.utilitySmokeOcclusionUnavailable {
		ledger.Abstentions["smoke_occlusion_source_unavailable"]++
		return
	}
	tick := h.ctx.Parser.GameState().IngameTick()
	if tick < h.currentRound.StartTick || objectID == "" || source == "" {
		ledger.Abstentions["invalid_point_provenance"]++
		return
	}
	smokes := make(map[int64]r3.Vector)
	for _, effect := range h.utilityVisualEffects {
		if effect.Type == "smoke" && effect.ID != ignoredSmokeID {
			smokes[int64(len(smokes)+1)] = effect.Position
		}
	}
	players := slices.Clone(h.ctx.Parser.GameState().Participants().Playing())
	slices.SortFunc(players, compareReplayParticipants)
	for _, player := range players {
		if player == nil || player.SteamID64 == 0 || !player.IsAlive() || !isCompetitiveTeam(getTeamSide(player)) {
			continue
		}
		visibility := analyzers.EstimateUtilityPointVisibility(h.ctx, player, point, smokes)
		if !visibility.Available {
			ledger.Abstentions["visibility_source_unavailable"]++
			continue
		}
		if !visibility.Visible {
			continue
		}
		ledger.Observations = append(ledger.Observations, models.ReplayUtilityObservation{
			ID:   fmt.Sprintf("r%d-utility-observation-%d", h.currentRound.Round, len(ledger.Observations)+1),
			Tick: tick, AvailabilityTick: tick, ObserverID: player.SteamID64, ObserverSide: getTeamSide(player),
			ObjectID: objectID, Kind: kind, UtilityType: utilityType, TypeStatus: "estimated_native_type_proxy",
			X: point.X, Y: point.Y, Z: point.Z, Source: source, EyeSource: visibility.EyeSource, PositionBasis: positionBasis,
		})
	}
}

func (h *ReplayHandler) startReplayUtilityVisualSmoke(e events.SmokeStart) {
	if h.utilityVisionLedger() == nil {
		return
	}
	key := playerstate.EquipmentID(e.Grenade)
	if key == 0 && e.GrenadeEntityID <= 0 {
		h.currentRound.UtilityVision.Abstentions["smoke_identity_unavailable"]++
		// An unlinked cloud is still an occluder. Until a round reset provides
		// a clean lifecycle, missing identity cannot mean clear visibility.
		h.utilitySmokeOcclusionUnavailable = true
		return
	}
	if h.utilityVisualEffects == nil {
		h.utilityVisualEffects = make(map[string]replayUtilityVisualEffect)
	}
	id := fmt.Sprintf("smoke:%d:start:%d", key, h.ctx.Parser.GameState().IngameTick())
	if e.GrenadeEntityID > 0 {
		id = fmt.Sprintf("smoke:entity:%d:start:%d", e.GrenadeEntityID, h.ctx.Parser.GameState().IngameTick())
	}
	h.utilityVisualEffects[id] = replayUtilityVisualEffect{ID: id, Type: "smoke", Position: e.Position,
		SmokeKey: key, SmokeEntityID: e.GrenadeEntityID, Source: "demoinfocs_event:SmokeStart"}
	h.observeReplayUtilityPoint("effect", id, "smoke", e.Position, "demoinfocs_event:SmokeStart", "effect_representative_point_not_full_geometry", id)
}

func (h *ReplayHandler) endReplayUtilityVisualSmoke(e events.SmokeExpired) {
	key := playerstate.EquipmentID(e.Grenade)
	for id, effect := range h.utilityVisualEffects {
		if effect.Type == "smoke" && ((e.GrenadeEntityID > 0 && effect.SmokeEntityID == e.GrenadeEntityID) || (key != 0 && effect.SmokeKey == key)) {
			delete(h.utilityVisualEffects, id)
		}
	}
}

func (h *ReplayHandler) startReplayUtilityVisualInferno(inferno *common.Inferno) {
	if h.utilityVisionLedger() == nil {
		return
	}
	key, ok := replayInfernoKey(inferno)
	if inferno == nil {
		return
	}
	fires := inferno.Fires().Active().List()
	if !ok || len(fires) == 0 {
		h.currentRound.UtilityVision.Abstentions["fire_point_unavailable"]++
		return
	}
	if h.utilityVisualEffects == nil {
		h.utilityVisualEffects = make(map[string]replayUtilityVisualEffect)
	}
	id := fmt.Sprintf("fire:%d:start:%d", key, h.ctx.Parser.GameState().IngameTick())
	point := r3.Vector{X: fires[0].X, Y: fires[0].Y, Z: fires[0].Z}
	h.utilityVisualEffects[id] = replayUtilityVisualEffect{ID: id, Type: "inferno", Position: point,
		InfernoKey: key, Inferno: inferno, Source: "demoinfocs_event:InfernoStart"}
	h.observeReplayUtilityPoint("effect", id, "inferno", point, "demoinfocs_event:InfernoStart", "effect_representative_point_not_full_geometry", "")
}

func (h *ReplayHandler) endReplayUtilityVisualInferno(key int64) {
	for id, effect := range h.utilityVisualEffects {
		if effect.Type == "inferno" && effect.InfernoKey == key {
			delete(h.utilityVisualEffects, id)
		}
	}
}

func isCompetitiveTeam(side string) bool { return side == "T" || side == "CT" }

// A callback can precede a kill within the same tick while the pawn still
// reports alive. Our tick-granularity consumer cannot certify that ordering;
// conservatively abstain that tick, while preserving earlier observations.
func reconcileReplayUtilityVisionLife(round *models.ReplayRoundData) {
	if round == nil || round.UtilityVision == nil {
		return
	}
	deaths := map[uint64]int{}
	for _, event := range round.Events {
		if event.Type != "kill" || event.VictimID == 0 || event.Tick < round.StartTick {
			continue
		}
		when, exists := deaths[event.VictimID]
		if !exists || event.Tick < when {
			deaths[event.VictimID] = event.Tick
		}
	}
	kept := round.UtilityVision.Observations[:0]
	for _, row := range round.UtilityVision.Observations {
		if death, exists := deaths[row.ObserverID]; exists && row.Tick >= death {
			round.UtilityVision.Abstentions["observer_death_at_or_before_observation_tick"]++
			continue
		}
		kept = append(kept, row)
	}
	round.UtilityVision.Observations = kept
}
