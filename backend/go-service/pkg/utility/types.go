package utility

type Type string

const (
	TypeFlashbang  Type = "flashbang"
	TypeSmoke      Type = "smoke"
	TypeHE         Type = "he"
	TypeMolotov    Type = "molotov"
	TypeIncendiary Type = "incendiary"
	TypeDecoy      Type = "decoy"
	TypeUnknown    Type = "unknown"
)

const (
	InitialVelocityReasonNativeUnavailable         = "native_velocity_property_unavailable"
	InitialVelocityReasonNativeNonFinite           = "native_velocity_non_finite"
	InitialVelocityReasonNativeSentinel            = "native_velocity_sentinel"
	InitialVelocityReasonNativeZeroAmbiguous       = "native_zero_ambiguous"
	InitialVelocityReasonLaunchPositionUnavailable = "launch_position_unavailable"
	InitialVelocityReasonTickRateUnavailable       = "tick_rate_unavailable"
	InitialVelocityReasonInsufficientSamples       = "insufficient_position_samples"
	InitialVelocityReasonSameTickConflict          = "same_tick_position_conflict"
	InitialVelocityReasonFlightEnded               = "flight_ended_before_second_sample"
	InitialVelocityReasonIdentityUnavailable       = "projectile_observation_identity_unavailable"
	InitialVelocityReasonIdentityConflict          = "projectile_observation_identity_conflict"
	InitialVelocityReasonEntityReused              = "projectile_entity_reused"
)

type TypeFamily string

const (
	TypeFamilyUnknown TypeFamily = "unknown"
	TypeFamilyFire    TypeFamily = "fire"
)

type Availability string

const (
	AvailabilityObserved      Availability = "observed"
	AvailabilityUnavailable   Availability = "unavailable"
	AvailabilityNotApplicable Availability = "not_applicable"
)

type TrajectoryStatus string

const (
	TrajectoryObserved    TrajectoryStatus = "observed"
	TrajectoryPartial     TrajectoryStatus = "partial"
	TrajectoryUnavailable TrajectoryStatus = "unavailable"
)

const (
	SourceUnavailable              = "unavailable"
	SourceWeaponInstance           = "weapon_instance"
	SourceCallbackType             = "callback_type"
	SourceProjectileEntity         = "projectile_entity"
	SourceGrenadeEntityID          = "grenade_entity_id"
	SourceEffectEntityID           = "effect_entity_id"
	SourceProjectileThrow          = "projectile_throw"
	SourceProjectileThrower        = "projectile_thrower"
	SourceProjectileOwner          = "projectile_owner"
	SourceProjectilePosition       = "projectile_position"
	SourceProjectileVelocity       = "projectile_velocity"
	SourceProjectilePositionDelta  = "projectile_position_delta"
	SourcePlayerView               = "player_view"
	SourcePlayerState              = "player_state"
	SourceVelocityNative           = "native"
	SourceVelocityPositionDelta    = "position_delta"
	SourceVelocityNoHistory        = "insufficient_history"
	SourceVelocityRejected         = "position_delta_rejected"
	SourceVelocityStaleGap         = "stale_gap"
	SourceVelocityEntityChanged    = "entity_changed"
	SourceVelocityNonMonotonicTick = "non_monotonic_tick"
	SourceVelocityNotApplicable    = "not_applicable"
	SourceMapCallout               = "map_callout"
	SourcePlayerLastPlace          = "player_last_place"
	SourceProjectileFrames         = "projectile_frames"
	SourceProjectileBounce         = "projectile_bounce"
	SourceProjectileDestroy        = "projectile_destroy"
	SourceFlashExplode             = "flash_explode"
	SourceHEExplode                = "he_explode"
	SourceSmokeStart               = "smoke_start"
	SourceSmokeExpired             = "smoke_expired"
	SourceInfernoStart             = "inferno_start"
	SourceInfernoExpired           = "inferno_expired"
	SourceDecoyStart               = "decoy_start"
	SourceDecoyExpired             = "decoy_expired"
	SourcePlayerFlashed            = "player_flashed"
	SourcePlayerHurt               = "player_hurt"
	SourceCallbackActor            = "callback_actor"
	SourceCallbackTicks            = "callback_ticks"
	SourceExpirationCallback       = "expiration_callback"
	SourceSpatialSmokeOverlap      = "spatial_smoke_overlap"
	SourceRoundBoundary            = "round_boundary"
	SourceThrowerTypePositionTick  = "thrower_type_position_tick"
	SourceTypePositionTick         = "type_position_tick"
)

type Relation string

