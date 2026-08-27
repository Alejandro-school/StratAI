package economy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

const (
	PriceTableVersion       = "stratai.cs2_prices@1"
	PriceTableEffectiveFrom = "2026-08-19"
	RulesVersion            = "stratai.cs2_economy_rules@1"
	RulesEffectiveFrom      = "2026-08-19"
	MaximumLossBonusLevel   = 4
)

var weaponPrices = map[string]int{
	"Glock-18": 200, "USP-S": 200, "P2000": 200, "Dual Berettas": 300,
	"P250": 300, "Tec-9": 500, "Five-SeveN": 500, "CZ75 Auto": 500,
	"CZ75-Auto": 500, "Desert Eagle": 700, "R8 Revolver": 600,
	"MAC-10": 1050, "MP9": 1250, "MP7": 1400, "MP5-SD": 1400,
	"UMP-45": 1200, "P90": 2350, "PP-Bizon": 1300,
	"Galil AR": 1800, "FAMAS": 1950, "AK-47": 2700, "M4A4": 2900,
	"M4A1-S": 2900, "M4A1": 2900, "SG 553": 3000, "AUG": 3300,
	"SSG 08": 1700, "AWP": 4750, "G3SG1": 5000, "SCAR-20": 5000,
	"Nova": 1050, "XM1014": 2000, "Sawed-Off": 1100, "MAG-7": 1300,
	"M249": 5200, "Negev": 1700,
	"Flashbang": 200, "Smoke Grenade": 300, "HE Grenade": 300,
	"Incendiary Grenade": 500, "Molotov": 400, "Decoy Grenade": 50,
	"Kevlar Vest": 650, "Kevlar + Helmet": 1000, "Zeus x27": 200,
	"Defuse Kit": 400, "Knife": 0, "C4": 0,
}

func PriceQuote(item string) (int, string) {
	price, ok := weaponPrices[item]
	if !ok {
		return 0, "unknown"
	}
	if price == 0 {
		return 0, "known_zero"
	}
	return price, "known"
}

func PriceTableChecksum() string {
	keys := make([]string, 0, len(weaponPrices))
	for item := range weaponPrices {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys)+2)
	entries = append(entries, PriceTableVersion, PriceTableEffectiveFrom)
	for _, item := range keys {
		entries = append(entries, fmt.Sprintf("%s=%d", item, weaponPrices[item]))
	}
	return checksum(strings.Join(entries, "\n"))
}

func LossBonus(level int) int {
	if level < 0 {
		level = 0
	}
	if level > MaximumLossBonusLevel {
		level = MaximumLossBonusLevel
	}
	return 1400 + level*500
}

func RulesChecksum() string {
	values := make([]string, MaximumLossBonusLevel+1)
	for level := range values {
		values[level] = fmt.Sprintf("%d=%d", level, LossBonus(level))
	}
	return checksum(
		strings.Join(
			append([]string{RulesVersion, RulesEffectiveFrom, "half_start_level=1", "winner_decrement=1"}, values...),
			"\n",
		),
	)
}

func checksum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
