package sound

import (
	"encoding/hex"
	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"slices"
)

func nativePoint(point *msg.CMsgVector) *r3.Vector {
	if point == nil || point.X == nil || point.Y == nil || point.Z == nil {
		return nil
	}
	p := r3.Vector{X: float64(*point.X), Y: float64(*point.Y), Z: float64(*point.Z)}
	if !finite(p.X) || !finite(p.Y) || !finite(p.Z) {
		return nil
	}
	return &p
}
func candidateActor(c *Candidate, player *common.Player) {
	if player == nil {
		return
	}
	c.EmitterID = player.SteamID64
	if player.Team == common.TeamTerrorists {
		c.EmitterSide = "T"
	}
	if player.Team == common.TeamCounterTerrorists {
		c.EmitterSide = "CT"
	}
}
func candidateEntity(p dem.Parser, c *Candidate, index int) {
	c.EntityIndex = index
	if player := p.GameState().Participants().ByEntityID()[index]; player != nil {
		candidateActor(c, player)
	}
	for _, player := range p.GameState().Participants().All() {
		if player != nil && player.PlayerPawnEntity() != nil && player.PlayerPawnEntity().ID() == index {
			candidateActor(c, player)
			break
		}
	}
	if c.Position == nil {
		if entity := p.GameState().Entities()[index]; entity != nil {
			pos := entity.Position()
			if finite(pos.X) && finite(pos.Y) && finite(pos.Z) {
				c.Position = &pos
				c.PositionSource = "native_sound_entity_position"
			}
		}
	}
}

// Native positions/life at the message callback, without future replay frames,
// FOV, visual occlusion, smoke or an attention/hearing assumption.
func NativeSoundObservers(p dem.Parser) []Observer {
	players := p.GameState().Participants().All()
	slices.SortFunc(players, func(a, b *common.Player) int {
		if a == nil && b == nil {
			return 0
		}
		if a == nil {
			return -1
		}
		if b == nil {
			return 1
		}
		if a.SteamID64 < b.SteamID64 {
			return -1
		}
		if a.SteamID64 > b.SteamID64 {
			return 1
		}
		return 0
	})
	result := []Observer{}
	for _, player := range players {
		if player == nil || player.SteamID64 == 0 || player.PlayerPawnEntity() == nil {
			continue
		}
		pos := player.PlayerPawnEntity().Position()
		if !finite(pos.X) || !finite(pos.Y) || !finite(pos.Z) {
			continue
		}
		side := ""
		if player.Team == common.TeamTerrorists {
			side = "T"
		}
		if player.Team == common.TeamCounterTerrorists {
			side = "CT"
		}
		if side == "" {
			continue
		}
		result = append(result, Observer{ID: player.SteamID64, Side: side, Alive: player.IsAlive(), Position: pos})
	}
	return result
}

// Pinned v5.2.0 proto APIs. Only positive native sound messages are adapted.
// Generic actions, movement speed and zoom state never synthesize emissions.
func RegisterNativeSounds(p dem.Parser, emit func(Candidate), abstain func(string)) {
	p.RegisterNetMessageHandler(func(e *msg.CMsgSosStartSoundEvent) {
		profile, ok := LookupHash(e.GetSoundeventHash())
		if !ok {
			abstain("native_sound_hash_not_in_catalog")
			return
		}
		params, err := DecodeParameters(e.PackedParams)
		if err != nil {
			abstain("native_sound_parameters_unsupported")
			return
		}
		if params.ImpactSpeed != nil {
			// Physics gain needs the event's separate impact-speed curve. A
			// geometric radius alone cannot establish a nonzero emission gain.
			abstain("native_dynamic_impact_gain_unsupported")
			return
		}
		c := Candidate{Tick: p.GameState().IngameTick(), Source: "net.CMsgSosStartSoundEvent", Profile: profile.Event,
			Position: params.Position, PositionSource: "native_sos_public.position", Volume: params.Volume,
			ParamsHex: hex.EncodeToString(e.PackedParams), NativeHash: e.SoundeventHash}
		if e.SourceEntityIndex != nil && *e.SourceEntityIndex >= 0 {
			candidateEntity(p, &c, int(*e.SourceEntityIndex))
		}
		emit(c)
	})
	p.RegisterNetMessageHandler(func(e *msg.CCSUsrMsg_WeaponSound) {
		if e.Sound == nil {
			abstain("native_weapon_sound_name_unavailable")
			return
		}
		var point *r3.Vector
		if e.OriginX != nil && e.OriginY != nil && e.OriginZ != nil {
			point = &r3.Vector{X: float64(*e.OriginX), Y: float64(*e.OriginY), Z: float64(*e.OriginZ)}
		}
		c := Candidate{Tick: p.GameState().IngameTick(), Source: "net.CCSUsrMsg_WeaponSound", Profile: *e.Sound,
			Position: point, PositionSource: "native_weapon_sound_origin_xyz"}
		if e.Entidx != nil && *e.Entidx >= 0 {
			candidateEntity(p, &c, int(*e.Entidx))
		}
		emit(c)
	})
	p.RegisterNetMessageHandler(func(e *msg.CMsgTEFireBullets) {
		if e.ItemDefIndex == nil || e.SoundType == nil {
			abstain("native_fire_sound_type_or_item_unavailable")
			return
		}
		profile, ok := FireProfile(*e.ItemDefIndex, *e.SoundType)
		if !ok || profile.Family != "gunshot" {
			abstain("native_fire_sound_mapping_unavailable")
			return
		}
		c := Candidate{Tick: p.GameState().IngameTick(), Source: "net.CMsgTEFireBullets", Profile: profile.Event,
			Position: nativePoint(e.EntOrigin), PositionSource: "native_fire_ent_origin", ItemDefinition: e.ItemDefIndex, SoundType: e.SoundType}
		if c.Position == nil {
			c.Position = nativePoint(e.Origin)
			c.PositionSource = "native_fire_origin"
		}
		if e.Player != nil {
			player := p.GameState().Participants().FindByPawnHandle(uint64(*e.Player))
			if player == nil {
				player = p.GameState().Participants().FindByHandle64(uint64(*e.Player))
			}
			candidateActor(&c, player)
		}
		emit(c)
	})
}