const (
	RelationSelf     Relation = "self"
	RelationTeammate Relation = "teammate"
	RelationEnemy    Relation = "enemy"
	RelationUnknown  Relation = "unknown"
)

type Stance string

const (
	StanceStanding      Stance = "standing"
	StanceWalking       Stance = "walking"
	StanceCrouching     Stance = "crouching"
	StanceCrouchWalking Stance = "crouch_walking"
	StanceAirborne      Stance = "airborne"
	StanceUnknown       Stance = "unknown"
)

type LifecycleStatus string

const (
	LifecycleThrown                     LifecycleStatus = "thrown"
	LifecycleDetonated                  LifecycleStatus = "detonated"
	LifecycleEffectActive               LifecycleStatus = "effect_active"
	LifecycleEffectExpired              LifecycleStatus = "effect_expired"
	LifecycleDestroyedWithoutDetonation LifecycleStatus = "destroyed_without_detonation"
	LifecycleRoundEndedUnresolved       LifecycleStatus = "round_ended_unresolved"
)

type EndReason string

const (
	EndReasonExpired           EndReason = "expired"
	EndReasonSmokeExtinguished EndReason = "smoke_extinguished"
	EndReasonDestroyed         EndReason = "destroyed"
	EndReasonRoundEnd          EndReason = "round_end"
	EndReasonUnavailable       EndReason = "unavailable"
)

type CorrelationStatus string

const (
	CorrelationObserved    CorrelationStatus = "observed"
	CorrelationInferred    CorrelationStatus = "inferred"
	CorrelationUnavailable CorrelationStatus = "unavailable"
)

type Vector struct {
	X float64
	Y float64
	Z float64
}

type PlayerRef struct {
	ID     uint64
	Name   string
	Side   string
	Status Availability
	Source string
}

type TickObservation struct {
	Tick   int
	Status Availability
	Source string
}

type VectorObservation struct {
	Value  Vector
	Status Availability
	Source string
}

type ScalarObservation struct {
	Value  float64
	Status Availability
	Source string
}

type StringObservation struct {
	Value  string
	Status Availability
	Source string
}

type StanceObservation struct {
	Value  Stance
	Status Availability
	Source string
}

type ViewObservation struct {
	Yaw    float64
	Pitch  float64
	Vector Vector
	Status Availability
	Source string
}

type VelocityObservation struct {
	Vector                 Vector
	HorizontalSpeed        float64
	ObservedTick           int
	MeasurementWindowTicks int
	Status                 Availability
	Source                 string
}

// InitialVelocityCandidate is populated only from information available while
// parsing the launch. It is deliberately separate from the canonical launch
// observation so a candidate contract can evolve without reinterpreting 3.8.
type InitialVelocityCandidate struct {
	Vector                 Vector
	HorizontalSpeed        float64
	ObservedTick           int
	AvailabilityTick       int
	MeasurementStartTick   int
	MeasurementEndTick     int
	MeasurementWindowTicks int
	TickRate               float64
	StartPosition          Vector
	EndPosition            Vector
	Status                 Availability
	Source                 string
	Reason                 string
	Terminal               bool
	IdentityStatus         Availability
	RuntimeEntityID        int
	RuntimeEntitySerial    int
	ActorID                uint64
	Type                   Type
	PositionSampleTick     int
	PositionSample         Vector
	PositionSampleObserved bool
}

type ThrowSnapshot struct {
	Tick                      TickObservation
	Position                  VectorObservation
	View                      ViewObservation
	ThrowerVelocity           VelocityObservation
	ProjectileInitialVelocity VelocityObservation
	Stance                    StanceObservation
	Area                      StringObservation
}

type Trajectory struct {
	Samples      []TrajectorySample
	Bounces      []BounceObservation
	Status       TrajectoryStatus
	Source       string
	BounceCount  int
	BounceStatus Availability
	BounceSource string
}

type TrajectorySample struct {
	Tick     int
	Position Vector
	Source   string
}

type BounceObservation struct {
	Tick           int
	Position       Vector
	PositionStatus Availability
	Number         int
	Source         string
}

type TickPositionObservation struct {
	Tick           int
	Position       Vector
	Status         Availability
	PositionStatus Availability
	Source         string
}

type Correlation struct {
	Status CorrelationStatus
	Source string
}

type Lifecycle struct {
	Status                LifecycleStatus
	Detonation            TickPositionObservation
	EffectStart           TickPositionObservation
	Expiration            TickPositionObservation
	Destroy               TickPositionObservation
	Extinguish            TickPositionObservation
	Duration              ScalarObservation
	Area                  StringObservation
	EndReason             EndReason
	EndReasonSource       string
	Correlation           Correlation
	ExtinguishedByThrowID StringObservation
	ExtinguishAttribution Correlation
}

