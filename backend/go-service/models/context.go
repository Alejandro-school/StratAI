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

	// Bomb tracking
	BombPlanted bool
	BombSite    string
	BombTick    int

	// Buy tracking (NEW)
	FreezeTimeEnded   bool
	PlayerMoneyBefore map[uint64]int // SteamID -> dinero antes de compra

	// SOLO tracking custom necesario (no redundante con demoinfocs)

	// Para calcular distancia acumulada (demoinfocs no lo hace)
	LastPositions map[uint64]r3.Vector
	LastTick      int // Para movement sampling

	// Spray detection
	CurrentSpray       map[uint64]*SprayData
	LastWeaponFireTick map[uint64]int

	// Reaction time tracking
	EnemyFirstSeenTick map[uint64]map[uint64]int
	LastVisibleEnemies map[uint64]map[uint64]bool
	ActiveSmokes       []r3.Vector // Posiciones de humos activos

	// Mechanics tracking per shot
	LastShotMechanics map[uint64]*ShotMechanics

	// Crosshair placement tracking
	CrosshairStats map[uint64]*CrosshairStats

	// Grenade Trajectories tracking (ProjectileID -> Event)
	ActiveGrenadeTrajectories map[int]*GrenadeTrajectoryEvent
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
		Timeline:                  []TimelineEvent{},
		CurrentRoundEvents:        []TimelineEvent{},
		RoundTimelines:            []RoundTimeline{},
		LastGameStateTick:         0,
		CTConsecutiveLosses:       0,
		TConsecutiveLosses:        0,
		PlayerMoneyBefore:         make(map[uint64]int),
		CurrentRound:              0,
		InRound:                   false,
		LastTick:                  0,
		LastPositions:             make(map[uint64]r3.Vector, 16),
		CurrentSpray:              make(map[uint64]*SprayData, 16),
		LastWeaponFireTick:        make(map[uint64]int, 16),
		EnemyFirstSeenTick:        make(map[uint64]map[uint64]int, 16),
		LastVisibleEnemies:        make(map[uint64]map[uint64]bool, 16),
		ActiveSmokes:              []r3.Vector{},
		LastShotMechanics:         make(map[uint64]*ShotMechanics, 16),
		CrosshairStats:            make(map[uint64]*CrosshairStats, 16),
		ActiveGrenadeTrajectories: make(map[int]*GrenadeTrajectoryEvent, 32),
	}
}

// ShotMechanics guarda las mecánicas del último disparo
type ShotMechanics struct {
	CounterStrafeRating float64
	Tick                int
}
