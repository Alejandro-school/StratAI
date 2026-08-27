package models

// CanonicalArtifact describes one file in the additive, machine-learning-safe
// match bundle. Paths are relative to the canonical directory.
type CanonicalArtifact struct {
	ArtifactType string   `json:"artifact_type"`
	Path         string   `json:"path"`
	SchemaID     string   `json:"schema_id"`
	Format       string   `json:"format"`
	Compression  string   `json:"compression,omitempty"`
	RecordCount  int      `json:"record_count"`
	SHA256       string   `json:"sha256"`
	Bytes        int64    `json:"bytes"`
	SortOrder    []string `json:"sort_order"`
}

type CanonicalManifest struct {
	SchemaID               string              `json:"schema_id"`
	ExportFormatVersion    string              `json:"export_format_version"`
	MatchID                string              `json:"match_id"`
	DemoChecksumSHA256     string              `json:"demo_checksum_sha256"`
	ParserVersion          string              `json:"parser_version"`
	ConfigurationHashes    map[string]string   `json:"configuration_hashes"`
	TransformationVersions map[string]string   `json:"transformation_versions"`
	Lineage                CanonicalLineage    `json:"lineage"`
	Artifacts              []CanonicalArtifact `json:"artifacts"`
}

type CanonicalMatchExport struct {
	SchemaID string `json:"schema_id"`
	MatchID  string `json:"match_id"`
	CanonicalMatch
}

type CanonicalMatch struct {
	MapName         string          `json:"map_name"`
	PlayedAt        *string         `json:"played_at"`
	TickRateHz      float64         `json:"tick_rate_hz"`
	DurationMS      int64           `json:"duration_ms"`
	RoundCount      int             `json:"round_count"`
	CTScore         int             `json:"ct_score"`
	TScore          int             `json:"t_score"`
	WinnerSide      string          `json:"winner_side"`
	WinnerTeamID    *string         `json:"winner_team_id"`
	Teams           []CanonicalTeam `json:"teams"`
	CoordinateUnits string          `json:"coordinate_units"`
}

type CanonicalTeam struct {
	TeamID       string `json:"team_id"`
	StartingSide string `json:"starting_side"`
	Score        int    `json:"score"`
}

type CanonicalParticipantsExport struct {
	SchemaID string                 `json:"schema_id"`
	MatchID  string                 `json:"match_id"`
	Players  []CanonicalParticipant `json:"players"`
}

type CanonicalParticipant struct {
	PlayerID    string `json:"player_id"`
	SteamID     string `json:"steam_id"`
	DisplayName string `json:"display_name"`
	TeamID      string `json:"team_id"`
}

type CanonicalRoundsExport struct {
	SchemaID string           `json:"schema_id"`
	MatchID  string           `json:"match_id"`
	Rounds   []CanonicalRound `json:"rounds"`
}

type CanonicalRound struct {
	RoundID          string                    `json:"round_id"`
	RoundNumber      int                       `json:"round_number"`
	StartTick        *int                      `json:"start_tick"`
	EndTick          *int                      `json:"end_tick"`
	WinnerSide       *string                   `json:"winner_side"`
	WinnerTeamID     *string                   `json:"winner_team_id"`
	SideAssignments  []CanonicalSideAssignment `json:"side_assignments"`
	TeamScoresAfter  []CanonicalTeamScore      `json:"team_scores_after"`
	WinReason        *string                   `json:"win_reason"`
	RawWinReasonCode *int                      `json:"raw_win_reason_code"`
	CTScoreAfter     *int                      `json:"ct_score_after"`
	TScoreAfter      *int                      `json:"t_score_after"`
	BombPlanted      *bool                     `json:"bomb_planted"`
	BombSite         *string                   `json:"bomb_site"`
	BombTick         *int                      `json:"bomb_tick"`
	Objective        CanonicalRoundObjective   `json:"objective"`
}

type CanonicalRoundObjective struct {
	WasBombPlanted    bool    `json:"was_bomb_planted"`
	PlantEventID      *string `json:"plant_event_id"`
	Site              *string `json:"site"`
	PlantTick         *int    `json:"plant_tick"`
	PlanterPlayerID   *string `json:"planter_player_id"`
	Outcome           string  `json:"outcome"`
	ResolutionEventID *string `json:"resolution_event_id"`
	ResolutionTick    *int    `json:"resolution_tick"`
	ResolverPlayerID  *string `json:"resolver_player_id"`
	PlantAttempts     int     `json:"plant_attempts"`
	PlantAborts       int     `json:"plant_aborts"`
	DefuseAttempts    int     `json:"defuse_attempts"`
	DefuseAborts      int     `json:"defuse_aborts"`
	BombDrops         int     `json:"bomb_drops"`
	BombPickups       int     `json:"bomb_pickups"`
}

type CanonicalTeamScore struct {
	TeamID string `json:"team_id"`
	Score  int    `json:"score"`
}

