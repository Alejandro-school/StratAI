package demo

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Mapa de precios aproximados de las armas / ítems en CS:GO.
// Nota: Ajusta valores si cambian en CS2 o si existe algún descuento puntual.
var weaponPrices = map[common.EquipmentType]int{
	// Pistolas
	common.EqGlock:        200,
	common.EqUSP:          200,
	common.EqP2000:        200,
	common.EqP250:         300,
	common.EqFiveSeven:    500,
	common.EqTec9:         500,
	common.EqCZ:           500,
	common.EqDeagle:       700,
	common.EqRevolver:     600,
	common.EqDualBerettas: 300,

	// SMGs
	common.EqMac10: 1050,
	common.EqMP9:   1250,
	common.EqMP7:   1500,
	common.EqMP5:   1500, // MP5-SD
	common.EqUMP:   1200,
	common.EqP90:   2350,
	common.EqBizon: 1400,

	// Rifles
	common.EqGalil:  1800,
	common.EqFamas:  1950,
	common.EqAK47:   2700,
	common.EqM4A4:   2900,
	common.EqM4A1:   2900,
	common.EqScout:  1700, // SSG08
	common.EqSG553:  3000, // SG 553
	common.EqAUG:    3300,
	common.EqAWP:    4750,
	common.EqG3SG1:  5000,
	common.EqScar20: 5000,

	// Escopetas
	common.EqNova:   1050,
	common.EqMag7:   1300,
	common.EqXM1014: 2000,

	// Ametralladoras
	common.EqM249:  5200,
	common.EqNegev: 1700,

	// Granadas
	common.EqMolotov:    400, // T
	common.EqIncendiary: 500, // CT
	common.EqFlash:      200,
	common.EqSmoke:      300,
	common.EqHE:         300,
	common.EqDecoy:      50,

	// Taser
	common.EqZeus: 200,

	// Kevlar
	common.EqKevlar:    650,
	common.EqDefuseKit: 400,
	common.EqHelmet:    1000,
}

// vectorDistance calcula la distancia euclídea entre dos vectores (X, Y, Z).
func vectorDistance(v1, v2 r3.Vector) float64 {
	return math.Sqrt(math.Pow(v1.X-v2.X, 2) +
		math.Pow(v1.Y-v2.Y, 2) +
		math.Pow(v1.Z-v2.Z, 2))
}

// calcLossBonus devuelve el bono de dinero según la racha de derrotas de un equipo.
func calcLossBonus(streak int) int {
	switch {
	case streak <= 0:
		return 1400
	case streak == 1:
		return 1900
	case streak == 2:
		return 2400
	case streak == 3:
		return 2900
	case streak == 4:
		return 3400
	default:
		return 1400
	}
}

