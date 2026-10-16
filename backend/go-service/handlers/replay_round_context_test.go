package handlers

import (
	"encoding/json"
	"math"
	"testing"

	"cs2-demo-service/models"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	dp "github.com/markus-wa/godispatch"
)

func replayRoundContextFixture() replayRoundContextInput {
	rules := map[string]models.SupportClockScalar{
		"m_fRoundStartTime": clockScalar(float32(100), "GameTime_t"),
		"m_iRoundTime":      clockScalar(int32(115), "int32"),
		"m_bBombPlanted":    clockScalar(false, "bool"),
	}
	for _, name := range []string{
		"m_bFreezePeriod", "m_bWarmupPeriod", "m_bMatchWaitingForResume",
		"m_bTerroristTimeOutActive", "m_bCTTimeOutActive", "m_bGameRestart",
		"m_bTechnicalTimeOut", "m_bGamePaused",
	} {
		rules[name] = clockScalar(false, "bool")
	}
	now := 150.0
	return replayRoundContextInput{
		Round: 6, Tick: 9600, FreezeEndTick: 6400, Phase: "playing", TickRate: 64,
		Rules: rules, NativeNow: &now,
	}
}

func TestReplayRoundContextCarriesCausalRoundMetadataAndNativeClock(t *testing.T) {
	in := replayRoundContextFixture()
	got := buildReplayRoundContext(in)
	if got.SchemaID != "stratai.replay_round_context@2" || got.Round != 6 || got.Tick != 9600 || got.AvailabilityTick != 9600 ||
		got.Phase != "preplant" || got.FreezeEndTick != 6400 {
		t.Fatalf("round frame lost causal identity or phase: %+v", got)
	}
	if got.ConfiguredDurationSeconds == nil || *got.ConfiguredDurationSeconds != 115 {
		t.Fatalf("native configured duration lost: %+v", got)
	}
	if got.PauseState != "running" || got.ClockStatus != "observed" || got.RoundRemainingSeconds == nil || *got.RoundRemainingSeconds != 65 {
		t.Fatalf("expected 100 + 115 - 150 = 65 native seconds: %+v", got)
	}
	if got.BombRemainingSeconds != nil || got.Winner != "" || got.EndTick != nil {
		t.Errorf("preplant frame contains a bomb clock or future round result: %+v", got)
	}
	if got.ElapsedSeconds == nil || *got.ElapsedSeconds != 50 {
		t.Errorf("elapsed time must be a separate fact: %+v", got.ElapsedSeconds)
	}
	encoded, err := json.Marshal(models.ReplayFrame{Tick: in.Tick, RoundContext: got})
	if err != nil {
		t.Fatal(err)
	}
	var frame map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &frame); err != nil {
		t.Fatal(err)
	}
	if _, exists := frame["round_context"]; !exists {
		t.Fatal("native round context is missing from the replay frame wire contract")
	}
}

func TestReplayRoundContextDoesNotPublishTheWinnerBeforeRoundEnd(t *testing.T) {
	in := replayRoundContextFixture()
	// A completed round container may already know the result. It is not
	// evidence available to this earlier frame.
	in.EndTick = 12800
	in.Winner = "CT"
	before := buildReplayRoundContext(in)
	if before.Winner != "" || before.EndTick != nil || before.Phase != "preplant" {
		t.Fatalf("future final-round metadata leaked into the prefix: %+v", before)
	}

	in.Tick = 12800
	in.Phase = "postround"
	// The native planted flag can outlive RoundEnd. It must not keep the
	// competitive round active or start an apparent post-round countdown.
	in.Rules["m_bBombPlanted"] = clockScalar(true, "bool")
	after := buildReplayRoundContext(in)
	if after.Phase != "ended" || after.Winner != "CT" || after.EndTick == nil || *after.EndTick != 12800 {
		t.Fatalf("RoundEnd did not publish its causal result: %+v", after)
	}
	if after.RoundRemainingSeconds != nil || after.BombRemainingSeconds != nil {
		t.Errorf("ended round still advertises a live objective countdown: %+v", after)
	}
	if before.Winner != "" || before.EndTick != nil {
		t.Error("later round-end state mutated an already captured frame")
	}
}

