package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"math"
)

type replayRoundContextInput struct {
	Round, Tick, FreezeEndTick, EndTick int
	Phase, Winner                       string
	TickRate                            float64
	Rules                               map[string]models.SupportClockScalar
	EnginePaused                        *bool
	NativeNow                           *float64
	BombFields                          []map[string]models.SupportClockScalar
}

func replayClockNumber(s models.SupportClockScalar, gameTime bool) *float64 {
	if s.Status != "observed" || (gameTime && (s.DeclaredType == nil || *s.DeclaredType != "GameTime_t")) {
		return nil
	}
	var n float64
	switch v := s.Value.(type) {
	case float64:
		n = v
	case float32:
		n = float64(v)
	case int:
		n = float64(v)
	case int32:
		n = float64(v)
	case uint32:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint64:
		n = float64(v)
	default:
		return nil
	}
	if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1e8 {
		return nil
	}
	return &n
}

func replayClockBool(s models.SupportClockScalar) *bool {
	if s.Status != "observed" {
		return nil
	}
	if b, ok := s.Value.(bool); ok {
		return &b
	}
	// Some Source 2 serializers expose bool as integer; require its declaration.
	if s.DeclaredType == nil || *s.DeclaredType != "bool" {
		return nil
	}
	if n := replayClockNumber(s, false); n != nil && (*n == 0 || *n == 1) {
		b := *n == 1
		return &b
	}
	return nil
}

func replayPauseState(rules map[string]models.SupportClockScalar, engine *bool) string {
	missing := false
	for _, name := range []string{"m_bFreezePeriod", "m_bWarmupPeriod", "m_bMatchWaitingForResume", "m_bTerroristTimeOutActive", "m_bCTTimeOutActive", "m_bGameRestart", "m_bTechnicalTimeOut"} {
		v := replayClockBool(rules[name])
		if v == nil {
			missing = true
		} else if *v {
			return "paused"
		}
	}
	general := replayClockBool(rules["m_bGamePaused"])
	if (general != nil && *general) || (engine != nil && *engine) {
		return "paused"
	}
	if missing || (general == nil && engine == nil) {
		return "unknown"
	}
	return "running"
}

func buildReplayRoundContext(in replayRoundContextInput) *models.ReplayRoundContext {
	r := &models.ReplayRoundContext{SchemaID: "stratai.replay_round_context@2", Round: in.Round,
		Tick: in.Tick, AvailabilityTick: in.Tick, Phase: "unknown", FreezeEndTick: in.FreezeEndTick,
		ConfiguredDurationSeconds: replayClockNumber(in.Rules["m_iRoundTime"], false),
		PauseState:                replayPauseState(in.Rules, in.EnginePaused), ClockStatus: "unavailable", ClockReason: "native_clock_alignment_unaccredited"}
	if in.TickRate > 0 && !math.IsInf(in.TickRate, 0) && in.FreezeEndTick > 0 && in.Tick >= in.FreezeEndTick {
		v := float64(in.Tick-in.FreezeEndTick) / in.TickRate
		r.ElapsedSeconds = &v
	}
	if in.Phase == "postround" {
		r.Phase, r.ClockStatus, r.ClockReason = "ended", "not_applicable", "round_ended"
		if in.EndTick > 0 && in.EndTick <= in.Tick {
			end := in.EndTick
			r.EndTick = &end
			if in.Winner == "T" || in.Winner == "CT" {
				r.Winner = in.Winner
			}
		}
		return r
	}
	if in.Phase == "freezetime" {
		r.Phase = "freezetime"
	}
	if in.Phase == "playing" {
		if planted := replayClockBool(in.Rules["m_bBombPlanted"]); planted != nil {
			if *planted {
				r.Phase = "planted"
			} else {
				r.Phase = "preplant"
			}
		}
	}
	if r.PauseState != "running" {
		r.ClockReason = "native_pause_controls_incomplete_or_paused"
		return r
	}
	if in.NativeNow == nil || math.IsNaN(*in.NativeNow) || math.IsInf(*in.NativeNow, 0) || *in.NativeNow < 0 {
		return r
	}
	var deadline *float64
	if r.Phase == "preplant" && r.ConfiguredDurationSeconds != nil {
		if start := replayClockNumber(in.Rules["m_fRoundStartTime"], true); start != nil && *start <= *in.NativeNow {
			v := *start + *r.ConfiguredDurationSeconds
			deadline = &v
		}
	} else if r.Phase == "planted" {
		active := 0
		for _, fields := range in.BombFields {
			ticking, defused := replayClockBool(fields["m_bBombTicking"]), replayClockBool(fields["m_bBombDefused"])
			if ticking != nil && *ticking && defused != nil && !*defused {
				active++
				if active > 1 {
					r.ClockReason = "multiple_active_bomb_clocks"
					return r
				}
				deadline = replayClockNumber(fields["m_flC4Blow"], true)
			}
		}
	}
	if deadline == nil {
		r.ClockReason = "native_deadline_unavailable"
		return r
	}
	v := math.Max(0, *deadline-*in.NativeNow)
	r.ClockStatus, r.ClockReason = "observed", ""
	if r.Phase == "preplant" {
		r.RoundRemainingSeconds = &v
	} else {
		r.BombRemainingSeconds = &v
	}
	return r
}