// getSteamData
func getSteamData(steamID string) (string, string, error) {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("steam_api_key no configurada")
	}
	urlAvatar := fmt.Sprintf("http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s", apiKey, steamID)
	resp, err := http.Get(urlAvatar)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	log.Printf("Respuesta avatar para %s: %s", steamID, string(body))
	var avatarResp struct {
		Response struct {
			Players []struct {
				AvatarFull string `json:"avatarfull"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &avatarResp); err != nil {
		return "", "", fmt.Errorf("error parseando JSON: %v", err)
	}
	avatarURL := ""
	if len(avatarResp.Response.Players) > 0 {
		avatarURL = avatarResp.Response.Players[0].AvatarFull
	}
	return avatarURL, "", nil
}

// ExtractMatchDuration
func ExtractMatchDuration(jsonData string) (string, error) {
	if jsonData == "{}" || jsonData == "" {
		return "00:00", nil
	}
	var response struct {
		MatchDuration int `json:"matchDuration"`
	}
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return "", fmt.Errorf("error parseando JSON: %v", err)
	}
	if response.MatchDuration == 0 {
		return "00:00", fmt.Errorf("⚠️ error: no se encontraron datos de matchDuration")
	}
	mins := response.MatchDuration / 60
	secs := response.MatchDuration % 60
	return fmt.Sprintf("%02d:%02d", mins, secs), nil
}

// Para aim placement
var aimOffsetsByPlayer = make(map[uint64][]float64)

// Hash del fichero
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	return "match_" + hash[:10], nil
}

var ExportDir = "../data/exports"

func ProcessDemoFile(filePath, userSteamID string, jsonData string) (*models.DemoParseResult, error) {
	log.Println("DEBUG: --> Entró a ProcessDemoFile")

	matchDuration, err := ExtractMatchDuration(jsonData)
	if err != nil {
		log.Printf("⚠️ error extrayendo match_duration: %v", err)
		matchDuration = "00:00"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la demo %s: %v", filePath, err)
	}
	defer file.Close()

	parser := dem.NewParser(file)
	defer parser.Close()

	// Estructuras
	playerStatsMap := make(map[uint64]*models.PlayerStats)
	damageDone := make(map[uint64]float64)
	headshotsCount := make(map[uint64]int)
	var eventLogs []models.EventLog

	lastPositions := make(map[uint64]r3.Vector)
	lastTick := parser.GameState().IngameTick()

	lastSpeed := make(map[uint64]float64)
	maxSpeed := make(map[uint64]float64)

	userID64, _ := strconv.ParseUint(userSteamID, 10, 64)
	var userTeam common.Team

	var roundNumber int
	var lossStreakT, lossStreakCT int

	// Ojo: Cambiamos a slice de *RoundEconomyStats
	var economyHistory []*models.RoundEconomyStats

	// Mapa para acceder a la economía de cada player en la ronda
	roundEconomyMap := make(map[int]map[uint64]*models.RoundEconomyStats)

	roundPlayerKills := make(map[uint64]int)
	type ClutchCandidate struct {
		steamID    uint64
		enemyCount int
	}
	var clutchCandidateT, clutchCandidateCT *ClutchCandidate

	// Control del freeze time
	inRound := false

	//--------------------------------------------------------------------
	// RoundStart => freeze time
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.RoundStart) {
		roundNumber++
		roundPlayerKills = make(map[uint64]int)
		clutchCandidateT = nil
		clutchCandidateCT = nil

		inRound = false

		gameState := parser.GameState()

		roundEconomyMap[roundNumber] = make(map[uint64]*models.RoundEconomyStats)

		for _, pl := range gameState.Participants().All() {
			if pl.SteamID64 == 0 || pl.Team == common.TeamUnassigned {
				continue
			}
			var lossBonus int
			if pl.Team == common.TeamTerrorists {
				lossBonus = calcLossBonus(lossStreakT)
			} else if pl.Team == common.TeamCounterTerrorists {
				lossBonus = calcLossBonus(lossStreakCT)
			}

			// Creamos un *RoundEconomyStats con initial/finalMoney
			econ := &models.RoundEconomyStats{
				RoundNumber:  roundNumber,
				SteamID:      fmt.Sprintf("%d", pl.SteamID64),
				InitialMoney: pl.Money(),
				FinalMoney:   pl.Money(),
				LossBonus:    lossBonus,
			}
			economyHistory = append(economyHistory, econ)

			roundEconomyMap[roundNumber][pl.SteamID64] = econ
		}

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: gameState.IngameTick(),
			EventType: "RoundStart",
			Details:   fmt.Sprintf("Round %d started (freeze time)", roundNumber),
		})
	})
	// Para cada ronda, guardamos en qué tick se termina la ventana de compra
	var buyWindowEndTickForRound = make(map[int]int)

	//--------------------------------------------------------------------
	// RoundFreezetimeEnd => arranca la acción
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		inRound = true

		// Tomamos el tick actual
		freezeTimeEndTick := parser.GameState().IngameTick()

		// Asumimos 15 segundos de buytime post-freeze
		buyTimeSeconds := 15
		// Convertir esos 15 seg a “ticks” => tickRate * 15
		// Ojo: parser.TickRate() retorna un float64 con la tasa de ticks
		buyWindowTicks := int(parser.TickRate() * float64(buyTimeSeconds))

		// guardamos
		buyWindowEndTickForRound[roundNumber] = freezeTimeEndTick + buyWindowTicks

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: freezeTimeEndTick,
			EventType: "FreezeTimeEnded",
			Details:   fmt.Sprintf("Round %d: freeze time ended, action begins (buy window lasts ~%ds)", roundNumber, buyTimeSeconds),
		})
	})

	//--------------------------------------------------------------------
	// Manejo de flashes
	//--------------------------------------------------------------------
	type flashData struct {
		explosionTick int
		enemyCount    int
		friendlyCount int
	}
	flashEvents := make(map[uint64][]flashData)

	parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower == nil {
			return
		}
		sid := e.Thrower.SteamID64
		currentTick := parser.GameState().IngameTick()
		flashEvents[sid] = append(flashEvents[sid], flashData{
			explosionTick: currentTick,
			enemyCount:    0,
			friendlyCount: 0,
		})
	})

	parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		if e.Attacker == nil || e.Player == nil {
			return
		}
		sid := e.Attacker.SteamID64
		currentTick := parser.GameState().IngameTick()
		const threshold = 20
		if dataList, exists := flashEvents[sid]; exists {
			for i := len(dataList) - 1; i >= 0; i-- {
				if currentTick-dataList[i].explosionTick < threshold {
					if e.Attacker.Team == e.Player.Team && e.Attacker.Team != common.TeamUnassigned {
						dataList[i].friendlyCount++
					} else {
						dataList[i].enemyCount++
					}
					flashEvents[sid][i] = dataList[i]
					break
				}
			}
		}
	})

	//--------------------------------------------------------------------
	// Kills
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.Kill) {
		killer := e.Killer
		victim := e.Victim

		if killer != nil {
			sidK := killer.SteamID64
			if playerStatsMap[sidK] == nil {
				playerStatsMap[sidK] = &models.PlayerStats{Name: killer.Name}
			}
			if victim != nil && killer.Team == victim.Team && killer.Team != common.TeamUnassigned {
				playerStatsMap[sidK].TeamKillCount++
				eventLogs = append(eventLogs, models.EventLog{
					Round:     roundNumber,
					Timestamp: parser.GameState().IngameTick(),
					EventType: "TeamKill",
					Details:   fmt.Sprintf("Player %s killed teammate %s", killer.Name, victim.Name),
				})
			} else {
				playerStatsMap[sidK].Kills++
				if e.IsHeadshot {
					headshotsCount[sidK]++
				}
			}
			roundPlayerKills[sidK]++
		}

		if victim != nil {
			sidV := victim.SteamID64
			if playerStatsMap[sidV] == nil {
				playerStatsMap[sidV] = &models.PlayerStats{Name: victim.Name}
			}
			playerStatsMap[sidV].Deaths++
			deathPos := victim.Position()
			eventLogs = append(eventLogs, models.EventLog{
				Round:     roundNumber,
				Timestamp: parser.GameState().IngameTick(),
				EventType: "DeathPosition",
				Details:   fmt.Sprintf("Player %s died at (%.1f, %.1f, %.1f)", victim.Name, deathPos.X, deathPos.Y, deathPos.Z),
			})
		}

		if e.Assister != nil && e.Assister != e.Killer {
			sidA := e.Assister.SteamID64
			if playerStatsMap[sidA] == nil {
				playerStatsMap[sidA] = &models.PlayerStats{Name: e.Assister.Name}
			}
			playerStatsMap[sidA].Assists++
		}

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: parser.GameState().IngameTick(),
			EventType: "Kill",
			Details: fmt.Sprintf("Killer=%s, Victim=%s, Headshot=%v",
				nameOrNil(killer),
				nameOrNil(victim),
				e.IsHeadshot),
		})

		// Verificamos si queda 1 vivo (clutch)
		gameState := parser.GameState()
		for _, team := range []common.Team{common.TeamTerrorists, common.TeamCounterTerrorists} {
			var alive []uint64
			for _, pl := range gameState.Participants().All() {
				if pl.Team == team && pl.IsAlive() {
					alive = append(alive, pl.SteamID64)
				}
			}
			if len(alive) == 1 {
				var enemyAlive []uint64
				var opposingTeam common.Team
				if team == common.TeamTerrorists {
					opposingTeam = common.TeamCounterTerrorists
				} else {
					opposingTeam = common.TeamTerrorists
				}
				for _, pl := range gameState.Participants().All() {
					if pl.Team == opposingTeam && pl.IsAlive() {
						enemyAlive = append(enemyAlive, pl.SteamID64)
					}
				}
				if team == common.TeamTerrorists && clutchCandidateT == nil {
					clutchCandidateT = &ClutchCandidate{
						steamID:    alive[0],
						enemyCount: len(enemyAlive),
					}
				} else if team == common.TeamCounterTerrorists && clutchCandidateCT == nil {
					clutchCandidateCT = &ClutchCandidate{
						steamID:    alive[0],
						enemyCount: len(enemyAlive),
					}
				}
			}
		}
	})

	//--------------------------------------------------------------------
	// PlayerHurt => daño
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil || e.Attacker == e.Player {
			return
		}
		sidA := e.Attacker.SteamID64
		if playerStatsMap[sidA] == nil {
			playerStatsMap[sidA] = &models.PlayerStats{Name: e.Attacker.Name}
		}
		if e.Attacker.Team == e.Player.Team && e.Attacker.Team != common.TeamUnassigned {
			playerStatsMap[sidA].TeamDamage += int(e.HealthDamage)
			eventLogs = append(eventLogs, models.EventLog{
				Round:     roundNumber,
				Timestamp: parser.GameState().IngameTick(),
				EventType: "TeamDamage",
				Details:   fmt.Sprintf("%s dealt %d dmg to teammate %s", e.Attacker.Name, e.HealthDamage, e.Player.Name),
			})
		} else {
			damageDone[sidA] += float64(e.HealthDamage)
		}
	})

	//--------------------------------------------------------------------
	// WeaponFire => disparos + aim
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil {
			return
		}
		sid := e.Shooter.SteamID64
		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{Name: e.Shooter.Name}
		}
		playerStatsMap[sid].ShotsFired++

		speed := lastSpeed[sid]
		const speedThreshold = 200.0
		if speed > speedThreshold {
			playerStatsMap[sid].AccidentalShots++
		}

		shooterPos := e.Shooter.Position()
		nearEnemy := false
		for _, pl := range parser.GameState().Participants().All() {
			if pl.Team != e.Shooter.Team && pl.IsAlive() {
				if vectorDistance(shooterPos, pl.Position()) < 800.0 {
					nearEnemy = true
					break
				}
			}
		}
		if !nearEnemy {
			playerStatsMap[sid].ShotsFiredNoReason++
		}

		// AimPlacement
		viewX := float64(e.Shooter.ViewDirectionX())
		viewY := float64(e.Shooter.ViewDirectionY())
		bestAngleDiff := 9999.0
		for _, potentialEnemy := range parser.GameState().Participants().All() {
			if potentialEnemy == nil || !potentialEnemy.IsAlive() {
				continue
			}
			if potentialEnemy.Team == e.Shooter.Team {
				continue
			}
			headPos := potentialEnemy.Position()
			headPos.Z += 64.0
			dx := headPos.X - shooterPos.X
			dy := headPos.Y - shooterPos.Y
			dz := headPos.Z - shooterPos.Z
			idealYaw := math.Atan2(dy, dx) * 180.0 / math.Pi
			distXY := math.Sqrt(dx*dx + dy*dy)
			idealPitch := math.Atan2(dz, distXY) * 180.0 / math.Pi
			angleDiff := models.AngularDifference(viewX, viewY, idealYaw, idealPitch)
			if angleDiff < bestAngleDiff {
				bestAngleDiff = angleDiff
			}
		}
		if bestAngleDiff < 9999.0 {
			aimOffsetsByPlayer[sid] = append(aimOffsetsByPlayer[sid], bestAngleDiff)
		}

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: parser.GameState().IngameTick(),
			EventType: "WeaponFire",
			Details:   fmt.Sprintf("Player %s fired %s (speed=%.2f)", e.Shooter.Name, e.Weapon.String(), speed),
		})
	})

	//--------------------------------------------------------------------
	// ItemPickup => comprar o recoger
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.ItemPickup) {
		if e.Player == nil || e.Player.SteamID64 == 0 || e.Weapon == nil {
			return
		}
		sid := e.Player.SteamID64
		wType := e.Weapon.Type

		// Buscamos su RoundEconomyStats en la ronda actual
		econMapForRound, ok := roundEconomyMap[roundNumber]
		if !ok {
			return
		}
		econ, ok := econMapForRound[sid]
		if !ok {
			return
		}

		cost, ok2 := weaponPrices[wType]
		if !ok2 {
			cost = 0
		}

		// Miramos si el tick actual está dentro de la ventana
		currentTick := parser.GameState().IngameTick()
		endTick := buyWindowEndTickForRound[roundNumber] // lo calculamos antes

		isPurchase := false
		if cost > 0 && econ.FinalMoney >= cost && currentTick <= endTick {
			// Solo se descuenta si estamos en buytime
			isPurchase = true
			econ.FinalMoney -= cost
		}

		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{Name: e.Player.Name}
		}
		if isPurchase {
			playerStatsMap[sid].Purchases++
		}

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: parser.GameState().IngameTick(),
			EventType: "ItemPickup",
			Details: fmt.Sprintf("Player %s %s %s [finalMoney(after)=%d cost=%d]",
				e.Player.Name,
				func() string {
					if isPurchase {
						return "bought"
					}
					return "picked_up"
				}(),
				e.Weapon.String(),
				econ.FinalMoney,
				cost,
			),
		})
	})

	//--------------------------------------------------------------------
	// FireGrenadeStart => molotov
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.FireGrenadeStart) {
		if e.Thrower == nil {
			return
		}
		sid := e.Thrower.SteamID64
		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{Name: e.Thrower.Name}
		}
		const airThreshold = 100.0
		if e.Position.Z > airThreshold {
			playerStatsMap[sid].BadMolotovCount++
			eventLogs = append(eventLogs, models.EventLog{
				Round:     roundNumber,
				Timestamp: parser.GameState().IngameTick(),
				EventType: "BadMolotov",
				Details:   fmt.Sprintf("Molotov exploded in the air by %s (Z=%.1f)", e.Thrower.Name, e.Position.Z),
			})
		}
	})

	//--------------------------------------------------------------------
	// RoundEnd
	//--------------------------------------------------------------------
	parser.RegisterEventHandler(func(e events.RoundEnd) {
		inRound = false

		// Procesamos flashes
		for sid, dataList := range flashEvents {
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: "Unknown"}
			}
			for _, fd := range dataList {
				playerStatsMap[sid].EnemiesFlashed += fd.enemyCount
				if fd.friendlyCount > fd.enemyCount && fd.friendlyCount > 0 {
					playerStatsMap[sid].BadFlashCount++
					eventLogs = append(eventLogs, models.EventLog{
						Round:     roundNumber,
						Timestamp: parser.GameState().IngameTick(),
						EventType: "BadFlash",
						Details: fmt.Sprintf("%s's flash blinded %d teammates vs %d enemies",
							playerStatsMap[sid].Name, fd.friendlyCount, fd.enemyCount),
					})
				}
			}
		}
		flashEvents = make(map[uint64][]flashData)

		// Multi-kill
		for sid, kills := range roundPlayerKills {
			switch kills {
			case 2:
				playerStatsMap[sid].DoubleKills++
			case 3:
				playerStatsMap[sid].TripleKills++
			case 4:
				playerStatsMap[sid].QuadKills++
			case 5:
				playerStatsMap[sid].Ace++
			}
		}

		// Clutch
		if e.Winner == common.TeamTerrorists && clutchCandidateT != nil {
			if clutchCandidateT.enemyCount >= 2 {
				playerStatsMap[clutchCandidateT.steamID].ClutchWins++
			}
		} else if e.Winner == common.TeamCounterTerrorists && clutchCandidateCT != nil {
			if clutchCandidateCT.enemyCount >= 2 {
				playerStatsMap[clutchCandidateCT.steamID].ClutchWins++
			}
		}

		// Actualización rachas
		if e.Winner == common.TeamTerrorists {
			lossStreakCT++
			lossStreakT = 0
		} else if e.Winner == common.TeamCounterTerrorists {
			lossStreakT++
			lossStreakCT = 0
		}

		eventLogs = append(eventLogs, models.EventLog{
			Round:     roundNumber,
			Timestamp: parser.GameState().IngameTick(),
			EventType: "RoundEnd",
			Details:   fmt.Sprintf("Round %d ended. Reason: %v (Winner=%v)", roundNumber, e.Reason, e.Winner),
		})
	})

	//--------------------------------------------------------------------
	// Bucle frames => movimiento
	//--------------------------------------------------------------------
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			more, errParse := parser.ParseNextFrame()
			if !more || errParse != nil {
				break
			}
			if !inRound {
				continue
			}

			currentTick := parser.GameState().IngameTick()
			if currentTick%10 != 0 {
				continue
			}

			deltaTicks := float64(currentTick - lastTick)
			if deltaTicks > 0 {
				for _, pl := range parser.GameState().Participants().All() {
					sid := pl.SteamID64
					if sid == 0 || !pl.IsAlive() {
						continue
					}
					currPos := pl.Position()

					if lastPos, ok := lastPositions[sid]; ok {
						deltaDist := vectorDistance(currPos, lastPos)
						speed := deltaDist * 128.0 / deltaTicks
						lastSpeed[sid] = speed
						if speed > maxSpeed[sid] {
							maxSpeed[sid] = speed
						}

						if playerStatsMap[sid] == nil {
							playerStatsMap[sid] = &models.PlayerStats{Name: pl.Name}
						}
						playerStatsMap[sid].DistanceTraveled += deltaDist

						if pl.IsDucking() {
							playerStatsMap[sid].CrouchTicks++
						} else if speed < 100 {
							playerStatsMap[sid].WalkTicks++
						} else {
							playerStatsMap[sid].RunTicks++
						}

						ps := playerStatsMap[sid]
						ps.Movement = append(ps.Movement, models.MovementLog{
							Tick:      currentTick,
							X:         currPos.X,
							Y:         currPos.Y,
							Z:         currPos.Z,
							Speed:     speed,
							IsDucking: pl.IsDucking(),
						})
					}
					lastPositions[sid] = currPos
				}
			}
			lastTick = currentTick
		}
	}()
	wg.Wait()

	//--------------------------------------------------------------------
	// Post-proceso final
	//--------------------------------------------------------------------
	header := parser.Header()
	mapName := header.MapName
	gs := parser.GameState()
	tScore := gs.TeamTerrorists().Score()
	ctScore := gs.TeamCounterTerrorists().Score()

	for _, pl := range gs.Participants().All() {
		sid := pl.SteamID64
		if sid == 0 {
			continue
		}
		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{}
		}
		ps := playerStatsMap[sid]
		ps.SteamID = fmt.Sprintf("%d", sid)
		ps.Name = pl.Name
		switch pl.Team {
		case common.TeamTerrorists:
			ps.Team = "T"
		case common.TeamCounterTerrorists:
			ps.Team = "CT"
		default:
			ps.Team = "UNK"
		}
		if sid == userID64 {
			userTeam = pl.Team
		}
	}

	var teamScore, opponentScore int
	if userTeam == common.TeamTerrorists {
		teamScore = tScore
		opponentScore = ctScore
	} else if userTeam == common.TeamCounterTerrorists {
		teamScore = ctScore
		opponentScore = tScore
	} else {
		teamScore = tScore
		opponentScore = ctScore
	}

	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Datos de Steam
	for sid, ps := range playerStatsMap {
		steamIDStr := fmt.Sprintf("%d", sid)
		avatarURL, rankVal, err := getSteamData(steamIDStr)
		if err != nil {
			log.Printf("error retrieving steam data for %s: %v", steamIDStr, err)
			ps.Rank = "N/A"
		} else {
			ps.Avatar = avatarURL
			ps.Rank = rankVal
		}
	}

	roundsPlayed := roundNumber
	for sid, ps := range playerStatsMap {
		if roundsPlayed > 0 {
			ps.ADR = damageDone[sid] / float64(roundsPlayed)
		}
		if ps.Deaths > 0 {
			ps.KDRatio = float64(ps.Kills) / float64(ps.Deaths)
		} else {
			ps.KDRatio = float64(ps.Kills)
		}
		if ps.Kills > 0 {
			ps.HSPercentage = float64(headshotsCount[sid]) / float64(ps.Kills) * 100
		}
		if offsets, exists := aimOffsetsByPlayer[sid]; exists && len(offsets) > 0 {
			var sum float64
			for _, val := range offsets {
				sum += val
			}
			ps.AimPlacement = sum / float64(len(offsets))
		}
	}

	// Ordenamos jugadores
	var sortedPlayers []models.PlayerStats
	for _, ps := range playerStatsMap {
		sortedPlayers = append(sortedPlayers, *ps)
	}
	sort.Slice(sortedPlayers, func(i, j int) bool {
		if sortedPlayers[i].KDRatio != sortedPlayers[j].KDRatio {
			return sortedPlayers[i].KDRatio > sortedPlayers[j].KDRatio
		}
		if sortedPlayers[i].ADR != sortedPlayers[j].ADR {
			return sortedPlayers[i].ADR > sortedPlayers[j].ADR
		}
		return sortedPlayers[i].Kills > sortedPlayers[j].Kills
	})
	for i := range sortedPlayers {
		sortedPlayers[i].Position = i + 1
	}

	matchID, err := ComputeFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("error generando matchID: %v", err)
	}

	parseResult := &models.DemoParseResult{
		MatchID:       matchID,
		MapName:       mapName,
		Duration:      matchDuration,
		Result:        result,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
		Players:       sortedPlayers,
		EventLogs:     eventLogs,
		Filename:      filepath.Base(filePath),
		Date:          time.Now().Format("2006-01-02 15:04:05"),
		// economyHistory es []*RoundEconomyStats
		EconomyHistory: economyHistory,
	}

	subFolder := filepath.Join(ExportDir, parseResult.MatchID)
	if err := os.MkdirAll(subFolder, 0755); err != nil {
		return nil, fmt.Errorf("error creando carpeta %s: %v", subFolder, err)
	}

	if err := exportMatchSummaryCSV(parseResult, subFolder); err != nil {
		return nil, err
	}
	if err := exportPlayersCSV(parseResult, subFolder); err != nil {
		return nil, err
	}
	if err := exportMovementCSV(parseResult, subFolder); err != nil {
		return nil, err
	}
	if err := exportEconomyCSV(parseResult, subFolder); err != nil {
		return nil, err
	}
	if err := exportEventsCSV(parseResult, subFolder); err != nil {
		return nil, err
	}

	return parseResult, nil
}

// Helpers para logs
func nameOrNil(p *common.Player) string {
	if p == nil {
		return "<nil>"
	}
	return p.Name
}

// -------------------- Export helpers -------------------- //
func exportMatchSummaryCSV(res *models.DemoParseResult, subFolder string) error {
	outPath := filepath.Join(subFolder, "match_summary.csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"match_id", "map_name", "match_duration", "result", "team_score", "opponent_score", "filename", "date"}); err != nil {
		return err
	}
	record := []string{
		res.MatchID,
		res.MapName,
		res.Duration,
		res.Result,
		strconv.Itoa(res.TeamScore),
		strconv.Itoa(res.OpponentScore),
		res.Filename,
		res.Date,
	}
	return w.Write(record)
}

func exportPlayersCSV(res *models.DemoParseResult, subFolder string) error {
	outPath := filepath.Join(subFolder, "players.csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"match_id", "steam_id", "name", "team", "kills", "assists", "deaths", "adr", "hs_percentage",
		"kd_ratio", "flash_assists", "avatar", "rank", "position", "distance_traveled", "enemies_flashed",
		"shots_fired", "shots_fired_no_reason", "purchases", "aim_placement", "bad_flash_count", "bad_molotov_count",
		"team_kill_count", "team_damage", "accidental_shots", "double_kills", "triple_kills", "quad_kills", "ace",
		"clutch_wins", "crouch_ticks", "walk_ticks", "run_ticks",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range res.Players {
		record := []string{
			res.MatchID,
			p.SteamID,
			p.Name,
			p.Team,
			strconv.Itoa(p.Kills),
			strconv.Itoa(p.Assists),
			strconv.Itoa(p.Deaths),
			fmt.Sprintf("%.2f", p.ADR),
			fmt.Sprintf("%.2f", p.HSPercentage),
			fmt.Sprintf("%.2f", p.KDRatio),
			strconv.Itoa(p.FlashAssists),
			p.Avatar,
			p.Rank,
			strconv.Itoa(p.Position),
			fmt.Sprintf("%.2f", p.DistanceTraveled),
			strconv.Itoa(p.EnemiesFlashed),
			strconv.Itoa(p.ShotsFired),
			strconv.Itoa(p.ShotsFiredNoReason),
			strconv.Itoa(p.Purchases),
			fmt.Sprintf("%.2f", p.AimPlacement),
			strconv.Itoa(p.BadFlashCount),
			strconv.Itoa(p.BadMolotovCount),
			strconv.Itoa(p.TeamKillCount),
			strconv.Itoa(p.TeamDamage),
			strconv.Itoa(p.AccidentalShots),
			strconv.Itoa(p.DoubleKills),
			strconv.Itoa(p.TripleKills),
			strconv.Itoa(p.QuadKills),
			strconv.Itoa(p.Ace),
			strconv.Itoa(p.ClutchWins),
			strconv.Itoa(p.CrouchTicks),
			strconv.Itoa(p.WalkTicks),
			strconv.Itoa(p.RunTicks),
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func exportMovementCSV(res *models.DemoParseResult, subFolder string) error {
	outPath := filepath.Join(subFolder, "movement_logs.csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"match_id", "steam_id", "tick", "x_position", "y_position", "z_position", "speed", "is_ducking"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range res.Players {
		for _, mv := range p.Movement {
			record := []string{
				res.MatchID,
				p.SteamID,
				strconv.Itoa(mv.Tick),
				fmt.Sprintf("%.2f", mv.X),
				fmt.Sprintf("%.2f", mv.Y),
				fmt.Sprintf("%.2f", mv.Z),
				fmt.Sprintf("%.2f", mv.Speed),
				strconv.FormatBool(mv.IsDucking),
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
	}
	return nil
}

// Export economy => ahora con initial_money y final_money en lugar de money
func exportEconomyCSV(res *models.DemoParseResult, subFolder string) error {
	outPath := filepath.Join(subFolder, "economy_history.csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Cambiamos cabecera: round_number, steam_id, initial_money, final_money, loss_bonus
	if err := w.Write([]string{"match_id", "round_number", "steam_id", "initial_money", "final_money", "loss_bonus"}); err != nil {
		return err
	}

	for _, econ := range res.EconomyHistory {
		record := []string{
			res.MatchID,
			strconv.Itoa(econ.RoundNumber),
			econ.SteamID,
			strconv.Itoa(econ.InitialMoney),
			strconv.Itoa(econ.FinalMoney),
			strconv.Itoa(econ.LossBonus),
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func exportEventsCSV(res *models.DemoParseResult, subFolder string) error {
	outPath := filepath.Join(subFolder, "event_logs.csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"match_id", "round", "timestamp", "event_type", "details"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, ev := range res.EventLogs {
		record := []string{
			res.MatchID,
			strconv.Itoa(ev.Round),
			strconv.Itoa(ev.Timestamp),
			ev.EventType,
			ev.Details,
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}
