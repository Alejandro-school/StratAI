// Package sound observes positive native sound sources separately from actions.
// This audit grants no audible reception: native range semantics must be proven
// before a spatial information producer can consume them.
package sound

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"slices"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

const SourceAuditSchema = "stratai.native_sound_source_audit@1"

type Signal struct {
	Tick             int        `json:"tick"`
	AvailabilityTick int        `json:"availability_tick"`
	Round            int        `json:"round"`
	Warmup           bool       `json:"warmup"`
	Source           string     `json:"source"`
	Family           string     `json:"family"`
	EmitterID        uint64     `json:"emitter_id,omitempty"`
	EmitterSide      string     `json:"emitter_side,omitempty"`
	Position         *r3.Vector `json:"native_position,omitempty"`
	PositionSource   string     `json:"position_source,omitempty"`
	Radius           *int       `json:"native_radius,omitempty"`
	RangeStatus      string     `json:"range_status"`
	DurationNS       *int64     `json:"native_duration_ns,omitempty"`
	EntityIndex      *int32     `json:"native_entity_index,omitempty"`
	SoundName        *string    `json:"native_sound_name,omitempty"`
	SoundHash        *uint32    `json:"native_sound_hash,omitempty"`
	SoundLevel       *int32     `json:"native_sound_level,omitempty"`
	PackedBytes      int        `json:"packed_bytes,omitempty"`
	PackedSHA256     string     `json:"packed_sha256,omitempty"`
}

type SourceAudit struct {
	SchemaID          string              `json:"schema_id"`
	Dependency        string              `json:"dependency"`
	ReceptionStatus   string              `json:"reception_status"`
	Counts            map[string]int      `json:"positive_signal_counts"`
	Packets           map[string]int      `json:"native_message_counts"`
	Actions           map[string]int      `json:"action_counts_not_sound"`
	RawEvents         map[string]int      `json:"raw_game_event_counts"`
	ResolvedEvents    map[string]int      `json:"resolved_game_event_counts"`
	ResolvedSoundKeys map[string][]string `json:"resolved_sound_event_keys"`
	Descriptors       map[string][]string `json:"sound_event_descriptor_keys"`
	Signals           []Signal            `json:"signals"`
}

func NewSourceAudit() *SourceAudit {
	a := &SourceAudit{SchemaID: SourceAuditSchema, Dependency: "demoinfocs-golang/v5@v5.2.0",
		ReceptionStatus: "unavailable", Counts: map[string]int{}, Packets: map[string]int{},
		Actions: map[string]int{}, RawEvents: map[string]int{}, ResolvedEvents: map[string]int{},
		ResolvedSoundKeys: map[string][]string{}, Descriptors: map[string][]string{}, Signals: []Signal{}}
	for _, source := range []string{"events.Footstep", "events.PlayerSound", "net.CSVCMsg_Sounds", "net.CCSUsrMsg_WeaponSound", "net.CMsgSosStartSoundEvent"} {
		a.Counts[source] = 0
	}
	for _, name := range []string{"CSVCMsg_Sounds", "CSVCMsg_StopSound", "CCSUsrMsg_WeaponSound", "CMsgSosStartSoundEvent", "CMsgSosStopSoundEvent", "CMsgSosSetSoundEventParams", "CMsgSosSetLibraryStackFields", "CMsgSosStopSoundEventHash"} {
		a.Packets[name] = 0
	}
	for _, name := range []string{"weapon_fire", "weapon_reload", "grenade_projectile_throw", "bomb_plant_begin", "bomb_planted", "bomb_defuse_start", "bomb_defused", "bomb_explode"} {
		a.Actions[name] = 0
	}
	return a
}

func intPointer(value int) *int { return &value }

func (a *SourceAudit) record(signal Signal) {
	signal.AvailabilityTick = signal.Tick
	signal.RangeStatus = "unavailable"
	if signal.Radius != nil {
		signal.RangeStatus = "native_radius_semantics_unverified"
		if *signal.Radius <= 0 {
			signal.RangeStatus = "invalid_native_radius"
		}
	}
	a.Counts[signal.Source]++
	a.Signals = append(a.Signals, signal)
}

