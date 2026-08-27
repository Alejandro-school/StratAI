package analyzers

import (
	"cs2-demo-service/models"
	"math"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestAnglesToR3VectorUsesSourcePitchSign(t *testing.T) {
	lookingDown := anglesToR3Vector(30, 0)
	lookingUp := anglesToR3Vector(330, 0)

	if math.Abs(lookingDown.Z+0.5) > 1e-9 {
		t.Fatalf("positive Source pitch must point down, got Z=%f", lookingDown.Z)
	}
	if math.Abs(lookingUp.Z-0.5) > 1e-9 {
		t.Fatalf("wrapped negative Source pitch must point up, got Z=%f", lookingUp.Z)
	}
}

func TestVisibilityFOVUsesRectangularCanonicalFrustum(t *testing.T) {
	state := visibilityPlayerState{}
	frustum := playerViewFrustum(state)
	pointAtAngles := func(horizontal, vertical, distance float64) r3.Vector {
		return r3.Vector{
			X: distance,
			Y: math.Tan(degreesToRadians(horizontal)) * distance,
			Z: math.Tan(degreesToRadians(vertical)) * distance,
		}
	}

	if !frustum.contains(state.eyes, pointAtAngles(53, 36, 5000)) {
		t.Fatal("a distant target inside both screen axes must remain visible")
	}
	if frustum.contains(state.eyes, pointAtAngles(54, 0, 1000)) {
		t.Fatal("a target outside the horizontal screen edge must be rejected")
	}
	if frustum.contains(state.eyes, pointAtAngles(0, 37, 1000)) {
		t.Fatal("a target outside the vertical screen edge must be rejected")
	}
	if frustum.contains(state.eyes, r3.Vector{X: -1000}) {
		t.Fatal("a target behind the player must be rejected")
	}
}

func TestVisibilityFOVUsesWeaponZoomLevel(t *testing.T) {
	state := visibilityPlayerState{isScoped: true, weaponType: common.EqAWP, zoomLevel: 1}
	firstZoom := playerViewFrustum(state)
	insideFirstZoom := anglesToR3Vector(0, 25).Mul(1000)
	outsideFirstZoom := anglesToR3Vector(0, 26).Mul(1000)
	if !firstZoom.contains(state.eyes, insideFirstZoom) || firstZoom.contains(state.eyes, outsideFirstZoom) {
		t.Fatal("AWP first zoom must use its scoped field of view")
	}

	state.zoomLevel = 2
	secondZoom := playerViewFrustum(state)
	insideSecondZoom := anglesToR3Vector(0, 6).Mul(1000)
	outsideSecondZoom := anglesToR3Vector(0, 7).Mul(1000)
	if !secondZoom.contains(state.eyes, insideSecondZoom) || secondZoom.contains(state.eyes, outsideSecondZoom) {
		t.Fatal("AWP second zoom must use its narrower field of view")
	}

	corner := r3.Vector{
		X: 1000,
		Y: firstZoom.tanHalfHorizontal * 800,
		Z: firstZoom.tanHalfVertical * 800,
	}
	if firstZoom.contains(state.eyes, corner) {
		t.Fatal("sniper scope must reject points hidden by the circular mask")
	}
}

func TestVisibilityHistoryKeepsOnlyRefinementWindow(t *testing.T) {
	history := newVisibilityHistory(visibilitySampleStride)
	for tick := 1; tick <= visibilitySampleStride+2; tick++ {
		history.add(visibilityFrame{tick: tick})
	}
	if len(history.frames) != visibilitySampleStride || history.frames[0].tick != 3 {
		t.Fatalf("unexpected history window: %+v", history.frames)
	}
}

func TestFirstSeenDataPreservesVelocityAvailability(t *testing.T) {
	shooter := visibilityPlayerState{
		eyes:       r3.Vector{},
		pitch:      0,
		yaw:        0,
		speed:      125,
		speedKnown: true,
	}
	enemy := visibilityPlayerState{
		head:  r3.Vector{X: 100},
		chest: r3.Vector{X: 100},
	}

	data := firstSeenDataFromStates(shooter, enemy, 42)
	if !data.ShooterVelocityAvailable || data.ShooterVelocity != 125 {
		t.Fatalf("velocity availability was lost: %+v", data)
	}

	shooter.speed = 0
	shooter.speedKnown = false
	data = firstSeenDataFromStates(shooter, enemy, 43)
	if data.ShooterVelocityAvailable {
		t.Fatal("unknown velocity must not be classified as stationary")
	}
}

func TestSmokeBlocksSightOnlyForDeployedSmokePositions(t *testing.T) {
	lineStart := r3.Vector{}
	lineEnd := r3.Vector{X: 1000}
	if !smokeBlocksSight(map[int64]r3.Vector{1: {X: 500, Y: 100}}, lineStart, lineEnd) {
		t.Fatal("smoke inside its radius should block sight")
	}
	if smokeBlocksSight(map[int64]r3.Vector{1: {X: 500, Y: 200}}, lineStart, lineEnd) {
		t.Fatal("smoke outside its radius should not block sight")
	}
}

func TestReactionDistanceHandlesZeroLengthSegment(t *testing.T) {
	if got := distancePointToSegment(r3.Vector{X: 3, Y: 4}, r3.Vector{}, r3.Vector{}); got != 5 {
		t.Fatalf("expected 5, got %f", got)
	}
}

func TestSelectReactionTargetUsesSteamIDAsExactTieBreaker(t *testing.T) {
	buildFirstSeen := func(order []uint64) map[uint64]models.FirstSeenData {
		result := make(map[uint64]models.FirstSeenData, len(order))
		for _, enemyID := range order {
			result[enemyID] = models.FirstSeenData{CrosshairPlacementError: 1.25}
		}
		return result
	}
	visible := map[uint64]bool{7: true, 42: true}

	for _, order := range [][]uint64{{42, 7}, {7, 42}} {
		target, found := selectReactionTarget(buildFirstSeen(order), visible)
		if !found || target != 7 {
			t.Fatalf("order %v selected target %d, found=%v", order, target, found)
		}
	}
}

func TestSelectReactionTargetPreservesEligibilityAndErrorPriority(t *testing.T) {
	firstSeen := map[uint64]models.FirstSeenData{
		3:  {CrosshairPlacementError: 0.5, FirstShotTick: 10},
		5:  {CrosshairPlacementError: 0.75},
		10: {CrosshairPlacementError: 0.25},
	}
	visible := map[uint64]bool{3: true, 5: true, 10: false}

	target, found := selectReactionTarget(firstSeen, visible)
	if !found || target != 5 {
		t.Fatalf("selected target %d, found=%v", target, found)
	}

	if _, found := selectReactionTarget(firstSeen, map[uint64]bool{}); found {
		t.Fatal("selected a target when no candidate was eligible")
	}
}

func TestSortVisibilityResultsUsesShooterAndEnemyIDs(t *testing.T) {
	results := []visibilityResult{
		{shooterID: 9, enemyID: 7, isVisible: true, raycasts: 1},
		{shooterID: 2, enemyID: 8, isVisible: false, raycasts: 2},
		{shooterID: 2, enemyID: 3, isVisible: true, raycasts: 2},
		{shooterID: 9, enemyID: 1, isVisible: false, raycasts: 1},
	}
	expected := []visibilityResult{
		{shooterID: 2, enemyID: 3, isVisible: true, raycasts: 2},
		{shooterID: 2, enemyID: 8, isVisible: false, raycasts: 2},
		{shooterID: 9, enemyID: 1, isVisible: false, raycasts: 1},
		{shooterID: 9, enemyID: 7, isVisible: true, raycasts: 1},
	}

	sortVisibilityResults(results)
	for index := range expected {
		if results[index] != expected[index] {
			t.Fatalf("result[%d] = %+v, expected %+v", index, results[index], expected[index])
		}
	}
}
