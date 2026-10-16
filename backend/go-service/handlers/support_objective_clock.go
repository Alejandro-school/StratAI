package handlers

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"google.golang.org/protobuf/proto"
)

var supportClockRules = []string{"m_fRoundStartTime", "m_iRoundTime", "m_flRestartRoundTime", "m_bFreezePeriod", "m_bWarmupPeriod", "m_bMatchWaitingForResume", "m_bTerroristTimeOutActive", "m_bCTTimeOutActive", "m_bTechnicalTimeOut", "m_bGamePaused", "m_bGameRestart"}
var supportClockPawn = []string{"m_flSimulationTime", "m_nTickBase"}
var supportClockBomb = []string{"m_flSimulationTime", "m_flC4Blow", "m_flTimerLength", "m_flDefuseCountDown", "m_flDefuseLength", "m_bBombTicking", "m_bBombDefused"}
var supportClockItem = []string{"m_flSimulationTime", "m_bStartedArming", "m_fArmedTime", "m_flArmingLength"}

type supportClockField struct{ typ, nested string }
type supportClockTracker struct {
	order   int
	network map[string]*models.SupportClockMessage
	types   map[string]map[string]supportClockField
	objects map[st.Entity]*supportC4Life
}

func clockScalar(raw any, declared string) models.SupportClockScalar {
	r := models.SupportClockScalar{Status: "absent"}
	if declared != "" {
		r.DeclaredType = &declared
	}
	if raw == nil {
		return r
	}
	typ := fmt.Sprintf("%T", raw)
	r.NativeType, r.Status = &typ, "observed"
	switch v := raw.(type) {
	case bool, int, int32, uint32, int64, uint64:
		r.Value = v
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			r.Status = "nonfinite"
		} else {
			r.Value = float64(v)
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			r.Status = "nonfinite"
		} else {
			r.Value = v
		}
	default:
		r.Status = "unsupported_type"
	}
	return r
}

// CDemoSendTables contains the same length-prefixed protobuf consumed by the
// pinned SDK. Resolve the complete class/property path, including game rules.
func (t *supportClockTracker) sendTables(raw []byte) {
	t.types = map[string]map[string]supportClockField{}
	size, offset := binary.Uvarint(raw)
	if offset <= 0 || size > 64<<20 || size != uint64(len(raw)-offset) {
		return
	}
	data := &msg.CSVCMsg_FlattenedSerializer{}
	if proto.Unmarshal(raw[offset:], data) != nil {
		return
	}
	symbol := func(i *int32) string {
		if i == nil || *i < 0 || int(*i) >= len(data.Symbols) {
			return ""
		}
		return data.Symbols[*i]
	}
	for _, s := range data.Serializers {
		name := symbol(s.SerializerNameSym)
		if name == "" {
			continue
		}
		fields := map[string]supportClockField{}
		for _, i := range s.FieldsIndex {
			if i < 0 || int(i) >= len(data.Fields) {
				continue
			}
			f := data.Fields[i]
			fields[symbol(f.VarNameSym)] = supportClockField{symbol(f.VarTypeSym), symbol(f.FieldSerializerNameSym)}
		}
		// Multiple versions must agree before a declaration is usable.
		if previous, exists := t.types[name]; exists {
			for field, value := range fields {
				if old, ok := previous[field]; !ok || old != value {
					fields[field] = supportClockField{}
				}
			}
		}
		t.types[name] = fields
	}
}

func (t *supportClockTracker) declared(class, path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		field, ok := t.types[class][part]
		if !ok {
			return ""
		}
		if i == len(parts)-1 {
			return field.typ
		}
		class = field.nested
	}
	return ""
}

func (t *supportClockTracker) fields(e st.Entity, class, prefix string, names []string) map[string]models.SupportClockScalar {
	r := map[string]models.SupportClockScalar{}
	for _, name := range names {
		r[name] = clockScalar(supportProperty(e, prefix+name), t.declared(class, prefix+name))
	}
	return r
}

func (h *ReplayHandler) clockMessage(kind string, raw any) {
	t := h.supportObjectiveClock
	t.order++
	frame := 1
	if h.support != nil {
		frame = h.support.frame + 1
	}
	t.network[kind] = &models.SupportClockMessage{Order: t.order, Frame: frame, Tick: h.ctx.Parser.GameState().IngameTick(), Value: clockScalar(raw, "")}
	h.attemptMessage(kind, t.network[kind])
}

