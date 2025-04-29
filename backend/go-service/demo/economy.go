package demo

import (
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

// Precios armas
var weaponPrices = map[common.EquipmentType]int{
	// Pistolas
	common.EqGlock:        200,
	common.EqUSP:          200,
	common.EqP2000:        200,
	common.EqP250:         300,
	common.EqFiveSeven:    500,
	common.EqTec9:         500,
	common.EqCZ:           500,
	common.EqDeagle:       700,
	common.EqRevolver:     600,
	common.EqDualBerettas: 300,

	// SMGs
	common.EqMac10: 1050,
	common.EqMP9:   1250,
	common.EqMP7:   1500,
	common.EqMP5:   1500,
	common.EqUMP:   1200,
	common.EqP90:   2350,
	common.EqBizon: 1400,

	// Rifles
	common.EqGalil:  1800,
	common.EqFamas:  1950,
	common.EqAK47:   2700,
	common.EqM4A4:   2900,
	common.EqM4A1:   2900,
	common.EqScout:  1700, // SSG08
	common.EqSG553:  3000, // SG 553
	common.EqAUG:    3300,
	common.EqAWP:    4750,
	common.EqG3SG1:  5000,
	common.EqScar20: 5000,

	// Escopetas
	common.EqNova:   1050,
	common.EqMag7:   1300,
	common.EqXM1014: 2000,

	// Ametralladoras
	common.EqM249:  5200,
	common.EqNegev: 1700,

	// Granadas
	common.EqMolotov:    400,
	common.EqIncendiary: 500,
	common.EqFlash:      200,
	common.EqSmoke:      300,
	common.EqHE:         300,
	common.EqDecoy:      50,

	// Taser
	common.EqZeus: 200,

	// Kevlar
	common.EqKevlar:    650,
	common.EqDefuseKit: 400,
	common.EqHelmet:    1000,
}

const (
	MinBonus       = 1400
	MaxBonus       = 3400
	BonusIncrement = 500
)

// getKillRewardType retorna la recompensa por kill basándose en el EquipmentType
// siguiendo la tabla proporcionada.
func getKillRewardType(eqType common.EquipmentType) int {
	switch eqType {
	// Cuchillo
	case common.EqKnife:
		return 1500

	// Taser
	case common.EqZeus:
		return 0

	// AWP y CZ75
	case common.EqAWP, common.EqCZ:
		return 100

	// SMG (Mac10, MP9, MP7, MP5, UMP, Bizon) y XM1014
	case common.EqMac10, common.EqMP9, common.EqMP7, common.EqMP5, common.EqUMP, common.EqBizon, common.EqXM1014:
		return 600

	// Escopetas (Nova, Mag7, SawedOff) -> 900
	case common.EqNova, common.EqMag7, common.EqSawedOff:
		return 900

	default:
		// Pistolas, rifles, P90, granadas y ametralladoras => 300
		return 300
	}
}

// UpdateLossBonus actualiza el bonus según si el equipo ganó o perdió la ronda.
// Si ganan, se reduce el bonus en 500 (sin bajar de 1400).
// Si pierden, se incrementa el bonus en 500 (hasta un máximo de 3400).
func UpdateLossBonus(currentBonus int, won bool, bombPlantedVictory bool) int {
	if won {
		next := currentBonus - BonusIncrement
		if next < MinBonus {
			return MinBonus
		}
		return next
	} else {
		next := currentBonus + BonusIncrement
		if next > MaxBonus {
			return MaxBonus
		}
		// Regla especial: si la victoria fue por bomba detonada (bombPlantedVictory)
		// y el bonus está en 3400, sumamos +600
		if bombPlantedVictory && next == MaxBonus {
			next += 600 // se convertiría en 4000
		}
		return next
	}
}

// GetWinAmount retorna el monto de victoria según el equipo y la condición de victoria.
// Para el equipo CT:
//   - "defuse" retorna 3500 (por desactivar la bomba)
//   - Cualquier otro valor (por tiempo o eliminación) retorna 3250
//
// Para el equipo T:
//   - "bomb" retorna 3500 (por detonación de bomba)
//   - Cualquier otro valor (por eliminación sin plantar) retorna 3250
func GetWinAmount(team common.Team, winType string) int {
	if team == common.TeamCounterTerrorists {
		if winType == "defuse" {
			return 3500
		}
		return 3250
	} else if team == common.TeamTerrorists {
		if winType == "bomb" {
			return 3500
		}
		return 3250
	}
	return 0
}
