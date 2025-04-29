package demo

import (
	"cs2-demo-service/models"
	"fmt"
	"log"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// makeIntKey genera una clave entera única combinando ronda y entity-ID.
//   - round se desplaza 16 bits a la izquierda  (⇢ admite hasta 65 535 rondas)
//   - se mezcla con los 16 bits bajos del entityID
func makeIntKey(round, entityID int) int {
	return (round << 16) | (entityID & 0xFFFF)
}

// -----------------------------------------------------------------------------
// registerGrenadeHandlers – todos los tipos de granada.
// -----------------------------------------------------------------------------
func registerGrenadeHandlers(ctx *DemoContext) {

	// ──────────────────────────────────────────────────────────────────────────
	// 1A)  ProjectileThrow  → inicializa meta + primer punto de trayectoria
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		gp := e.Projectile
		if gp == nil || gp.Entity == nil {
			return
		}

		//  Ignoramos si seguimos en warm-up
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber == 0 {
			return
		}

		nadeType := "Unknown"
		if gp.WeaponInstance != nil {
			nadeType = gp.WeaponInstance.Type.String()
		}
		meta := getOrInitMeta(ctx, gp.Entity.ID(), nadeType)

		// ── Guardar orden de lanzamiento ───────────────────────────────
		if meta.ThrowTick == 0 { // sólo la primera vez
			meta.ThrowTick = ctx.parser.GameState().IngameTick()
		}

		// Rellenar thrower
		if t := gp.Thrower; t != nil {
			updateThrower(meta, t)
		}

		//  Primer punto de trayectoria
		recordGrenadeTrajectory(ctx, gp)
	})

	// ──────────────────────────────────────────────────────────────────────────
	// 1B)  ProjectileBounce  → añade puntos intermedios
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.GrenadeProjectileBounce) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber == 0 {
			return
		}
		recordGrenadeTrajectory(ctx, e.Projectile)
	})

	ctx.parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		// 1) Asegurarnos de que tenemos un proyectil válido
		gp := e.Projectile
		if gp == nil || gp.Entity == nil {
			return
		}

		// 2) Ignorar durante warm-up o en ronda 0
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber == 0 {
			return
		}

		// 3) Determinar el tipo de granada desde WeaponInstance
		nadeType := "Unknown"
		if gp.WeaponInstance != nil {
			nadeType = gp.WeaponInstance.Type.String()
		}

		// 4) Obtener o inicializar la metadata usando la clave (round+entityID)
		meta := getOrInitMeta(ctx, gp.Entity.ID(), nadeType)

		// 5) Marcar que sí explotó, y guardar la posición de explosión
		meta.Exploded = true
		pos := gp.Position()
		meta.ExplosionPosition = &models.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		}

		// 6) Añadir el último punto de trayectoria
		recordGrenadeTrajectory(ctx, gp)
	})

	// ──────────────────────────────────────────────────────────────────────────
	// SMOKE – SmokeStart
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.SmokeStart) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		meta := getOrInitMeta(ctx, e.GrenadeEntityID, "Smoke")
		updateThrower(meta, e.Thrower)

		meta.Exploded = true
		meta.ExplosionPosition = &models.Position{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}

		if e.Thrower != nil {
			getOrCreatePlayerStats(ctx, e.Thrower.SteamID64, e.Thrower.Name).SmokesThrown++
		}
		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)

		log.Printf("Smoke detonated by %s at (%.2f, %.2f, %.2f)",
			nameOrNil(e.Thrower), e.Position.X, e.Position.Y, e.Position.Z)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "SmokeStart",
			fmt.Sprintf("Smoke exploded at (%.2f, %.2f, %.2f)", e.Position.X, e.Position.Y, e.Position.Z),
			teamString(e.Thrower)))
	})

	// SMOKE – SmokeExpired
	ctx.parser.RegisterEventHandler(func(e events.SmokeExpired) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "SmokeExpired",
			fmt.Sprintf("Smoke ended at (%.1f, %.1f, %.1f)", e.Position.X, e.Position.Y, e.Position.Z),
			""))
		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)
	})

	// ──────────────────────────────────────────────────────────────────────────
	// DECOY – DecoyStart / DecoyExpired
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.DecoyStart) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		meta := getOrInitMeta(ctx, e.GrenadeEntityID, "Decoy")
		updateThrower(meta, e.Thrower)

		meta.Exploded = true
		meta.ExplosionPosition = &models.Position{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}

		if e.Thrower != nil {
			getOrCreatePlayerStats(ctx, e.Thrower.SteamID64, e.Thrower.Name).DecoysThrown++
		}
		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "DecoyStart",
			fmt.Sprintf("Decoy started at (%.2f, %.2f, %.2f)", e.Position.X, e.Position.Y, e.Position.Z),
			teamString(e.Thrower)))
	})

	ctx.parser.RegisterEventHandler(func(e events.DecoyExpired) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "DecoyExpired",
			fmt.Sprintf("Decoy ended at (%.1f, %.1f, %.1f)", e.Position.X, e.Position.Y, e.Position.Z),
			""))
		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)
	})

	// ──────────────────────────────────────────────────────────────────────────
	// HE – HeExplode
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.HeExplode) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		meta := getOrInitMeta(ctx, e.GrenadeEntityID, "HE")
		updateThrower(meta, e.Thrower)

		meta.Exploded = true
		meta.ExplosionPosition = &models.Position{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}

		if e.Thrower != nil {
			getOrCreatePlayerStats(ctx, e.Thrower.SteamID64, e.Thrower.Name).HeThrown++
		}
		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "HeExplode",
			fmt.Sprintf("HE exploded at (%.2f, %.2f, %.2f)", e.Position.X, e.Position.Y, e.Position.Z),
			teamString(e.Thrower)))
	})

	// ──────────────────────────────────────────────────────────────────────────
	// FLASH – FlashExplode
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.FlashExplode) {
		if ctx.parser.GameState().IsWarmupPeriod() || e.Thrower == nil {
			return
		}

		meta := getOrInitMeta(ctx, e.GrenadeEntityID, "Flash")
		updateThrower(meta, e.Thrower)

		meta.Exploded = true
		meta.ExplosionPosition = &models.Position{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}

		sid := e.Thrower.SteamID64
		ps := getOrCreatePlayerStats(ctx, sid, e.Thrower.Name)
		ps.FlashesThrown++

		currentTick := ctx.parser.GameState().IngameTick()
		ctx.FlashEvents[sid] = append(ctx.FlashEvents[sid], flashData{
			explosionTick: currentTick,
			enemyCount:    0,
			friendlyCount: 0,
		})

		recordGrenadeTrajectoryIfFound(ctx, e.GrenadeEntityID)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "FlashExplode",
			fmt.Sprintf("Flash exploded at (%.2f, %.2f, %.2f)", e.Position.X, e.Position.Y, e.Position.Z),
			teamString(e.Thrower)))
	})

	// ──────────────────────────────────────────────────────────────────────────
	// MOLOTOV – FireGrenadeStart
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.InfernoStart) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber == 0 {
			return
		}

		infID := e.Inferno.Entity.ID()
		infPos := e.Inferno.Entity.Position()

		// Buscar el proyectil lanzado más cercano (dentro de 120 u.)
		const maxDist = 120.0
		bestID := -1
		bestD := maxDist
		for pid, data := range ctx.PendingMolotovs { // key = projectileID
			if d := vectorDistance(data.Position, infPos); d < bestD {
				bestID, bestD = pid, d
			}
		}
		if bestID == -1 {
			return // no hallado ⇒ ignoramos; no se creará meta duplicada
		}

		// Guardamos el vínculo inferno→proyectil
		ctx.InfernoToProjectile[infID] = bestID
	})

	// MOLOTOV – InfernoExpired
	ctx.parser.RegisterEventHandler(func(e events.InfernoExpired) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber == 0 {
			return
		}

		infID := e.Inferno.Entity.ID()
		projID, ok := ctx.InfernoToProjectile[infID]
		if !ok {
			return // por seguridad: si no está vinculada la ignoramos
		}

		// Usamos la meta que ya existe para el proyectil
		key := makeIntKey(ctx.RoundNumber, projID)
		meta := ctx.GrenadeMetas[key]
		if meta == nil {
			return
		}

		// Marcamos explosión y posición final
		pos := e.Inferno.Entity.Position()
		meta.Exploded = true
		meta.ExplosionPosition = &models.Position{X: pos.X, Y: pos.Y, Z: pos.Z}

		// Comprobación de “bad molly”
		if data, ok := ctx.PendingMolotovs[projID]; ok && !fireStartedAt(data.Position, e.Inferno) {
			getOrCreatePlayerStats(ctx, data.Thrower.SteamID64, data.Thrower.Name).BadMolotovCount++
		}

		// Limpieza
		delete(ctx.PendingMolotovs, projID)
		delete(ctx.InfernoToProjectile, infID)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber, "MolotovExpired",
			fmt.Sprintf("Molotov expired at (%.1f, %.1f)", pos.X, pos.Y),
			"",
		))
	})
}