type FlashEffect struct {
	Victim      PlayerRef
	Relation    Relation
	Duration    ScalarObservation
	Tick        int
	Source      string
	Correlation Correlation
}

type DamageEffect struct {
	Victim       PlayerRef
	Relation     Relation
	HealthDamage int
	ArmorDamage  int
	Kill         bool
	Tick         int
	Source       string
	Correlation  Correlation
}

type Throw struct {
	ID                       string
	Round                    int
	Sequence                 int
	SourceEntityID           int
	SourceEntityGeneration   int
	SourceEntitySerial       int
	SourceEntitySerialStatus Availability
	EntityStatus             Availability
	EntitySource             string
	Type                     Type
	TypeSource               string
	Actor                    PlayerRef
	Launch                   ThrowSnapshot
	InitialVelocityCandidate InitialVelocityCandidate
	Trajectory               Trajectory
	Lifecycle                Lifecycle
	Flashes                  []FlashEffect
	Damage                   []DamageEffect
}

type ThrowInput struct {
	Round                     int
	RuntimeEntityID           int
	RuntimeEntitySerial       int
	RuntimeEntitySerialStatus Availability
	EntitySource              string
	Type                      Type
	TypeSource                string
	Actor                     PlayerRef
	Launch                    ThrowSnapshot
}

type InitialVelocityPositionInput struct {
	Round                     int
	RuntimeEntityID           int
	RuntimeEntitySerial       int
	RuntimeEntitySerialStatus Availability
	ActorID                   uint64
	Type                      Type
	Tick                      int
	Position                  Vector
	TickRate                  float64
}

type CallbackHint struct {
	Round                     int
	RuntimeEntityID           int
	RuntimeEntitySerial       int
	RuntimeEntitySerialStatus Availability
	EntitySource              string
	Type                      Type
	TypeFamily                TypeFamily
	ActorID                   uint64
	Actor                     PlayerRef
	Tick                      int
	TickRate                  float64
	Position                  Vector
	PositionStatus            Availability
	Area                      StringObservation
}

type EffectInput struct {
	Hint                  CallbackHint
	RuntimeEffectEntityID int
	Source                string
}

type FlashInput struct {
	Round           int
	RuntimeEntityID int
	ActorID         uint64
	Actor           PlayerRef
	Tick            int
	TickRate        float64
	Victim          PlayerRef
	Relation        Relation
	Duration        ScalarObservation
}

type DamageInput struct {
	Round                int
	Type                 Type
	ActorID              uint64
	Actor                PlayerRef
	Tick                 int
	TickRate             float64
	Victim               PlayerRef
	VictimPosition       Vector
	VictimPositionStatus Availability
	Relation             Relation
	HealthDamage         int
	ArmorDamage          int
	Kill                 bool
}

type CallbackDiagnostics struct {
	Observed           int
	ExactCorrelated    int
	InferredCorrelated int
	Orphaned           int
	Deduplicated       int
	Unmatched          int
}

type CallbackGroup string

const (
	CallbackGroupThrows        CallbackGroup = "throws"
	CallbackGroupBounces       CallbackGroup = "bounces"
	CallbackGroupLifecycle     CallbackGroup = "lifecycle"
	CallbackGroupPlayerFlashed CallbackGroup = "player_flashed"
	CallbackGroupDamage        CallbackGroup = "damage"
)

type CallbackDiscardReason string

const (
	CallbackDiscardWarmup       CallbackDiscardReason = "warmup"
	CallbackDiscardOutsideRound CallbackDiscardReason = "outside_round"
	CallbackDiscardInvalid      CallbackDiscardReason = "invalid_payload"
)

type CallbackDiscardDiagnostics struct {
	Warmup       int
	OutsideRound int
	Invalid      int
}

type DiscardedCallbackDiagnostics struct {
	Throws        CallbackDiscardDiagnostics
	Bounces       CallbackDiscardDiagnostics
	Lifecycle     CallbackDiscardDiagnostics
	PlayerFlashed CallbackDiscardDiagnostics
	Damage        CallbackDiscardDiagnostics
}

type Diagnostics struct {
	Throws         CallbackDiagnostics
	Bounces        CallbackDiagnostics
	Lifecycle      CallbackDiagnostics
	Flashes        CallbackDiagnostics
	Damage         CallbackDiagnostics
	Discarded      DiscardedCallbackDiagnostics
	ActorEnriched  int
	ActorConflicts int
}
