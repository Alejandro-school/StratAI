package handlers

import (
	"encoding/json"
	"fmt"
	"sort"

	"cs2-demo-service/models"
)

// This is a fresh read after the SDK processed the frame's queued messages and
// processFrameGameEvents. It never reinterprets an earlier unresolved reference.
func (t *economyIdentityTracker) captureOwnership() {
	gs := t.ctx.Parser.GameState()
	rounds := t.ctx.EconomyObservations.Rounds
	if len(rounds) == 0 || gs.IsWarmupPeriod() || !gs.IsFreezetimePeriod() {
		return
	}
	round := rounds[len(rounds)-1]
	tick := gs.IngameTick()
	if tick <= round.StartTick || (round.FreezeEndTick != nil && tick >= *round.FreezeEndTick) {
		return
	}
	state := models.EconomyOwnershipState{Pawns: []models.EconomyOwnershipPawn{}, Equipment: []models.EconomyOwnershipEquipment{}}
	entities := make([]*identityEntity, 0, len(t.live))
	for _, e := range t.live {
		if e.alive {
			entities = append(entities, e)
		}
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].alias < entities[j].alias })
	for _, e := range entities {
		if e.kind == "equipment" {
			var raw any
			if prop := e.entity.Property("m_hOwnerEntity"); prop != nil {
				raw = prop.Value().Any
			}
			state.Equipment = append(state.Equipment, models.EconomyOwnershipEquipment{EntityAlias: e.alias, Owner: t.reference(raw, "pawn")})
			continue
		}
		pawn := models.EconomyOwnershipPawn{EntityAlias: e.alias, Status: "absent", Slots: []models.EconomyEntityReference{}}
		if p := t.playerForPawn(e); p != nil && p.SteamID64 != 0 {
			if alias := t.ctx.EconomyObservations.ActorAliases[p.SteamID64]; alias != "" {
				pawn.ActorAlias = &alias
			}
			if side := economySide(p); side == "CT" || side == "T" {
				pawn.Side = &side
			}
		}
		if prop := e.entity.Property(identityInventory); prop != nil && prop.Value().Any != nil {
			pawn.Status = "unsupported_type"
			if array, ok := prop.Value().Any.([]any); ok {
				length := len(array)
				pawn.Length = &length
				pawn.Status = "outside_probe"
				if length <= 64 {
					pawn.Status = "observed"
					for slot := 0; slot < length; slot++ {
						var raw any
						if prop := e.entity.Property(fmt.Sprintf("%s.%04d", identityInventory, slot)); prop != nil {
							raw = prop.Value().Any
						}
						pawn.Slots = append(pawn.Slots, t.reference(raw, "equipment"))
					}
				}
			}
		}
		state.Pawns = append(state.Pawns, pawn)
	}
	// Keep a marker for EVERY selected FrameDone. Only identical freshly read
	// payloads share storage; a state is never referenced before its first cut.
	fingerprint, _ := json.Marshal(state)
	ledger := t.ctx.EconomyOwnershipObservations
	cut := models.EconomyOwnershipCut{Sequence: len(ledger.Cuts) + 1, Frame: t.frame, DemoTick: tick, Instance: round.Instance, IdentitySequence: t.sequence}
	if n := len(t.ledger.Clocks); n > 0 {
		seq := t.ledger.Clocks[n-1].Sequence
		cut.ClockSequence = &seq
	}
	if len(ledger.Cuts) == 0 || ledger.Cuts[len(ledger.Cuts)-1].Instance != round.Instance || string(fingerprint) != t.ownershipFingerprint {
		state.Sequence, state.FirstCut = len(ledger.States)+1, cut.Sequence
		ledger.States = append(ledger.States, state)
		t.ownershipFingerprint = string(fingerprint)
	}
	cut.StateSequence = len(ledger.States)
	ledger.Cuts = append(ledger.Cuts, cut)
	t.captureEntries(entities, cut)
}