func TestReplayRoundContextUnknownNativeClockDoesNotBecomeZeroOrDefault(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*replayRoundContextInput)
	}{
		{"no_native_now", func(in *replayRoundContextInput) { in.NativeNow = nil }},
		{"no_native_duration", func(in *replayRoundContextInput) { delete(in.Rules, "m_iRoundTime") }},
		{"no_native_origin", func(in *replayRoundContextInput) { delete(in.Rules, "m_fRoundStartTime") }},
		{"unaccredited_origin_type", func(in *replayRoundContextInput) {
			in.Rules["m_fRoundStartTime"] = clockScalar(float32(100), "float32")
		}},
		{"nonfinite_native_now", func(in *replayRoundContextInput) { now := math.NaN(); in.NativeNow = &now }},
		{"nonfinite_native_duration", func(in *replayRoundContextInput) {
			in.Rules["m_iRoundTime"] = clockScalar(math.Inf(1), "int32")
		}},
		{"string_duration", func(in *replayRoundContextInput) {
			in.Rules["m_iRoundTime"] = clockScalar("115", "int32")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := replayRoundContextFixture()
			tt.mutate(&in)
			got := buildReplayRoundContext(in)
			if got.RoundRemainingSeconds != nil || got.BombRemainingSeconds != nil || got.ClockStatus == "observed" || got.ClockReason == "" {
				t.Fatalf("unknown native clock became an observed countdown: %+v", got)
			}
			if _, err := json.Marshal(got); err != nil {
				t.Fatalf("unknown/nonfinite clock must still have a valid JSON representation: %v", err)
			}
		})
	}
}

func TestReplayRoundContextNativeZeroRemainsDistinctFromUnknown(t *testing.T) {
	in := replayRoundContextFixture()
	now := 215.0
	in.NativeNow = &now
	got := buildReplayRoundContext(in)
	if got.RoundRemainingSeconds == nil || *got.RoundRemainingSeconds != 0 || got.ClockStatus != "observed" {
		t.Fatalf("an accredited expired deadline must preserve observed zero: %+v", got)
	}
}

func TestReplayRoundContextRequiresPositivePauseEvidence(t *testing.T) {
	tests := []struct {
		name  string
		state string
		edit  func(*replayRoundContextInput)
	}{
		{"missing_timeout_flag", "unknown", func(in *replayRoundContextInput) { delete(in.Rules, "m_bTechnicalTimeOut") }},
		{"missing_general_pause", "unknown", func(in *replayRoundContextInput) { delete(in.Rules, "m_bGamePaused") }},
		{"timeout_active", "paused", func(in *replayRoundContextInput) { in.Rules["m_bCTTimeOutActive"] = clockScalar(true, "bool") }},
		{"engine_paused", "paused", func(in *replayRoundContextInput) { paused := true; in.EnginePaused = &paused }},
		{"integer_without_boolean_type", "unknown", func(in *replayRoundContextInput) { in.Rules["m_bTechnicalTimeOut"] = clockScalar(uint64(0), "uint64") }},
		{"invalid_boolean_integer", "unknown", func(in *replayRoundContextInput) { in.Rules["m_bTechnicalTimeOut"] = clockScalar(uint64(2), "bool") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := replayRoundContextFixture()
			tt.edit(&in)
			got := buildReplayRoundContext(in)
			if got.PauseState != tt.state || got.RoundRemainingSeconds != nil || got.BombRemainingSeconds != nil || got.ClockStatus == "observed" {
				t.Fatalf("unproved/running clock survived a pause uncertainty: %+v", got)
			}
		})
	}
	for _, nativeFalse := range []any{false, uint64(0), int32(0)} {
		in := replayRoundContextFixture()
		in.Rules["m_bTechnicalTimeOut"] = clockScalar(nativeFalse, "bool")
		delete(in.Rules, "m_bGamePaused")
		engineRunning := false
		in.EnginePaused = &engineRunning
		got := buildReplayRoundContext(in)
		if got.PauseState != "running" || got.RoundRemainingSeconds == nil || *got.RoundRemainingSeconds != 65 {
			t.Errorf("declared false plus observed engine running was lost (%T): %+v", nativeFalse, got)
		}
	}
}

