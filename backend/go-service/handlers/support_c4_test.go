package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"testing"
)

type c4TestState struct {
	dem.GameState
	current map[int]st.Entity
}

func (s c4TestState) Entities() map[int]st.Entity { return s.current }

type c4TestParser struct {
	dem.Parser
	gs c4TestState
}

func (p c4TestParser) GameState() dem.GameState { return p.gs }

func c4Fixture() (*ReplayHandler, *common.Player, *physicalPawn, *cashflowEntity, models.SupportFrame, map[int]st.Entity) {
	h, p, pawn, item, base, _ := physicalFixture()
	life, handle := "support-pawn-000001", uint64(3<<14|5)
	base.PawnLife, base.PawnHandle = &life, &handle
	base.PositionStatus, base.AliveStatus = "observed", "observed"
	item.values["m_iItemDefinitionIndex"] = uint64(49)
	item.values["m_hOwnerEntity"] = handle
	pawn.values[identityInventory] = []any{nil}
	pawn.values[identityInventory+".0000"] = uint64(2<<14 | 10)
	current := map[int]st.Entity{5: pawn, 10: item}
	h.ctx.Parser = c4TestParser{gs: c4TestState{current: current}}
	h.ctx.SupportC4Observations = &models.SupportC4Observations{}
	h.support = &supportTracker{lives: map[st.Entity]string{pawn: life}}
	h.supportC4State = &supportC4Tracker{objects: map[st.Entity]*supportC4Life{}}
	h.supportC4State.created(item, "CC4")
	h.supportObjectiveOrder = 7
	return h, p, pawn, item, models.SupportFrame{Sequence: 1, Frame: 100, Tick: 200, Instance: 1, Players: []models.SupportPlayer{base}}, current
}

func TestSupportC4FreshInventoryAndOwnerEachSameTickCut(t *testing.T) {
	h, p, pawn, item, base, _ := c4Fixture()
	capture := func() {
		h.captureSupportC4(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	first, _ := json.Marshal(h.ctx.SupportC4Observations.States[0])
	capture() // Same freshly read payload shares storage.
	if len(h.ctx.SupportC4Observations.States) != 1 {
		t.Fatal("identical payload not shared")
	}
	item.values["m_hOwnerEntity"] = uint64(4294967295)
	capture() // Same demo tick; owner now contradicts the still-current inventory.
	if len(h.ctx.SupportC4Observations.States) != 2 || h.ctx.SupportC4Observations.States[1].Entities[0].Owner.Status != "invalid_sentinel" {
		t.Fatal("owner was carried forward")
	}
	pawn.values[identityInventory] = []any{}
	capture()
	last := h.ctx.SupportC4Observations.States[2].Players[0]
	if last.InventoryStatus != "observed" || *last.InventoryLength != 0 || len(last.Slots) != 0 {
		t.Fatal("zero inventory lost")
	}
	base.Instance++
	capture()
	base.Epoch++
	capture()
	if len(h.ctx.SupportC4Observations.States) != 5 {
		t.Fatal("state crossed round or epoch")
	}
	after, _ := json.Marshal(h.ctx.SupportC4Observations.States[0])
	if string(first) != string(after) || h.ctx.SupportC4Observations.Cuts[0].ObjectiveOrder != 7 {
		t.Fatal("past cut mutated")
	}
}

func TestSupportC4ReferenceFullSerialDestroyedAndReusedObject(t *testing.T) {
	h, _, _, item, _, current := c4Fixture()
	tt := h.supportC4State
	ref := tt.itemReference(uint64(2<<14|10), current)
	if ref.Status != "resolved" {
		t.Fatal(ref)
	}
	for _, tc := range []struct {
		raw    any
		status string
	}{{nil, "absent"}, {false, "unsupported_type"}, {int64(-1), "unsupported_type"}, {uint64(1 << 32), "unsupported_type"}, {uint64(0), "entity_unavailable"}, {uint64(1026<<14 | 10), "serial_mismatch"}, {uint64(4294967295), "invalid_sentinel"}, {uint64(3<<14 | 5), "kind_or_life_unavailable"}} {
		if r := tt.itemReference(tc.raw, current); r.Status != tc.status {
			t.Fatalf("%v: %+v", tc.raw, r)
		}
	}
	item.destroy()
	if tt.itemReference(uint64(2<<14|10), current).Status != "destroyed" {
		t.Fatal("destroyed item accepted")
	}
	replacement := &cashflowEntity{ownershipEntity: item.ownershipEntity}
	current[10] = replacement
	tt.created(replacement, "CC4")
	second := tt.itemReference(uint64(2<<14|10), current)
	if second.Status != "resolved" || *second.TargetLife == *ref.TargetLife {
		t.Fatal("full handle reuse inherited object life")
	}
	current[0] = &cashflowEntity{ownershipEntity: ownershipEntity{index: 0, serial: 0}}
	tt.created(current[0], "CC4")
	if tt.itemReference(uint64(0), current).Status != "resolved" {
		t.Fatal("zero handle treated as missing")
	}
}

func TestSupportC4PartialInventoryAndPawnAvailability(t *testing.T) {
	for _, mode := range []string{"absent", "unsupported_type", "outside_probe", "slot_absent", "slot_serial", "dead", "disconnected", "life", "pawn_replaced", "owner_serial"} {
		t.Run(mode, func(t *testing.T) {
			h, p, pawn, item, base, current := c4Fixture()
			expect := "pawn_unavailable"
			switch mode {
			case "absent":
				delete(pawn.values, identityInventory)
				expect = "absent"
			case "unsupported_type":
				pawn.values[identityInventory] = 0
				expect = "unsupported_type"
			case "outside_probe":
				pawn.values[identityInventory] = make([]any, 65)
				expect = "outside_probe"
			case "slot_absent":
				delete(pawn.values, identityInventory+".0000")
				expect = "observed"
			case "slot_serial":
				pawn.values[identityInventory+".0000"] = uint64(1026<<14 | 10)
				expect = "observed"
			case "dead":
				*base.Players[0].Alive = false
			case "disconnected":
				base.Players[0].Connected = false
			case "life":
				delete(h.support.lives, pawn)
			case "pawn_replaced":
				current[5] = &physicalPawn{pawn.cashflowEntity}
			case "owner_serial":
				item.values["m_hOwnerEntity"] = uint64(1027<<14 | 5)
				expect = "observed"
			}
			h.captureSupportC4(base, map[uint64]*common.Player{p.SteamID64: p})
			state := h.ctx.SupportC4Observations.States[0]
			if state.Players[0].InventoryStatus != expect {
				t.Fatal(state.Players[0])
			}
			if mode == "slot_absent" && state.Players[0].Slots[0].Status != "absent" {
				t.Fatal("SDK inventory filled gap")
			}
			if mode == "slot_serial" && state.Players[0].Slots[0].Status != "serial_mismatch" {
				t.Fatal("slot serial masked")
			}
			if mode == "owner_serial" && state.Entities[0].Owner.Status != "serial_mismatch" {
				t.Fatal("owner serial masked")
			}
		})
	}
}

func TestSupportC4DisabledDoesNotReadParser(t *testing.T) {
	h := NewReplayHandler(&models.DemoContext{})
	h.registerSupportC4()
	h.captureSupportC4(models.SupportFrame{}, nil)
	if h.supportC4State != nil {
		t.Fatal("non-opt-in capture")
	}
}
