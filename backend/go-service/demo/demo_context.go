package demo

import (
	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// ClutchCandidate para guardar quién está en situación de clutch
type ClutchCandidate struct {
	SteamID    uint64
	EnemyCount int
}

// flashData se utiliza para registrar cuántos enemigos/aliados fueron cegados por cada flash
type flashData struct {
	explosionTick int
	enemyCount    int
	friendlyCount int
}

// DemoContext agrupa el estado global de la demo.
type DemoContext struct {
	parser dem.Parser

	PlayerStatsMap map[uint64]*models.PlayerStats

	DamageDone     map[uint64]float64
	HeadshotsCount map[uint64]int

	EventLogs []models.EventLog

	LastPositions map[uint64]r3.Vector
	LastTick      int
	LastSpeed     map[uint64]float64

	// Control de rondas
	RoundNumber      int
	InRound          bool
	InRoundEndPeriod bool

	DiedPostRound    map[uint64]bool
	RoundPlayerKills map[uint64]int
	RoundFirstKill   bool
	LastTeamKillTick map[int]int

	EconomyHistory  []*models.RoundEconomyStats
	RoundEconomyMap map[int]map[uint64]*models.RoundEconomyStats

	BuyWindowEndTickForRound map[int]int

	ClutchCandidateT  *ClutchCandidate
	ClutchCandidateCT *ClutchCandidate

	FlashEvents map[uint64][]flashData

	LastScopedState map[uint64]bool

	LossBonusT  int
	LossBonusCT int

	BombPlanted     bool
	LastRoundWinner common.Team
	LastRoundReason events.RoundEndReason

	//Guardamos el dinero actual
	LastKnownMoney map[uint64]int

	// First Shot Tracking - para calcular first shot accuracy
	LastWeaponFireTick map[uint64]int  // Último tick de disparo por jugador
	LastShotWasFirst   map[uint64]bool // Si el último disparo fue "primero" del engagement

	// Reaction Time Tracking - cuando un enemigo se vuelve visible
	EnemyFirstSeenTick map[uint64]map[uint64]int  // [playerID][enemyID] = tick primera vez visto
	ReactionRegistered map[uint64]map[uint64]bool // [playerID][enemyID] = true si ya se registró evento de reacción
	LastVisibleEnemies map[uint64]map[uint64]bool // [playerID][enemyID] = true si estaba visible en frame anterior

	// Rotation Tracking - última zona de cada jugador para detectar cambios
	LastZone        map[uint64]string // [playerID] = última zona
	ZoneEnteredTick map[uint64]int    // [playerID] = tick al entrar a última zona

	BombPlantedSite string // A qué sitio se plantó la bomba ("A", "B", "Unknown")
	BombPlantedTick int    // Tick cuando se plantó

	// Recoil Analysis Tracking - para detectar sprays
	CurrentSpray  map[uint64]*models.RecoilSpray // [playerID] = spray actual en progreso
	LastShotPitch map[uint64]float32             // [playerID] = último pitch al disparar
	LastShotYaw   map[uint64]float32             // [playerID] = último yaw al disparar

	// Grenade Tracking - todas las granadas lanzadas
	Grenades []models.GrenadeEvent
}

// NewDemoContext crea e inicializa un DemoContext vacío.
func NewDemoContext() *DemoContext {
	return &DemoContext{
		PlayerStatsMap: make(map[uint64]*models.PlayerStats),

		DamageDone:     make(map[uint64]float64),
		HeadshotsCount: make(map[uint64]int),
		EventLogs:      []models.EventLog{},
		LastPositions:  make(map[uint64]r3.Vector),
		LastSpeed:      make(map[uint64]float64),

		DiedPostRound:    make(map[uint64]bool),
		RoundPlayerKills: make(map[uint64]int),
		RoundFirstKill:   true,
		LastTeamKillTick: make(map[int]int),

		EconomyHistory:           []*models.RoundEconomyStats{},
		RoundEconomyMap:          make(map[int]map[uint64]*models.RoundEconomyStats),
		BuyWindowEndTickForRound: make(map[int]int),
		FlashEvents:              make(map[uint64][]flashData),
		LastScopedState:          make(map[uint64]bool),

		LossBonusT:  1400,
		LossBonusCT: 1400,

		LastKnownMoney: make(map[uint64]int),

		LastWeaponFireTick: make(map[uint64]int),
		LastShotWasFirst:   make(map[uint64]bool),

		EnemyFirstSeenTick: make(map[uint64]map[uint64]int),
		ReactionRegistered: make(map[uint64]map[uint64]bool),
		LastVisibleEnemies: make(map[uint64]map[uint64]bool),

		LastZone:        make(map[uint64]string),
		ZoneEnteredTick: make(map[uint64]int),
		BombPlantedSite: "",
		BombPlantedTick: 0,

		CurrentSpray:  make(map[uint64]*models.RecoilSpray),
		LastShotPitch: make(map[uint64]float32),
		LastShotYaw:   make(map[uint64]float32),
	}
}
