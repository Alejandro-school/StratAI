package handlers

import (
	"cs2-demo-service/pkg/playerstate"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type replayReloadSignal string

const (
	replayReloadSignalStart      replayReloadSignal = "weapon_reload"
	replayReloadSignalItemEquip  replayReloadSignal = "item_equip"
	replayReloadSignalWeaponFire replayReloadSignal = "weapon_fire"
	replayReloadSignalClipUpdate replayReloadSignal = "clip_update"
)

type replayReloadObservation struct {
	value            bool
	available        bool
	availabilityTick int
	weaponID         int64
	signal           replayReloadSignal
	signalTick       int // Original qualifying callback, not the refreshed sample tick.
}

// replayReloadTracker separates the native bool's zero value from a callback-
// backed observation. Clip callback bindings live for the match; observations
// are cleared at every round boundary so no state carries across rounds.
type replayReloadTracker struct {
	observations    map[uint64]replayReloadObservation
	boundClipWeapon map[*common.Equipment]struct{}
}

func newReplayReloadTracker() replayReloadTracker {
	return replayReloadTracker{
		observations:    make(map[uint64]replayReloadObservation),
		boundClipWeapon: make(map[*common.Equipment]struct{}),
	}
}

func (tracker *replayReloadTracker) resetRound() {
	if tracker == nil {
		return
	}
	tracker.observations = make(map[uint64]replayReloadObservation)
}

func (tracker *replayReloadTracker) invalidatePlayer(steamID uint64) {
	if tracker == nil || steamID == 0 {
		return
	}
	delete(tracker.observations, steamID)
}

func (tracker *replayReloadTracker) invalidateWeapon(steamID uint64, weaponID int64) {
	if tracker == nil || steamID == 0 || weaponID == 0 {
		return
	}
	observation, exists := tracker.observations[steamID]
	if exists && observation.weaponID == weaponID {
		delete(tracker.observations, steamID)
	}
}

func (tracker *replayReloadTracker) record(
	steamID uint64,
	weaponID int64,
	value bool,
	tick int,
	signal replayReloadSignal,
) {
	if tracker == nil || steamID == 0 || weaponID == 0 || tick < 0 || !replayReloadSignalAccredits(signal, value) {
		return
	}
	if tracker.observations == nil {
		tracker.observations = make(map[uint64]replayReloadObservation)
	}
	if current, exists := tracker.observations[steamID]; exists && current.available && tick < current.availabilityTick {
		return
	}
	tracker.observations[steamID] = replayReloadObservation{
		value:            value,
		available:        true,
		availabilityTick: tick,
		weaponID:         weaponID,
		signal:           signal,
		signalTick:       tick,
	}
}

func replayReloadSignalAccredits(signal replayReloadSignal, value bool) bool {
	if value {
		return signal == replayReloadSignalStart
	}
	switch signal {
	case replayReloadSignalItemEquip, replayReloadSignalWeaponFire, replayReloadSignalClipUpdate:
		return true
	default:
		return false
	}
}

func (tracker *replayReloadTracker) recordActiveSignal(
	steamID uint64,
	alive bool,
	activeWeaponID int64,
	eventWeaponID int64,
	value bool,
	tick int,
	signal replayReloadSignal,
) {
	if tracker == nil || steamID == 0 {
		return
	}
	if !alive || activeWeaponID == 0 || activeWeaponID != eventWeaponID {
		tracker.invalidatePlayer(steamID)
		return
	}
	tracker.record(steamID, activeWeaponID, value, tick, signal)
}

// refreshSnapshot can only refresh state initialized by a valid callback. It
// binds the native demoinfocs bool to the exact active entity and stamps the
// current replay tick, which is the source tick consumed by the ledger.
func (tracker *replayReloadTracker) refreshSnapshot(
	steamID uint64,
	alive bool,
	activeWeaponID int64,
	nativeValue bool,
	sampleTick int,
) replayReloadObservation {
	if tracker == nil || steamID == 0 || sampleTick < 0 {
		return replayReloadObservation{}
	}
	if !alive || activeWeaponID == 0 {
		tracker.invalidatePlayer(steamID)
		return replayReloadObservation{}
	}
	observation, exists := tracker.observations[steamID]
	if !exists || !observation.available || observation.availabilityTick > sampleTick {
		return replayReloadObservation{}
	}
	if observation.weaponID != activeWeaponID {
		tracker.invalidatePlayer(steamID)
		return replayReloadObservation{}
	}
	observation.value = nativeValue
	observation.availabilityTick = sampleTick
	tracker.observations[steamID] = observation
	return observation
}

func (handler *ReplayHandler) recordReloadStart(player *common.Player, tick int) {
	if handler == nil || player == nil {
		return
	}
	weapon := player.ActiveWeapon()
	weaponID := playerstate.EquipmentID(weapon)
	handler.bindReloadClipCallback(weapon)
	handler.reloadState.recordActiveSignal(
		player.SteamID64,
		player.IsAlive(),
		weaponID,
		weaponID,
		true,
		tick,
		replayReloadSignalStart,
	)
}

func (handler *ReplayHandler) recordReloadFalse(
	player *common.Player,
	weapon *common.Equipment,
	tick int,
	signal replayReloadSignal,
) {
	if handler == nil || player == nil {
		return
	}
	eventWeaponID := playerstate.EquipmentID(weapon)
	activeWeaponID := playerstate.EquipmentID(player.ActiveWeapon())
	handler.bindReloadClipCallback(weapon)
	handler.reloadState.recordActiveSignal(
		player.SteamID64,
		player.IsAlive(),
		activeWeaponID,
		eventWeaponID,
		false,
		tick,
		signal,
	)
}

func (handler *ReplayHandler) bindReloadClipCallbacks(players []*common.Player) {
	if handler == nil {
		return
	}
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, weapon := range player.Weapons() {
			handler.bindReloadClipCallback(weapon)
		}
	}
}

