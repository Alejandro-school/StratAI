package handlers

import "cs2-demo-service/pkg/sound"

func (h *ReplayHandler) soundInformationLedger() *sound.Information {
	if h.ctx == nil || !h.ctx.CaptureSoundInformation || h.currentRound == nil {
		return nil
	}
	if h.currentRound.SoundInformation == nil {
		h.currentRound.SoundInformation = sound.NewInformation()
	}
	return h.currentRound.SoundInformation
}
func (h *ReplayHandler) registerReplaySoundInformation() {
	if !h.ctx.CaptureSoundInformation {
		return
	}
	sound.RegisterNativeSounds(h.ctx.Parser, func(c sound.Candidate) {
		if ledger := h.soundInformationLedger(); ledger != nil {
			ledger.Observe(c, sound.NativeSoundObservers(h.ctx.Parser))
		}
	}, func(reason string) {
		if ledger := h.soundInformationLedger(); ledger != nil {
			ledger.Abstain(reason)
		}
	})
}
func (h *ReplayHandler) reconcileReplaySoundInformation(end int) {
	if h.currentRound.SoundInformation == nil {
		return
	}
	deaths := map[uint64]int{}
	for _, event := range h.currentRound.Events {
		if event.Type != "kill" || event.Tick < h.currentRound.StartTick {
			continue
		}
		if before, ok := deaths[event.VictimID]; !ok || event.Tick < before {
			deaths[event.VictimID] = event.Tick
		}
	}
	h.currentRound.SoundInformation.Reconcile(h.currentRound.StartTick, end, deaths)
}