type CanonicalSideAssignment struct {
	TeamID string `json:"team_id"`
	Side   string `json:"side"`
}

type CanonicalVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type CanonicalCombatEvent struct {
	SchemaID        string   `json:"schema_id"`
	MatchID         string   `json:"match_id"`
	EventID         string   `json:"event_id"`
	RoundID         string   `json:"round_id"`
	RoundNumber     int      `json:"round_number"`
	Tick            int      `json:"tick"`
	SequenceInTick  int      `json:"sequence_in_tick"`
	SequenceInRound int      `json:"sequence_in_round"`
	EventType       string   `json:"event_type"`
	Source          string   `json:"source"`
	SourceEventIDs  []string `json:"source_event_ids"`

	TickStatus        string   `json:"tick_status"`
	Subtick           *float64 `json:"subtick"`
	SubtickStatus     string   `json:"subtick_status"`
	TimeSeconds       *float64 `json:"time_seconds"`
	TimeSecondsStatus string   `json:"time_seconds_status"`
	TimeSecondsSource string   `json:"time_seconds_source"`

	ActorPlayerID    *string `json:"actor_player_id"`
	ActorSide        *string `json:"actor_side"`
	ActorStatus      string  `json:"actor_status"`
	ActorSource      string  `json:"actor_source"`
	TargetPlayerID   *string `json:"target_player_id"`
	TargetSide       *string `json:"target_side"`
	TargetStatus     string  `json:"target_status"`
	TargetSource     string  `json:"target_source"`
	AssisterPlayerID *string `json:"assister_player_id"`
	AssisterSide     *string `json:"assister_side"`
	AssisterStatus   string  `json:"assister_status"`
	AssisterSource   string  `json:"assister_source"`
	Relation         string  `json:"relation"`

	Weapon               *string          `json:"weapon"`
	WeaponStatus         string           `json:"weapon_status"`
	WeaponSource         string           `json:"weapon_source"`
	WeaponIsUtility      *bool            `json:"weapon_is_utility"`
	ActorPosition        *CanonicalVector `json:"actor_position"`
	ActorPositionStatus  string           `json:"actor_position_status"`
	ActorPositionSource  string           `json:"actor_position_source"`
	TargetPosition       *CanonicalVector `json:"target_position"`
	TargetPositionStatus string           `json:"target_position_status"`
	TargetPositionSource string           `json:"target_position_source"`

	ShotID                     *string  `json:"shot_id"`
	CorrelationStatus          string   `json:"correlation_status"`
	CorrelationSource          string   `json:"correlation_source"`
	ShotResult                 *string  `json:"shot_result"`
	ShotResultStatus           string   `json:"shot_result_status"`
	ShotResultSource           string   `json:"shot_result_source"`
	ShotResultAvailabilityTick *int     `json:"shot_result_availability_tick"`
	ViewYaw                    *float64 `json:"view_yaw"`
	ViewPitch                  *float64 `json:"view_pitch"`

	ImpactPosition           *CanonicalVector `json:"impact_position"`
	ImpactPositionStatus     string           `json:"impact_position_status"`
	ImpactPositionSource     string           `json:"impact_position_source"`
	BulletDistanceWorldUnits *float64         `json:"bullet_distance_world_units"`
	DamageDirection          *CanonicalVector `json:"damage_direction"`
	PenetratedObjects        *int             `json:"penetrated_objects"`
	NoScope                  *bool            `json:"no_scope"`
	AttackerInAir            *bool            `json:"attacker_in_air"`
	ThroughSmoke             *bool            `json:"through_smoke"`
	AttackerBlind            *bool            `json:"attacker_blind"`
	KillDistanceWorldUnits   *float64         `json:"kill_distance_world_units"`

	HealthDamage      *int    `json:"health_damage"`
	HealthDamageTaken *int    `json:"health_damage_taken"`
	ArmorDamage       *int    `json:"armor_damage"`
	ArmorDamageTaken  *int    `json:"armor_damage_taken"`
	HealthBefore      *int    `json:"health_before"`
	HealthAfter       *int    `json:"health_after"`
	ArmorBefore       *int    `json:"armor_before"`
	ArmorAfter        *int    `json:"armor_after"`
	DamageStatus      string  `json:"damage_status"`
	DamageSource      string  `json:"damage_source"`
	Hitgroup          *string `json:"hitgroup"`
	HitgroupStatus    string  `json:"hitgroup_status"`
	HitgroupSource    string  `json:"hitgroup_source"`
	IsHeadshot        *bool   `json:"is_headshot"`
	IsKill            bool    `json:"is_kill"`
	AssistedFlash     *bool   `json:"assisted_flash"`

	ReloadPhase          *string `json:"reload_phase"`
	ReloadEndTick        *int    `json:"reload_end_tick"`
	ReloadEndStatus      string  `json:"reload_end_status"`
	PreviousWeapon       *string `json:"previous_weapon"`
	PreviousWeaponStatus string  `json:"previous_weapon_status"`
	IsWeaponSwitch       *bool   `json:"is_weapon_switch"`
	AmmoInMagazine       *int    `json:"ammo_in_magazine"`
	AmmoReserve          *int    `json:"ammo_reserve"`
	AmmoStatus           string  `json:"ammo_status"`
	AmmoSource           string  `json:"ammo_source"`

	ReactionTimeMS *float64 `json:"reaction_time_ms"`
	TimeToDamageMS *float64 `json:"time_to_damage_ms"`
}