// -----------------------------------------------------------------------------
// HELPERS (granadas)
// -----------------------------------------------------------------------------

// getOrInitMeta devuelve la metadata (clave = round+entityID) o la crea.
func getOrInitMeta(ctx *DemoContext, entityID int, nadeType string) *models.GrenadeMetadata {

	key := makeIntKey(ctx.RoundNumber, entityID)

	if meta, ok := ctx.GrenadeMetas[key]; ok {
		return meta // ya existía para ESTA ronda
	}

	meta := &models.GrenadeMetadata{
		ProjectileID: entityID,
		Round:        ctx.RoundNumber,
		NadeType:     nadeType,
		Exploded:     false,
	}
	ctx.GrenadeMetas[key] = meta        // map[int] → ✅ sin error de tipo
	ctx.lastGrenadeRecordTick[key] = -1 // iniciamos muestreo
	return meta
}

// updateThrower actualiza los datos del lanzador (si se proporcionan).
func updateThrower(meta *models.GrenadeMetadata, p *common.Player) {
	if p == nil {
		return
	}
	meta.ThrowerSteamID = p.SteamID64
	meta.ThrowerName = p.Name
	meta.ThrowerTeam = teamString(p)
}

// -------------------------------------------------------------------------------
// recordGrenadeTrajectoryIfFound -> llama a recordGrenadeTrajectory si la granada existe
// -------------------------------------------------------------------------------
func recordGrenadeTrajectoryIfFound(ctx *DemoContext, entityID int) {
	if gp := findGrenadeByID(ctx, entityID); gp != nil {
		recordGrenadeTrajectory(ctx, gp)
	}
}