func (h *ReplayHandler) registerSupportObjectiveClock() {
	if h.ctx.SupportObjectiveClockObservations == nil {
		return
	}
	t := &supportClockTracker{network: map[string]*models.SupportClockMessage{"tick": nil, "packet_tick": nil, "interval": nil, "paused": nil}, types: map[string]map[string]supportClockField{}, objects: map[st.Entity]*supportC4Life{}}
	h.supportObjectiveClock = t
	h.registerSupportRoundContext()
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CDemoSendTables) { t.sendTables(m.Data) })
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CNETMsg_Tick) {
		var raw any
		if m.Tick != nil {
			raw = *m.Tick
		}
		h.clockMessage("tick", raw)
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_PacketEntities) {
		var raw any
		if m.ServerTick != nil {
			raw = *m.ServerTick
		}
		h.clockMessage("packet_tick", raw)
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		var raw any
		if m.TickInterval != nil {
			raw = *m.TickInterval
		}
		h.clockMessage("interval", raw)
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_SetPause) {
		var raw any
		if m.Paused != nil {
			raw = *m.Paused
		}
		h.clockMessage("paused", raw)
	})
	h.ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range h.ctx.Parser.ServerClasses().All() {
			if class.Name() == "CPlantedC4" || class.Name() == "CCSGameRulesProxy" {
				class.OnEntityCreated(func(e st.Entity) {
					life := &supportC4Life{life: fmt.Sprintf("support-clock-object-%06d", len(t.objects)+1), class: class.Name(), alive: true}
					t.objects[e] = life
					e.OnDestroy(func() { life.alive = false })
					if life.class == "CPlantedC4" {
						h.bindSupportPosition(e, life)
					}
				})
			}
		}
	})
}

func (h *ReplayHandler) captureSupportObjectiveClock(base models.SupportFrame, players map[uint64]*common.Player) {
	data := h.ctx.SupportObjectiveClockObservations
	if data == nil {
		return
	}
	t := h.supportObjectiveClock
	t.order++
	r := models.SupportObjectiveClockFrame{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ObjectiveOrder: h.supportObjectiveOrder, C4StateSequence: len(h.ctx.SupportC4Observations.States), Order: t.order, Network: map[string]*models.SupportClockMessage{}, Players: []models.SupportClockPlayer{}, Bombs: []models.SupportClockObject{}, Items: []models.SupportClockObject{}}
	for k, v := range t.network {
		r.Network[k] = v
	} // Message records are immutable.
	current := h.ctx.Parser.GameState().Entities()
	for e, life := range t.objects {
		if !life.alive || current[e.ID()] != e {
			continue
		}
		object := models.SupportClockObject{Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.class}
		if life.class == "CCSGameRulesProxy" {
			rules := h.ctx.Parser.GameState().Rules()
			if rules != nil && rules.Entity() == e {
				object.Fields = t.fields(e, life.class, "m_pGameRules.", supportClockRules)
				r.Rules = &object
			}
		} else {
			object.Fields = t.fields(e, life.class, "", supportClockBomb)
			ref, pawn := c4Reference(supportProperty(e, "m_hBombDefuser"), current)
			if pawn != nil {
				if alias := h.support.lives[pawn]; alias != "" {
					ref.TargetLife = &alias
				} else {
					ref.Status = "kind_or_life_unavailable"
				}
			}
			object.Defuser = &ref
			r.Bombs = append(r.Bombs, object)
		}
	}
	for e, life := range h.supportC4State.objects {
		if life.alive && life.class == "CC4" && current[e.ID()] == e {
			r.Items = append(r.Items, models.SupportClockObject{Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.class, Fields: t.fields(e, life.class, "", supportClockItem)})
		}
	}
	for _, actor := range base.Players {
		p := players[actor.NativeActor]
		proof := objectivePlayer(p, actor)
		var pawn st.Entity
		if proof.PawnLife != nil {
			e := p.PlayerPawnEntity()
			if current[e.ID()] == e && h.support.lives[e] == *proof.PawnLife {
				pawn = e
			}
		}
		player := models.SupportClockPlayer{NativeActor: actor.NativeActor, Fields: t.fields(pawn, "CCSPlayerPawn", "", supportClockPawn)}
		if pawn != nil {
			player.PawnLife, player.PawnHandle = proof.PawnLife, actor.PawnHandle
		}
		r.Players = append(r.Players, player)
	}
	sort.Slice(r.Bombs, func(i, j int) bool { return r.Bombs[i].Life < r.Bombs[j].Life })
	sort.Slice(r.Items, func(i, j int) bool { return r.Items[i].Life < r.Items[j].Life })
	data.Frames = append(data.Frames, r)
	h.captureSupportRoundContext(base)
}
