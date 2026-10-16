package handlers

import (
	"cs2-demo-service/models"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"testing"
)

type cashflowEntity struct {
	ownershipEntity
	destroy func()
}

func (e *cashflowEntity) OnDestroy(fn func()) { e.destroy = fn }

func cashflowFixture() (*economyIdentityTracker, *economyTraceTracker, *ownershipGameState, *cashflowEntity) {
	t, gs, _, _ := entryTrackerFixture()
	controller := &cashflowEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hPlayerPawn": uint64(2<<14 | 1), economyTraceProperties[0]: int32(1000), economyTraceProperties[1]: int32(0)}}, index: 100, serial: 20000}}
	gs.players[0].Entity = controller
	t.ctx.EconomyCashflowObservations = &models.EconomyCashflowObservations{}
	t.ctx.EconomyCashflowRuntime = &models.EconomyCashflowRuntime{NextFrame: 1, Controllers: map[st.Entity]*models.EconomyCashflowController{}, Live: map[int]st.Entity{}}
	trace := &economyTraceTracker{ctx: t.ctx, ledger: &models.EconomyTransactionTrace{}, entities: map[st.Entity]string{}}
	t.ctx.EconomyTransactionTrace = trace.ledger
	alias := trace.entityAlias(controller)
	trace.cashflowController(controller, *alias)
	return t, trace, gs, controller
}

func TestCashflowFreshAnchorsAndDeliveryReferencesPreserveCompensation(t *testing.T) {
	tracker, trace, gs, controller := cashflowFixture()
	tracker.captureOwnership()
	for _, value := range []int32{700, 1000} {
		controller.values[economyTraceProperties[0]] = value
		trace.append("property_update", economyTraceProperties[0], gs.players[0], trace.entityAlias(controller), map[string]any{"value": value, "status": "observed"})
		trace.cashflowDelivery(controller)
	}
	tracker.frame = 1
	trace.frame = 1
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 2
	tracker.captureOwnership()
	ledger := tracker.ctx.EconomyCashflowObservations
	if len(ledger.Cuts) != 2 || len(ledger.Deliveries) != 2 || ledger.Cuts[1].TraceSequence != 2 || ledger.Cuts[1].TraceNextFrame != 2 {
		t.Fatal("common cut or compensated deliveries lost")
	}
	if ledger.Deliveries[0].TraceSequence != 1 || ledger.Deliveries[1].TraceSequence != 2 || ledger.States[0].Pawns[0].Account.Value != int32(1000) {
		t.Fatal("baseline or ordering changed")
	}
	if ledger.States[0].Pawns[0].Spent.Value != int32(0) {
		t.Fatal("zero became absent")
	}
	delete(controller.values, economyTraceProperties[0])
	tracker.frame = 2
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 3
	tracker.captureOwnership()
	last := ledger.States[len(ledger.States)-1]
	if last.Pawns[0].Account.Status != "absent" {
		t.Fatal("missing fresh account inherited old value")
	}
}

func TestCashflowControllerReplacementAndFullPawnSerialDoNotInherit(t *testing.T) {
	tracker, trace, gs, controller := cashflowFixture()
	tracker.captureOwnership()
	controller.values["m_hPlayerPawn"] = uint64(1026<<14 | 1)
	tracker.frame = 1
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 2
	tracker.captureOwnership()
	ledger := tracker.ctx.EconomyCashflowObservations
	if ledger.States[len(ledger.States)-1].Pawns[0].ControllerAlias != nil {
		t.Fatal("pawn serial truncated")
	}
	controller.values["m_hPlayerPawn"] = uint64(2<<14 | 1)
	replacement := &cashflowEntity{ownershipEntity: controller.ownershipEntity}
	replacement.serial++
	trace.frame = 2
	trace.cashflowController(replacement, *trace.entityAlias(replacement))
	tracker.frame = 2
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 3
	tracker.captureOwnership()
	if ledger.States[len(ledger.States)-1].Pawns[0].ControllerAlias != nil {
		t.Fatal("replaced controller still live")
	}
	gs.players[0].Entity = replacement
	tracker.frame = 3
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 4
	tracker.captureOwnership()
	if *ledger.States[0].Pawns[0].ControllerAlias == *ledger.States[len(ledger.States)-1].Pawns[0].ControllerAlias {
		t.Fatal("new controller reused prior life")
	}
	trace.frame = 4
	replacement.destroy()
	tracker.frame = 4
	tracker.ctx.EconomyCashflowRuntime.NextFrame = 5
	tracker.captureOwnership()
	if ledger.States[len(ledger.States)-1].Pawns[0].ControllerAlias != nil || *ledger.Controllers[1].DestroyedFrame != 4 {
		t.Fatal("destroyed controller inherited")
	}
}

func TestCashflowDeliveryRetainsCreationFrameAndUnavailablePawnHandle(t *testing.T) {
	tracker, trace, gs, controller := cashflowFixture()
	trace.frame = 0
	delete(controller.values, "m_hPlayerPawn")
	trace.append("property_baseline", economyTraceProperties[0], gs.players[0], trace.entityAlias(controller), map[string]any{"value": int32(1000), "status": "observed"})
	trace.cashflowDelivery(controller)
	ledger := tracker.ctx.EconomyCashflowObservations
	if ledger.Controllers[0].CreatedFrame != 0 || ledger.Deliveries[0].PawnHandle.Status != "absent" {
		t.Fatal("creation timing or absence lost")
	}
	controller.values["m_hPlayerPawn"] = uint64(2<<14 | 1)
	tracker.ctx.EconomyObservations.ActorAliases[7] = "p0001"
	tracker.captureOwnership()
	if ledger.Deliveries[0].PawnHandle.Status != "absent" {
		t.Fatal("delivery backfilled")
	}
}
