package combat

type Availability string

const (
	AvailabilityObserved    Availability = "observed"
	AvailabilityDerived     Availability = "derived"
	AvailabilityUnavailable Availability = "unavailable"
)

const (
	DiscardWarmup             = "warmup"
	DiscardOutsideRound       = "outside_official_round"
	DiscardInvalidObservation = "invalid_round_or_tick"
)

const (
	SourceUnavailable       = "unavailable"
	SourceWeaponFire        = "demoinfocs.weapon_fire"
	SourceBulletDamage      = "demoinfocs.bullet_damage"
	SourcePlayerHurt        = "demoinfocs.player_hurt"
	SourceKill              = "demoinfocs.kill"
	SourceWeaponReload      = "demoinfocs.weapon_reload"
	SourceItemEquip         = "demoinfocs.item_equip"
	SourceCallbackPlayer    = "callback_player"
	SourceCallbackWeapon    = "callback_weapon"
	SourceCallbackPosition  = "callback_player_position"
	SourceActiveWeaponAmmo  = "active_weapon_snapshot"
	SourceDeterministicSort = "deterministic_total_order"
	SourceFireCorrelation   = "actor_weapon_prior_fire"
	SourceBulletCorrelation = "same_tick_actor_target"
	SourceFatalHurt         = "same_tick_fatal_hurt"
	SourceRoundClosure      = "round_end_observed"
)

type EventType string

const (
	EventWeaponEquip  EventType = "weapon_equip"
	EventWeaponReload EventType = "weapon_reload"
	EventWeaponFire   EventType = "weapon_fire"
	EventBulletDamage EventType = "bullet_damage"
	EventPlayerHurt   EventType = "player_hurt"
	EventKill         EventType = "kill"
)

type Relation string

const (
	RelationEnemy    Relation = "enemy"
	RelationFriendly Relation = "friendly"
	RelationSelf     Relation = "self"
	RelationWorld    Relation = "world"
	RelationUnknown  Relation = "unknown"
)

type CorrelationStatus string

const (
	CorrelationExact       CorrelationStatus = "exact"
	CorrelationInferred    CorrelationStatus = "inferred"
	CorrelationUnavailable CorrelationStatus = "unavailable"
)

type ShotResult string

const (
	ShotHit         ShotResult = "hit"
	ShotMiss        ShotResult = "miss"
	ShotUnavailable ShotResult = "unavailable"
)

type Vector struct {
	X float64
	Y float64
	Z float64
}

type PlayerRef struct {
	ID             uint64
	Name           string
	Side           string
	Status         Availability
	Source         string
	Position       *Vector
	PositionStatus Availability
	PositionSource string
}

type WeaponRef struct {
	Name      string
	Status    Availability
	Source    string
	IsUtility *bool
}

type AmmoObservation struct {
	InMagazine *int
	Reserve    *int
	Status     Availability
	Source     string
}

type Event struct {
	LocalID         string
	Round           int
	Tick            int
	SequenceInTick  int
	SequenceInRound int
	Type            EventType
	Source          string
	SourceEventIDs  []string

	Actor    PlayerRef
	Target   PlayerRef
	Assister PlayerRef
	Relation Relation
	Weapon   WeaponRef

	ShotID                     string
	CorrelationStatus          CorrelationStatus
	CorrelationSource          string
	ShotResult                 ShotResult
	ShotResultStatus           Availability
	ShotResultSource           string
	ShotResultAvailabilityTick *int

	ViewYaw   *float64
	ViewPitch *float64
	Ammo      AmmoObservation

	HealthDamage      *int
	HealthDamageTaken *int
	ArmorDamage       *int
	ArmorDamageTaken  *int
	HealthBefore      *int
	HealthAfter       *int
	ArmorBefore       *int
	ArmorAfter        *int
	DamageStatus      Availability
	DamageSource      string
	Hitgroup          *string
	HitgroupStatus    Availability
	HitgroupSource    string
	IsHeadshot        *bool
	IsKill            bool

	AssistedFlash *bool

	ImpactPosition       *Vector
	ImpactPositionStatus Availability
	ImpactPositionSource string
	BulletDistance       *float64
	DamageDirection      *Vector
	PenetratedObjects    *int
	NoScope              *bool
	AttackerInAir        *bool
	ThroughSmoke         *bool
	AttackerBlind        *bool
	KillDistance         *float64

	ReloadPhase          *string
	ReloadEndTick        *int
	ReloadEndStatus      Availability
	PreviousWeapon       *string
	PreviousWeaponStatus Availability
	IsWeaponSwitch       *bool
}

type FireInput struct {
	Round     int
	Tick      int
	Actor     PlayerRef
	Weapon    WeaponRef
	ViewYaw   *float64
	ViewPitch *float64
	Ammo      AmmoObservation
}

type BulletDamageInput struct {
	Round             int
	Tick              int
	Actor             PlayerRef
	Target            PlayerRef
	Distance          float64
	Direction         Vector
	PenetratedObjects int
	NoScope           bool
	AttackerInAir     bool
}

type HurtInput struct {
	Round             int
	Tick              int
	Actor             PlayerRef
	Target            PlayerRef
	Weapon            WeaponRef
	HealthDamage      int
	HealthDamageTaken int
	ArmorDamage       int
	ArmorDamageTaken  int
	HealthAfter       int
	ArmorAfter        int
	Hitgroup          string
}

type KillInput struct {
	Round             int
	Tick              int
	Actor             PlayerRef
	Target            PlayerRef
	Assister          PlayerRef
	Weapon            WeaponRef
	IsHeadshot        bool
	AssistedFlash     bool
	PenetratedObjects int
	NoScope           bool
	ThroughSmoke      bool
	AttackerBlind     bool
	Distance          float64
}

type ReloadInput struct {
	Round  int
	Tick   int
	Actor  PlayerRef
	Weapon WeaponRef
	Ammo   AmmoObservation
}

type EquipInput struct {
	Round  int
	Tick   int
	Actor  PlayerRef
	Weapon WeaponRef
	Ammo   AmmoObservation
}

type Diagnostics struct {
	ObservedByType    map[EventType]int
	RecordedByType    map[EventType]int
	DiscardedByType   map[EventType]int
	DiscardedByReason map[string]int
	InvalidLinks      int
	FutureLinks       int
	DuplicateLocalIDs int
	DuplicateShotIDs  int
}

type WeaponSummary struct {
	Kills       int
	Headshots   int
	Damage      int
	ShotsFired  int
	ShotsHit    int
	ShotsMissed int
}

type PlayerSummary struct {
	Kills          int
	Deaths         int
	Assists        int
	FlashAssists   int
	Headshots      int
	EnemyDamage    int
	FriendlyDamage int
	SelfDamage     int
	ShotsFired     int
	ShotsHit       int
	ShotsMissed    int
	BodyPartHits   map[string]int
	WeaponStats    map[string]WeaponSummary
}
