package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"fmt"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"math"
	"strconv"
)

func (h *ReplayHandler) recordReplayContextEvent(e models.ReplayContextEvent) {
	if h.ctx == nil || !h.ctx.CaptureCalculatedVision || h.currentRound == nil || e.Source == "" || e.Tick < h.currentRound.StartTick {
		return
	}
	if h.currentRound.EventContext == nil {
		h.currentRound.EventContext = &models.ReplayEventContext{SchemaID: "stratai.replay_events@1", Events: []models.ReplayContextEvent{}}
	}
	c := h.currentRound.EventContext
	e.ID = fmt.Sprintf("r%d-action-%d", h.currentRound.Round, len(c.Events)+1)
	e.AvailabilityTick = e.Tick
	c.Events = append(c.Events, e)
}

func replayEventPosition(e *models.ReplayContextEvent, p r3.Vector) {
	e.X, e.Y, e.Z = nil, nil, nil
	if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsNaN(p.Z) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || math.IsInf(p.Z, 0) {
		return
	}
	e.X, e.Y, e.Z = &p.X, &p.Y, &p.Z
}

// A reused inventory item is not a unique projectile identity (e.g. two
// flashes). Ambiguous keys abstain; no later launch overwrites an earlier one.
func (h *ReplayHandler) replayUtilityLaunch(entityID int, equipmentID int64) (models.ReplayContextEvent, bool) {
	candidates := h.replayEventThrows[equipmentID]
	if entityID > 0 {
		entities := h.replayEventProjectileEntities[entityID]
		if len(entities) != 1 {
			return models.ReplayContextEvent{}, false
		}
		if equipmentID == 0 {
			return entities[0], true
		}
		for _, candidate := range candidates {
			if candidate.ProjectileID == entities[0].ProjectileID {
				return entities[0], true
			}
		}
		return models.ReplayContextEvent{}, false
	}
	if equipmentID != 0 && len(candidates) == 1 {
		return candidates[0], true
	}
	return models.ReplayContextEvent{}, false
}

func replayEventParticipant(e *models.ReplayContextEvent, p *common.Player) {
	if p != nil {
		e.ActorID = p.SteamID64
		e.ActorTeam = getTeamSide(p)
	}
}

func (h *ReplayHandler) registerReplayEvents() {
	if !h.ctx.CaptureCalculatedVision {
		return
	}
	p := h.ctx.Parser
	p.RegisterEventHandler(func(events.RoundStart) {
		h.replayEventThrows = make(map[int64][]models.ReplayContextEvent)
		h.replayEventProjectileEntities = make(map[int][]models.ReplayContextEvent)
		h.replayEventInfernos = make(map[int64]models.ReplayContextEvent)
	})
	p.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil {
			return
		}
		row := models.ReplayContextEvent{Type: "weapon_fire", Tick: p.GameState().IngameTick(), Source: "demoinfocs_event:WeaponFire"}
		replayEventParticipant(&row, e.Shooter)
		replayEventPosition(&row, e.Shooter.Position())
		if e.Weapon != nil {
			row.Weapon = e.Weapon.String()
		}
		h.recordReplayContextEvent(row)
	})
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Player == nil {
			return
		}
		row := models.ReplayContextEvent{Type: "player_hurt", Tick: p.GameState().IngameTick(), Source: "demoinfocs_event:PlayerHurt", VictimID: e.Player.SteamID64, VictimTeam: getTeamSide(e.Player), Damage: e.HealthDamageTaken, Health: e.Health}
		replayEventParticipant(&row, e.Attacker)
		replayEventPosition(&row, e.Player.Position())
		if e.Weapon != nil {
			row.Weapon = e.Weapon.String()
		}
		h.recordReplayContextEvent(row)
	})
	p.RegisterEventHandler(func(e events.HeExplode) {
		h.recordReplayUtilityEffect(e.Grenade, e.GrenadeEntityID, e.Position, "utility_start", "demoinfocs_event:HeExplode")
		h.observeReplayUtilityPoint("effect", fmt.Sprintf("he:%d:tick:%d", e.GrenadeEntityID, p.GameState().IngameTick()), "he", e.Position, "demoinfocs_event:HeExplode", "native_explosion_point", "")
	})
	p.RegisterEventHandler(func(e events.FlashExplode) {
		h.recordReplayUtilityEffect(e.Grenade, e.GrenadeEntityID, e.Position, "utility_start", "demoinfocs_event:FlashExplode")
		h.observeReplayUtilityPoint("effect", fmt.Sprintf("flash:%d:tick:%d", e.GrenadeEntityID, p.GameState().IngameTick()), "flashbang", e.Position, "demoinfocs_event:FlashExplode", "native_explosion_point", "")
	})
	p.RegisterEventHandler(func(e events.DecoyStart) {
		h.recordReplayUtilityEffect(e.Grenade, e.GrenadeEntityID, e.Position, "utility_start", "demoinfocs_event:DecoyStart")
	})
	p.RegisterEventHandler(func(e events.DecoyExpired) {
		h.recordReplayUtilityEffect(e.Grenade, e.GrenadeEntityID, e.Position, "utility_end", "demoinfocs_event:DecoyExpired")
	})
}

