package models

const (
	ObservedWeaponStateSchemaID        = "stratai.observed_weapon_state@1"
	ObservedWeaponStateContractVersion = "1.0.0"
	ObservedWeaponStateCausalRole      = "model_input_observation"

	ObservedWeaponFactObserved    = "observed"
	ObservedWeaponFactUnavailable = "unavailable"

	ObservedWeaponSourceActiveWeapon = "replay_active_weapon_snapshot"
	ObservedWeaponSourceAmmo         = "demoinfocs_active_weapon_ammo"
	ObservedWeaponSourceReload       = "demoinfocs_causal_reload_tracker"
	ObservedWeaponSourceUnavailable  = "unavailable"

	ObservedWeaponReconciliationNotApplicable = "not_applicable"
	ObservedWeaponReconciliationFire          = "confirmed_weapon_fire"
	ObservedWeaponReconciliationReload        = "confirmed_weapon_reload"
	ObservedWeaponReconciliationAmbiguous     = "ambiguous"
	ObservedWeaponReconciliationUnavailable   = "unavailable"
)

// ObservedWeaponStateLedger is a candidate contract. It intentionally remains
// separate from the canonical 3.8.0 tactical export.
type ObservedWeaponStateLedger struct {
	SchemaID          string                   `json:"schema_id"`
	ContractVersion   string                   `json:"contract_version"`
	IdentitySemantics string                   `json:"identity_semantics"`
	CausalRole        string                   `json:"causal_role"`
	Sampling          ObservedWeaponSampling   `json:"sampling"`
	Rows              []ObservedWeaponStateRow `json:"rows"`
}

type ObservedWeaponSampling struct {
	TargetHz int    `json:"target_hz"`
	Strategy string `json:"strategy"`
}

// ObservedWeaponStateJoinKeys is the only place where identities are exposed.
// Consumers must use every field exclusively for joins, never as a feature.
type ObservedWeaponStateJoinKeys struct {
	MatchID       string `json:"match_id"`
	RoundID       string `json:"round_id"`
	PlayerID      string `json:"player_id"`
	ObservationID string `json:"observation_id"`
}

type ObservedWeaponFactProvenance struct {
	Source         string `json:"source"`
	SourceTick     *int   `json:"source_tick"`
	Reconciliation string `json:"reconciliation"`
}

type ObservedWeaponStringFact struct {
	Status           string                       `json:"status"`
	Value            *string                      `json:"value"`
	AvailabilityTick *int                         `json:"availability_tick"`
	Provenance       ObservedWeaponFactProvenance `json:"provenance"`
}

type ObservedWeaponIntFact struct {
	Status           string                       `json:"status"`
	Value            *int                         `json:"value"`
	AvailabilityTick *int                         `json:"availability_tick"`
	Provenance       ObservedWeaponFactProvenance `json:"provenance"`
}

type ObservedWeaponBoolFact struct {
	Status           string                       `json:"status"`
	Value            *bool                        `json:"value"`
	AvailabilityTick *int                         `json:"availability_tick"`
	Provenance       ObservedWeaponFactProvenance `json:"provenance"`
}

type ObservedWeaponStateRow struct {
	SchemaID          string                      `json:"schema_id"`
	ContractVersion   string                      `json:"contract_version"`
	IdentitySemantics string                      `json:"identity_semantics"`
	CausalRole        string                      `json:"causal_role"`
	JoinKeys          ObservedWeaponStateJoinKeys `json:"join_keys"`
	RoundNumber       int                         `json:"round_number"`
	Tick              int                         `json:"tick"`
	T0Tick            int                         `json:"t0_tick"`
	ActiveWeapon      ObservedWeaponStringFact    `json:"active_weapon"`
	AmmoInMagazine    ObservedWeaponIntFact       `json:"ammo_in_magazine"`
	AmmoReserve       ObservedWeaponIntFact       `json:"ammo_reserve"`
	IsReloading       ObservedWeaponBoolFact      `json:"is_reloading"`
	AbstentionReasons []string                    `json:"abstention_reasons"`
}

// ReplayWeaponStateObservation is retained only in memory. The current replay
// and canonical 3.8.0 JSON shapes remain unchanged.
type ReplayWeaponStateObservation struct {
	ActiveWeapon                 *string `json:"-"`
	ActiveWeaponStatus           string  `json:"-"`
	ActiveWeaponAvailabilityTick *int    `json:"-"`

	AmmoInMagazine                 *int   `json:"-"`
	AmmoInMagazineStatus           string `json:"-"`
	AmmoInMagazineAvailabilityTick *int   `json:"-"`

	AmmoReserve                 *int   `json:"-"`
	AmmoReserveStatus           string `json:"-"`
	AmmoReserveAvailabilityTick *int   `json:"-"`

	IsReloading                 *bool  `json:"-"`
	IsReloadingStatus           string `json:"-"`
	IsReloadingAvailabilityTick *int   `json:"-"`

	NonAmmunitionWeapon bool     `json:"-"`
	AbstentionReasons   []string `json:"-"`
}
