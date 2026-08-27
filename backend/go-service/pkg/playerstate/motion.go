package playerstate

import (
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	defaultTickRate = 64.0

	MaxPlausibleHorizontalSpeedUPS = 2000.0
	MaxPlausibleVerticalSpeedUPS   = 4000.0
	MaxPositionDeltaIntervalTicks  = 8
)

type VelocitySource string

const (
	VelocitySourceNative              VelocitySource = "native"
	VelocitySourcePositionDelta       VelocitySource = "position_delta"
	VelocitySourceInsufficientHistory VelocitySource = "insufficient_history"
	VelocitySourceNotApplicable       VelocitySource = "not_applicable"
	VelocitySourceRejected            VelocitySource = "position_delta_rejected"
	VelocitySourceStaleGap            VelocitySource = "stale_gap"
	VelocitySourceEntityChanged       VelocitySource = "entity_changed"
	VelocitySourceNonMonotonicTick    VelocitySource = "non_monotonic_tick"
)

type MotionObservation struct {
	PlayerID       uint64
	PawnID         int
	PawnSerial     int
	Round          int
	Tick           int
	TickRate       float64
	Position       r3.Vector
	IsAlive        bool
	NativeVelocity *r3.Vector
}

type MotionEstimate struct {
	Vector        r3.Vector
	Source        VelocitySource
	IntervalTicks int
	ObservedTick  int
	Available     bool
}

func (estimate MotionEstimate) HorizontalSpeed() float64 {
	return math.Hypot(estimate.Vector.X, estimate.Vector.Y)
}

func (estimate MotionEstimate) Speed3D() float64 {
	return math.Sqrt(
		estimate.Vector.X*estimate.Vector.X +
			estimate.Vector.Y*estimate.Vector.Y +
			estimate.Vector.Z*estimate.Vector.Z,
	)
}

type motionSample struct {
	round      int
	tick       int
	pawnID     int
	pawnSerial int
	position   r3.Vector
	estimate   MotionEstimate
}

type MotionTracker struct {
	samples   map[uint64]motionSample
	lastAlive map[uint64]motionSample
}

func (tracker *MotionTracker) ObservePlayer(
	player *common.Player,
	round int,
	tick int,
	tickRate float64,
) MotionEstimate {
	if player == nil {
		return unavailableEstimate(VelocitySourceNotApplicable)
	}

	nativeVelocity, hasNativeVelocity := NativeVelocity(player)
	observation := MotionObservation{
		PlayerID: player.SteamID64,
		Round:    round,
		Tick:     tick,
		TickRate: tickRate,
		Position: player.Position(),
		IsAlive:  player.IsAlive(),
	}
	if pawn := player.PlayerPawnEntity(); pawn != nil {
		observation.PawnID = pawn.ID()
		observation.PawnSerial = pawn.SerialNum()
	}
	if hasNativeVelocity {
		observation.NativeVelocity = &nativeVelocity
	}
	return tracker.Observe(observation)
}

func (tracker *MotionTracker) Observe(observation MotionObservation) MotionEstimate {
	if observation.PlayerID == 0 || observation.Round <= 0 || observation.Tick < 0 {
		return estimateAt(unavailableEstimate(VelocitySourceInsufficientHistory), observation.Tick)
	}
	tracker.ensureSamples()

	if !observation.IsAlive {
		delete(tracker.samples, observation.PlayerID)
		return estimateAt(unavailableEstimate(VelocitySourceNotApplicable), observation.Tick)
	}
	if !isFiniteVector(observation.Position) {
		return estimateAt(unavailableEstimate(VelocitySourceRejected), observation.Tick)
	}

	previous, hasPrevious := tracker.samples[observation.PlayerID]
	if hasPrevious && previous.round == observation.Round && entityChanged(previous, observation) {
		estimate := estimateAt(unavailableEstimate(VelocitySourceEntityChanged), observation.Tick)
		tracker.store(observation, estimate)
		return estimate
	}
	if hasPrevious && previous.round == observation.Round && previous.tick == observation.Tick {
		return previous.estimate
	}
	if hasPrevious && previous.round == observation.Round && observation.Tick < previous.tick {
		return estimateAt(unavailableEstimate(VelocitySourceNonMonotonicTick), observation.Tick)
	}
	if hasPrevious && previous.round == observation.Round && observation.Tick-previous.tick > MaxPositionDeltaIntervalTicks {
		estimate := estimateAt(unavailableEstimate(VelocitySourceStaleGap), observation.Tick)
		tracker.store(observation, estimate)
		return estimate
	}

	positionEstimate, hasPositionEstimate := estimateFromPrevious(previous, hasPrevious, observation)
	if hasPositionEstimate {
		tracker.store(observation, positionEstimate)
		return positionEstimate
	}

	if observation.NativeVelocity != nil &&
		isFiniteVector(*observation.NativeVelocity) &&
		isPlausibleVelocity(*observation.NativeVelocity) &&
		!isZeroVector(*observation.NativeVelocity) {
		estimate := MotionEstimate{
			Vector:       *observation.NativeVelocity,
			Source:       VelocitySourceNative,
			ObservedTick: observation.Tick,
			Available:    true,
		}
		tracker.store(observation, estimate)
		return estimate
	}

	estimate := estimateAt(unavailableEstimate(VelocitySourceInsufficientHistory), observation.Tick)
	tracker.store(observation, estimate)
	return estimate
}