func nativeEmitter(signal *Signal, player *common.Player) {
	if player == nil {
		return
	}
	signal.EmitterID = player.SteamID64
	switch player.Team {
	case common.TeamTerrorists:
		signal.EmitterSide = "T"
	case common.TeamCounterTerrorists:
		signal.EmitterSide = "CT"
	}
	// Player.Position() returns zero when the pawn is missing. Do not convert
	// that fallback into observed coordinates.
	if pawn := player.PlayerPawnEntity(); pawn != nil {
		pos := pawn.Position()
		if finite(pos.X) && finite(pos.Y) && finite(pos.Z) {
			signal.Position = &pos
			signal.PositionSource = "demoinfocs_state:PlayerPawnEntity.Position"
		}
	}
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func (a *SourceAudit) recordResolved(e events.GenericGameEvent) {
	a.ResolvedEvents[e.Name]++
	if e.Name == "player_sound" || e.Name == "player_footstep" {
		keys := make([]string, 0, len(e.Data))
		for key := range e.Data {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		a.ResolvedSoundKeys[e.Name] = keys
	}
}

// RegisterSourceAudit is deliberately opt-in. It observes the pinned decoder's
// typed callbacks AND lower-level decoded packets, with independent raw counts.
// A descriptor is a supported definition, never a positive emitted sound.
func RegisterSourceAudit(p dem.Parser) *SourceAudit {
	a := NewSourceAudit()
	// CS2 may omit the descriptor packet. The pinned parser then loads its
	// internal event list; GenericGameEvent still provides the resolved name.
	// Keep this count separate from the raw packet's available name.
	p.RegisterEventHandler(func(e events.GenericGameEvent) { a.recordResolved(e) })
	native := func(source, family string, player *common.Player) Signal {
		state := p.GameState()
		signal := Signal{Tick: state.IngameTick(), Round: state.TotalRoundsPlayed() + 1,
			Warmup: state.IsWarmupPeriod(), Source: source, Family: family}
		nativeEmitter(&signal, player)
		return signal
	}
	p.RegisterEventHandler(func(e events.Footstep) {
		a.record(native("events.Footstep", "footstep", e.Player))
	})
	p.RegisterEventHandler(func(e events.PlayerSound) {
		signal := native("events.PlayerSound", "unknown", e.Player)
		signal.Radius = intPointer(e.Radius)
		duration := int64(e.Duration)
		signal.DurationNS = &duration
		a.record(signal)
	})
	p.RegisterEventHandler(func(e events.WeaponFire) { a.Actions["weapon_fire"]++ })
	p.RegisterEventHandler(func(e events.WeaponReload) { a.Actions["weapon_reload"]++ })
	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) { a.Actions["grenade_projectile_throw"]++ })
	p.RegisterEventHandler(func(e events.BombPlantBegin) { a.Actions["bomb_plant_begin"]++ })
	p.RegisterEventHandler(func(e events.BombPlanted) { a.Actions["bomb_planted"]++ })
	p.RegisterEventHandler(func(e events.BombDefuseStart) { a.Actions["bomb_defuse_start"]++ })
	p.RegisterEventHandler(func(e events.BombDefused) { a.Actions["bomb_defused"]++ })
	p.RegisterEventHandler(func(e events.BombExplode) { a.Actions["bomb_explode"]++ })
	eventNames := map[int32]string{}
	p.RegisterNetMessageHandler(func(e *msg.CMsgSource1LegacyGameEventList) {
		for _, desc := range e.GetDescriptors() {
			name := desc.GetName()
			eventNames[desc.GetEventid()] = name
			if name == "player_sound" || name == "player_footstep" {
				keys := []string{}
				for _, key := range desc.GetKeys() {
					keys = append(keys, key.GetName())
				}
				slices.Sort(keys)
				a.Descriptors[name] = keys
			}
		}
	})
	p.RegisterNetMessageHandler(func(e *msg.CMsgSource1LegacyGameEvent) {
		name := e.GetEventName()
		if name == "" {
			name = eventNames[e.GetEventid()]
		}
		if name == "" {
			name = "unknown_descriptor"
		}
		a.RawEvents[name]++
	})
	p.RegisterNetMessageHandler(func(e *msg.CCSUsrMsg_WeaponSound) {
		a.Packets["CCSUsrMsg_WeaponSound"]++
		signal := native("net.CCSUsrMsg_WeaponSound", "unknown", nil)
		signal.EntityIndex, signal.SoundName = e.Entidx, e.Sound
		if e.OriginX != nil && e.OriginY != nil && e.OriginZ != nil {
			pos := r3.Vector{X: float64(*e.OriginX), Y: float64(*e.OriginY), Z: float64(*e.OriginZ)}
			if finite(pos.X) && finite(pos.Y) && finite(pos.Z) {
				signal.Position, signal.PositionSource = &pos, "net.CCSUsrMsg_WeaponSound:origin_xyz"
			}
		}
		a.record(signal)
	})
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosStartSoundEvent) {
		a.Packets["CMsgSosStartSoundEvent"]++
		signal := native("net.CMsgSosStartSoundEvent", "unknown", nil)
		signal.EntityIndex, signal.SoundHash = e.SourceEntityIndex, e.SoundeventHash
		signal.PackedBytes = len(e.PackedParams)
		hash := sha256.Sum256(e.PackedParams)
		signal.PackedSHA256 = hex.EncodeToString(hash[:])
		a.record(signal)
	})
	p.RegisterNetMessageHandler(func(e *msg.CSVCMsg_Sounds) {
		a.Packets["CSVCMsg_Sounds"]++
		for _, data := range e.GetSounds() {
			signal := native("net.CSVCMsg_Sounds", "unknown", nil)
			signal.EntityIndex, signal.SoundLevel = data.EntityIndex, data.SoundLevel
			if data.OriginX != nil && data.OriginY != nil && data.OriginZ != nil {
				pos := r3.Vector{X: float64(*data.OriginX), Y: float64(*data.OriginY), Z: float64(*data.OriginZ)}
				signal.Position, signal.PositionSource = &pos, "net.CSVCMsg_Sounds:origin_xyz"
			}
			a.record(signal)
		}
	})
	p.RegisterNetMessageHandler(func(e *msg.CSVCMsg_StopSound) { a.Packets["CSVCMsg_StopSound"]++ })
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosStopSoundEvent) { a.Packets["CMsgSosStopSoundEvent"]++ })
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosSetSoundEventParams) { a.Packets["CMsgSosSetSoundEventParams"]++ })
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosSetLibraryStackFields) { a.Packets["CMsgSosSetLibraryStackFields"]++ })
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosStopSoundEventHash) { a.Packets["CMsgSosStopSoundEventHash"]++ })
	return a
}