func TestReplayRoundContextUsesCurrentPlantedDeadlineWithoutRoundTimer(t *testing.T) {
	in := replayRoundContextFixture()
	in.Rules["m_bBombPlanted"] = clockScalar(true, "bool")
	in.BombFields = []map[string]models.SupportClockScalar{{
		"m_bBombTicking":  clockScalar(true, "bool"),
		"m_bBombDefused":  clockScalar(false, "bool"),
		"m_flC4Blow":      clockScalar(float32(180), "GameTime_t"),
		"m_flTimerLength": clockScalar(float32(40), "float32"),
	}}
	got := buildReplayRoundContext(in)
	if got.Phase != "planted" || got.RoundRemainingSeconds != nil || got.BombRemainingSeconds == nil || *got.BombRemainingSeconds != 30 || got.ClockStatus != "observed" {
		t.Fatalf("planted clock must be 180 - 150 = 30 seconds, independent of round countdown: %+v", got)
	}

	delete(in.BombFields[0], "m_flC4Blow")
	unknown := buildReplayRoundContext(in)
	if unknown.BombRemainingSeconds != nil || unknown.RoundRemainingSeconds != nil || unknown.ClockStatus == "observed" {
		t.Fatalf("missing planted deadline fell back to round time or default 40 seconds: %+v", unknown)
	}
}

func TestReplayRoundContextElapsedNeverInventsTickRateOrRoundStart(t *testing.T) {
	for _, mutate := range []func(*replayRoundContextInput){
		func(in *replayRoundContextInput) { in.TickRate = 0 },
		func(in *replayRoundContextInput) { in.TickRate = math.NaN() },
		func(in *replayRoundContextInput) { in.FreezeEndTick = 0 },
		func(in *replayRoundContextInput) { in.FreezeEndTick = in.Tick + 1 },
	} {
		in := replayRoundContextFixture()
		mutate(&in)
		got := buildReplayRoundContext(in)
		if got.ElapsedSeconds != nil {
			t.Errorf("missing/invalid round-start clock became elapsed seconds: %+v", got)
		}
	}
}

func TestReplayRoundContextRejectsAmbiguousActiveBombsEvenWithMissingDeadline(t *testing.T) {
	for _, missing := range []int{-1, 0, 1} {
		in := replayRoundContextFixture()
		in.Rules["m_bBombPlanted"] = clockScalar(true, "bool")
		for i := 0; i < 2; i++ {
			fields := map[string]models.SupportClockScalar{
				"m_bBombTicking": clockScalar(true, "bool"),
				"m_bBombDefused": clockScalar(false, "bool"),
			}
			if i != missing {
				fields["m_flC4Blow"] = clockScalar(float32(180+i), "GameTime_t")
			}
			in.BombFields = append(in.BombFields, fields)
		}
		got := buildReplayRoundContext(in)
		if got.BombRemainingSeconds != nil || got.ClockStatus == "observed" {
			t.Errorf("two active bombs became one accredited countdown (missing deadline index %d): %+v", missing, got)
		}
	}
}

func replayRoundNativeClockFixture() (*replayRoundClock, st.Entity, replayRoundContextInput) {
	in := replayRoundContextFixture()
	in.Tick, in.NativeNow = in.FreezeEndTick, nil
	step := float32(1.0 / 64)
	clock := &replayRoundClock{frame: 10, freezeFrame: 10, interval: &step}
	clock.tick = &models.SupportClockMessage{Frame: 10, Value: clockScalar(uint32(6400), "")}
	clock.packet = &models.SupportClockMessage{Frame: 10, Value: clockScalar(uint32(6400), "")}
	return clock, &sourceTestEntity{}, in
}

func TestReplayRoundContextNativeClockRequiresCurrentMatchingNetworkMessagesAtOrigin(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*replayRoundClock, *st.Entity, *replayRoundContextInput)
	}{
		{"missing_NET_tick", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.tick = nil }},
		{"missing_packet_tick", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.packet = nil }},
		{"NET_tick_from_previous_frame", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.tick.Frame-- }},
		{"packet_tick_from_previous_frame", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.packet.Frame-- }},
		{"NET_and_packet_disagree", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			c.packet.Value = clockScalar(uint32(6401), "")
		}},
		{"native_interval_missing", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.interval = nil }},
		{"native_interval_invalid", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			step := float32(0)
			c.interval = &step
		}},
		{"native_interval_nonfinite", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			step := float32(math.NaN())
			c.interval = &step
		}},
		{"native_origin_does_not_match_server", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) {
			in.Rules["m_fRoundStartTime"] = clockScalar(float32(101), "GameTime_t")
		}},
		{"rules_entity_unavailable", func(_ *replayRoundClock, entity *st.Entity, _ *replayRoundContextInput) { *entity = nil }},
		{"freeze_has_not_ended", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) { in.Phase = "freezetime" }},
		{"freeze_origin_already_passed", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) { in.Tick++ }},
		{"same_demo_tick_but_frame_after_freeze", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			c.frame++
			c.tick.Frame, c.packet.Frame = c.frame, c.frame
		}},
		{"pause_at_origin", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) {
			in.Rules["m_bCTTimeOutActive"] = clockScalar(true, "bool")
		}},
		{"bomb_already_planted", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) {
			in.Rules["m_bBombPlanted"] = clockScalar(true, "bool")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock, entity, in := replayRoundNativeClockFixture()
			tt.mutate(clock, &entity, &in)
			if got := clock.nativeNow(entity, in); got != nil || clock.aligned {
				t.Fatalf("native clock was accredited without a current freeze-origin witness: now=%v aligned=%v", got, clock.aligned)
			}
		})
	}
}

