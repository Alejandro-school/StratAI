package models

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
)

// The probe specifies names, never default values or economic formulas.
//
//go:embed economy_rule_context_probe.json
var economyRuleProbeJSON []byte

type EconomyRuleProbe struct {
	Version string          `json:"version"`
	Convars []string        `json:"convars"`
	Rules   map[string]bool `json:"rules"`
}

func LoadEconomyRuleProbe() (EconomyRuleProbe, string) {
	var probe EconomyRuleProbe
	if err := json.Unmarshal(economyRuleProbeJSON, &probe); err != nil {
		panic(err)
	}
	digest := sha256.Sum256(economyRuleProbeJSON)
	return probe, hex.EncodeToString(digest[:])
}

type EconomyConvarEntry struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
}

type EconomyConvarMessage struct {
	Sequence int                  `json:"sequence"`
	Tick     int                  `json:"tick"`
	Entries  []EconomyConvarEntry `json:"entries"`
}

type EconomyEffectiveConvar struct {
	Value          *string `json:"value"`
	SourceSequence *int    `json:"source_sequence"`
	SourceTick     *int    `json:"source_tick"`
}

type EconomyRuleBoundary struct {
	Sequence int                               `json:"sequence"`
	Tick     int                               `json:"tick"`
	Instance int                               `json:"instance"`
	Phase    string                            `json:"phase"`
	Convars  map[string]EconomyEffectiveConvar `json:"convars"`
	Rules    map[string]EconomyFact            `json:"rules"`
}

type EconomyOpaqueField struct {
	Present bool    `json:"present"`
	Bytes   *int    `json:"bytes"`
	SHA256  *string `json:"sha256"`
}

type EconomySessionProbe struct {
	Sequence             int                `json:"sequence"`
	Tick                 int                `json:"tick"`
	ConfigurationPresent bool               `json:"configuration_present"`
	Manifest             EconomyOpaqueField `json:"manifest"`
	ConfigurationData    EconomyOpaqueField `json:"configuration_data"`
}

type EconomyRuleContext struct {
	Messages   []EconomyConvarMessage `json:"messages"`
	Boundaries []EconomyRuleBoundary  `json:"boundaries"`
	Sessions   []EconomySessionProbe  `json:"sessions"`
}
