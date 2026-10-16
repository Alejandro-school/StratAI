package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func (t *economyTraceTracker) cashflowController(entity st.Entity, alias string) {
	rt := t.ctx.EconomyCashflowRuntime
	if rt == nil {
		return
	}
	c := &models.EconomyCashflowController{Alias: alias, Index: entity.ID(), Serial: entity.SerialNum(), CreatedFrame: t.frame}
	rt.Controllers[entity], rt.Live[entity.ID()] = c, entity
	t.ctx.EconomyCashflowObservations.Controllers = append(t.ctx.EconomyCashflowObservations.Controllers, c)
	entity.OnDestroy(func() {
		frame := t.frame
		c.DestroyedFrame = &frame
		if rt.Live[entity.ID()] == entity {
			delete(rt.Live, entity.ID())
		}
	})
}

func (t *economyTraceTracker) cashflowDelivery(entity st.Entity) {
	if t.ctx.EconomyCashflowObservations == nil {
		return
	}
	t.ctx.EconomyCashflowObservations.Deliveries = append(t.ctx.EconomyCashflowObservations.Deliveries, models.EconomyCashflowDelivery{
		TraceSequence: len(t.ledger.Rows), PawnHandle: entryScalar(entity, "m_hPlayerPawn"),
	})
}

func (t *economyIdentityTracker) captureCashflow(entities []*identityEntity, cut models.EconomyOwnershipCut, entryState int) {
	ledger, rt := t.ctx.EconomyCashflowObservations, t.ctx.EconomyCashflowRuntime
	if ledger == nil {
		return
	}
	state := models.EconomyCashflowState{EntryState: entryState, Pawns: []models.EconomyCashflowPawn{}}
	for _, e := range entities {
		if e.kind != "pawn" {
			continue
		}
		absent := models.EconomyEntryScalar{Status: "absent"}
		row := models.EconomyCashflowPawn{PawnAlias: e.alias, PawnHandle: absent, Account: absent, Spent: absent}
		if p := t.playerForPawn(e); p != nil && p.Entity != nil {
			if controller := rt.Controllers[p.Entity]; controller != nil && rt.Live[p.Entity.ID()] == p.Entity {
				alias := controller.Alias
				row.ControllerAlias = &alias
				row.PawnHandle = entryScalar(p.Entity, "m_hPlayerPawn")
				row.Account = entryScalar(p.Entity, economyTraceProperties[0])
				row.Spent = entryScalar(p.Entity, economyTraceProperties[1])
			}
		}
		state.Pawns = append(state.Pawns, row)
	}
	fingerprint, _ := json.Marshal(state)
	if len(ledger.States) == 0 || string(fingerprint) != rt.Fingerprint {
		state.Sequence, state.FirstCut = len(ledger.States)+1, cut.Sequence
		ledger.States = append(ledger.States, state)
		rt.Fingerprint = string(fingerprint)
	}
	ledger.Cuts = append(ledger.Cuts, models.EconomyCashflowCut{OwnershipCut: cut.Sequence, StateSequence: len(ledger.States), TraceSequence: len(t.ctx.EconomyTransactionTrace.Rows), TraceNextFrame: rt.NextFrame})
}

func RegisterEconomyCashflowHandler(ctx *models.DemoContext) {
	ctx.EconomyCashflowObservations = &models.EconomyCashflowObservations{Controllers: []*models.EconomyCashflowController{}, Deliveries: []models.EconomyCashflowDelivery{}, Cuts: []models.EconomyCashflowCut{}, States: []models.EconomyCashflowState{}}
	ctx.EconomyCashflowRuntime = &models.EconomyCashflowRuntime{Controllers: map[st.Entity]*models.EconomyCashflowController{}, Live: map[int]st.Entity{}}
	RegisterEconomyEntryHandler(ctx)
}
