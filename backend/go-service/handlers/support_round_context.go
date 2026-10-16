package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

var supportRoundConvars = map[string]bool{
	"mp_give_player_c4":    true,
	"mp_t_default_primary": true, "mp_ct_default_primary": true,
	"mp_t_default_secondary": true, "mp_ct_default_secondary": true,
	"mp_t_default_grenades": true, "mp_ct_default_grenades": true,
	"mp_t_default_melee": true, "mp_ct_default_melee": true,
}

type supportRoundTracker struct {
	order   int
	session int
	data    *models.SupportRoundContextObservations
}

func (t *supportRoundTracker) convars(m *msg.CNETMsg_SetConVar, frame, tick int) {
	entries := []models.SupportRoundConvarEntry{}
	for _, entry := range m.GetConvars().GetCvars() {
		if entry == nil || entry.Name == nil || !supportRoundConvars[*entry.Name] {
			continue
		}
		var value *string
		if entry.Value != nil {
			v := *entry.Value
			value = &v
		}
		entries = append(entries, models.SupportRoundConvarEntry{Name: *entry.Name, Value: value})
	}
	if len(entries) == 0 {
		return
	}
	t.order++
	t.data.Messages = append(t.data.Messages, models.SupportRoundConvarMessage{
		Sequence: len(t.data.Messages) + 1, Order: t.order, Frame: frame, Tick: tick, Session: t.session, Entries: entries,
	})
}

func (h *ReplayHandler) registerSupportRoundContext() {
	data := &models.SupportRoundContextObservations{Messages: []models.SupportRoundConvarMessage{}, Frames: []models.SupportRoundContextFrame{}}
	h.ctx.SupportRoundContextObservations = data
	t := &supportRoundTracker{data: data}
	h.supportRoundContext = t
	h.ctx.Parser.RegisterNetMessageHandler(func(*msg.CSVCMsg_ServerInfo) { t.session++ })
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CNETMsg_SetConVar) {
		t.convars(m, h.support.frame+1, h.ctx.Parser.GameState().IngameTick())
	})
}

func (h *ReplayHandler) captureSupportRoundContext(base models.SupportFrame) {
	t := h.supportRoundContext
	if t == nil {
		return
	}
	t.order++
	rules := h.ctx.Parser.GameState().Rules()
	var fields map[string]models.SupportClockScalar
	names := []string{"m_bTCantBuy", "m_bCTCantBuy", "m_bWarmupPeriod"}
	if rules != nil {
		fields = h.supportObjectiveClock.fields(rules.Entity(), "CCSGameRulesProxy", "m_pGameRules.", names)
	} else {
		fields = h.supportObjectiveClock.fields(nil, "CCSGameRulesProxy", "m_pGameRules.", names)
	}
	t.data.Frames = append(t.data.Frames, models.SupportRoundContextFrame{
		SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch,
		Order: t.order, Session: t.session, MessageSequence: len(t.data.Messages), Rules: fields,
	})
}
