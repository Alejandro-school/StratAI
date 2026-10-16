package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// Read both native reserve variants at the exact C4 cut. No SDK default, cached
// replay list or future callback fills an absent or invalid property.
func (h *ReplayHandler) captureSupportInventory(base models.SupportFrame, ownership models.SupportC4State, current map[int]st.Entity, c4Sequence int) {
	data := h.ctx.SupportInventoryObservations
	if data == nil {
		return
	}
	state := models.SupportInventoryState{Items: []models.SupportInventoryItem{}}
	for _, item := range ownership.Entities {
		e := current[int(item.Handle&16383)]
		life := h.supportC4State.objects[e]
		if e == nil || life == nil || !life.alive || life.life != item.Life || uint64(e.SerialNum()) != item.Handle>>14 {
			continue
		}
		state.Items = append(state.Items, models.SupportInventoryItem{Life: item.Life, Handle: item.Handle, Fields: map[string]models.EconomyEntryScalar{
			"m_pReserveAmmo.0000":        entryScalar(e, "m_pReserveAmmo.0000"),
			"m_iPrimaryReserveAmmoCount": entryScalar(e, "m_iPrimaryReserveAmmoCount"),
		}})
	}
	raw, _ := json.Marshal(state.Items)
	changed := len(data.Cuts) == 0 || string(raw) != h.supportC4State.inventoryFingerprint
	if len(data.Cuts) > 0 {
		previous := data.Cuts[len(data.Cuts)-1]
		changed = changed || previous.Instance != base.Instance || previous.Epoch != base.Epoch
	}
	if changed {
		state.Sequence, state.FirstCut = len(data.States)+1, base.Sequence
		data.States = append(data.States, state)
		h.supportC4State.inventoryFingerprint = string(raw)
	}
	data.Cuts = append(data.Cuts, models.SupportInventoryCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, C4StateSequence: c4Sequence, StateSequence: len(data.States)})
}