type CanonicalUtilityVectorObservation struct {
	Value  *CanonicalVector `json:"value"`
	Status string           `json:"status"`
	Source string           `json:"source"`
}

type CanonicalUtilityViewObservation struct {
	YawDeg   *float64         `json:"yaw_deg"`
	PitchDeg *float64         `json:"pitch_deg"`
	Vector   *CanonicalVector `json:"vector"`
	Status   string           `json:"status"`
	Source   string           `json:"source"`
}

type CanonicalUtilityVelocityObservation struct {
	Vector                   *CanonicalVector `json:"vector_world_units_per_second"`
	HorizontalWorldUnitsPerS *float64         `json:"horizontal_world_units_per_second"`
	ObservedTick             *int             `json:"observed_tick"`
	MeasurementWindowTicks   *int             `json:"measurement_window_ticks"`
	Status                   string           `json:"status"`
	Source                   string           `json:"source"`
}

type CanonicalUtilityStringObservation struct {
	Value  *string `json:"value"`
	Status string  `json:"status"`
	Source string  `json:"source"`
}

type CanonicalUtilityMomentObservation struct {
	Tick           *int             `json:"tick"`
	Position       *CanonicalVector `json:"position"`
	Status         string           `json:"status"`
	PositionStatus string           `json:"position_status"`
	Source         string           `json:"source"`
}

type CanonicalUtilityDurationObservation struct {
	Milliseconds *float64 `json:"milliseconds"`
	Status       string   `json:"status"`
	Source       string   `json:"source"`
}

type CanonicalUtilityLaunch struct {
	Tick                      *int                                `json:"tick"`
	TickStatus                string                              `json:"tick_status"`
	TickSource                string                              `json:"tick_source"`
	Position                  CanonicalUtilityVectorObservation   `json:"position"`
	View                      CanonicalUtilityViewObservation     `json:"view"`
	ThrowerVelocity           CanonicalUtilityVelocityObservation `json:"thrower_velocity"`
	ProjectileInitialVelocity CanonicalUtilityVelocityObservation `json:"projectile_initial_velocity"`
	Stance                    CanonicalUtilityStringObservation   `json:"stance"`
	Area                      CanonicalUtilityStringObservation   `json:"area"`
}

type CanonicalUtilityTrajectory struct {
	BounceCount  *int                                `json:"bounce_count"`
	BounceStatus string                              `json:"bounce_status"`
	BounceSource string                              `json:"bounce_source"`
	Samples      []CanonicalUtilityTrajectorySample  `json:"samples"`
	Bounces      []CanonicalUtilityBounceObservation `json:"bounces"`
	Status       string                              `json:"status"`
	Source       string                              `json:"source"`
}

type CanonicalUtilityTrajectorySample struct {
	Tick     int             `json:"tick"`
	Position CanonicalVector `json:"position"`
	Source   string          `json:"source"`
}

type CanonicalUtilityBounceObservation struct {
	Tick           int              `json:"tick"`
	Position       *CanonicalVector `json:"position"`
	PositionStatus string           `json:"position_status"`
	Number         int              `json:"number"`
	Source         string           `json:"source"`
}

type CanonicalUtilityLifecycle struct {
	Status                string                              `json:"status"`
	Detonation            CanonicalUtilityMomentObservation   `json:"detonation"`
	EffectStart           CanonicalUtilityMomentObservation   `json:"effect_start"`
	Expiration            CanonicalUtilityMomentObservation   `json:"expiration"`
	Destroy               CanonicalUtilityMomentObservation   `json:"destroy"`
	Extinguish            CanonicalUtilityMomentObservation   `json:"extinguish"`
	Area                  CanonicalUtilityStringObservation   `json:"area"`
	ExtinguishedByThrowID CanonicalUtilityStringObservation   `json:"extinguished_by_throw_id"`
	ExtinguishCorrelation CanonicalUtilityCorrelation         `json:"extinguish_correlation"`
	Duration              CanonicalUtilityDurationObservation `json:"duration"`
	EndReason             CanonicalUtilityStringObservation   `json:"end_reason"`
}

type CanonicalUtilityDamageEffect struct {
	Tick         int                         `json:"tick"`
	HealthDamage int                         `json:"health_damage"`
	ArmorDamage  int                         `json:"armor_damage"`
	IsKill       bool                        `json:"is_kill"`
	Source       string                      `json:"source"`
	Correlation  CanonicalUtilityCorrelation `json:"correlation"`
}

