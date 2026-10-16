package handlers

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"google.golang.org/protobuf/proto"
)

type traceParser struct {
	dem.Parser
	state dem.GameState
}

func (p traceParser) GameState() dem.GameState { return p.state }

type traceState struct{ dem.GameState }

func (traceState) IngameTick() int      { return 12 }
func (traceState) IsWarmupPeriod() bool { return false }

func TestTraceFieldsPreservePresenceTypePrivacyAndOrdinal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		typ    int32
		key    *msg.CMsgSource1LegacyGameEventKeyT
		status string
		value  any
	}{
		{"silent", 6, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(6), ValBool: proto.Bool(false)}, "observed", false},
		{"defindex", 4, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(4), ValShort: proto.Int32(0)}, "observed", int32(0)},
		{"item", 1, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(1), ValString: proto.String("")}, "observed", ""},
		{"item", 1, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(1)}, "absent", nil},
		{"userid", 4, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(4), ValShort: proto.Int32(123)}, "identity_redacted", nil},
		{"name", 1, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(1), ValString: proto.String("private")}, "outside_probe", nil},
		{"defindex", 4, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(3), ValLong: proto.Int32(7)}, "type_mismatch", nil},
		{"defindex", 4, nil, "absent", nil},
	} {
		d := &msg.CMsgSource1LegacyGameEventListKeyT{Name: &tc.name, Type: &tc.typ}
		got := traceField(3, d, tc.key)
		if got.Ordinal != 3 || got.Status != tc.status || !reflect.DeepEqual(got.Value, tc.value) {
			t.Fatalf("%s: %+v", tc.name, got)
		}
	}
}

func TestTraceNeverBackfillsAliasesOrCollapsesSameTickCallbacks(t *testing.T) {
	ctx := &models.DemoContext{Parser: traceParser{state: traceState{}}, EconomyObservations: &models.EconomyObservationLedger{ActorAliases: map[uint64]string{}, Rounds: []models.EconomyObservedRound{{Instance: 2, RoundNumber: 1, StartTick: 10}}}}
	tracker := &economyTraceTracker{ctx: ctx, ledger: &models.EconomyTransactionTrace{}}
	p := &common.Player{SteamID64: 123, Team: common.TeamCounterTerrorists}
	tracker.append("sdk_callback", "ItemPickup", p, nil, nil)
	ctx.EconomyObservations.ActorAliases[123] = "p0001"
	tracker.append("sdk_callback", "ItemPickup", p, nil, nil)
	tracker.append("sdk_callback", "ItemPickup", p, nil, nil)
	rows := tracker.ledger.Rows
	if len(rows) != 3 || rows[0].ActorAlias != nil || *rows[1].ActorAlias != "p0001" || rows[2].Sequence != 3 || *rows[2].Instance != 2 || rows[0].Tick != rows[2].Tick {
		t.Fatal(rows)
	}
}

func TestTraceEntityLifetimeNotReusedEntityIndex(t *testing.T) {
	tracker := &economyTraceTracker{entities: map[st.Entity]string{}}
	a, b := &sourceTestEntity{values: map[string]any{}}, &sourceTestEntity{values: map[string]any{}}
	if *tracker.entityAlias(a) == *tracker.entityAlias(b) || *tracker.entityAlias(a) != *tracker.entityAlias(a) || tracker.entityAlias(nil) != nil {
		t.Fatal("lifetime alias collision")
	}
}

func TestTraceScalarNoCoercion(t *testing.T) {
	for _, v := range []any{true, "800", float64(800), uint64(1 << 63)} {
		if value, status := traceScalar(v); value != nil || status != "unsupported_type" {
			t.Fatal(v)
		}
	}
	if v, s := traceScalar(nil); v != nil || s != "absent" {
		t.Fatal(v, s)
	}
	if v, s := traceScalar(uint64(0)); v != uint64(0) || s != "observed" {
		t.Fatal(v, s)
	}
}

func TestTraceUserIDObservedCS2TypeNinePreservesMissingness(t *testing.T) {
	for _, typ := range []int32{4, 9} {
		d := &msg.CMsgSource1LegacyGameEventListKeyT{Type: proto.Int32(typ)}
		k := &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(typ), ValShort: proto.Int32(0)}
		if !traceUserIDSupported(d, k) {
			t.Fatal("supported actor field rejected", typ)
		}
		k.ValShort = nil
		if traceUserIDSupported(d, k) {
			t.Fatal("absent identity became zero")
		}
	}
	if traceUserIDSupported(&msg.CMsgSource1LegacyGameEventListKeyT{Type: proto.Int32(9)}, &msg.CMsgSource1LegacyGameEventKeyT{Type: proto.Int32(4), ValShort: proto.Int32(1)}) {
		t.Fatal("type disagreement accepted")
	}
}
