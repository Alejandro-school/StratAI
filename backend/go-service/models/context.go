package models

import (
	"cs2-demo-service/pkg/combat"
	"cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/objective"
	"cs2-demo-service/pkg/playerstate"
	"cs2-demo-service/pkg/utility"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

// DemoContext mantiene el estado mínimo necesario durante el parsing
type DemoContext struct {
	Parser dem.Parser

	// Map Manager for visibility checks
	MapManager maps.VisibilityChecker
	MapName    string

	// Output final
	MatchData *MatchData

	// Timeline tracking - NEW for coaching system
	Timeline           []TimelineEvent // Todos los eventos en orden cronológico
	CurrentRoundEvents []TimelineEvent // Eventos de la ronda actual
	RoundTimelines     []RoundTimeline // Timelines por ronda (para export)
	LastGameStateTick  int             // Último tick donde se sampleo game state

	// Estado de ronda actual
	CurrentRound      int // Contador de rondas procesadas
	ActualRoundNumber int // Número de ronda ACTUAL para eventos (fijo durante toda la ronda)
	InRound           bool

	// Loss Bonus Tracking
	CTConsecutiveLosses int
	TConsecutiveLosses  int

	// Round Win Tracking
	CTRoundsWon int
	TRoundsWon  int

	// Single source of truth for the C4 lifecycle and objective ledger.
	Objectives *objective.Tracker

	// Single source of truth for utility throws, effects, attribution and damage.
	Utilities *utility.Tracker

	// Single source of truth for combat callbacks and causal correlations.
	Combat *combat.Tracker

	// Buy tracking (NEW)
	FreezeTimeEnded   bool
	FreezeTimeEndTick int            // Tick when freeze time ended (round action start)
	PlayerMoneyBefore map[uint64]int // SteamID -> dinero antes de compra

	// SOLO tracking custom necesario (no redundante con demoinfocs)

	// Para calcular distancia acumulada (demoinfocs no lo hace)
	LastPositions map[uint64]r3.Vector
	LastTick      int // Para movement sampling

	// Spray detection
	CurrentSpray       map[uint64]*SprayData
	LastWeaponFireTick map[uint64]int
	LastCombatFireTick map[uint64]int

	// Reaction time tracking
	EnemyFirstSeenTick               map[uint64]map[uint64]FirstSeenData
	FirstDamageTick                  map[uint64]map[uint64]int // Attacker -> Victim -> Tick of first damage
	LastVisibleEnemies               map[uint64]map[uint64]bool
	ActiveSmokes                     map[int64]r3.Vector
	VisibilitySampledTicks           int
	VisibilitySkippedTicks           int
	VisibilityRaycasts               int
	VisibilityRefinementRaycasts     int
	NativeEyePositions               int
	FallbackEyePositions             int
	MapLoadError                     string
	ObjectiveNativeRoleDisagreements int

	// Mechanics tracking per shot
	LastShotMechanics map[uint64]*ShotMechanics

	// Crosshair placement tracking
	CrosshairStats map[uint64]*CrosshairStats

	// Grenade Trajectories tracking (ProjectileID -> Event)
	ActiveGrenadeTrajectories    map[int]*GrenadeTrajectoryEvent
	CompletedGrenadeTrajectories map[int]*GrenadeTrajectoryEvent // Buffer for recently destroyed grenades
	GrenadeBounces               map[int]int                     // ProjectileID -> Bounce Count

	// Economy Tracking
	RoundPurchases map[uint64][]AI_WeaponItem // SteamID -> List of items purchased in current round

	// AI Data Collection
	AI_TrackingEvents          []AI_TrackingEvent
	AI_TrackingEventsWithRound []AI_TrackingEventWithRound // Events with round info for grouping
	LastTrackingTick           int
	AI_CombatDuels             []AI_CombatDuel
	PendingCombatEvents        []AI_CombatDuel // Buffer for non-fatal events
	AI_EconomyRounds           []AI_EconomyRound
	AI_GrenadeEvents           []AI_GrenadeEvent // Derived compatibility projection of Utilities.

	// Economy Drop Tracking
	PendingDrops        []AI_EconomyDrop    // Drops waiting to be matched with pickups
	RoundDrops          []AI_EconomyDrop    // All drops in current round (matched)
	RoundPickups        []AI_EconomyPickup  // Non-purchase pickups in current round
	RoundRefunds        []AI_EconomyRefund  // Refunds in current round
	PlayerDropsGiven    map[uint64][]string // SteamID -> weapons dropped
	PlayerDropsReceived map[uint64][]string // SteamID -> weapons received
	PlayerRefunds       map[uint64][]string // SteamID -> items refunded

	// Entity-based weapon tracking (UniqueID -> Original Owner SteamID)
	WeaponOriginalOwner map[int64]uint64 // Weapon UniqueID -> Original Owner SteamID at round start

	// NEW: Consolidated Duel System
	AI_Duels               []AI_Duel        // Final consolidated duels
	AI_PlayersSummary      []AI_PlayerStats // Comprehensive player stats
	RawCombatEvents        []RawCombatEvent // Buffer for raw events to consolidate at round end
	AI_CombatEvents        []RawCombatEvent // Persistent atomic combat events for canonical export
	PendingBulletDamage    []BulletDamageSnapshot
	BulletDamageEvents     int
	BulletDamageCorrelated int

	// PHASE 1: Weapon state tracking per player (for combat integration)
	LastWeaponState     map[uint64]*WeaponStateSnapshot // Current tick weapon state
	PreviousWeaponState map[uint64]*WeaponStateSnapshot // Previous tick weapon state (true BEFORE state)
	SprayStartState     map[uint64]*WeaponStateSnapshot // State at the start of the current firing sequence

	// Shared causal motion estimates, keyed by player, round, and tick.
	PlayerMotion playerstate.MotionTracker

	// Damage tracking for AI Duels
	RoundDamage map[uint64]map[uint64]int // AttackerID -> VictimID -> TotalDamage

	// FIX 5: Monotonic counter for unique event IDs
	CombatEventCounter int

	// FIX Bug 3: Track player HP before damage events
	// This allows showing victim's actual HP before they took damage
	LastKnownHealth map[uint64]int // SteamID -> HP before current damage

	// Last directly observed active weapon, retained separately from current state.
	LastActiveWeapon map[uint64]ActiveWeaponObservation

	// 2D Replay Data (for frontend visualization)
	ReplayData     *ReplayData
	ParseCompleted bool
	ParserWarnings []string
}

// FirstSeenData stores metadata when an enemy is first seen
type FirstSeenData struct {
	Tick                     int
	LastSeenTick             int // Último tick donde fue visible (para jiggle peek grace period)
	FirstShotTick            int
	FirstDamageTick          int
	CrosshairPlacementError  float64
	PitchError               float64
	YawError                 float64
	ShooterVelocity          float64 // Velocity at first sight (u/s) for peek/hold classification
	ShooterVelocityAvailable bool
}

type ActiveWeaponObservation struct {
	Weapon      string
	Tick        int
	RoundNumber int
}

const (
	ActiveWeaponStatusObserved      = "observed"
	ActiveWeaponStatusUnavailable   = "unavailable"
	ActiveWeaponStatusNotApplicable = "not_applicable"
)

// NewDemoContext crea un nuevo contexto inicializado
func NewDemoContext(p dem.Parser) *DemoContext {
	return &DemoContext{
		Parser: p,
		MatchData: &MatchData{
			Players:     make(map[uint64]*PlayerData),
			PlayerStats: []PlayerStats{},
			Rounds:      []RoundData{},
			Economy:     []RoundEconomyStats{},
			Kills:       []KillEvent{},
			Damage:      []DamageEvent{},
			Flashes:     []FlashEvent{},
			HEGrenades:  []HEEvent{},
			Smokes:      []SmokeEvent{},
			Molotovs:    []MolotovEvent{},
			BombEvents:  []BombEvent{},
		},
		Objectives:                   objective.NewTracker(),
		Utilities:                    utility.NewTracker(),
		Combat:                       combat.NewTracker(),
		Timeline:                     []TimelineEvent{},
		CurrentRoundEvents:           []TimelineEvent{},
		RoundTimelines:               []RoundTimeline{},
		AI_TrackingEvents:            []AI_TrackingEvent{},
		AI_TrackingEventsWithRound:   []AI_TrackingEventWithRound{},
		LastTrackingTick:             0,
		AI_CombatDuels:               []AI_CombatDuel{},
		PendingCombatEvents:          []AI_CombatDuel{},
		AI_EconomyRounds:             []AI_EconomyRound{},
		AI_GrenadeEvents:             []AI_GrenadeEvent{},
		ParserWarnings:               []string{},
		PendingDrops:                 []AI_EconomyDrop{},
		RoundDrops:                   []AI_EconomyDrop{},
		RoundPickups:                 []AI_EconomyPickup{},
		RoundRefunds:                 []AI_EconomyRefund{},
		PlayerDropsGiven:             make(map[uint64][]string),
		PlayerDropsReceived:          make(map[uint64][]string),
		PlayerRefunds:                make(map[uint64][]string),
		WeaponOriginalOwner:          make(map[int64]uint64),
		AI_Duels:                     []AI_Duel{},
		AI_PlayersSummary:            []AI_PlayerStats{},
		RawCombatEvents:              []RawCombatEvent{},
		AI_CombatEvents:              []RawCombatEvent{},
		CTConsecutiveLosses:          1,
		TConsecutiveLosses:           1,
		CTRoundsWon:                  0,
		TRoundsWon:                   0,
		PlayerMoneyBefore:            make(map[uint64]int),
		CurrentRound:                 0,
		InRound:                      false,
		LastTick:                     0,
		LastPositions:                make(map[uint64]r3.Vector, 16),
		CurrentSpray:                 make(map[uint64]*SprayData, 16),
		LastWeaponFireTick:           make(map[uint64]int, 16),
		LastCombatFireTick:           make(map[uint64]int, 16),
		EnemyFirstSeenTick:           make(map[uint64]map[uint64]FirstSeenData, 16),
		FirstDamageTick:              make(map[uint64]map[uint64]int, 16),
		LastVisibleEnemies:           make(map[uint64]map[uint64]bool, 16),
		ActiveSmokes:                 make(map[int64]r3.Vector, 8),
		LastShotMechanics:            make(map[uint64]*ShotMechanics, 16),
		CrosshairStats:               make(map[uint64]*CrosshairStats, 16),
		ActiveGrenadeTrajectories:    make(map[int]*GrenadeTrajectoryEvent, 32),
		CompletedGrenadeTrajectories: make(map[int]*GrenadeTrajectoryEvent, 32),
		GrenadeBounces:               make(map[int]int, 32),
		RoundPurchases:               make(map[uint64][]AI_WeaponItem),
		// PHASE 1: Weapon state tracking
		LastWeaponState:     make(map[uint64]*WeaponStateSnapshot),
		PreviousWeaponState: make(map[uint64]*WeaponStateSnapshot),
		SprayStartState:     make(map[uint64]*WeaponStateSnapshot),
		RoundDamage:         make(map[uint64]map[uint64]int),
		LastKnownHealth:     make(map[uint64]int),
		LastActiveWeapon:    make(map[uint64]ActiveWeaponObservation),
		PendingBulletDamage: make([]BulletDamageSnapshot, 0, 8),
	}
}

// ShotMechanics guarda las mecánicas del último disparo
type ShotMechanics struct {
	CounterStrafeRating float64
	Tick                int
}