type CanonicalAffectedPlayer struct {
	PlayerID         *string                             `json:"player_id"`
	Side             *string                             `json:"side"`
	PlayerStatus     string                              `json:"player_status"`
	PlayerSource     string                              `json:"player_source"`
	Relation         string                              `json:"relation"`
	IsEnemy          bool                                `json:"is_enemy"`
	IsSelf           bool                                `json:"is_self"`
	BlindDuration    CanonicalUtilityDurationObservation `json:"blind_duration"`
	BlindCorrelation CanonicalUtilityCorrelation         `json:"blind_correlation"`
	Damage           *int                                `json:"damage"`
	ArmorDamage      *int                                `json:"armor_damage"`
	IsKill           *bool                               `json:"is_kill"`
	DamageEvents     []CanonicalUtilityDamageEffect      `json:"damage_events"`
}

type CanonicalUtilityDamageSummary struct {
	TotalDamage           int  `json:"total_damage"`
	EnemyDamage           int  `json:"enemy_damage"`
	TeammateDamage        int  `json:"teammate_damage"`
	SelfDamage            int  `json:"self_damage"`
	UnknownDamage         int  `json:"unknown_damage"`
	TotalArmorDamage      int  `json:"total_armor_damage"`
	EnemyArmorDamage      int  `json:"enemy_armor_damage"`
	TeammateArmorDamage   int  `json:"teammate_armor_damage"`
	SelfArmorDamage       int  `json:"self_armor_damage"`
	UnknownArmorDamage    int  `json:"unknown_armor_damage"`
	EnemiesDamaged        int  `json:"enemies_damaged"`
	TeammatesDamaged      int  `json:"teammates_damaged"`
	SelfDamaged           bool `json:"self_damaged"`
	UnknownPlayersDamaged int  `json:"unknown_players_damaged"`
	EnemyKills            int  `json:"enemy_kills"`
	TeammateKills         int  `json:"teammate_kills"`
	SelfKills             int  `json:"self_kills"`
	UnknownKills          int  `json:"unknown_kills"`
}

type CanonicalUtilityFlashSummary struct {
	PlayersTotal       int      `json:"players_total"`
	EnemiesFlashed     int      `json:"enemies_flashed"`
	TeammatesFlashed   int      `json:"teammates_flashed"`
	SelfFlashed        int      `json:"self_flashed"`
	UnknownFlashed     int      `json:"unknown_flashed"`
	TotalDurationMS    *float64 `json:"total_duration_ms"`
	EnemyDurationMS    *float64 `json:"enemy_duration_ms"`
	TeammateDurationMS *float64 `json:"teammate_duration_ms"`
	SelfDurationMS     *float64 `json:"self_duration_ms"`
	UnknownDurationMS  *float64 `json:"unknown_duration_ms"`
}

type CanonicalUtilityCorrelation struct {
	Status string `json:"status"`
	Source string `json:"source"`
}

type CanonicalUtilityEntityReference struct {
	RoundNumber int `json:"round_number"`
	EntityID    int `json:"entity_id"`
	Generation  int `json:"generation"`
}

type CanonicalUtilityEvent struct {
	SchemaID           string                           `json:"schema_id"`
	MatchID            string                           `json:"match_id"`
	EventID            string                           `json:"event_id"`
	SourceThrowID      string                           `json:"source_throw_id"`
	SourceEntity       *CanonicalUtilityEntityReference `json:"source_entity"`
	SourceEntityStatus string                           `json:"source_entity_status"`
	SourceEntitySource string                           `json:"source_entity_source"`
	RoundID            string                           `json:"round_id"`
	RoundNumber        int                              `json:"round_number"`
	SequenceInRound    int                              `json:"sequence_in_round"`
	EventType          string                           `json:"event_type"`
	UtilityType        string                           `json:"utility_type"`
	UtilityTypeStatus  string                           `json:"utility_type_status"`
	UtilityTypeSource  string                           `json:"utility_type_source"`
	ThrowerPlayerID    *string                          `json:"thrower_player_id"`
	ThrowerSide        *string                          `json:"thrower_side"`
	ThrowerStatus      string                           `json:"thrower_status"`
	ThrowerSource      string                           `json:"thrower_source"`
	Correlation        CanonicalUtilityCorrelation      `json:"correlation"`
	Launch             CanonicalUtilityLaunch           `json:"launch"`
	Trajectory         CanonicalUtilityTrajectory       `json:"trajectory"`
	Lifecycle          CanonicalUtilityLifecycle        `json:"lifecycle"`
	AffectedPlayers    []CanonicalAffectedPlayer        `json:"affected_players"`
	FlashSummary       CanonicalUtilityFlashSummary     `json:"flash_summary"`
	DamageSummary      CanonicalUtilityDamageSummary    `json:"damage_summary"`
	Details            AI_GrenadeEvent                  `json:"details"`
}