func TestReplayRoundContextNativeClockAdvancesOnlyFromAnObservedOrigin(t *testing.T) {
	clock, entity, in := replayRoundNativeClockFixture()
	if now := clock.nativeNow(entity, in); now == nil || *now != 100 {
		t.Fatalf("exact native freeze-end origin was not accredited: %v", now)
	}
	clock.frame++
	in.Tick = 8000
	clock.tick = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8000), "")}
	clock.packet = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8000), "")}
	in.NativeNow = clock.nativeNow(entity, in)
	got := buildReplayRoundContext(in)
	if in.NativeNow == nil || *in.NativeNow != 125 || got.RoundRemainingSeconds == nil || *got.RoundRemainingSeconds != 90 {
		t.Fatalf("current native server time did not preserve the witnessed domain (100 + 115 - 125): %+v", got)
	}
	// Restarting the reader at this same later cut cannot use a future or
	// retrospectively fitted origin to authorize its countdown.
	clock.aligned = false
	if now := clock.nativeNow(entity, in); now != nil {
		t.Errorf("reader without an earlier native witness fitted a clock at a later cut: %v", *now)
	}
}

func TestReplayRoundContextNativeClockDropsProofOnFreshnessOrIdentityBreak(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*replayRoundClock, *st.Entity, *replayRoundContextInput)
	}{
		{"stale_NET_tick", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.tick.Frame-- }},
		{"stale_packet_tick", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) { c.packet.Frame-- }},
		{"disagreeing_ticks", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			c.packet.Value = clockScalar(uint32(8001), "")
		}},
		{"rules_entity_replaced", func(_ *replayRoundClock, entity *st.Entity, _ *replayRoundContextInput) {
			*entity = &sourceTestEntity{}
		}},
		{"rules_entity_absent", func(_ *replayRoundClock, entity *st.Entity, _ *replayRoundContextInput) { *entity = nil }},
		{"round_origin_changed", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) {
			in.Rules["m_fRoundStartTime"] = clockScalar(float32(101), "GameTime_t")
		}},
		{"round_origin_absent", func(_ *replayRoundClock, _ *st.Entity, in *replayRoundContextInput) {
			delete(in.Rules, "m_fRoundStartTime")
		}},
		{"interval_changed", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			step := float32(1.0 / 128)
			c.interval = &step
		}},
		{"server_rewind", func(c *replayRoundClock, _ *st.Entity, _ *replayRoundContextInput) {
			c.tick.Value, c.packet.Value = clockScalar(uint32(6399), ""), clockScalar(uint32(6399), "")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock, entity, in := replayRoundNativeClockFixture()
			if now := clock.nativeNow(entity, in); now == nil {
				t.Fatal("fixture did not accredit its initial native origin")
			}
			originalEntity := entity
			clock.frame++
			in.Tick = 8000
			clock.tick = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8000), "")}
			clock.packet = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8000), "")}
			tt.mutate(clock, &entity, &in)
			if now := clock.nativeNow(entity, in); now != nil || clock.aligned {
				t.Fatalf("broken clock witness survived: now=%v aligned=%v", now, clock.aligned)
			}
			// A subsequent internally consistent packet must not revive proof
			// lost at an earlier cut. Only a newly witnessed round origin can.
			clock.frame++
			step := float32(1.0 / 64)
			clock.interval = &step
			in.Rules["m_fRoundStartTime"] = clockScalar(float32(100), "GameTime_t")
			in.Tick = 8064
			clock.tick = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8064), "")}
			clock.packet = &models.SupportClockMessage{Frame: clock.frame, Value: clockScalar(uint32(8064), "")}
			if now := clock.nativeNow(originalEntity, in); now != nil {
				t.Errorf("later packet revived discarded clock proof: %v", *now)
			}
		})
	}
}