// Creation and factual recording run in one callback; dispatcher map order
// must never decide whether a positive native launch or effect is retained.
func (h *ReplayHandler) recordReplayGrenadeThrow(e events.GrenadeProjectileThrow) {
	if !h.ctx.CaptureCalculatedVision {
		return
	}
	p := h.ctx.Parser
	if e.Projectile == nil || h.currentRound == nil {
		return
	}
	projectile := h.activeProjectiles[e.Projectile.UniqueID()]
	if projectile == nil {
		return
	}
	row := models.ReplayContextEvent{Type: "grenade_throw", Tick: p.GameState().IngameTick(), Source: "demoinfocs_event:GrenadeProjectileThrow", UtilityType: projectile.Type, ProjectileID: strconv.FormatInt(projectile.ID, 10)}
	replayEventParticipant(&row, e.Projectile.Thrower)
	replayEventPosition(&row, e.Projectile.Position())
	h.recordReplayContextEvent(row)
	if key := playerstate.EquipmentID(e.Projectile.WeaponInstance); key != 0 {
		h.replayEventThrows[key] = append(h.replayEventThrows[key], row)
	}
	if entity := e.Projectile.Entity; entity != nil {
		h.replayEventProjectileEntities[entity.ID()] = append(h.replayEventProjectileEntities[entity.ID()], row)
	}
}

// Equipment identity is a positive launch/effect link. Never substitute the
// final utility ledger's nearest-position correlation or expiry countdown.
func (h *ReplayHandler) recordReplayUtilityEffect(grenade *common.Equipment, entityID int, position r3.Vector, kind, source string) {
	if !h.ctx.CaptureCalculatedVision {
		return
	}
	p := h.ctx.Parser
	key := playerstate.EquipmentID(grenade)
	launch, ok := h.replayUtilityLaunch(entityID, key)
	if !ok {
		if h.currentRound != nil && h.currentRound.EventContext != nil {
			c := h.currentRound.EventContext
			if c.Abstentions == nil {
				c.Abstentions = make(map[string]int)
			}
			c.Abstentions["launch_link_unavailable:"+source]++
		}
		return
	}
	row := launch
	row.Type = kind
	row.Tick = p.GameState().IngameTick()
	row.Source = source
	replayEventPosition(&row, position)
	h.recordReplayContextEvent(row)
	if kind == "utility_start" && launch.UtilityType == "smoke" {
		if smoke := h.activeSmokes[key]; smoke != nil {
			smoke.ProjectileID = launch.ProjectileID
			smoke.ThrowerID = launch.ActorID
		}
	}
}

func (h *ReplayHandler) recordReplayInfernoEvent(inferno *common.Inferno, kind, source string) {
	if !h.ctx.CaptureCalculatedVision {
		return
	}
	p := h.ctx.Parser
	key, ok := replayInfernoKey(inferno)
	if !ok || h.currentRound == nil {
		return
	}
	row := models.ReplayContextEvent{Type: kind, Tick: p.GameState().IngameTick(), UtilityType: "inferno", Source: source,
		EffectID: fmt.Sprintf("r%d-inferno-%d", h.currentRound.Round, key)}
	replayEventParticipant(&row, inferno.Thrower())
	if kind == "utility_end" {
		if start, ok := h.replayEventInfernos[key]; ok {
			row.ActorID, row.ActorTeam = start.ActorID, start.ActorTeam
		}
	}
	// The native owner is available even when a unique thrown projectile is
	// not. Keep the effect identity separate instead of guessing a link.
	if active := h.activeInfernos[key]; active != nil {
		active.EffectID = row.EffectID
		active.ThrowerID = row.ActorID
		// Expiry does not observe a fresh position of the historical fire center.
		if kind == "utility_start" {
			x, y := active.X, active.Y
			row.X, row.Y = &x, &y
		}
	}
	h.recordReplayContextEvent(row)
	if kind == "utility_start" {
		h.replayEventInfernos[key] = row
	}
}
