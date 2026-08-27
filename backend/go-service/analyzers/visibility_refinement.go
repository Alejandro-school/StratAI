package analyzers

import (
	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

type visibilityFrame struct {
	tick    int
	players map[uint64]visibilityPlayerState
	smokes  map[int64]r3.Vector
}

type visibilityHistory struct {
	capacity int
	frames   []visibilityFrame
}

func newVisibilityHistory(capacity int) *visibilityHistory {
	return &visibilityHistory{capacity: capacity, frames: make([]visibilityFrame, 0, capacity)}
}

func (history *visibilityHistory) add(frame visibilityFrame) {
	if len(history.frames) > 0 && history.frames[len(history.frames)-1].tick == frame.tick {
		history.frames[len(history.frames)-1] = frame
		return
	}
	history.frames = append(history.frames, frame)
	if len(history.frames) > history.capacity {
		history.frames = history.frames[len(history.frames)-history.capacity:]
	}
}

func (history *visibilityHistory) clear() {
	history.frames = history.frames[:0]
}

func captureVisibilityFrame(ctx *models.DemoContext, tick int) visibilityFrame {
	players := ctx.Parser.GameState().Participants().Playing()
	frame := visibilityFrame{
		tick:    tick,
		players: make(map[uint64]visibilityPlayerState, len(players)),
		smokes:  make(map[int64]r3.Vector, len(ctx.ActiveSmokes)),
	}
	for _, player := range players {
		if player == nil || player.SteamID64 == 0 {
			continue
		}
		frame.players[player.SteamID64] = visibilityStateFromPlayer(ctx, player, tick)
	}
	for id, position := range ctx.ActiveSmokes {
		frame.smokes[id] = position
	}
	return frame
}

func refineShotVisibility(ctx *models.DemoContext, history *visibilityHistory, shooterID uint64, tick int) {
	if !canRefineVisibility(ctx) {
		return
	}
	current := captureVisibilityFrame(ctx, tick)
	shooter, ok := current.players[shooterID]
	if !ok || !shooter.alive {
		return
	}
	for enemyID, enemy := range current.players {
		if enemy.alive && enemy.team != shooter.team {
			refineVisibilityPair(ctx, history, current, shooterID, enemyID, false)
		}
	}
}

func refineDamageVisibility(ctx *models.DemoContext, history *visibilityHistory, attackerID, victimID uint64, tick int) {
	if !canRefineVisibility(ctx) {
		return
	}
	current := captureVisibilityFrame(ctx, tick)
	victim, ok := current.players[victimID]
	if !ok {
		return
	}
	victim.alive = true
	current.players[victimID] = victim
	refineVisibilityPair(ctx, history, current, attackerID, victimID, true)
}

func canRefineVisibility(ctx *models.DemoContext) bool {
	return ctx.InRound && ctx.ActualRoundNumber > 0 && ctx.MapManager != nil && ctx.MapManager.IsLoaded()
}

func refineVisibilityPair(ctx *models.DemoContext, history *visibilityHistory, current visibilityFrame, shooterID, enemyID uint64, allowDeadEnemy bool) {
	ensureVisibilityMaps(ctx, shooterID)
	if data, ok := ctx.EnemyFirstSeenTick[shooterID][enemyID]; ok &&
		ctx.LastVisibleEnemies[shooterID][enemyID] && data.LastSeenTick == current.tick {
		return
	}

	isVisible, raycasts := isVisibleInFrame(ctx, current, shooterID, enemyID, allowDeadEnemy)
	recordRefinementRaycasts(ctx, raycasts)
	if !isVisible {
		ctx.LastVisibleEnemies[shooterID][enemyID] = false
		return
	}
	recordVisiblePair(ctx, history, current, shooterID, enemyID)
}

func recordVisiblePair(ctx *models.DemoContext, history *visibilityHistory, current visibilityFrame, shooterID, enemyID uint64) {
	ensureVisibilityMaps(ctx, shooterID)
	if data, ok := ctx.EnemyFirstSeenTick[shooterID][enemyID]; ok {
		if ctx.LastVisibleEnemies[shooterID][enemyID] || current.tick-data.LastSeenTick < jiggleGraceTicks(ctx) {
			data.LastSeenTick = current.tick
			ctx.EnemyFirstSeenTick[shooterID][enemyID] = data
			ctx.LastVisibleEnemies[shooterID][enemyID] = true
			return
		}
	}

	firstVisible := current
	expectedTick := current.tick - 1
	index := len(history.frames) - 1
	if index >= 0 && history.frames[index].tick == current.tick {
		index--
	}
	for ; index >= 0 && current.tick-expectedTick < visibilitySampleStride; index-- {
		frame := history.frames[index]
		if frame.tick != expectedTick {
			break
		}
		visible, historicalRaycasts := isVisibleInFrame(ctx, frame, shooterID, enemyID, false)
		recordRefinementRaycasts(ctx, historicalRaycasts)
		if !visible {
			break
		}
		firstVisible = frame
		expectedTick--
	}

	shooter := firstVisible.players[shooterID]
	enemy := firstVisible.players[enemyID]
	data := firstSeenDataFromStates(shooter, enemy, firstVisible.tick)
	data.LastSeenTick = current.tick
	ctx.EnemyFirstSeenTick[shooterID][enemyID] = data
	ctx.LastVisibleEnemies[shooterID][enemyID] = true
}

func isVisibleInFrame(ctx *models.DemoContext, frame visibilityFrame, shooterID, enemyID uint64, allowDeadEnemy bool) (bool, int) {
	shooter, shooterExists := frame.players[shooterID]
	enemy, enemyExists := frame.players[enemyID]
	if !shooterExists || !enemyExists || !shooter.alive || shooter.blinded || shooter.team == enemy.team {
		return false, 0
	}
	if !enemy.alive && !allowDeadEnemy {
		return false, 0
	}
	if !isInsideVisibilityFrustum(shooter, enemy.head, enemy.chest) {
		return false, 0
	}

	raycasts := 0
	if !smokeBlocksSight(frame.smokes, shooter.eyes, enemy.head) {
		raycasts++
		if ctx.MapManager.IsVisible(shooter.eyes, enemy.head) {
			return true, raycasts
		}
	}
	if !smokeBlocksSight(frame.smokes, shooter.eyes, enemy.chest) {
		raycasts++
		if ctx.MapManager.IsVisible(shooter.eyes, enemy.chest) {
			return true, raycasts
		}
	}
	return false, raycasts
}

func ensureVisibilityMaps(ctx *models.DemoContext, shooterID uint64) {
	if ctx.EnemyFirstSeenTick[shooterID] == nil {
		ctx.EnemyFirstSeenTick[shooterID] = make(map[uint64]models.FirstSeenData)
	}
	if ctx.LastVisibleEnemies[shooterID] == nil {
		ctx.LastVisibleEnemies[shooterID] = make(map[uint64]bool)
	}
}

func recordRefinementRaycasts(ctx *models.DemoContext, count int) {
	ctx.VisibilityRaycasts += count
	ctx.VisibilityRefinementRaycasts += count
}

func jiggleGraceTicks(ctx *models.DemoContext) int {
	tickRate := ctx.Parser.TickRate()
	if tickRate <= 0 {
		tickRate = 64
	}
	return int(0.5 * tickRate)
}