// Reuses the native scalar/declaration reader without enabling the much larger
// support export. The clock origin is witnessed at freeze end, never fitted.
type replayRoundClock struct {
	reader         supportClockTracker
	frame          int
	freezeFrame    int
	tick, packet   *models.SupportClockMessage
	interval       *float32
	paused         *bool
	rules          st.Entity
	origin, step   float64
	aligned        bool
	lastServerTick float64
	originWitness  map[string]any
}

func (h *ReplayHandler) registerReplayRoundClock() {
	if !h.ctx.CaptureCalculatedVision {
		return
	}
	t := &replayRoundClock{freezeFrame: -1}
	h.replayRoundClock = t
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CDemoSendTables) { t.reader.sendTables(m.Data) })
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CNETMsg_Tick) {
		var v any
		if m.Tick != nil {
			v = *m.Tick
		}
		t.tick = &models.SupportClockMessage{Frame: t.frame, Value: clockScalar(v, "")}
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_PacketEntities) {
		var v any
		if m.ServerTick != nil {
			v = *m.ServerTick
		}
		t.packet = &models.SupportClockMessage{Frame: t.frame, Value: clockScalar(v, "")}
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		t.interval = m.TickInterval
		t.aligned = false
		t.paused = nil
		t.tick, t.packet, t.rules = nil, nil, nil
		t.origin, t.step, t.lastServerTick = 0, 0, 0
		t.freezeFrame = -1
		t.originWitness = nil
	})
	h.ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_SetPause) { t.paused = m.Paused })
}

func (t *replayRoundClock) nativeNow(entity st.Entity, in replayRoundContextInput) *float64 {
	rules := in.Rules
	start := replayClockNumber(rules["m_fRoundStartTime"], true)
	var now *float64
	if t.tick != nil && t.packet != nil && t.tick.Frame == t.frame && t.packet.Frame == t.frame && t.interval != nil {
		a, b := replayClockNumber(t.tick.Value, false), replayClockNumber(t.packet.Value, false)
		step := float64(*t.interval)
		if a != nil && b != nil && *a == *b && step >= 1.0/256 && step <= 1.0/16 {
			v := *a * step
			now = &v
			if t.rules != entity || start == nil || *start != t.origin || step != t.step || *a < t.lastServerTick {
				t.aligned = false
			}
			if entity != nil && start != nil && *start == v && in.Tick == in.FreezeEndTick && t.frame == t.freezeFrame && in.Phase == "playing" && replayPauseState(rules, in.EnginePaused) == "running" {
				if planted := replayClockBool(rules["m_bBombPlanted"]); planted != nil && !*planted {
					t.aligned, t.rules, t.origin, t.step = true, entity, *start, step
					t.originWitness = map[string]any{"frame": t.frame, "tick": in.Tick, "freeze_end_tick": in.FreezeEndTick,
						"server_tick": *a, "packet_tick": *b, "interval_seconds": step, "round_start_seconds": *start,
						"pause_state": "running", "phase": "preplant"}
				}
			}
			t.lastServerTick = *a
		}
	}
	if now == nil {
		t.aligned = false
	}
	if t.aligned {
		return now
	}
	return nil
}

func (h *ReplayHandler) captureReplayRoundContext() *models.ReplayRoundContext {
	t := h.replayRoundClock
	if t == nil || h.currentRound == nil {
		return nil
	}
	gs := h.ctx.Parser.GameState()
	var entity st.Entity
	if rules := gs.Rules(); rules != nil {
		entity = rules.Entity()
	}
	names := append(append([]string{}, supportClockRules...), "m_bBombPlanted")
	rules := t.reader.fields(entity, "CCSGameRulesProxy", "m_pGameRules.", names)
	in := replayRoundContextInput{Round: h.currentRound.Round, Tick: gs.IngameTick(), FreezeEndTick: h.freezeTimeEndTick,
		EndTick: h.roundEndTick, Phase: h.roundPhase, Winner: h.currentRound.Winner, TickRate: h.ctx.Parser.TickRate(), Rules: rules, EnginePaused: t.paused}
	in.NativeNow = t.nativeNow(entity, in)
	if planted := replayClockBool(rules["m_bBombPlanted"]); planted != nil && *planted {
		for _, e := range gs.Entities() {
			if e.ServerClass().Name() == "CPlantedC4" {
				in.BombFields = append(in.BombFields, t.reader.fields(e, "CPlantedC4", "", supportClockBomb))
			}
		}
	}
	r := buildReplayRoundContext(in)
	var server *float64
	if t.tick != nil && t.tick.Frame == t.frame {
		server = replayClockNumber(t.tick.Value, false)
	}
	r.ClockEvidence = map[string]any{"rules": rules, "server_tick": server, "interval_seconds": t.interval,
		"current_frame": t.frame, "freeze_frame": t.freezeFrame, "tick_message": t.tick, "packet_message": t.packet,
		"origin_witness":     t.originWitness,
		"native_now_seconds": in.NativeNow, "origin_accredited": t.aligned, "bombs": in.BombFields, "engine_paused": t.paused,
		"source": "demoinfocs_native_rules_and_current_network_messages"}
	return r
}
