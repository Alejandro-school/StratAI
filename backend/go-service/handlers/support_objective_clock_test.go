package handlers

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"google.golang.org/protobuf/proto"
)

func TestSupportClockScalarPreservesZeroPresenceTypeAndNonfinite(t *testing.T) {
	for _, tc := range []struct {
		raw    any
		status string
	}{{nil, "absent"}, {float32(0), "observed"}, {float64(0), "observed"}, {uint32(0), "observed"}, {false, "observed"}, {"12", "unsupported_type"}, {math.Inf(1), "nonfinite"}, {float32(math.NaN()), "nonfinite"}} {
		r := clockScalar(tc.raw, "GameTime_t")
		if r.Status != tc.status || r.DeclaredType == nil || *r.DeclaredType != "GameTime_t" || (r.Value != nil) != (tc.status == "observed") {
			t.Fatal(r)
		}
		if _, err := json.Marshal(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSupportClockDeclarationsResolveExactNestedPathAndRejectMalformed(t *testing.T) {
	tt := &supportClockTracker{}
	data := &msg.CSVCMsg_FlattenedSerializer{Symbols: []string{"CCSGameRulesProxy", "m_pGameRules", "CCSGameRules", "m_fRoundStartTime", "GameTime_t", "CCSGameRules*"},
		Fields:      []*msg.ProtoFlattenedSerializerFieldT{{VarNameSym: proto.Int32(1), VarTypeSym: proto.Int32(5), FieldSerializerNameSym: proto.Int32(2)}, {VarNameSym: proto.Int32(3), VarTypeSym: proto.Int32(4)}},
		Serializers: []*msg.ProtoFlattenedSerializerT{{SerializerNameSym: proto.Int32(0), FieldsIndex: []int32{0}}, {SerializerNameSym: proto.Int32(2), FieldsIndex: []int32{1}}},
	}
	raw, err := proto.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	payload := append(binary.AppendUvarint(nil, uint64(len(raw))), raw...)
	tt.sendTables(payload)
	if tt.declared("CCSGameRulesProxy", "m_pGameRules.m_fRoundStartTime") != "GameTime_t" || tt.declared("CCSGameRulesProxy", "m_fRoundStartTime") != "" {
		t.Fatal("declaration path not resolved")
	}
	for _, bad := range [][]byte{nil, {255}, {1, 0, 0}, {1, 255}} {
		tt.sendTables(bad)
		if tt.declared("CCSGameRulesProxy", "m_pGameRules.m_fRoundStartTime") != "" {
			t.Fatal("stale declaration survived malformed payload")
		}
	}
}

func TestSupportClockFreshCutsLifecycleAndSnapshotIsolation(t *testing.T) {
	h, p, pawn, item, base, current := c4Fixture()
	h.ctx.SupportObjectiveClockObservations = &models.SupportObjectiveClockObservations{}
	h.supportObjectiveClock = &supportClockTracker{network: map[string]*models.SupportClockMessage{"tick": nil, "packet_tick": nil, "interval": nil, "paused": nil}, types: map[string]map[string]supportClockField{}, objects: map[st.Entity]*supportC4Life{}}
	pawn.values["m_flSimulationTime"] = float32(0)
	item.values["m_bStartedArming"] = true
	item.values["m_fArmedTime"] = float32(2)
	capture := func() {
		h.captureSupportC4(base, map[uint64]*common.Player{p.SteamID64: p})
		h.captureSupportObjectiveClock(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	first := h.ctx.SupportObjectiveClockObservations.Frames[0]
	if first.Players[0].Fields["m_flSimulationTime"].Value != float64(0) || first.Items[0].Fields["m_bStartedArming"].Value != true {
		t.Fatal(first)
	}
	raw, _ := json.Marshal(first)
	item.values["m_bStartedArming"] = false
	pawn.values["m_flSimulationTime"] = float32(1)
	capture() // Same demo tick, later frame observes cancellation and new time.
	second := h.ctx.SupportObjectiveClockObservations.Frames[1]
	if second.Items[0].Fields["m_bStartedArming"].Value != false || second.Order <= first.Order {
		t.Fatal(second)
	}
	item.destroy()
	capture()
	if len(h.ctx.SupportObjectiveClockObservations.Frames[2].Items) != 0 {
		t.Fatal("destroyed item reused")
	}
	replacement := &cashflowEntity{ownershipEntity: item.ownershipEntity}
	current[10] = replacement
	h.supportC4State.created(replacement, "CC4")
	capture()
	if h.ctx.SupportObjectiveClockObservations.Frames[3].Items[0].Life == first.Items[0].Life {
		t.Fatal("reused full handle inherited life")
	}
	delete(h.support.lives, pawn)
	capture()
	if h.ctx.SupportObjectiveClockObservations.Frames[4].Players[0].Fields["m_flSimulationTime"].Status != "absent" {
		t.Fatal("unavailable life retained clock")
	}
	after, _ := json.Marshal(h.ctx.SupportObjectiveClockObservations.Frames[0])
	if string(after) != string(raw) {
		t.Fatal("past clock changed")
	}
}

func TestSupportClockDisabledHasNoSideEffects(t *testing.T) {
	h := NewReplayHandler(&models.DemoContext{})
	h.registerSupportObjectiveClock()
	h.captureSupportObjectiveClock(models.SupportFrame{}, nil)
	if h.supportObjectiveClock != nil {
		t.Fatal("non-opt-in clock capture")
	}
}