func (handler *ReplayHandler) bindReloadClipCallback(weapon *common.Equipment) {
	if handler == nil || weapon == nil || weapon.Entity == nil {
		return
	}
	if handler.reloadState.boundClipWeapon == nil {
		handler.reloadState.boundClipWeapon = make(map[*common.Equipment]struct{})
	}
	if _, exists := handler.reloadState.boundClipWeapon[weapon]; exists {
		return
	}
	clip := weapon.Entity.Property("m_iClip1")
	if clip == nil {
		return
	}
	lastClip, lastClipAvailable := replayReloadClipValue(clip.Value())
	handler.reloadState.boundClipWeapon[weapon] = struct{}{}
	clip.OnUpdate(func(value st.PropertyValue) {
		currentClip, currentClipAvailable := replayReloadClipValue(value)
		changed := lastClipAvailable && currentClipAvailable && currentClip != lastClip
		lastClip = currentClip
		lastClipAvailable = currentClipAvailable
		if handler.currentRound == nil || handler.ctx == nil || handler.ctx.Parser == nil {
			return
		}
		owner := weapon.Owner
		weaponID := playerstate.EquipmentID(weapon)
		if owner == nil || weaponID == 0 {
			return
		}
		if !currentClipAvailable {
			handler.reloadState.invalidateWeapon(owner.SteamID64, weaponID)
			return
		}
		if !changed {
			return
		}
		handler.reloadState.recordActiveSignal(
			owner.SteamID64,
			owner.IsAlive(),
			playerstate.EquipmentID(owner.ActiveWeapon()),
			weaponID,
			false,
			handler.ctx.Parser.GameState().IngameTick(),
			replayReloadSignalClipUpdate,
		)
	})
}

func replayReloadClipValue(value st.PropertyValue) (int, bool) {
	raw, ok := value.Any.(int32)
	if !ok || raw < 0 {
		return 0, false
	}
	return int(raw), true
}