type replayRoundRegistrationParser struct {
	dem.Parser
	netHandlers   []any
	eventHandlers []any
	state         dem.GameState
}

func (p *replayRoundRegistrationParser) RegisterNetMessageHandler(handler any) dp.HandlerIdentifier {
	p.netHandlers = append(p.netHandlers, handler)
	return nil
}

func (p *replayRoundRegistrationParser) RegisterEventHandler(handler any) dp.HandlerIdentifier {
	p.eventHandlers = append(p.eventHandlers, handler)
	return nil
}

func (p *replayRoundRegistrationParser) GameState() dem.GameState { return p.state }

func TestReplayRoundContextServerInfoCannotReuseMessagesFromPreviousSession(t *testing.T) {
	parser := &replayRoundRegistrationParser{}
	handler := NewReplayHandler(&models.DemoContext{Parser: parser, CaptureCalculatedVision: true})
	handler.registerReplayRoundClock()
	var onServerInfo func(*msg.CSVCMsg_ServerInfo)
	var onTick func(*msg.CNETMsg_Tick)
	var onPacket func(*msg.CSVCMsg_PacketEntities)
	for _, registered := range parser.netHandlers {
		switch callback := registered.(type) {
		case func(*msg.CSVCMsg_ServerInfo):
			onServerInfo = callback
		case func(*msg.CNETMsg_Tick):
			onTick = callback
		case func(*msg.CSVCMsg_PacketEntities):
			onPacket = callback
		}
	}
	if onServerInfo == nil || onTick == nil || onPacket == nil {
		t.Fatal("native clock message handlers were not registered")
	}
	_, entity, in := replayRoundNativeClockFixture()
	step, tick := float32(1.0/64), uint32(6400)
	onServerInfo(&msg.CSVCMsg_ServerInfo{TickInterval: &step})
	handler.replayRoundClock.freezeFrame = handler.replayRoundClock.frame
	onTick(&msg.CNETMsg_Tick{Tick: &tick})
	onPacket(&msg.CSVCMsg_PacketEntities{ServerTick: &tick})
	if now := handler.replayRoundClock.nativeNow(entity, in); now == nil {
		t.Fatal("initial session did not accredit its origin")
	}
	// ServerInfo can be delivered in the same frame. Equal ticks from the old
	// session cannot accredit the new one just because their frame IDs match.
	onServerInfo(&msg.CSVCMsg_ServerInfo{TickInterval: &step})
	if now := handler.replayRoundClock.nativeNow(entity, in); now != nil {
		t.Errorf("ServerInfo reused previous-session tick messages: %v", *now)
	}
	onTick(&msg.CNETMsg_Tick{Tick: &tick})
	onPacket(&msg.CSVCMsg_PacketEntities{ServerTick: &tick})
	if now := handler.replayRoundClock.nativeNow(entity, in); now != nil {
		t.Errorf("fresh packets reused a freeze witness from the previous session: %v", *now)
	}
	handler.replayRoundClock.freezeFrame = handler.replayRoundClock.frame
	if now := handler.replayRoundClock.nativeNow(entity, in); now == nil || *now != 100 {
		t.Errorf("fresh new-session messages could not establish their own origin: %v", now)
	}
}

func TestReplayRoundContextAdvancesFrameFreshnessWhenNoReplayRoundIsSampled(t *testing.T) {
	parser := &replayRoundRegistrationParser{state: &objectiveTestState{tick: 6400}}
	handler := RegisterReplayHandlers(&models.DemoContext{Parser: parser, CaptureCalculatedVision: true})
	var frameDone func(events.FrameDone)
	for _, registered := range parser.eventHandlers {
		if callback, ok := registered.(func(events.FrameDone)); ok {
			if frameDone != nil {
				t.Fatal("test requires the single replay FrameDone capture callback")
			}
			frameDone = callback
		}
	}
	if frameDone == nil {
		t.Fatal("replay FrameDone callback was not registered")
	}
	before := handler.replayRoundClock.frame
	frameDone(events.FrameDone{})
	if handler.replayRoundClock.frame != before+1 {
		t.Fatal("an unsampled frame retained the previous native-message freshness ID")
	}
}