type CanonicalObjectiveEvent struct {
	SchemaID             string           `json:"schema_id"`
	MatchID              string           `json:"match_id"`
	EventID              string           `json:"event_id"`
	RoundID              string           `json:"round_id"`
	RoundNumber          int              `json:"round_number"`
	Tick                 int              `json:"tick"`
	SequenceInTick       int              `json:"sequence_in_tick"`
	EventType            string           `json:"event_type"`
	ActorPlayerID        *string          `json:"actor_player_id"`
	ActorSide            *string          `json:"actor_side"`
	Site                 *string          `json:"site"`
	Position             *CanonicalVector `json:"position"`
	PositionStatus       string           `json:"position_status"`
	Source               string           `json:"source"`
	StateAfter           string           `json:"state_after"`
	PhaseAfter           string           `json:"phase_after"`
	AttemptID            *string          `json:"attempt_id"`
	AttemptOutcome       *string          `json:"attempt_outcome"`
	AttemptStartObserved *bool            `json:"attempt_start_observed"`
	ActionDurationMS     *int64           `json:"action_duration_ms"`
	HasDefuseKit         *bool            `json:"has_defuse_kit"`
	BombEntityID         *int             `json:"bomb_entity_id"`
}

type CanonicalPlayerState struct {
	SchemaID                     string           `json:"schema_id"`
	MatchID                      string           `json:"match_id"`
	StateID                      string           `json:"state_id"`
	RoundID                      string           `json:"round_id"`
	RoundNumber                  int              `json:"round_number"`
	Tick                         int              `json:"tick"`
	PlayerID                     string           `json:"player_id"`
	TeamID                       string           `json:"team_id"`
	Side                         string           `json:"side"`
	Position                     CanonicalVector  `json:"position"`
	Area                         string           `json:"area"`
	ViewYawDeg                   float32          `json:"view_yaw_deg"`
	ViewPitchDeg                 float32          `json:"view_pitch_deg"`
	HorizontalVelocityWorldUPS   *float64         `json:"horizontal_velocity_world_units_per_second"`
	VelocityVectorWorldUPS       *CanonicalVector `json:"velocity_vector_world_units_per_second"`
	VelocitySource               string           `json:"velocity_source"`
	VelocityMeasurementTicks     *int             `json:"velocity_measurement_window_ticks"`
	Health                       int              `json:"health"`
	Armor                        int              `json:"armor"`
	IsAlive                      bool             `json:"is_alive"`
	ActiveWeapon                 *string          `json:"active_weapon"`
	ActiveWeaponStatus           string           `json:"active_weapon_status"`
	LastObservedActiveWeapon     *string          `json:"last_observed_active_weapon"`
	LastObservedActiveWeaponTick *int             `json:"last_observed_active_weapon_tick"`
	HasC4                        bool             `json:"has_c4"`
	HasDefuseKit                 bool             `json:"has_defuse_kit"`
	IsPlanting                   bool             `json:"is_planting"`
	IsDefusing                   bool             `json:"is_defusing"`
	IsWalking                    bool             `json:"is_walking"`
	IsDucking                    bool             `json:"is_ducking"`
	NearbyTeammates              int              `json:"nearby_teammates"`
	RoundTimeMS                  int64            `json:"round_time_remaining_ms"`
	ObjectivePhase               string           `json:"objective_phase"`
	PhaseTimeRemainingMS         *int64           `json:"phase_time_remaining_ms"`
	RoundClockRemainingMS        *int64           `json:"round_clock_remaining_ms"`
	BombTimeRemainingMS          *int64           `json:"bomb_time_remaining_ms"`
}

type CanonicalEngagementsExport struct {
	SchemaID    string                    `json:"schema_id"`
	MatchID     string                    `json:"match_id"`
	Config      CanonicalEngagementConfig `json:"config"`
	Engagements []CanonicalEngagement     `json:"engagements"`
}

type CanonicalEngagementConfig struct {
	AlgorithmVersion          string  `json:"algorithm_version"`
	TickRateHz                float64 `json:"tick_rate_hz"`
	PairContinuationWindowMS  int     `json:"pair_continuation_window_ms"`
	PairContinuationTicks     int     `json:"pair_continuation_window_ticks"`
	MultiTargetWindowMS       int     `json:"multi_target_window_ms"`
	MultiTargetWindowTicks    int     `json:"multi_target_window_ticks"`
	MaxEngagementDurationMS   int     `json:"max_engagement_duration_ms"`
	MaxEngagementDurationTick int     `json:"max_engagement_duration_ticks"`
	AggressorPreludeWindowMS  int     `json:"aggressor_prelude_window_ms"`
	AggressorPreludeTicks     int     `json:"aggressor_prelude_window_ticks"`
}