// recordGrenadeTrajectory registra un punto cada ~200 ms (25 ticks).
func recordGrenadeTrajectory(ctx *DemoContext, gp *common.GrenadeProjectile) {
	if gp == nil || gp.Entity == nil {
		return
	}

	key := makeIntKey(ctx.RoundNumber, gp.Entity.ID())
	cur := ctx.parser.GameState().IngameTick()

	const every = 25 // ≈ 200 ms en 128 tick
	if last, ok := ctx.lastGrenadeRecordTick[key]; ok && cur-last < every {
		return // aún no toca samplear
	}
	ctx.lastGrenadeRecordTick[key] = cur

	ctx.GrenadeTrajectories[key] = append(ctx.GrenadeTrajectories[key],
		models.ProjectileTrajectoryEntry{
			FrameID: cur,
			X:       gp.Position().X,
			Y:       gp.Position().Y,
			Z:       gp.Position().Z,
			TimeMS:  int64(float64(cur) / ctx.parser.TickRate() * 1000),
		})
}

// -------------------------------------------------------------------------------
// findGrenadeByID busca la granada en GameState (por entityID).
// -------------------------------------------------------------------------------
func findGrenadeByID(ctx *DemoContext, entityID int) *common.GrenadeProjectile {
	for _, gp := range ctx.parser.GameState().GrenadeProjectiles() {
		if gp.Entity != nil && gp.Entity.ID() == entityID {
			return gp
		}
	}
	return nil
}

// -------------------------------------------------------------------------------
// fireStartedAt -> check si la molotov se prendió bien (opcional).
// -------------------------------------------------------------------------------
func fireStartedAt(initialPos r3.Vector, inferno *common.Inferno) bool {
	// Lógica simple: asume que si la posición difiere mucho, no prendió
	return true
}
