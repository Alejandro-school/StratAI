package models

import (
	"cs2-demo-service/pkg/maps"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
)

// DemoContext mantiene el estado mínimo necesario durante el parsing
type DemoContext struct {
	Parser dem.Parser

	// Map Manager for visibility checks
	MapManager maps.VisibilityChecker

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

	// Bomb tracking
	BombPlanted bool
	BombSite    string
	BombTick    int

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
	EnemyFirstSeenTick map[uint64]map[uint64]FirstSeenData
	FirstDamageTick    map[uint64]map[uint64]int // Attacker -> Victim -> Tick of first damage
	LastVisibleEnemies map[uint64]map[uint64]bool
	ActiveSmokes       []r3.Vector // Posiciones de humos activos

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
	AI_GrenadeEvents           []AI_GrenadeEvent

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
	AI_Duels          []AI_Duel        // Final consolidated duels
	AI_PlayersSummary []AI_PlayerStats // Comprehensive player stats
	RawCombatEvents   []RawCombatEvent // Buffer for raw events to consolidate at round end

	// Grenade Damage Tracking
	ActiveInfernos map[int]int    // InfernoID -> Index in AI_GrenadeEvents
	PendingHEs     map[uint64]int // ThrowerSteamID -> Index in AI_GrenadeEvents (for immediate damage attribution)

	// Flashbang dedupe (CS2 can emit duplicate FlashExplode callbacks)
	ProcessedFlashExplodes map[string]struct{} // key: round|tick_explode|thrower_steam_id

	// PHASE 1: Weapon state tracking per player (for combat integration)
	LastWeaponState     map[uint64]*WeaponStateSnapshot // Current tick weapon state
	PreviousWeaponState map[uint64]*WeaponStateSnapshot // Previous tick weapon state (true BEFORE state)
	SprayStartState     map[uint64]*WeaponStateSnapshot // State at the start of the current firing sequence

	// Velocity tracking (Manual calculation fallback)
	PreviousPlayerPosition map[uint64]r3.Vector

	// Damage tracking for AI Duels
	RoundDamage map[uint64]map[uint64]int // AttackerID -> VictimID -> TotalDamage

	// FIX 5: Monotonic counter for unique event IDs
	CombatEventCounter int

	// FIX Bug 3: Track player HP before damage events
	// This allows showing victim's actual HP before they took damage
	LastKnownHealth map[uint64]int // SteamID -> HP before current damage

	// FIX: Track active weapon name per player each tick (for victim weapon in kills)
	LastActiveWeapon map[uint64]string // SteamID -> Active weapon name (updated each tick)
}

// FirstSeenData stores metadata when an enemy is first seen
type FirstSeenData struct {
	Tick                    int
	LastSeenTick            int // Último tick donde fue visible (para jiggle peek grace period)
	CrosshairPlacementError float64
	PitchError              float64
	YawError                float64
}

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
		TConsecutiveLosses:           0,
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
		ActiveSmokes:                 []r3.Vector{},
		LastShotMechanics:            make(map[uint64]*ShotMechanics, 16),
		CrosshairStats:               make(map[uint64]*CrosshairStats, 16),
		ActiveGrenadeTrajectories:    make(map[int]*GrenadeTrajectoryEvent, 32),
		CompletedGrenadeTrajectories: make(map[int]*GrenadeTrajectoryEvent, 32),
		GrenadeBounces:               make(map[int]int, 32),
		RoundPurchases:               make(map[uint64][]AI_WeaponItem),
		ActiveInfernos:               make(map[int]int),
		PendingHEs:                   make(map[uint64]int),
		ProcessedFlashExplodes:       make(map[string]struct{}),

		// PHASE 1: Weapon state tracking
		LastWeaponState:        make(map[uint64]*WeaponStateSnapshot),
		PreviousWeaponState:    make(map[uint64]*WeaponStateSnapshot),
		SprayStartState:        make(map[uint64]*WeaponStateSnapshot),
		PreviousPlayerPosition: make(map[uint64]r3.Vector),
		RoundDamage:            make(map[uint64]map[uint64]int),
		LastKnownHealth:        make(map[uint64]int),
		LastActiveWeapon:       make(map[uint64]string),
	}
}

// ShotMechanics guarda las mecánicas del último disparo
type ShotMechanics struct {
	CounterStrafeRating float64
	Tick                int
}