type CanonicalRoleAssignment struct {
	PlayerID         *string  `json:"player_id"`
	Status           string   `json:"status"`
	Source           string   `json:"source"`
	AvailabilityTick *int     `json:"availability_tick"`
	SourceEventIDs   []string `json:"source_event_ids"`
	Confidence       *float64 `json:"confidence"`
}

type CanonicalEngagementParticipant struct {
	PlayerID string   `json:"player_id"`
	Side     *string  `json:"side"`
	Roles    []string `json:"roles"`
}

type CanonicalEngagementExchange struct {
	ExchangeID                   string           `json:"exchange_id"`
	Tick                         int              `json:"tick"`
	SequenceInTick               int              `json:"sequence_in_tick"`
	SequenceInRound              int              `json:"sequence_in_round"`
	ActorPlayerID                string           `json:"actor_player_id"`
	TargetPlayerID               string           `json:"target_player_id"`
	Weapon                       *string          `json:"weapon"`
	WeaponStatus                 string           `json:"weapon_status"`
	WeaponSource                 string           `json:"weapon_source"`
	HealthDamage                 *int             `json:"health_damage"`
	HealthDamageTaken            *int             `json:"health_damage_taken"`
	ArmorDamage                  *int             `json:"armor_damage"`
	ArmorDamageTaken             *int             `json:"armor_damage_taken"`
	HealthBefore                 *int             `json:"health_before"`
	HealthAfter                  *int             `json:"health_after"`
	ArmorBefore                  *int             `json:"armor_before"`
	ArmorAfter                   *int             `json:"armor_after"`
	Hitgroup                     *string          `json:"hitgroup"`
	HitgroupStatus               string           `json:"hitgroup_status"`
	IsHeadshot                   *bool            `json:"is_headshot"`
	IsKill                       bool             `json:"is_kill"`
	KillEventID                  *string          `json:"kill_event_id"`
	ShotID                       *string          `json:"shot_id"`
	FirstImpactEventID           *string          `json:"first_impact_event_id"`
	ActorPosition                *CanonicalVector `json:"actor_position"`
	ActorPositionStatus          string           `json:"actor_position_status"`
	ActorPositionSource          string           `json:"actor_position_source"`
	TargetPosition               *CanonicalVector `json:"target_position"`
	TargetPositionStatus         string           `json:"target_position_status"`
	TargetPositionSource         string           `json:"target_position_source"`
	DistanceWorldUnits           *float64         `json:"distance_world_units"`
	DistanceStatus               string           `json:"distance_status"`
	DistanceSource               string           `json:"distance_source"`
	ReactionTimeMS               *float64         `json:"reaction_time_ms"`
	ReactionStatus               string           `json:"reaction_status"`
	ReactionSource               string           `json:"reaction_source"`
	ReactionAvailabilityTick     *int             `json:"reaction_availability_tick"`
	TimeToDamageMS               *float64         `json:"time_to_damage_ms"`
	TimeToDamageStatus           string           `json:"time_to_damage_status"`
	TimeToDamageSource           string           `json:"time_to_damage_source"`
	TimeToDamageAvailabilityTick *int             `json:"time_to_damage_availability_tick"`
	SourceEventIDs               []string         `json:"source_event_ids"`
}

type CanonicalEngagementParticipantState struct {
	PlayerID                   string           `json:"player_id"`
	StateID                    *string          `json:"state_id"`
	AvailabilityTick           *int             `json:"availability_tick"`
	Status                     string           `json:"status"`
	Source                     string           `json:"source"`
	Side                       *string          `json:"side"`
	Position                   *CanonicalVector `json:"position"`
	PositionStatus             string           `json:"position_status"`
	HorizontalVelocityWorldUPS *float64         `json:"horizontal_velocity_world_units_per_second"`
	VelocityStatus             string           `json:"velocity_status"`
	VelocitySource             string           `json:"velocity_source"`
	VelocityMeasurementTicks   *int             `json:"velocity_measurement_window_ticks"`
	MovementClassification     *string          `json:"movement_classification"`
	ActiveWeapon               *string          `json:"active_weapon"`
	ActiveWeaponStatus         string           `json:"active_weapon_status"`
	ActiveWeaponSource         string           `json:"active_weapon_source"`
	Health                     *int             `json:"health"`
	Armor                      *int             `json:"armor"`
	IsAlive                    *bool            `json:"is_alive"`
	ObjectivePhase             *string          `json:"objective_phase"`
	RoundClockRemainingMS      *int64           `json:"round_clock_remaining_ms"`
	BombTimeRemainingMS        *int64           `json:"bomb_time_remaining_ms"`
}

