package models

type CanonicalAvailabilityValue struct {
	Amount *int   `json:"amount"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type CanonicalEconomyPrice struct {
	Amount       *int   `json:"amount"`
	Status       string `json:"status"`
	TableVersion string `json:"table_version"`
}

type CanonicalEconomyItem struct {
	ObservedItem       *string               `json:"observed_item"`
	PurchasedItem      *string               `json:"purchased_item"`
	EntityID           *string               `json:"entity_id"`
	OriginalOwnerID    *string               `json:"original_owner_player_id"`
	OriginalOwnerState string                `json:"original_owner_status"`
	ObservationStatus  string                `json:"observation_status"`
	Price              CanonicalEconomyPrice `json:"price"`
}

type CanonicalEconomyTransaction struct {
	TransactionID  string               `json:"transaction_id"`
	Type           string               `json:"type"`
	Tick           *int                 `json:"tick"`
	ActorPlayerID  string               `json:"actor_player_id"`
	OtherPlayerID  *string              `json:"other_player_id"`
	Item           CanonicalEconomyItem `json:"item"`
	Amount         *int                 `json:"amount"`
	Status         string               `json:"status"`
	Source         string               `json:"source"`
	SourceEventIDs []string             `json:"source_event_ids"`
}

type CanonicalEconomyInventory struct {
	Status          string                 `json:"status"`
	NativeValue     *int                   `json:"native_value"`
	CalculatedValue *int                   `json:"calculated_value"`
	Items           []CanonicalEconomyItem `json:"items"`
}

type CanonicalEconomyMoney struct {
	RoundStartObserved    CanonicalAvailabilityValue `json:"round_start_observed"`
	FreezeEndObserved     CanonicalAvailabilityValue `json:"freeze_end_observed"`
	AfterBuyObserved      CanonicalAvailabilityValue `json:"after_buy_observed"`
	AfterBuyCalculated    CanonicalAvailabilityValue `json:"after_buy_calculated"`
	RoundEndObserved      CanonicalAvailabilityValue `json:"round_end_observed"`
	NextRoundObserved     CanonicalAvailabilityValue `json:"next_round_observed"`
	NextRoundCalculated   CanonicalAvailabilityValue `json:"next_round_calculated"`
	NativeCalculatedDelta *int                       `json:"native_calculated_delta"`
}

type CanonicalEconomyPlayer struct {
	RoundID         string                        `json:"round_id"`
	RoundNumber     int                           `json:"round_number"`
	PlayerID        string                        `json:"player_id"`
	TeamID          string                        `json:"team_id"`
	Side            string                        `json:"side"`
	Outcome         string                        `json:"outcome"`
	Survived        bool                          `json:"survived"`
	Money           CanonicalEconomyMoney         `json:"money"`
	InventoryStart  CanonicalEconomyInventory     `json:"inventory_start"`
	InventoryFreeze CanonicalEconomyInventory     `json:"inventory_freeze_end"`
	InventoryEnd    CanonicalEconomyInventory     `json:"inventory_round_end"`
	SpentInBuy      CanonicalAvailabilityValue    `json:"spent_in_buy"`
	Transactions    []CanonicalEconomyTransaction `json:"transactions"`
	Warnings        []string                      `json:"warnings"`
}

type CanonicalEconomyPlayerExport struct {
	SchemaID     string                   `json:"schema_id"`
	MatchID      string                   `json:"match_id"`
	PriceTable   CanonicalVersionedConfig `json:"price_table"`
	EconomyRules CanonicalVersionedConfig `json:"economy_rules"`
	Players      []CanonicalEconomyPlayer `json:"players"`
}

type CanonicalLossBonus struct {
	Level        int    `json:"level"`
	Amount       int    `json:"amount"`
	Status       string `json:"status"`
	RulesVersion string `json:"rules_version"`
}

type CanonicalEconomyReward struct {
	RewardID         string   `json:"reward_id"`
	Type             string   `json:"type"`
	PlayerID         *string  `json:"player_id"`
	ObservedAmount   *int     `json:"observed_amount"`
	CalculatedAmount *int     `json:"calculated_amount"`
	Status           string   `json:"status"`
	SourceEventIDs   []string `json:"source_event_ids"`
}

type CanonicalEconomyRound struct {
	RoundID        string                     `json:"round_id"`
	RoundNumber    int                        `json:"round_number"`
	TeamID         string                     `json:"team_id"`
	Side           string                     `json:"side"`
	Outcome        string                     `json:"outcome"`
	WinReason      *string                    `json:"win_reason"`
	LossBonus      CanonicalLossBonus         `json:"loss_bonus"`
	MoneyStart     CanonicalAvailabilityValue `json:"money_start"`
	MoneyFreezeEnd CanonicalAvailabilityValue `json:"money_freeze_end"`
	MoneyRoundEnd  CanonicalAvailabilityValue `json:"money_round_end"`
	Rewards        []CanonicalEconomyReward   `json:"rewards"`
	Diagnostics    map[string]int             `json:"diagnostics"`
}

type CanonicalEconomyRoundExport struct {
	SchemaID     string                   `json:"schema_id"`
	MatchID      string                   `json:"match_id"`
	EconomyRules CanonicalVersionedConfig `json:"economy_rules"`
	Rounds       []CanonicalEconomyRound  `json:"rounds"`
}

type CanonicalVersionedConfig struct {
	Version             string  `json:"version"`
	ChecksumSHA256      string  `json:"checksum_sha256"`
	EffectiveFrom       *string `json:"effective_from"`
	ApplicabilityStatus string  `json:"applicability_status"`
	Source              string  `json:"source"`
}

type CanonicalClutchEvent struct {
	ClutchID         string   `json:"clutch_id"`
	RoundID          string   `json:"round_id"`
	RoundNumber      int      `json:"round_number"`
	PlayerID         string   `json:"player_id"`
	TeamID           string   `json:"team_id"`
	Side             string   `json:"side"`
	EnemiesAtStart   int      `json:"enemies_at_start"`
	State            string   `json:"state"`
	Attempt          bool     `json:"attempt"`
	Result           string   `json:"result"`
	StartTick        int      `json:"start_tick"`
	TriggerEventID   string   `json:"trigger_event_id"`
	SourceEventIDs   []string `json:"source_event_ids"`
	OutcomeSource    string   `json:"outcome_source"`
	EvaluationStatus string   `json:"evaluation_status"`
}

type CanonicalClutchEventExport struct {
	SchemaID     string                   `json:"schema_id"`
	MatchID      string                   `json:"match_id"`
	Algorithm    CanonicalVersionedConfig `json:"algorithm"`
	ClutchEvents []CanonicalClutchEvent   `json:"clutch_events"`
}

type CanonicalClutchSummary struct {
	Attempts     int            `json:"attempts"`
	Wins         int            `json:"wins"`
	Losses       int            `json:"losses"`
	NotEvaluable int            `json:"not_evaluable"`
	ByState      map[string]int `json:"by_state"`
}

type CanonicalStatsDerived struct {
	KillsObserved         int            `json:"kills_observed"`
	DeathsObserved        int            `json:"deaths_observed"`
	AssistsObserved       int            `json:"assists_observed"`
	CombatDamageObserved  int            `json:"combat_damage_observed"`
	UtilityDamageObserved int            `json:"utility_damage_observed"`
	GrenadeDamageObserved map[string]int `json:"grenade_damage_observed"`
	TradeKills            int            `json:"trade_kills"`
	TradedDeaths          int            `json:"traded_deaths"`
	OpeningDuelsAttempted int            `json:"opening_duels_attempted"`
	OpeningDuelsWon       int            `json:"opening_duels_won"`
	OpeningDuelsLost      int            `json:"opening_duels_lost"`
	KDRatioObserved       *float64       `json:"kd_ratio_observed"`
}

type CanonicalStatsReconciliation struct {
	KillsNativeMinusObserved   *int `json:"kills_native_minus_observed"`
	DeathsNativeMinusObserved  *int `json:"deaths_native_minus_observed"`
	AssistsNativeMinusObserved *int `json:"assists_native_minus_observed"`
	DamageNativeMinusObserved  *int `json:"damage_native_minus_observed"`
	UtilityNativeMinusObserved *int `json:"utility_native_minus_observed"`
}

type CanonicalApproximateRating struct {
	Value            *float64 `json:"value"`
	Status           string   `json:"status"`
	Approximate      bool     `json:"approximate"`
	AlgorithmVersion string   `json:"algorithm_version"`
	Formula          string   `json:"formula"`
	Source           string   `json:"source"`
}

type CanonicalPlayerStats struct {
	PlayerID               string                       `json:"player_id"`
	TeamID                 string                       `json:"team_id"`
	NativeScoreboardStatus string                       `json:"native_scoreboard_status"`
	NativeScoreboard       *AI_NativePlayerStats        `json:"native_scoreboard"`
	Derived                CanonicalStatsDerived        `json:"derived"`
	Reconciliation         CanonicalStatsReconciliation `json:"reconciliation"`
	Clutch                 CanonicalClutchSummary       `json:"clutch"`
	Rating                 CanonicalApproximateRating   `json:"rating"`
	Metrics                AI_PlayerStats               `json:"metrics"`
	Provenance             map[string]string            `json:"provenance"`
}

type CanonicalPlayerStatsExport struct {
	SchemaID string                 `json:"schema_id"`
	MatchID  string                 `json:"match_id"`
	Players  []CanonicalPlayerStats `json:"players"`
}

type CanonicalSourceProvenance struct {
	Source    string  `json:"source"`
	Endpoint  *string `json:"endpoint"`
	QueriedAt *string `json:"queried_at"`
	Version   *string `json:"version"`
	Checksum  string  `json:"checksum_sha256"`
}

type CanonicalMatchMetadata struct {
	SchemaID               string                              `json:"schema_id"`
	MatchID                string                              `json:"match_id"`
	PlayedAt               *string                             `json:"played_at"`
	PlayedAtStatus         string                              `json:"played_at_status"`
	PlayedAtSource         *string                             `json:"played_at_source"`
	OriginDate             *string                             `json:"origin_date"`
	OriginDateStatus       string                              `json:"origin_date_status"`
	Source                 CanonicalSourceProvenance           `json:"source"`
	ParserVersion          string                              `json:"parser_version"`
	ExportFormatVersion    string                              `json:"export_format_version"`
	QualitySchemaVersion   int                                 `json:"quality_schema_version"`
	PriceTable             CanonicalVersionedConfig            `json:"price_table"`
	EconomyRules           CanonicalVersionedConfig            `json:"economy_rules"`
	Algorithms             map[string]CanonicalVersionedConfig `json:"algorithms"`
	ConfigurationHashes    map[string]string                   `json:"configuration_hashes"`
	TransformationVersions map[string]string                   `json:"transformation_versions"`
	Availability           map[string]string                   `json:"availability"`
	Warnings               []string                            `json:"warnings"`
}

type CanonicalExportProvenance struct {
	DemoChecksum    string
	Source          string
	Endpoint        string
	QueriedAt       string
	SourceVersion   string
	BuildIdentifier string
}
