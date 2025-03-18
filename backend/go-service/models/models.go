package models

import (
	"math"

	"github.com/golang/geo/r3"
)

// RoundEconomyStats almacena la info económica de un jugador en una ronda concreta.
type RoundEconomyStats struct {
	RoundNumber  int    `json:"round_number"`
	SteamID      string `json:"steam_id"`
	InitialMoney int    `json:"initial_money"`
	FinalMoney   int    `json:"final_money"`
	LossBonus    int    `json:"loss_bonus"`
}

// MovementLog guarda la posición, velocidad y estado (agachado o no) del jugador en un tick concreto.
type MovementLog struct {
	Tick      int     `json:"tick"`       // Tick del juego
	X         float64 `json:"x_position"` // Posición X
	Y         float64 `json:"y_position"` // Posición Y
	Z         float64 `json:"z_position"` // Posición Z
	Speed     float64 `json:"speed"`      // Velocidad en este tick (aprox. unidades/seg)
	IsDucking bool    `json:"is_ducking"` // Verdadero si el jugador está agachado
}

// PlayerStats representa las estadísticas finales de un jugador.
type PlayerStats struct {
	SteamID            string  `json:"steamID"`
	Name               string  `json:"name"`
	Team               string  `json:"team"`
	Kills              int     `json:"kills"`
	Assists            int     `json:"assists"`
	Deaths             int     `json:"deaths"`
	ADR                float64 `json:"adr"`           // Average Damage per Round
	HSPercentage       float64 `json:"hs_percentage"` // Porcentaje de kills que fueron headshot
	KDRatio            float64 `json:"kd_ratio"`      // Kills/Deaths
	FlashAssists       int     `json:"flash_assists"`
	Avatar             string  `json:"avatar"`
	Rank               string  `json:"rank"`
	Position           int     `json:"position"`          // Orden final (1º, 2º, etc.)
	DistanceTraveled   float64 `json:"distance_traveled"` // Suma de distancias recorridas
	EnemiesFlashed     int     `json:"enemiesFlashed"`
	ShotsFired         int     `json:"shots_fired"`
	ShotsFiredNoReason int     `json:"shots_fired_no_reason"`
	Purchases          int     `json:"purchases"`
	InitialMoney       int     `json:"initial_money"`

	// AimPlacement: Grados promedio de desviación con respecto a la cabeza del enemigo
	AimPlacement float64 `json:"aim_placement"`

	// Errores obvios
	BadFlashCount   int `json:"bad_flash_count"`
	BadMolotovCount int `json:"bad_molotov_count"`

	// Fuego amigo
	TeamKillCount int `json:"team_kill_count"`
	TeamDamage    int `json:"team_damage"`

	// Disparos "raros"
	AccidentalShots int `json:"accidental_shots"`

	// Estadísticas de multi-kill (por ronda, se acumulan a lo largo del match)
	DoubleKills int `json:"double_kills"`
	TripleKills int `json:"triple_kills"`
	QuadKills   int `json:"quad_kills"`
	Ace         int `json:"ace"`

	// Clutch: ronda ganada siendo el único vivo de tu equipo
	ClutchWins int `json:"clutch_wins"`

	// Contadores de ticks agachado, andando y corriendo
	CrouchTicks int `json:"crouch_ticks"`
	WalkTicks   int `json:"walk_ticks"`
	RunTicks    int `json:"run_ticks"`

	// Registro detallado de movimiento a cada tick (o cada frame)
	Movement []MovementLog `json:"movement"`
}

// EventLog registra un evento relevante ocurrido durante la partida.
type EventLog struct {
	Round     int    `json:"round"`
	Timestamp int    `json:"timestamp"` // tick del juego
	EventType string `json:"event_type"`
	Details   string `json:"details"`
}

// DemoParseResult almacena la información de toda la partida.
type DemoParseResult struct {
	MatchID        string               `json:"match_id"`
	MapName        string               `json:"map_name"`
	Duration       string               `json:"match_duration"`
	Result         string               `json:"result"`
	TeamScore      int                  `json:"team_score"`
	OpponentScore  int                  `json:"opponent_score"`
	Players        []PlayerStats        `json:"players"`
	EventLogs      []EventLog           `json:"event_logs"`
	Filename       string               `json:"filename"`
	Date           string               `json:"date"`
	EconomyHistory []*RoundEconomyStats `json:"economy_history"`
}

// deg2rad convierte grados a radianes (auxiliar para cálculos de ángulos).
func deg2rad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// angularDifference calcula el ángulo (en grados) entre dos direcciones dadas por (yaw, pitch).
func AngularDifference(yaw1, pitch1, yaw2, pitch2 float64) float64 {
	a1 := deg2rad(yaw1)
	b1 := deg2rad(pitch1)
	a2 := deg2rad(yaw2)
	b2 := deg2rad(pitch2)

	// Convertir ángulos a vectores unitarios
	v1 := r3.Vector{
		X: math.Cos(b1) * math.Cos(a1),
		Y: math.Cos(b1) * math.Sin(a1),
		Z: math.Sin(b1),
	}
	v2 := r3.Vector{
		X: math.Cos(b2) * math.Cos(a2),
		Y: math.Cos(b2) * math.Sin(a2),
		Z: math.Sin(b2),
	}

	// Calcular el producto punto y asegurarse de que esté en [-1, 1]
	dot := v1.X*v2.X + v1.Y*v2.Y + v1.Z*v2.Z
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}

	// Retorna el ángulo en grados
	return math.Acos(dot) * 180.0 / math.Pi
}