type CanonicalEngagementCausalContext struct {
	T0Tick                    int                                   `json:"t0_tick"`
	T0SequenceInTick          int                                   `json:"t0_sequence_in_tick"`
	ParticipantStates         []CanonicalEngagementParticipantState `json:"participant_states"`
	InitialDistanceWorldUnits *float64                              `json:"initial_distance_world_units"`
	InitialDistanceStatus     string                                `json:"initial_distance_status"`
	InitialDistanceSource     string                                `json:"initial_distance_source"`
	BombContextStatus         string                                `json:"bomb_context_status"`
	EconomyContextStatus      string                                `json:"economy_context_status"`
	EnemiesExposedCount       *int                                  `json:"enemies_exposed_count"`
	EnemiesExposedStatus      string                                `json:"enemies_exposed_status"`
	SourceStateIDs            []string                              `json:"source_state_ids"`
}

type CanonicalEngagementOutcomeContext struct {
	Outcome              string   `json:"outcome"`
	WinnerPlayerID       *string  `json:"winner_player_id"`
	LoserPlayerIDs       []string `json:"loser_player_ids"`
	TerminalKillEventIDs []string `json:"terminal_kill_event_ids"`
	TradeCandidateIDs    []string `json:"trade_candidate_ids"`
	TradeCompletionIDs   []string `json:"trade_completion_ids"`
	SurvivalStatus       string   `json:"survival_status"`
	DisengagementStatus  string   `json:"disengagement_status"`
}

type CanonicalEngagement struct {
	EngagementID        string                            `json:"engagement_id"`
	RoundID             string                            `json:"round_id"`
	RoundNumber         int                               `json:"round_number"`
	StartTick           int                               `json:"start_tick"`
	StartSequenceInTick int                               `json:"start_sequence_in_tick"`
	EndTick             int                               `json:"end_tick"`
	EndSequenceInTick   int                               `json:"end_sequence_in_tick"`
	DurationMS          float64                           `json:"duration_ms"`
	EngagementType      string                            `json:"engagement_type"`
	Initiator           CanonicalRoleAssignment           `json:"initiator"`
	FirstAggressor      CanonicalRoleAssignment           `json:"first_aggressor"`
	FirstDamageDealer   CanonicalRoleAssignment           `json:"first_damage_dealer"`
	Participants        []CanonicalEngagementParticipant  `json:"participants"`
	Exchanges           []CanonicalEngagementExchange     `json:"exchanges"`
	CausalContext       CanonicalEngagementCausalContext  `json:"causal_context"`
	OutcomeContext      CanonicalEngagementOutcomeContext `json:"outcome_context"`
	SourceEventIDs      []string                          `json:"source_event_ids"`
	AlgorithmVersion    string                            `json:"algorithm_version"`
	AttackerPlayerID    string                            `json:"-"`
	VictimPlayerIDs     []string                          `json:"-"`
	Outcome             string                            `json:"-"`
	Details             AI_Duel                           `json:"-"`
}

type CanonicalTradesExport struct {
	SchemaID    string                     `json:"schema_id"`
	MatchID     string                     `json:"match_id"`
	Config      CanonicalTradeConfig       `json:"config"`
	Candidates  []CanonicalTradeCandidate  `json:"trade_candidates"`
	Completions []CanonicalTradeCompletion `json:"trade_completions"`
}

type CanonicalTradeConfig struct {
	AlgorithmVersion             string  `json:"algorithm_version"`
	TickRateHz                   float64 `json:"tick_rate_hz"`
	TradeWindowMS                int     `json:"trade_window_ms"`
	TradeWindowTicks             int     `json:"trade_window_ticks"`
	MaxDistanceWorldUnits        float64 `json:"max_distance_world_units"`
	AssumedMovementSpeedWorldUPS float64 `json:"assumed_movement_speed_world_units_per_second"`
	MaxFacingDeltaDeg            float64 `json:"max_facing_delta_deg"`
	PhysicalEvidenceRequirement  string  `json:"physical_evidence_requirement"`
}

// CanonicalTradeConnection records only facts available at the original death
// tick. A nil value with status=unavailable is an abstention, never a zero.
type CanonicalTradeConnection struct {
	TeammatePlayerID      string   `json:"teammate_player_id"`
	PlayerIDUsage         string   `json:"player_id_usage"`
	PlayerStateID         *string  `json:"player_state_id"`
	StateAvailabilityTick *int     `json:"state_availability_tick"`
	StateStatus           string   `json:"state_status"`
	Alive                 *bool    `json:"alive"`
	AliveStatus           string   `json:"alive_status"`
	DistanceWorldUnits    *float64 `json:"distance_world_units"`
	DistanceStatus        string   `json:"distance_status"`
	ConnectionTimeMS      *float64 `json:"connection_time_ms"`
	ConnectionTimeStatus  string   `json:"connection_time_status"`
	LineOfSight           *bool    `json:"line_of_sight"`
	LineOfSightStatus     string   `json:"line_of_sight_status"`
	FacingDeltaDeg        *float64 `json:"facing_delta_deg"`
	FacingStatus          string   `json:"facing_status"`
	MapGeometryStatus     string   `json:"map_geometry_status"`
	Eligible              *bool    `json:"eligible"`
	EligibilityStatus     string   `json:"eligibility_status"`
	IneligibilityReasons  []string `json:"ineligibility_reasons"`
}

