package handlers

import (
	"strings"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type rawWeaponAmmoObservation struct {
	class            common.EquipmentClass
	entityAvailable  bool
	magazine         int
	reserve          int
	reserveAvailable bool
}

const replayReloadAbstentionUnavailable = "reload_callback_unavailable"

func observeReplayWeaponState(
	player *common.Player,
	activeState activeWeaponState,
	reloadObservation replayReloadObservation,
	tick int,
) models.ReplayWeaponStateObservation {
	result := unavailableReplayWeaponState()
	if player == nil || !player.IsAlive() || activeState.Status != models.ActiveWeaponStatusObserved ||
		activeState.CurrentWeapon == nil {
		return result
	}

	weaponName := strings.TrimSpace(*activeState.CurrentWeapon)
	if weaponName == "" {
		return result
	}
	result.ActiveWeapon = stringPointer(weaponName)
	result.ActiveWeaponStatus = models.ObservedWeaponFactObserved
	result.ActiveWeaponAvailabilityTick = intPointer(tick)

	activeWeapon := player.ActiveWeapon()
	if activeWeapon == nil || strings.TrimSpace(activeWeapon.String()) != weaponName {
		result.AbstentionReasons = append(result.AbstentionReasons, "active_weapon_snapshot_unavailable")
		return result
	}

	magazine, reserve, applicable, reasons := observedWeaponAmmoWithReasons(activeWeapon)
	result.AbstentionReasons = append(result.AbstentionReasons, reasons...)
	if !applicable {
		result.NonAmmunitionWeapon = true
		return result
	}
	if magazine != nil {
		result.AmmoInMagazine = magazine
		result.AmmoInMagazineStatus = models.ObservedWeaponFactObserved
		result.AmmoInMagazineAvailabilityTick = intPointer(tick)
	}
	if reserve != nil {
		result.AmmoReserve = reserve
		result.AmmoReserveStatus = models.ObservedWeaponFactObserved
		result.AmmoReserveAvailabilityTick = intPointer(tick)
	}

	applyReplayReloadObservation(&result, reloadObservation, tick)
	return result
}

func applyReplayReloadObservation(
	result *models.ReplayWeaponStateObservation,
	observation replayReloadObservation,
	t0Tick int,
) {
	if result == nil {
		return
	}
	if !observation.available || observation.availabilityTick < 0 || observation.availabilityTick != t0Tick {
		result.AbstentionReasons = append(result.AbstentionReasons, replayReloadAbstentionUnavailable)
		return
	}
	result.IsReloading = boolPointer(observation.value)
	result.IsReloadingStatus = models.ObservedWeaponFactObserved
	result.IsReloadingAvailabilityTick = intPointer(observation.availabilityTick)
}

func unavailableReplayWeaponState() models.ReplayWeaponStateObservation {
	return models.ReplayWeaponStateObservation{
		ActiveWeaponStatus:   models.ObservedWeaponFactUnavailable,
		AmmoInMagazineStatus: models.ObservedWeaponFactUnavailable,
		AmmoReserveStatus:    models.ObservedWeaponFactUnavailable,
		IsReloadingStatus:    models.ObservedWeaponFactUnavailable,
	}
}

// observedWeaponAmmo rejects demoinfocs' ambiguous zero defaults. In v5.2.0
// AmmoInMagazine returns -1 when m_iClip1 is missing, but returns zero when the
// entity itself is nil. AmmoReserve returns zero for both a true empty reserve
// and a missing entity/property, so property presence must be checked directly.
func observedWeaponAmmo(weapon *common.Equipment) (magazine, reserve *int, applicable bool) {
	magazine, reserve, applicable, _ = observedWeaponAmmoWithReasons(weapon)
	return magazine, reserve, applicable
}

func observedWeaponAmmoWithReasons(
	weapon *common.Equipment,
) (magazine, reserve *int, applicable bool, reasons []string) {
	if weapon == nil {
		return nil, nil, false, nil
	}
	if !weaponClassUsesAmmunition(weapon.Class()) {
		return nil, nil, false, nil
	}

	raw := rawWeaponAmmoObservation{
		class:           weapon.Class(),
		entityAvailable: weapon.Entity != nil,
		magazine:        weapon.AmmoInMagazine(),
	}
	if weapon.Entity != nil {
		if property := weapon.Entity.Property("m_pReserveAmmo.0000"); property != nil && property.Value().Any != nil {
			raw.reserve = property.Value().Int()
			raw.reserveAvailable = true
		} else if value, ok := weapon.Entity.PropertyValue("m_iPrimaryReserveAmmoCount"); ok && value.Any != nil {
			raw.reserve = value.Int()
			raw.reserveAvailable = true
		}
	}
	magazine, reserve, applicable = normalizeRawWeaponAmmo(raw)
	if !raw.entityAvailable {
		reasons = append(reasons, "ammo_in_magazine_snapshot_unavailable", "ammo_reserve_snapshot_unavailable")
		return magazine, reserve, applicable, reasons
	}
	if raw.magazine < 0 {
		reasons = append(reasons, "invalid_ammo_sentinel:magazine")
	}
	if !raw.reserveAvailable {
		reasons = append(reasons, "ammo_reserve_snapshot_unavailable")
	} else if raw.reserve < 0 {
		reasons = append(reasons, "invalid_ammo_sentinel:reserve")
	}
	return magazine, reserve, applicable, reasons
}

func normalizeRawWeaponAmmo(raw rawWeaponAmmoObservation) (magazine, reserve *int, applicable bool) {
	if !weaponClassUsesAmmunition(raw.class) {
		return nil, nil, false
	}
	applicable = true
	if raw.entityAvailable && raw.magazine >= 0 {
		magazine = intPointer(raw.magazine)
	}
	if raw.entityAvailable && raw.reserveAvailable && raw.reserve >= 0 {
		reserve = intPointer(raw.reserve)
	}
	return magazine, reserve, applicable
}

func weaponClassUsesAmmunition(class common.EquipmentClass) bool {
	switch class {
	case common.EqClassPistols, common.EqClassSMG, common.EqClassHeavy, common.EqClassRifle:
		return true
	default:
		return false
	}
}

func intPointer(value int) *int {
	copy := value
	return &copy
}

func boolPointer(value bool) *bool {
	copy := value
	return &copy
}

func stringPointer(value string) *string {
	copy := value
	return &copy
}
