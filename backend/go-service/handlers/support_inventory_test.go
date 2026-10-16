package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"testing"
)

func TestInventoryFreshReserveAndMissingAtSameTick(t *testing.T) {
	h, p, _, item, base, _ := c4Fixture()
	h.ctx.SupportInventoryObservations = &models.SupportInventoryObservations{}
	item.values["m_pReserveAmmo.0000"] = int32(1)
	capture := func() {
		h.captureSupportC4(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	capture()
	data := h.ctx.SupportInventoryObservations
	if len(data.States) != 1 || len(data.Cuts) != 2 {
		t.Fatal("identical fresh reads not shared")
	}
	item.values["m_pReserveAmmo.0000"] = int32(0)
	capture()
	if len(data.States) != 2 || len(h.ctx.SupportC4Observations.States) != 1 {
		t.Fatal("quantity change not captured independently of ownership")
	}
	delete(item.values, "m_pReserveAmmo.0000")
	capture()
	if data.States[2].Items[0].Fields["m_pReserveAmmo.0000"].Status != "absent" {
		t.Fatal("missing reserve carried forward")
	}
	item.values["m_pReserveAmmo.0000"] = "bad"
	capture()
	if data.States[3].Items[0].Fields["m_pReserveAmmo.0000"].Status != "unsupported_type" {
		t.Fatal("invalid type lost")
	}
	base.Instance++
	capture()
	base.Epoch++
	capture()
	if len(data.States) != 6 {
		t.Fatal("inventory crossed boundary")
	}
	if data.States[0].Items[0].Fields["m_pReserveAmmo.0000"].Value != int32(1) {
		t.Fatal("past mutated")
	}
}