func (tracker *MotionTracker) ensureSamples() {
	if tracker.samples == nil {
		tracker.samples = make(map[uint64]motionSample, 16)
	}
	if tracker.lastAlive == nil {
		tracker.lastAlive = make(map[uint64]motionSample, 16)
	}
}

func (tracker *MotionTracker) store(observation MotionObservation, estimate MotionEstimate) {
	tracker.samples[observation.PlayerID] = motionSample{
		round:      observation.Round,
		tick:       observation.Tick,
		pawnID:     observation.PawnID,
		pawnSerial: observation.PawnSerial,
		position:   observation.Position,
		estimate:   estimate,
	}
	if estimate.Available {
		tracker.lastAlive[observation.PlayerID] = tracker.samples[observation.PlayerID]
	}
}

func (tracker *MotionTracker) Reset() {
	tracker.samples = make(map[uint64]motionSample, 16)
	tracker.lastAlive = make(map[uint64]motionSample, 16)
}

func (tracker *MotionTracker) LastAlive(playerID uint64, round int) (MotionEstimate, bool) {
	tracker.ensureSamples()
	sample, exists := tracker.lastAlive[playerID]
	if !exists || sample.round != round || !sample.estimate.Available {
		return MotionEstimate{}, false
	}
	return sample.estimate, true
}

func estimateFromPrevious(
	previous motionSample,
	hasPrevious bool,
	current MotionObservation,
) (MotionEstimate, bool) {
	if !hasPrevious || previous.round != current.Round || current.Tick <= previous.tick {
		return MotionEstimate{}, false
	}
	tickRate := current.TickRate
	if tickRate <= 0 {
		tickRate = defaultTickRate
	}
	intervalTicks := current.Tick - previous.tick
	scale := tickRate / float64(intervalTicks)
	velocity := r3.Vector{
		X: (current.Position.X - previous.position.X) * scale,
		Y: (current.Position.Y - previous.position.Y) * scale,
		Z: (current.Position.Z - previous.position.Z) * scale,
	}
	if !isFiniteVector(velocity) || !isPlausibleVelocity(velocity) {
		return unavailableEstimate(VelocitySourceRejected), true
	}
	return MotionEstimate{
		Vector:        velocity,
		Source:        VelocitySourcePositionDelta,
		IntervalTicks: intervalTicks,
		ObservedTick:  current.Tick,
		Available:     true,
	}, true
}

func unavailableEstimate(source VelocitySource) MotionEstimate {
	return MotionEstimate{Source: source}
}

func estimateAt(estimate MotionEstimate, tick int) MotionEstimate {
	estimate.ObservedTick = tick
	return estimate
}

func isFiniteVector(vector r3.Vector) bool {
	return !math.IsNaN(vector.X) && !math.IsInf(vector.X, 0) &&
		!math.IsNaN(vector.Y) && !math.IsInf(vector.Y, 0) &&
		!math.IsNaN(vector.Z) && !math.IsInf(vector.Z, 0)
}

func isZeroVector(vector r3.Vector) bool {
	return vector.X == 0 && vector.Y == 0 && vector.Z == 0
}

func isPlausibleVelocity(vector r3.Vector) bool {
	return math.Hypot(vector.X, vector.Y) <= MaxPlausibleHorizontalSpeedUPS &&
		math.Abs(vector.Z) <= MaxPlausibleVerticalSpeedUPS
}

func entityChanged(previous motionSample, current MotionObservation) bool {
	if previous.pawnID == 0 || current.PawnID == 0 {
		return false
	}
	return previous.pawnID != current.PawnID || previous.pawnSerial != current.PawnSerial
}
