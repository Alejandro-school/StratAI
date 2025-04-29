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
	MaxSpeed      map[uint64]float64

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

	PendingMolotovs map[int]models.MolotovData

	BombPlanted     bool
	LastRoundWinner common.Team
	LastRoundReason events.RoundEndReason

	LastReloadTick map[uint64]int

	//Registra el mismo ID para la granada cuando es lanzada y cuando explota. Si no registra 2 veces la misma granada.
	InfernoToProjectile map[int]int // infernoID → projectileID

	//Guardamos el dinero actual
	LastKnownMoney map[uint64]int

	// -----------------------------
	// Campo para trayectorias
	// -----------------------------
	ProjectileTrajectories map[int][]models.ProjectileTrajectoryEntry
	GrenadeMetas           map[int]*models.GrenadeMetadata
	GrenadeTrajectories    map[int][]models.ProjectileTrajectoryEntry

	// -----------------------------
	// Granadas (recopilación final)
	// -----------------------------

	AllGrenadeTrajectories []models.GrenadeMetadata

	// Control de muestreo -> último tick en que guardamos la posición
	lastGrenadeRecordTick map[int]int
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
		MaxSpeed:       make(map[uint64]float64),

		DiedPostRound:    make(map[uint64]bool),
		RoundPlayerKills: make(map[uint64]int),
		RoundFirstKill:   true,
		LastTeamKillTick: make(map[int]int),

		EconomyHistory:           []*models.RoundEconomyStats{},
		RoundEconomyMap:          make(map[int]map[uint64]*models.RoundEconomyStats),
		BuyWindowEndTickForRound: make(map[int]int),
		FlashEvents:              make(map[uint64][]flashData),
		LastScopedState:          make(map[uint64]bool),
		PendingMolotovs:          make(map[int]models.MolotovData),

		LossBonusT:  1400,
		LossBonusCT: 1400,

		ProjectileTrajectories: make(map[int][]models.ProjectileTrajectoryEntry),

		GrenadeMetas:          make(map[int]*models.GrenadeMetadata),
		GrenadeTrajectories:   make(map[int][]models.ProjectileTrajectoryEntry),
		lastGrenadeRecordTick: make(map[int]int),

		AllGrenadeTrajectories: []models.GrenadeMetadata{},

		LastReloadTick:      make(map[uint64]int),
		LastKnownMoney:      make(map[uint64]int),
		InfernoToProjectile: make(map[int]int),
	}
}
