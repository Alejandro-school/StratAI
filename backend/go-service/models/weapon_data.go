package models

// WeaponMaxSpeed defines the maximum player movement speed (units/s) for each weapon
// Based on official CS2 game files and community data
// Source: SlothSquadron CS2 Weapon Spreadsheet + CS2 game files
var WeaponMaxSpeed = map[string]float64{
	// Rifles
	"AK-47":    215.0,
	"M4A4":     225.0,
	"M4A1":     225.0, // M4A1-S
	"M4A1-S":   225.0,
	"Galil AR": 220.0,
	"FAMAS":    220.0,
	"SG 553":   210.0,
	"AUG":      220.0,

	// Sniper Rifles
	"AWP":     200.0,
	"SSG 08":  230.0, // Scout
	"G3SG1":   215.0,
	"SCAR-20": 215.0,

	// SMGs
	"MP9":      240.0,
	"MAC-10":   240.0,
	"MP7":      220.0,
	"MP5-SD":   235.0,
	"UMP-45":   230.0,
	"PP-Bizon": 240.0,
	"P90":      230.0,

	// Heavy
	"Nova":      220.0,
	"XM1014":    215.0,
	"MAG-7":     225.0,
	"Sawed-Off": 220.0,
	"M249":      195.0,
	"Negev":     150.0,

	// Pistols
	"Glock-18":      240.0,
	"USP-S":         240.0,
	"P2000":         240.0,
	"P250":          240.0,
	"Five-SeveN":    240.0,
	"Tec-9":         240.0,
	"CZ75-Auto":     240.0,
	"Desert Eagle":  230.0,
	"Dual Berettas": 240.0,
	"R8 Revolver":   220.0,

	// Utility
	"Knife":    250.0,
	"Zeus x27": 240.0,
}

// GetWeaponMaxSpeed returns the MaxPlayerSpeed for a given weapon
// Returns 250.0 (default running speed) if weapon not found
func GetWeaponMaxSpeed(weaponName string) float64 {
	if speed, exists := WeaponMaxSpeed[weaponName]; exists {
		return speed
	}
	// Default to standard running speed if weapon unknown
	return 250.0
}

// GetAccuracyThreshold returns the speed threshold (34% of MaxSpeed)
// below which there is NO movement inaccuracy penalty
func GetAccuracyThreshold(weaponName string) float64 {
	return 0.34 * GetWeaponMaxSpeed(weaponName)
}