type CanonicalTradeCandidate struct {
	TradeCandidateID           string                     `json:"trade_candidate_id"`
	RoundID                    string                     `json:"round_id"`
	RoundNumber                int                        `json:"round_number"`
	DeathTick                  int                        `json:"death_tick"`
	DeathSequenceInTick        int                        `json:"death_sequence_in_tick"`
	OriginalKillEventID        string                     `json:"original_kill_event_id"`
	OriginalVictimPlayerID     string                     `json:"original_victim_player_id"`
	OriginalKillerPlayerID     string                     `json:"original_killer_player_id"`
	PlayerIDUsage              string                     `json:"player_id_usage"`
	EligibleTeammatePlayerIDs  []string                   `json:"eligible_teammate_player_ids"`
	EligibilityStatus          string                     `json:"eligibility_status"`
	EligibilitySource          string                     `json:"eligibility_source"`
	EligibilityStateIDs        []string                   `json:"eligibility_state_ids"`
	Connections                []CanonicalTradeConnection `json:"connections"`
	TradePossible              *bool                      `json:"trade_possible"`
	TradePossibleStatus        string                     `json:"trade_possible_status"`
	AttemptEventIDs            []string                   `json:"attempt_event_ids"`
	Evaluation                 string                     `json:"evaluation"`
	TradeCompletionID          *string                    `json:"trade_completion_id"`
	CounterTradeOfCompletionID *string                    `json:"counter_trade_of_completion_id"`
	WindowMS                   int                        `json:"window_ms"`
	WindowTicks                int                        `json:"window_ticks"`
	WindowEndTick              int                        `json:"window_end_tick"`
	SourceEventIDs             []string                   `json:"source_event_ids"`
}

type CanonicalTradeCompletion struct {
	TradeCompletionID      string   `json:"trade_completion_id"`
	TradeCandidateID       string   `json:"trade_candidate_id"`
	RoundID                string   `json:"round_id"`
	RoundNumber            int      `json:"round_number"`
	OriginalKillEventID    string   `json:"original_kill_event_id"`
	ResponseKillEventID    string   `json:"response_kill_event_id"`
	OriginalVictimPlayerID string   `json:"original_victim_player_id"`
	OriginalKillerPlayerID string   `json:"original_killer_player_id"`
	TraderPlayerID         string   `json:"trader_player_id"`
	TradeRelation          string   `json:"trade_relation"`
	ElapsedTicks           int      `json:"elapsed_ticks"`
	ElapsedMS              float64  `json:"elapsed_ms"`
	SourceEventIDs         []string `json:"source_event_ids"`
}

type CanonicalPlayerRoundEconomyExport struct {
	SchemaID string                         `json:"schema_id"`
	MatchID  string                         `json:"match_id"`
	Rounds   []CanonicalEconomyRoundContext `json:"rounds"`
	Records  []CanonicalPlayerRoundEconomy  `json:"records"`
}

type CanonicalEconomyRoundContext struct {
	RoundID     string                    `json:"round_id"`
	RoundNumber int                       `json:"round_number"`
	Teams       map[string]AI_EconomyTeam `json:"teams"`
	Events      *AI_EconomyRoundEvents    `json:"events,omitempty"`
}

type CanonicalPlayerRoundEconomy struct {
	RoundID                 string           `json:"round_id"`
	RoundNumber             int              `json:"round_number"`
	PlayerID                string           `json:"player_id"`
	TeamID                  string           `json:"team_id"`
	Side                    string           `json:"side"`
	InitialMoney            int              `json:"initial_money"`
	MoneyAfterBuy           int              `json:"money_after_buy"`
	SpentInBuy              int              `json:"spent_in_buy"`
	EquipmentValueStart     int              `json:"equipment_value_start_calculated"`
	EquipmentValueFreezeEnd int              `json:"equipment_value_freeze_end_calculated"`
	MoneyAtRoundEnd         int              `json:"money_at_round_end"`
	EquipmentValueRoundEnd  int              `json:"equipment_value_round_end_calculated"`
	Survived                bool             `json:"survived"`
	Outcome                 string           `json:"outcome"`
	Source                  string           `json:"source"`
	Details                 AI_EconomyPlayer `json:"details"`
}

type CanonicalPlayerMatchStatsExport struct {
	SchemaID string                      `json:"schema_id"`
	MatchID  string                      `json:"match_id"`
	Players  []CanonicalPlayerMatchStats `json:"players"`
}

type CanonicalPlayerMatchStats struct {
	PlayerID string         `json:"player_id"`
	Source   string         `json:"source"`
	Metrics  AI_PlayerStats `json:"metrics"`
}

type CanonicalQualityReportExport struct {
	SchemaID string      `json:"schema_id"`
	MatchID  string      `json:"match_id"`
	Report   interface{} `json:"report"`
}
