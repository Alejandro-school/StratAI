package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"strings"
	"testing"
)

type objectiveTestRules struct {
	dem.GameRules
	entity st.Entity
}

func (r objectiveTestRules) Entity() st.Entity { return r.entity }

type objectiveTestState struct {
	dem.GameState
	tick  int
	rules dem.GameRules
}

func (s *objectiveTestState) IngameTick() int      { return s.tick }
func (s *objectiveTestState) Rules() dem.GameRules { return s.rules }

type objectiveTestParser struct {
	dem.Parser
	state *objectiveTestState
}

func (p objectiveTestParser) GameState() dem.GameState { return p.state }

func TestSupportObjectiveCapturesFreshNativeZeroFalseAndOrder(t *testing.T) {
	gs := &objectiveTestState{tick: 100, rules: objectiveTestRules{entity: sourceTestEntity{values: map[string]any{"m_pGameRules.m_bBombPlanted": false, "m_pGameRules.m_iRoundTime": int32(0), "cs_gamerules_data.m_bBombPlanted": true, "cs_gamerules_data.m_iRoundTime": int32(999)}}}}
	ctx := &models.DemoContext{Parser: objectiveTestParser{state: gs}, SupportObservations: &models.SupportObservations{}, SupportObjectiveObservations: &models.SupportObjectiveObservations{}}
	h := NewReplayHandler(ctx)
	h.currentRound = &models.ReplayRoundData{Round: 1}
	h.supportRoundStart()
	h.objectiveEvent("round_start")
	h.supportClock(100)
	base := models.SupportFrame{Sequence: 1, Frame: 1, Tick: 100, Instance: 1, Epoch: 0}
	h.captureSupportObjective(base, map[uint64]*common.Player{})
	h.objectiveEvent("bomb_planted") // Same tick, after first cut.
	h.supportClock(100)
	base.Sequence, base.Frame = 2, 2
	h.captureSupportObjective(base, nil)
	data := ctx.SupportObjectiveObservations
	if data.Frames[0].EventSequence != 1 || data.Frames[1].EventSequence != 2 || data.Events[1].Order <= data.Frames[0].Order {
		t.Fatal("same tick delivery order lost")
	}
	rules := data.Frames[0].Rules
	if rules["m_bBombPlanted"].Value != false || rules["m_iRoundTime"].Value != int32(0) || rules["m_bWarmupPeriod"].Status != "absent" {
		t.Fatal("native presence lost")
	}
	gs.tick = 90
	h.objectiveEvent("round_end")
	h.supportClock(90)
	if data.Events[2].Epoch != 1 || h.support.epoch != 1 {
		t.Fatal("rewind event epoch mismatch")
	}
	h.supportRoundStart()
	h.objectiveEvent("round_start")
	if data.Events[3].Instance != 2 {
		t.Fatal("round instance reused")
	}
	raw, _ := json.Marshal(data.Events)
	if strings.Contains(string(raw), "actor") || strings.Contains(string(raw), "position") || strings.Contains(string(raw), "site") {
		t.Fatal("public trace leaked server-only information")
	}
}

func TestSupportObjectivePawnFullIdentityAndPresentTypes(t *testing.T) {
	for _, mode := range []string{"valid", "series", "life_missing", "disconnected", "dead", "position_missing", "native_missing", "native_wrong_type"} {
		t.Run(mode, func(t *testing.T) {
			_, p, pawn, _, base, _ := physicalFixture()
			life := "support-pawn-000001"
			handle := uint64(3<<14 | 5)
			base.PawnLife, base.PawnHandle = &life, &handle
			base.PositionStatus, base.AliveStatus = "observed", "observed"
			pawn.values["m_bIsDefusing"] = false
			switch mode {
			case "series":
				handle = uint64(1027<<14 | 5)
			case "life_missing":
				base.PawnLife = nil
			case "disconnected":
				base.Connected = false
			case "dead":
				*base.Alive = false
			case "position_missing":
				base.Position = nil
			case "native_missing":
				delete(pawn.values, "m_bIsDefusing")
			case "native_wrong_type":
				pawn.values["m_bIsDefusing"] = float32(0)
			}
			r := objectivePlayer(p, base)
			s := r.Fields["m_bIsDefusing"]
			if mode == "valid" {
				if s.Status != "observed" || s.Value != false || r.PawnLife == nil {
					t.Fatal("false lost")
				}
			} else if mode == "native_wrong_type" {
				if s.Status != "unsupported_type" || *s.NativeType != "float32" || s.Value != nil {
					t.Fatal("wrong native type normalized")
				}
			} else if s.Status != "absent" || s.Value != nil {
				t.Fatal("invalid identity inherited state")
			}
		})
	}
}

func TestSupportObjectiveIsOptInAndDoesNotReadParserWhenDisabled(t *testing.T) {
	h := NewReplayHandler(&models.DemoContext{})
	h.objectiveEvent("bomb_planted")
	h.registerSupportObjective()
	h.captureSupportObjective(models.SupportFrame{}, nil)
	if h.supportObjectiveOrder != 0 {
		t.Fatal("disabled objective has side effects")
	}
}
