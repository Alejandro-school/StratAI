package utility

import (
	"fmt"
	"math"
	"slices"
	"sync"
)

const (
	ExtinguishConfirmationWindowTicks = 16
	maxInferredFlightSeconds          = 10.0
	maxInferredPositionDistance       = 384.0
	maxDamageCallbackReorderTicks     = 1
	maxInferredEffectSeconds          = 30.0
)

type entityScope struct {
	round    int
	entityID int
}

type extinguishCandidate struct {
	smokeThrowID string
	tick         int
	position     Vector
	distance     float64
	correlation  Correlation
}

type trajectorySampleKey struct {
	tick     int
	position Vector
	source   string
}

type Tracker struct {
	mu sync.RWMutex

	throws               []Throw
	currentRound         int
	sequenceByRound      map[int]int
	entityGenerations    map[entityScope]int
	projectileByEntity   map[entityScope]int
	effectByEntity       map[entityScope]int
	extinguishCandidates map[int]extinguishCandidate
	trajectorySamples    map[int]map[trajectorySampleKey]struct{}
	pendingDamage        []DamageInput
	diagnostics          Diagnostics
}

func NewTracker() *Tracker {
	return &Tracker{
		sequenceByRound:      make(map[int]int),
		entityGenerations:    make(map[entityScope]int),
		projectileByEntity:   make(map[entityScope]int),
		effectByEntity:       make(map[entityScope]int),
		extinguishCandidates: make(map[int]extinguishCandidate),
		trajectorySamples:    make(map[int]map[trajectorySampleKey]struct{}),
	}
}

func (tracker *Tracker) BeginRound(round int) {
	if round <= 0 {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.currentRound > 0 && tracker.currentRound != round {
		tracker.endRoundLocked(tracker.currentRound)
	}
	tracker.currentRound = round
}

func (tracker *Tracker) EndRound(round int) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.endRoundLocked(round)
	if tracker.currentRound == round {
		tracker.currentRound = 0
	}
}

func (tracker *Tracker) RecordThrow(input ThrowInput) (string, bool) {
	if input.Round <= 0 {
		tracker.mu.Lock()
		tracker.diagnostics.Throws.Observed++
		tracker.diagnostics.Throws.Unmatched++
		tracker.mu.Unlock()
		return "", false
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Throws.Observed++
	tracker.ensureMapsLocked()

	scope := entityScope{round: input.Round, entityID: input.RuntimeEntityID}
	if input.RuntimeEntityID != 0 {
		if index, exists := tracker.projectileByEntity[scope]; exists &&
			tracker.throws[index].Launch.Tick.Tick == input.Launch.Tick.Tick {
			tracker.diagnostics.Throws.Deduplicated++
			return tracker.throws[index].ID, false
		}
	}

	sequence := tracker.sequenceByRound[input.Round] + 1
	tracker.sequenceByRound[input.Round] = sequence
	generation := 0
	if input.RuntimeEntityID != 0 {
		generation = tracker.entityGenerations[scope] + 1
		tracker.entityGenerations[scope] = generation
	}
	entry := newThrow(input, sequence, generation)
	tracker.throws = append(tracker.throws, entry)
	index := len(tracker.throws) - 1
	if input.RuntimeEntityID != 0 {
		tracker.projectileByEntity[scope] = index
	}
	tracker.diagnostics.Throws.ExactCorrelated++
	tracker.reconcilePendingDamageLocked()
	return entry.ID, true
}

func (tracker *Tracker) RecordTrajectoryPosition(round, runtimeEntityID, tick int, position Vector) bool {
	if !isFiniteVector(position) {
		return false
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	index, exists := tracker.projectileByEntity[entityScope{round: round, entityID: runtimeEntityID}]
	if !exists {
		return false
	}
	entry := &tracker.throws[index]
	if entry.Launch.Tick.Status == AvailabilityObserved && tick < entry.Launch.Tick.Tick {
		return false
	}
	if terminalTick, observed := flightTerminalTick(entry.Lifecycle); observed && tick >= terminalTick {
		return false
	}
	if entry.Lifecycle.Destroy.Status == AvailabilityObserved && tick >= entry.Lifecycle.Destroy.Tick {
		return false
	}
	tracker.appendTrajectorySampleLocked(index, tick, position, SourceProjectileFrames)
	tracker.updateTrajectoryStatusLocked(index)
	return true
}

func (tracker *Tracker) RecordBounce(
	round, runtimeEntityID, tick int,
	position Vector,
	positionStatus Availability,
	bounceNumber int,
) bool {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Bounces.Observed++
	if bounceNumber <= 0 {
		tracker.diagnostics.Bounces.Unmatched++
		return false
	}
	index, exists := tracker.projectileByEntity[entityScope{round: round, entityID: runtimeEntityID}]
	if !exists {
		tracker.diagnostics.Bounces.Unmatched++
		return false
	}
	entry := &tracker.throws[index]
	if entry.Launch.Tick.Status == AvailabilityObserved && tick < entry.Launch.Tick.Tick {
		tracker.diagnostics.Bounces.Unmatched++
		return false
	}
	if terminalTick, observed := bounceTerminalTick(entry.Lifecycle); observed && tick >= terminalTick {
		tracker.diagnostics.Bounces.ExactCorrelated++
		return false
	}
	trajectory := &entry.Trajectory
	if positionStatus != AvailabilityObserved || !isFiniteVector(position) {
		position, positionStatus = Vector{}, AvailabilityUnavailable
	}
	for _, bounce := range trajectory.Bounces {
		if bounce.Tick == tick && bounce.Number == bounceNumber {
			tracker.diagnostics.Bounces.Deduplicated++
			return false
		}
	}
	trajectory.Bounces = append(trajectory.Bounces, BounceObservation{
		Tick: tick, Position: position, PositionStatus: positionStatus,
		Number: bounceNumber, Source: SourceProjectileBounce,
	})
	trajectory.BounceCount = len(trajectory.Bounces)
	trajectory.BounceStatus = AvailabilityObserved
	trajectory.BounceSource = SourceProjectileBounce
	tracker.diagnostics.Bounces.ExactCorrelated++
	return true
}

func (tracker *Tracker) RecordDestroy(hint CallbackHint) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Lifecycle.Observed++
	index, correlation := tracker.resolveThrowLocked(hint, true)
	orphaned := index < 0
	if index < 0 {
		index = tracker.recordLifecycleOnlyLocked(hint, SourceProjectileDestroy)
		correlation = Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable}
	}
	entry := &tracker.throws[index]
	tracker.enrichActorLocked(entry, callbackActor(hint))
	if sameObservation(entry.Lifecycle.Destroy, hint.Tick, SourceProjectileDestroy) {
		tracker.diagnostics.Lifecycle.Deduplicated++
		return entry.ID, false
	}
	entry.Lifecycle.Destroy = tickPosition(hint.Tick, hint.Position, hint.PositionStatus, SourceProjectileDestroy)
	_, flightAlreadyEnded := flightTerminalTick(entry.Lifecycle)
	if !flightAlreadyEnded && hint.PositionStatus == AvailabilityObserved {
		tracker.appendTrajectorySampleLocked(index, hint.Tick, hint.Position, SourceProjectileDestroy)
	}
	if terminalTick, observed := bounceTerminalTick(entry.Lifecycle); observed {
		tracker.pruneBouncesAtOrAfterLocked(index, terminalTick)
	}
	tracker.updateTrajectoryStatusLocked(index)
	if entry.Lifecycle.Correlation.Status == CorrelationUnavailable {
		entry.Lifecycle.Correlation = correlation
	}
	tracker.recordCallbackResultLocked(&tracker.diagnostics.Lifecycle, correlation, orphaned)
	return entry.ID, true
}

func (tracker *Tracker) RecordDetonation(hint CallbackHint, source string) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Lifecycle.Observed++
	index, correlation := tracker.resolveThrowLocked(hint, false)
	orphaned := index < 0
	if index < 0 {
		index = tracker.recordLifecycleOnlyLocked(hint, source)
		correlation = Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable}
	}
	entry := &tracker.throws[index]
	tracker.enrichActorLocked(entry, callbackActor(hint))
	if sameObservation(entry.Lifecycle.Detonation, hint.Tick, source) {
		tracker.diagnostics.Lifecycle.Deduplicated++
		return entry.ID, false
	}
	entry.Lifecycle.Detonation = tickPosition(hint.Tick, hint.Position, hint.PositionStatus, source)
	for index := range entry.Flashes {
		entry.Flashes[index].Tick = hint.Tick
	}
	entry.Lifecycle.Area = normalizeStringObservation(hint.Area)
	entry.Lifecycle.Status = LifecycleDetonated
	entry.Lifecycle.Correlation = correlation
	tracker.closeFlightTrajectoryLocked(index)
	tracker.reconcilePendingDamageLocked()
	tracker.recordCallbackResultLocked(&tracker.diagnostics.Lifecycle, correlation, orphaned)
	return entry.ID, true
}

func (tracker *Tracker) StartEffect(input EffectInput) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Lifecycle.Observed++
	index, correlation := tracker.resolveThrowLocked(input.Hint, false)
	orphaned := index < 0
	if index < 0 {
		index = tracker.recordLifecycleOnlyLocked(input.Hint, input.Source)
		correlation = Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable}
	}
	entry := &tracker.throws[index]
	tracker.enrichActorLocked(entry, callbackActor(input.Hint))
	if sameObservation(entry.Lifecycle.EffectStart, input.Hint.Tick, input.Source) {
		tracker.diagnostics.Lifecycle.Deduplicated++
		return entry.ID, false
	}
	observation := tickPosition(
		input.Hint.Tick,
		input.Hint.Position,
		input.Hint.PositionStatus,
		input.Source,
	)
	entry.Lifecycle.Detonation = observation
	entry.Lifecycle.EffectStart = observation
	entry.Lifecycle.Area = normalizeStringObservation(input.Hint.Area)
	entry.Lifecycle.Status = LifecycleEffectActive
	entry.Lifecycle.Correlation = correlation
	tracker.closeFlightTrajectoryLocked(index)
	if input.RuntimeEffectEntityID != 0 {
		tracker.effectByEntity[entityScope{
			round:    input.Hint.Round,
			entityID: input.RuntimeEffectEntityID,
		}] = index
	}
	tracker.reconcilePendingDamageLocked()
	tracker.recordCallbackResultLocked(&tracker.diagnostics.Lifecycle, correlation, orphaned)
	return entry.ID, true
}

func (tracker *Tracker) ExpireEffect(input EffectInput, tickRate float64) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Lifecycle.Observed++
	index := -1
	correlation := Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable}
	if input.RuntimeEffectEntityID != 0 {
		mappedIndex, exists := tracker.effectByEntity[entityScope{
			round:    input.Hint.Round,
			entityID: input.RuntimeEffectEntityID,
		}]
		if exists {
			index = mappedIndex
			correlation = Correlation{Status: CorrelationObserved, Source: SourceEffectEntityID}
		}
	}
	if index < 0 {
		index = tracker.resolveActiveEffectLocked(input.Hint)
		if index >= 0 {
			correlation = Correlation{Status: CorrelationInferred, Source: SourceTypePositionTick}
		}
	}
	orphaned := index < 0
	if index < 0 {
		index = tracker.recordLifecycleOnlyLocked(input.Hint, input.Source)
	}
	entry := &tracker.throws[index]
	tracker.enrichActorLocked(entry, callbackActor(input.Hint))
	if sameObservation(entry.Lifecycle.Expiration, input.Hint.Tick, input.Source) {
		tracker.diagnostics.Lifecycle.Deduplicated++
		return entry.ID, false
	}
	entry.Lifecycle.Expiration = tickPosition(
		input.Hint.Tick,
		input.Hint.Position,
		input.Hint.PositionStatus,
		input.Source,
	)
	entry.Lifecycle.Status = LifecycleEffectExpired
	entry.Lifecycle.EndReason = EndReasonExpired
	entry.Lifecycle.EndReasonSource = SourceExpirationCallback
	entry.Lifecycle.Area = preferObservation(
		normalizeStringObservation(input.Hint.Area),
		entry.Lifecycle.Area,
	)
	tracker.reconcilePendingDamageLocked()
	if entry.Lifecycle.EffectStart.Status == AvailabilityObserved &&
		tickRate > 0 && input.Hint.Tick >= entry.Lifecycle.EffectStart.Tick {
		entry.Lifecycle.Duration = ScalarObservation{
			Value:  float64(input.Hint.Tick-entry.Lifecycle.EffectStart.Tick) / tickRate,
			Status: AvailabilityObserved,
			Source: SourceCallbackTicks,
		}
	}
	tracker.confirmExtinguishLocked(index, input.Hint.Tick)
	if input.RuntimeEffectEntityID != 0 {
		delete(tracker.effectByEntity, entityScope{
			round:    input.Hint.Round,
			entityID: input.RuntimeEffectEntityID,
		})
	}
	tracker.recordCallbackResultLocked(&tracker.diagnostics.Lifecycle, correlation, orphaned)
	return entry.ID, true
}

func (tracker *Tracker) MarkExtinguishCandidates(
	smokeThrowID string,
	round int,
	tick int,
	position Vector,
	radius float64,
	correlation Correlation,
) int {
	if smokeThrowID == "" || radius <= 0 || !isFiniteVector(position) {
		return 0
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	marked := 0
	for index := range tracker.throws {
		entry := &tracker.throws[index]
		if entry.Round != round || !isFire(entry.Type) ||
			entry.Lifecycle.Status != LifecycleEffectActive ||
			entry.Lifecycle.EffectStart.PositionStatus != AvailabilityObserved {
			continue
		}
		distance := vectorDistance(position, entry.Lifecycle.EffectStart.Position)
		if distance > radius {
			continue
		}
		candidate, exists := tracker.extinguishCandidates[index]
		if exists && (candidate.distance < distance ||
			(candidate.distance == distance && candidate.smokeThrowID <= smokeThrowID)) {
			continue
		}
		tracker.extinguishCandidates[index] = extinguishCandidate{
			smokeThrowID: smokeThrowID,
			tick:         tick,
			position:     position,
			distance:     distance,
			correlation:  correlation,
		}
		marked++
	}
	return marked
}

func (tracker *Tracker) RecordFlash(input FlashInput) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Flashes.Observed++
	input.Duration = normalizeScalarObservation(input.Duration)
	input.Relation = normalizeRelation(input.Relation)
	actorID := input.ActorID
	if actorID == 0 && input.Actor.Status == AvailabilityObserved {
		actorID = input.Actor.ID
	}
	hint := CallbackHint{
		Round:           input.Round,
		RuntimeEntityID: input.RuntimeEntityID,
		EntitySource:    SourceProjectileEntity,
		Type:            TypeFlashbang,
		ActorID:         actorID,
		Actor:           input.Actor,
		Tick:            input.Tick,
		TickRate:        input.TickRate,
		PositionStatus:  AvailabilityUnavailable,
	}
	index, correlation := tracker.resolveThrowLocked(hint, true)
	orphaned := index < 0
	if index < 0 {
		index = tracker.recordLifecycleOnlyLocked(hint, SourcePlayerFlashed)
	}
	entry := &tracker.throws[index]
	tracker.enrichActorLocked(entry, input.Actor)
	if entry.Lifecycle.Detonation.Status != AvailabilityObserved {
		entry.Lifecycle.Detonation = TickPositionObservation{
			Tick: input.Tick, Status: AvailabilityObserved,
			PositionStatus: AvailabilityUnavailable, Source: SourcePlayerFlashed,
		}
		entry.Lifecycle.Status = LifecycleDetonated
	}
	tracker.closeFlightTrajectoryLocked(index)
	if entry.Lifecycle.Correlation.Status == CorrelationUnavailable &&
		correlation.Status != CorrelationUnavailable {
		entry.Lifecycle.Correlation = correlation
	}
	effectTick := input.Tick
	if entry.Lifecycle.Detonation.Status == AvailabilityObserved {
		effectTick = entry.Lifecycle.Detonation.Tick
	}
	for effectIndex := range entry.Flashes {
		effect := &entry.Flashes[effectIndex]
		if effect.Tick == effectTick && effect.Victim.ID != 0 && effect.Victim.ID == input.Victim.ID {
			if input.Duration.Status == AvailabilityObserved &&
				(effect.Duration.Status != AvailabilityObserved || input.Duration.Value > effect.Duration.Value) {
				effect.Duration = input.Duration
			}
			tracker.diagnostics.Flashes.Deduplicated++
			return entry.ID, false
		}
	}
	entry.Flashes = append(entry.Flashes, FlashEffect{
		Victim:      playerWithSource(input.Victim, SourcePlayerFlashed),
		Relation:    input.Relation,
		Duration:    input.Duration,
		Tick:        effectTick,
		Source:      SourcePlayerFlashed,
		Correlation: correlation,
	})
	tracker.recordCallbackResultLocked(&tracker.diagnostics.Flashes, correlation, orphaned)
	return entry.ID, true
}

func (tracker *Tracker) RecordDamage(input DamageInput) (string, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.Damage.Observed++
	input.Relation = normalizeRelation(input.Relation)
	input.HealthDamage = max(0, input.HealthDamage)
	input.ArmorDamage = max(0, input.ArmorDamage)
	if input.ActorID == 0 && input.Actor.Status == AvailabilityObserved {
		input.ActorID = input.Actor.ID
	}
	index := tracker.resolveDamageThrowLocked(input)
	if index < 0 {
		tracker.pendingDamage = append(tracker.pendingDamage, input)
		tracker.diagnostics.Damage.Unmatched++
		return "", false
	}
	tracker.enrichActorLocked(&tracker.throws[index], input.Actor)
	tracker.appendDamageLocked(index, input, Correlation{
		Status: CorrelationInferred, Source: SourceThrowerTypePositionTick,
	})
	tracker.diagnostics.Damage.InferredCorrelated++
	return tracker.throws[index].ID, true
}

func (tracker *Tracker) Snapshot() []Throw {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	result := make([]Throw, len(tracker.throws))
	for index := range tracker.throws {
		result[index] = cloneThrow(tracker.throws[index])
	}
	slices.SortFunc(result, func(left, right Throw) int {
		if left.Round != right.Round {
			return left.Round - right.Round
		}
		return left.Sequence - right.Sequence
	})
	return result
}

func (tracker *Tracker) Diagnostics() Diagnostics {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	return tracker.diagnostics
}

func (tracker *Tracker) RecordDiscardedCallback(group CallbackGroup, reason CallbackDiscardReason) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	discarded, callbacks := callbackDiagnosticsForGroup(&tracker.diagnostics, group)
	if discarded == nil {
		return
	}
	switch reason {
	case CallbackDiscardWarmup:
		discarded.Warmup++
	case CallbackDiscardOutsideRound:
		discarded.OutsideRound++
	case CallbackDiscardInvalid:
		discarded.Invalid++
		callbacks.Observed++
		callbacks.Unmatched++
	}
}

func (tracker *Tracker) Lookup(id string) (Throw, bool) {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	for _, entry := range tracker.throws {
		if entry.ID == id {
			return cloneThrow(entry), true
		}
	}
	return Throw{}, false
}

func callbackDiagnosticsForGroup(
	diagnostics *Diagnostics,
	group CallbackGroup,
) (*CallbackDiscardDiagnostics, *CallbackDiagnostics) {
	switch group {
	case CallbackGroupThrows:
		return &diagnostics.Discarded.Throws, &diagnostics.Throws
	case CallbackGroupBounces:
		return &diagnostics.Discarded.Bounces, &diagnostics.Bounces
	case CallbackGroupLifecycle:
		return &diagnostics.Discarded.Lifecycle, &diagnostics.Lifecycle
	case CallbackGroupPlayerFlashed:
		return &diagnostics.Discarded.PlayerFlashed, &diagnostics.Flashes
	case CallbackGroupDamage:
		return &diagnostics.Discarded.Damage, &diagnostics.Damage
	default:
		return nil, nil
	}
}

func (tracker *Tracker) endRoundLocked(round int) {
	if round <= 0 {
		return
	}
	tracker.reconcilePendingDamageLocked()
	tracker.flushPendingDamageLocked(round)
	for index := range tracker.throws {
		entry := &tracker.throws[index]
		if entry.Round != round {
			continue
		}
		switch entry.Lifecycle.Status {
		case LifecycleThrown:
			if entry.Lifecycle.Destroy.Status == AvailabilityObserved {
				entry.Lifecycle.Status = LifecycleDestroyedWithoutDetonation
				entry.Lifecycle.EndReason = EndReasonDestroyed
				entry.Lifecycle.EndReasonSource = SourceProjectileDestroy
			} else {
				entry.Lifecycle.Status = LifecycleRoundEndedUnresolved
				entry.Lifecycle.EndReason = EndReasonRoundEnd
				entry.Lifecycle.EndReasonSource = SourceRoundBoundary
			}
		case LifecycleEffectActive:
			entry.Lifecycle.Status = LifecycleRoundEndedUnresolved
			entry.Lifecycle.EndReason = EndReasonRoundEnd
			entry.Lifecycle.EndReasonSource = SourceRoundBoundary
		}
		delete(tracker.extinguishCandidates, index)
	}
	for scope := range tracker.projectileByEntity {
		if scope.round == round {
			delete(tracker.projectileByEntity, scope)
		}
	}
	for scope := range tracker.effectByEntity {
		if scope.round == round {
			delete(tracker.effectByEntity, scope)
		}
	}
}

func (tracker *Tracker) resolveThrowLocked(hint CallbackHint, includeCompleted bool) (int, Correlation) {
	if hint.RuntimeEntityID != 0 {
		if index, exists := tracker.projectileByEntity[entityScope{
			round: hint.Round, entityID: hint.RuntimeEntityID,
		}]; exists && compatibleHintType(tracker.throws[index].Type, hint) {
			source := hint.EntitySource
			if source == "" {
				source = SourceProjectileEntity
			}
			return index, Correlation{Status: CorrelationObserved, Source: source}
		}
	}
	index := tracker.findCandidateLocked(hint, includeCompleted)
	if index < 0 {
		return -1, Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable}
	}
	source := SourceTypePositionTick
	if hint.ActorID != 0 && tracker.throws[index].Actor.ID == hint.ActorID {
		source = SourceThrowerTypePositionTick
	}
	return index, Correlation{Status: CorrelationInferred, Source: source}
}

func (tracker *Tracker) findCandidateLocked(hint CallbackHint, includeCompleted bool) int {
	bestIndex := -1
	bestScore := candidateScore{actorRank: math.MaxInt, positionRank: math.MaxInt, distance: math.Inf(1), tickGap: math.MaxInt, sequence: math.MaxInt}
	for index := range tracker.throws {
		entry := &tracker.throws[index]
		if entry.Round != hint.Round || !compatibleHintType(entry.Type, hint) {
			continue
		}
		if entry.Launch.Tick.Status == AvailabilityObserved && entry.Launch.Tick.Tick > hint.Tick {
			continue
		}
		if !causallyClose(*entry, hint) {
			continue
		}
		if !includeCompleted &&
			(entry.Lifecycle.Detonation.Status == AvailabilityObserved || entry.Lifecycle.EffectStart.Status == AvailabilityObserved) {
			continue
		}
		score := scoreCandidate(*entry, hint)
		if score.less(bestScore) {
			bestIndex = index
			bestScore = score
		}
	}
	return bestIndex
}

func (tracker *Tracker) resolveActiveEffectLocked(hint CallbackHint) int {
	bestIndex := -1
	bestDistance := math.Inf(1)
	for index := range tracker.throws {
		entry := tracker.throws[index]
		if entry.Round != hint.Round || !compatibleHintType(entry.Type, hint) ||
			entry.Lifecycle.Status != LifecycleEffectActive {
			continue
		}
		gap := hint.Tick - entry.Lifecycle.EffectStart.Tick
		if gap < 0 || gap > ticksForSeconds(hint.TickRate, maxInferredEffectSeconds) {
			continue
		}
		distance := math.Inf(1)
		if hint.PositionStatus == AvailabilityObserved &&
			entry.Lifecycle.EffectStart.PositionStatus == AvailabilityObserved {
			distance = vectorDistance(hint.Position, entry.Lifecycle.EffectStart.Position)
			if distance > maxInferredPositionDistance {
				continue
			}
		}
		if distance < bestDistance || (distance == bestDistance &&
			(bestIndex < 0 || entry.Sequence < tracker.throws[bestIndex].Sequence)) {
			bestIndex = index
			bestDistance = distance
		}
	}
	return bestIndex
}

func (tracker *Tracker) resolveDamageThrowLocked(input DamageInput) int {
	bestIndex := -1
	bestTick := -1
	bestDistance := math.Inf(1)
	for index := range tracker.throws {
		entry := tracker.throws[index]
		if entry.Round != input.Round || !compatibleType(entry.Type, input.Type) ||
			input.ActorID == 0 || entry.Actor.ID != input.ActorID {
			continue
		}
		if input.Type == TypeHE {
			if entry.Lifecycle.Detonation.Status != AvailabilityObserved {
				continue
			}
			tick := entry.Lifecycle.Detonation.Tick
			if absolute(tick-input.Tick) > maxDamageCallbackReorderTicks {
				continue
			}
			if tick > bestTick || (tick == bestTick &&
				(bestIndex < 0 || entry.Sequence < tracker.throws[bestIndex].Sequence)) {
				bestIndex, bestTick = index, tick
			}
			continue
		}
		if !isFire(entry.Type) {
			continue
		}
		if entry.Lifecycle.EffectStart.Status != AvailabilityObserved ||
			(entry.Lifecycle.Status != LifecycleEffectActive && entry.Lifecycle.Status != LifecycleEffectExpired) {
			continue
		}
		if input.Tick+maxDamageCallbackReorderTicks < entry.Lifecycle.EffectStart.Tick {
			continue
		}
		if entry.Lifecycle.Expiration.Status == AvailabilityObserved && input.Tick > entry.Lifecycle.Expiration.Tick {
			continue
		}
		if input.VictimPositionStatus != AvailabilityObserved {
			tick := entry.Lifecycle.EffectStart.Tick
			if tick > bestTick || (tick == bestTick &&
				(bestIndex < 0 || entry.Sequence < tracker.throws[bestIndex].Sequence)) {
				bestIndex, bestTick = index, tick
			}
			continue
		}
		distance := vectorDistance(input.VictimPosition, entry.Lifecycle.EffectStart.Position)
		if distance < bestDistance || (distance == bestDistance &&
			(bestIndex < 0 || entry.Sequence < tracker.throws[bestIndex].Sequence)) {
			bestIndex, bestDistance = index, distance
		}
	}
	return bestIndex
}

func (tracker *Tracker) reconcilePendingDamageLocked() {
	if len(tracker.pendingDamage) == 0 {
		return
	}
	remaining := tracker.pendingDamage[:0]
	for _, input := range tracker.pendingDamage {
		index := tracker.resolveDamageThrowLocked(input)
		if index < 0 {
			remaining = append(remaining, input)
			continue
		}
		tracker.appendDamageLocked(index, input, Correlation{
			Status: CorrelationInferred, Source: SourceThrowerTypePositionTick,
		})
		tracker.enrichActorLocked(&tracker.throws[index], input.Actor)
		tracker.diagnostics.Damage.Unmatched--
		tracker.diagnostics.Damage.InferredCorrelated++
	}
	tracker.pendingDamage = remaining
}

func (tracker *Tracker) flushPendingDamageLocked(round int) {
	type sparseGroup struct {
		index    int
		actorID  uint64
		typeName Type
		lastTick int
	}
	groups := make([]sparseGroup, 0)
	remaining := tracker.pendingDamage[:0]
	for _, input := range tracker.pendingDamage {
		if input.Round != round {
			remaining = append(remaining, input)
			continue
		}
		groupIndex := -1
		for index := len(groups) - 1; index >= 0; index-- {
			group := groups[index]
			if group.actorID != input.ActorID || !compatibleType(group.typeName, input.Type) {
				continue
			}
			window := ticksForSeconds(input.TickRate, maxInferredFlightSeconds)
			if input.Tick >= group.lastTick && input.Tick-group.lastTick <= window {
				groupIndex = index
				break
			}
		}
		if groupIndex < 0 {
			entryIndex := tracker.recordLifecycleOnlyLocked(CallbackHint{
				Round: input.Round, Type: input.Type, ActorID: input.ActorID, Actor: input.Actor,
				Tick: input.Tick, TickRate: input.TickRate,
				PositionStatus: AvailabilityUnavailable,
			}, SourcePlayerHurt)
			groups = append(groups, sparseGroup{
				index: entryIndex, actorID: input.ActorID, typeName: input.Type,
				lastTick: input.Tick,
			})
			groupIndex = len(groups) - 1
		}
		groups[groupIndex].lastTick = input.Tick
		tracker.appendDamageLocked(groups[groupIndex].index, input, Correlation{
			Status: CorrelationUnavailable, Source: SourceUnavailable,
		})
		tracker.diagnostics.Damage.Unmatched--
		tracker.diagnostics.Damage.Orphaned++
	}
	tracker.pendingDamage = remaining
}

func (tracker *Tracker) appendDamageLocked(index int, input DamageInput, correlation Correlation) {
	entry := &tracker.throws[index]
	entry.Damage = append(entry.Damage, DamageEffect{
		Victim:       playerWithSource(input.Victim, SourcePlayerHurt),
		Relation:     input.Relation,
		HealthDamage: input.HealthDamage,
		ArmorDamage:  input.ArmorDamage,
		Kill:         input.Kill,
		Tick:         input.Tick,
		Source:       SourcePlayerHurt,
		Correlation:  correlation,
	})
}

func (tracker *Tracker) recordLifecycleOnlyLocked(hint CallbackHint, _ string) int {
	tracker.ensureMapsLocked()
	sequence := tracker.sequenceByRound[hint.Round] + 1
	tracker.sequenceByRound[hint.Round] = sequence
	typeName := normalizedType(hint.Type)
	typeSource := SourceCallbackType
	if typeName == TypeUnknown {
		typeSource = SourceUnavailable
	}
	input := ThrowInput{
		Round:           hint.Round,
		RuntimeEntityID: hint.RuntimeEntityID,
		EntitySource:    hint.EntitySource,
		Type:            typeName,
		TypeSource:      typeSource,
		Actor:           callbackActor(hint),
		Launch:          unavailableThrowSnapshot(),
	}
	generation := 0
	if hint.RuntimeEntityID != 0 {
		scope := entityScope{round: hint.Round, entityID: hint.RuntimeEntityID}
		generation = tracker.entityGenerations[scope] + 1
		tracker.entityGenerations[scope] = generation
	}
	tracker.throws = append(tracker.throws, newThrow(input, sequence, generation))
	index := len(tracker.throws) - 1
	if hint.RuntimeEntityID != 0 {
		tracker.projectileByEntity[entityScope{
			round: hint.Round, entityID: hint.RuntimeEntityID,
		}] = index
	}
	return index
}

func (tracker *Tracker) confirmExtinguishLocked(index, expirationTick int) {
	candidate, exists := tracker.extinguishCandidates[index]
	delete(tracker.extinguishCandidates, index)
	if !exists || expirationTick < candidate.tick ||
		expirationTick-candidate.tick > ExtinguishConfirmationWindowTicks {
		return
	}
	entry := &tracker.throws[index]
	entry.Lifecycle.Extinguish = tickPosition(
		candidate.tick,
		candidate.position,
		AvailabilityObserved,
		SourceSpatialSmokeOverlap,
	)
	entry.Lifecycle.ExtinguishedByThrowID = StringObservation{
		Value: candidate.smokeThrowID, Status: AvailabilityObserved, Source: SourceSpatialSmokeOverlap,
	}
	entry.Lifecycle.ExtinguishAttribution = Correlation{
		Status: CorrelationInferred,
		Source: SourceSpatialSmokeOverlap,
	}
	entry.Lifecycle.EndReason = EndReasonSmokeExtinguished
	entry.Lifecycle.EndReasonSource = SourceSpatialSmokeOverlap
}

func (tracker *Tracker) ensureMapsLocked() {
	if tracker.sequenceByRound == nil {
		tracker.sequenceByRound = make(map[int]int)
	}
	if tracker.entityGenerations == nil {
		tracker.entityGenerations = make(map[entityScope]int)
	}
	if tracker.projectileByEntity == nil {
		tracker.projectileByEntity = make(map[entityScope]int)
	}
	if tracker.effectByEntity == nil {
		tracker.effectByEntity = make(map[entityScope]int)
	}
	if tracker.extinguishCandidates == nil {
		tracker.extinguishCandidates = make(map[int]extinguishCandidate)
	}
	if tracker.trajectorySamples == nil {
		tracker.trajectorySamples = make(map[int]map[trajectorySampleKey]struct{})
	}
}

func (tracker *Tracker) recordCallbackResultLocked(
	diagnostics *CallbackDiagnostics,
	correlation Correlation,
	orphaned bool,
) {
	if orphaned {
		diagnostics.Orphaned++
		return
	}
	switch correlation.Status {
	case CorrelationObserved:
		diagnostics.ExactCorrelated++
	case CorrelationInferred:
		diagnostics.InferredCorrelated++
	default:
		diagnostics.Unmatched++
	}
}

func (tracker *Tracker) enrichActorLocked(entry *Throw, actor PlayerRef) {
	actor = normalizePlayer(actor)
	if actor.Status != AvailabilityObserved {
		return
	}
	if entry.Actor.Status != AvailabilityObserved {
		entry.Actor = actor
		tracker.diagnostics.ActorEnriched++
		return
	}
	if entry.Actor.ID != actor.ID {
		tracker.diagnostics.ActorConflicts++
	}
}

func callbackActor(hint CallbackHint) PlayerRef {
	if hint.Actor.Status == AvailabilityObserved {
		return playerWithSource(hint.Actor, SourceCallbackActor)
	}
	if hint.ActorID == 0 {
		return PlayerRef{Status: AvailabilityUnavailable, Source: SourceUnavailable}
	}
	return PlayerRef{
		ID: hint.ActorID, Status: AvailabilityObserved, Source: SourceCallbackActor,
	}
}

type candidateScore struct {
	actorRank    int
	positionRank int
	distance     float64
	tickGap      int
	sequence     int
}

func (score candidateScore) less(other candidateScore) bool {
	if score.actorRank != other.actorRank {
		return score.actorRank < other.actorRank
	}
	if score.positionRank != other.positionRank {
		return score.positionRank < other.positionRank
	}
	if score.distance != other.distance {
		return score.distance < other.distance
	}
	if score.tickGap != other.tickGap {
		return score.tickGap < other.tickGap
	}
	return score.sequence < other.sequence
}

func scoreCandidate(entry Throw, hint CallbackHint) candidateScore {
	actorRank := 0
	if hint.ActorID != 0 {
		switch {
		case entry.Actor.ID == hint.ActorID:
			actorRank = 0
		case entry.Actor.Status == AvailabilityUnavailable:
			actorRank = 1
		default:
			actorRank = 2
		}
	}
	reference, positionAvailable := evidencePosition(entry)
	positionRank := 1
	distance := math.Inf(1)
	if hint.PositionStatus == AvailabilityObserved && positionAvailable {
		positionRank = 0
		distance = vectorDistance(hint.Position, reference)
	}
	return candidateScore{
		actorRank:    actorRank,
		positionRank: positionRank,
		distance:     distance,
		tickGap:      absolute(hint.Tick - evidenceTick(entry)),
		sequence:     entry.Sequence,
	}
}

func evidencePosition(entry Throw) (Vector, bool) {
	if entry.Lifecycle.Destroy.PositionStatus == AvailabilityObserved {
		return entry.Lifecycle.Destroy.Position, true
	}
	if count := len(entry.Trajectory.Samples); count > 0 {
		return entry.Trajectory.Samples[count-1].Position, true
	}
	if entry.Launch.Position.Status == AvailabilityObserved {
		return entry.Launch.Position.Value, true
	}
	return Vector{}, false
}

func causallyClose(entry Throw, hint CallbackHint) bool {
	gap := hint.Tick - evidenceTick(entry)
	if gap < 0 || gap > ticksForSeconds(hint.TickRate, maxInferredFlightSeconds) {
		return false
	}
	if hint.ActorID != 0 && entry.Actor.Status == AvailabilityObserved && entry.Actor.ID != hint.ActorID {
		return false
	}
	reference, positionAvailable := evidencePosition(entry)
	if hint.PositionStatus == AvailabilityObserved && positionAvailable {
		return vectorDistance(hint.Position, reference) <= maxInferredPositionDistance
	}
	if hint.ActorID != 0 {
		return entry.Actor.ID == hint.ActorID || entry.Actor.Status == AvailabilityUnavailable
	}
	return false
}

func evidenceTick(entry Throw) int {
	if entry.Lifecycle.Destroy.Status == AvailabilityObserved {
		return entry.Lifecycle.Destroy.Tick
	}
	if count := len(entry.Trajectory.Samples); count > 0 {
		latest := entry.Trajectory.Samples[0].Tick
		for _, sample := range entry.Trajectory.Samples[1:] {
			if sample.Tick > latest {
				latest = sample.Tick
			}
		}
		return latest
	}
	if entry.Launch.Tick.Status == AvailabilityObserved {
		return entry.Launch.Tick.Tick
	}
	return 0
}

func newThrow(input ThrowInput, sequence, generation int) Throw {
	typeValue := normalizedType(input.Type)
	typeSource := input.TypeSource
	if typeValue == TypeUnknown {
		typeSource = SourceUnavailable
	} else if typeSource != SourceWeaponInstance && typeSource != SourceCallbackType {
		typeValue = TypeUnknown
		typeSource = SourceUnavailable
	}
	entityObserved := input.RuntimeEntityID != 0 &&
		(input.EntitySource == SourceProjectileEntity || input.EntitySource == SourceGrenadeEntityID)
	entityStatus := availability(entityObserved)
	entitySource := SourceUnavailable
	sourceEntityID, sourceEntityGeneration := 0, 0
	if entityObserved {
		entitySource = input.EntitySource
		sourceEntityID = input.RuntimeEntityID
		sourceEntityGeneration = generation
	}
	return Throw{
		ID:                     fmt.Sprintf("r%d-u%04d", input.Round, sequence),
		Round:                  input.Round,
		Sequence:               sequence,
		SourceEntityID:         sourceEntityID,
		SourceEntityGeneration: sourceEntityGeneration,
		EntityStatus:           entityStatus,
		EntitySource:           entitySource,
		Type:                   typeValue,
		TypeSource:             typeSource,
		Actor:                  normalizePlayer(input.Actor),
		Launch:                 normalizeThrowSnapshot(input.Launch),
		Trajectory: Trajectory{
			Samples: []TrajectorySample{}, Bounces: []BounceObservation{},
			Status: TrajectoryUnavailable, Source: SourceUnavailable,
			BounceStatus: AvailabilityUnavailable, BounceSource: SourceUnavailable,
		},
		Lifecycle: Lifecycle{
			Status:                LifecycleThrown,
			Detonation:            unavailableTickPosition(),
			EffectStart:           unavailableTickPosition(),
			Expiration:            unavailableTickPosition(),
			Destroy:               unavailableTickPosition(),
			Extinguish:            unavailableTickPosition(),
			Duration:              unavailableScalar(),
			Area:                  unavailableString(),
			EndReason:             EndReasonUnavailable,
			EndReasonSource:       SourceUnavailable,
			Correlation:           Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable},
			ExtinguishedByThrowID: unavailableString(),
			ExtinguishAttribution: Correlation{Status: CorrelationUnavailable, Source: SourceUnavailable},
		},
		Flashes: []FlashEffect{},
		Damage:  []DamageEffect{},
	}
}

func unavailableThrowSnapshot() ThrowSnapshot {
	return ThrowSnapshot{
		Tick:                      TickObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable},
		Position:                  VectorObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable},
		View:                      ViewObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable},
		ThrowerVelocity:           VelocityObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable},
		ProjectileInitialVelocity: VelocityObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable},
		Stance:                    StanceObservation{Value: StanceUnknown, Status: AvailabilityUnavailable, Source: SourceUnavailable},
		Area:                      unavailableString(),
	}
}

func normalizeThrowSnapshot(snapshot ThrowSnapshot) ThrowSnapshot {
	snapshot.Tick = normalizeTickObservation(snapshot.Tick)
	snapshot.Position = normalizeVectorObservation(snapshot.Position)
	snapshot.View = normalizeViewObservation(snapshot.View)
	snapshot.ThrowerVelocity = normalizeVelocityObservation(snapshot.ThrowerVelocity, false)
	snapshot.ProjectileInitialVelocity = normalizeVelocityObservation(snapshot.ProjectileInitialVelocity, true)
	if snapshot.Tick.Status == AvailabilityObserved {
		if snapshot.ThrowerVelocity.Status == AvailabilityObserved &&
			snapshot.ThrowerVelocity.ObservedTick != snapshot.Tick.Tick {
			snapshot.ThrowerVelocity = VelocityObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
		}
		if snapshot.ProjectileInitialVelocity.Status == AvailabilityObserved &&
			snapshot.ProjectileInitialVelocity.ObservedTick != snapshot.Tick.Tick {
			snapshot.ProjectileInitialVelocity = VelocityObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
		}
	}
	snapshot.Stance = normalizeStanceObservation(snapshot.Stance)
	snapshot.Area = normalizeStringObservation(snapshot.Area)
	return snapshot
}

func normalizeTickObservation(observation TickObservation) TickObservation {
	if observation.Status == AvailabilityObserved && observation.Tick >= 0 &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return TickObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func normalizeVectorObservation(observation VectorObservation) VectorObservation {
	if observation.Status == AvailabilityObserved && isFiniteVector(observation.Value) &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return VectorObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func normalizeViewObservation(observation ViewObservation) ViewObservation {
	if observation.Status == AvailabilityObserved && isFiniteVector(observation.Vector) &&
		isFinite(observation.Yaw) && isFinite(observation.Pitch) &&
		observation.Yaw >= -180 && observation.Yaw < 180 &&
		observation.Pitch >= -90 && observation.Pitch <= 90 &&
		math.Abs(vectorMagnitude(observation.Vector)-1) <= 1e-6 &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return ViewObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func normalizeVelocityObservation(observation VelocityObservation, projectile bool) VelocityObservation {
	if observation.Status == AvailabilityObserved && isFiniteVector(observation.Vector) &&
		isFinite(observation.HorizontalSpeed) && observation.HorizontalSpeed >= 0 &&
		observation.ObservedTick >= 0 && observation.MeasurementWindowTicks >= 0 &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		if projectile && observation.Source == SourceProjectileVelocity && observation.MeasurementWindowTicks == 0 {
			return observation
		}
		if !projectile && ((observation.Source == SourceVelocityNative && observation.MeasurementWindowTicks == 0) ||
			(observation.Source == SourceVelocityPositionDelta && observation.MeasurementWindowTicks > 0)) {
			return observation
		}
	}
	if observation.Status == AvailabilityNotApplicable && observation.Source == SourceVelocityNotApplicable {
		return VelocityObservation{Status: AvailabilityNotApplicable, Source: SourceVelocityNotApplicable}
	}
	source := observation.Source
	if !isUnavailableVelocitySource(source) {
		source = SourceUnavailable
	}
	return VelocityObservation{Status: AvailabilityUnavailable, Source: source}
}

func normalizeStanceObservation(observation StanceObservation) StanceObservation {
	if observation.Status == AvailabilityObserved && validStance(observation.Value) &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return StanceObservation{
		Value: StanceUnknown, Status: AvailabilityUnavailable, Source: SourceUnavailable,
	}
}

func normalizeStringObservation(observation StringObservation) StringObservation {
	if observation.Status == AvailabilityObserved && observation.Value != "" &&
		observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return unavailableString()
}

func isUnavailableVelocitySource(source string) bool {
	switch source {
	case SourceUnavailable, SourceVelocityNoHistory, SourceVelocityRejected,
		SourceVelocityStaleGap, SourceVelocityEntityChanged, SourceVelocityNonMonotonicTick:
		return true
	default:
		return false
	}
}

func validStance(value Stance) bool {
	switch value {
	case StanceStanding, StanceWalking, StanceCrouching, StanceCrouchWalking, StanceAirborne:
		return true
	default:
		return false
	}
}

func cloneThrow(entry Throw) Throw {
	entry.Trajectory.Samples = slices.Clone(entry.Trajectory.Samples)
	entry.Trajectory.Bounces = slices.Clone(entry.Trajectory.Bounces)
	entry.Flashes = slices.Clone(entry.Flashes)
	entry.Damage = slices.Clone(entry.Damage)
	slices.SortFunc(entry.Trajectory.Samples, compareTrajectorySamples)
	slices.SortFunc(entry.Trajectory.Bounces, compareBounceObservations)
	slices.SortFunc(entry.Flashes, compareFlashEffects)
	slices.SortFunc(entry.Damage, compareDamageEffects)
	return entry
}

func compareFlashEffects(left, right FlashEffect) int {
	if left.Tick != right.Tick {
		return left.Tick - right.Tick
	}
	if compared := comparePlayerRefs(left.Victim, right.Victim); compared != 0 {
		return compared
	}
	if left.Relation < right.Relation {
		return -1
	}
	if left.Relation > right.Relation {
		return 1
	}
	if left.Duration.Value < right.Duration.Value {
		return -1
	}
	if left.Duration.Value > right.Duration.Value {
		return 1
	}
	if left.Duration.Status != right.Duration.Status {
		return compareStrings(string(left.Duration.Status), string(right.Duration.Status))
	}
	if left.Duration.Source != right.Duration.Source {
		return compareStrings(left.Duration.Source, right.Duration.Source)
	}
	if left.Correlation.Status != right.Correlation.Status {
		return compareStrings(string(left.Correlation.Status), string(right.Correlation.Status))
	}
	if left.Correlation.Source != right.Correlation.Source {
		return compareStrings(left.Correlation.Source, right.Correlation.Source)
	}
	return compareStrings(left.Source, right.Source)
}

func compareDamageEffects(left, right DamageEffect) int {
	if left.Tick != right.Tick {
		return left.Tick - right.Tick
	}
	if compared := comparePlayerRefs(left.Victim, right.Victim); compared != 0 {
		return compared
	}
	if left.HealthDamage != right.HealthDamage {
		return left.HealthDamage - right.HealthDamage
	}
	if left.ArmorDamage != right.ArmorDamage {
		return left.ArmorDamage - right.ArmorDamage
	}
	if left.Kill != right.Kill {
		if !left.Kill {
			return -1
		}
		return 1
	}
	if left.Relation != right.Relation {
		return compareStrings(string(left.Relation), string(right.Relation))
	}
	if left.Correlation.Status != right.Correlation.Status {
		return compareStrings(string(left.Correlation.Status), string(right.Correlation.Status))
	}
	if left.Correlation.Source != right.Correlation.Source {
		return compareStrings(left.Correlation.Source, right.Correlation.Source)
	}
	return compareStrings(left.Source, right.Source)
}

func compareTrajectorySamples(left, right TrajectorySample) int {
	if left.Tick != right.Tick {
		return left.Tick - right.Tick
	}
	if compared := compareVectors(left.Position, right.Position); compared != 0 {
		return compared
	}
	return compareStrings(left.Source, right.Source)
}

func compareBounceObservations(left, right BounceObservation) int {
	if left.Tick != right.Tick {
		return left.Tick - right.Tick
	}
	if left.Number != right.Number {
		return left.Number - right.Number
	}
	if left.PositionStatus != right.PositionStatus {
		return compareStrings(string(left.PositionStatus), string(right.PositionStatus))
	}
	if compared := compareVectors(left.Position, right.Position); compared != 0 {
		return compared
	}
	return compareStrings(left.Source, right.Source)
}

func (tracker *Tracker) appendTrajectorySampleLocked(index, tick int, position Vector, source string) bool {
	tracker.ensureMapsLocked()
	keys := tracker.trajectorySamples[index]
	if keys == nil {
		keys = make(map[trajectorySampleKey]struct{})
		tracker.trajectorySamples[index] = keys
	}
	trajectory := &tracker.throws[index].Trajectory
	if source == SourceProjectileDestroy {
		retained := trajectory.Samples[:0]
		keys = make(map[trajectorySampleKey]struct{}, len(trajectory.Samples))
		for _, sample := range trajectory.Samples {
			if sample.Source == SourceProjectileDestroy || sample.Tick >= tick {
				continue
			}
			retained = append(retained, sample)
			keys[trajectorySampleKey{tick: sample.Tick, position: sample.Position, source: sample.Source}] = struct{}{}
		}
		trajectory.Samples = retained
		tracker.trajectorySamples[index] = keys
	}
	key := trajectorySampleKey{tick: tick, position: position, source: source}
	if _, exists := keys[key]; exists {
		return false
	}
	keys[key] = struct{}{}
	trajectory.Samples = append(trajectory.Samples, TrajectorySample{
		Tick: tick, Position: position, Source: source,
	})
	trajectory.Source = source
	for _, sample := range trajectory.Samples {
		if sample.Source == SourceProjectileFrames {
			trajectory.Source = SourceProjectileFrames
			break
		}
	}
	return true
}

func (tracker *Tracker) closeFlightTrajectoryLocked(index int) {
	terminalTick, observed := flightTerminalTick(tracker.throws[index].Lifecycle)
	if !observed {
		return
	}
	trajectory := &tracker.throws[index].Trajectory
	retained := trajectory.Samples[:0]
	keys := make(map[trajectorySampleKey]struct{}, len(trajectory.Samples))
	for _, sample := range trajectory.Samples {
		if sample.Source == SourceProjectileDestroy || sample.Tick >= terminalTick {
			continue
		}
		retained = append(retained, sample)
		keys[trajectorySampleKey{tick: sample.Tick, position: sample.Position, source: sample.Source}] = struct{}{}
	}
	trajectory.Samples = retained
	tracker.trajectorySamples[index] = keys
	tracker.pruneBouncesAtOrAfterLocked(index, terminalTick)
	tracker.updateTrajectoryStatusLocked(index)
}

func (tracker *Tracker) pruneBouncesAtOrAfterLocked(index, terminalTick int) {
	trajectory := &tracker.throws[index].Trajectory
	retained := trajectory.Bounces[:0]
	for _, bounce := range trajectory.Bounces {
		if bounce.Tick >= terminalTick {
			continue
		}
		retained = append(retained, bounce)
	}
	trajectory.Bounces = retained
	tracker.updateBounceStatusLocked(index)
}

func (tracker *Tracker) updateBounceStatusLocked(index int) {
	trajectory := &tracker.throws[index].Trajectory
	trajectory.BounceCount = len(trajectory.Bounces)
	if trajectory.BounceCount == 0 {
		trajectory.BounceStatus = AvailabilityUnavailable
		trajectory.BounceSource = SourceUnavailable
		return
	}
	trajectory.BounceStatus = AvailabilityObserved
	trajectory.BounceSource = SourceProjectileBounce
}

func (tracker *Tracker) updateTrajectoryStatusLocked(index int) {
	entry := &tracker.throws[index]
	frameSamples, destroySamples := 0, 0
	for _, sample := range entry.Trajectory.Samples {
		switch sample.Source {
		case SourceProjectileFrames:
			frameSamples++
		case SourceProjectileDestroy:
			destroySamples++
		}
	}
	switch {
	case frameSamples > 0:
		entry.Trajectory.Source = SourceProjectileFrames
		if _, terminalObserved := flightTerminalTick(entry.Lifecycle); terminalObserved || destroySamples > 0 {
			entry.Trajectory.Status = TrajectoryObserved
		} else {
			entry.Trajectory.Status = TrajectoryPartial
		}
	case destroySamples > 0:
		entry.Trajectory.Status = TrajectoryPartial
		entry.Trajectory.Source = SourceProjectileDestroy
	default:
		entry.Trajectory.Status = TrajectoryUnavailable
		entry.Trajectory.Source = SourceUnavailable
	}
}

func flightTerminalTick(lifecycle Lifecycle) (int, bool) {
	terminalTick := 0
	observed := false
	for _, moment := range []TickPositionObservation{lifecycle.Detonation, lifecycle.EffectStart} {
		if moment.Status != AvailabilityObserved {
			continue
		}
		if !observed || moment.Tick < terminalTick {
			terminalTick, observed = moment.Tick, true
		}
	}
	return terminalTick, observed
}

func bounceTerminalTick(lifecycle Lifecycle) (int, bool) {
	if terminalTick, observed := flightTerminalTick(lifecycle); observed {
		return terminalTick, true
	}
	if lifecycle.Destroy.Status == AvailabilityObserved {
		return lifecycle.Destroy.Tick, true
	}
	return 0, false
}

func tickPosition(tick int, position Vector, status Availability, source string) TickPositionObservation {
	if tick < 0 || source == "" || source == SourceUnavailable {
		return unavailableTickPosition()
	}
	if status != AvailabilityObserved || !isFiniteVector(position) {
		return TickPositionObservation{
			Tick: tick, Status: AvailabilityObserved,
			PositionStatus: AvailabilityUnavailable, Source: source,
		}
	}
	return TickPositionObservation{
		Tick: tick, Position: position, Status: AvailabilityObserved,
		PositionStatus: AvailabilityObserved, Source: source,
	}
}

func unavailableTickPosition() TickPositionObservation {
	return TickPositionObservation{
		Status: AvailabilityUnavailable, PositionStatus: AvailabilityUnavailable, Source: SourceUnavailable,
	}
}

func unavailableScalar() ScalarObservation {
	return ScalarObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func normalizeScalarObservation(observation ScalarObservation) ScalarObservation {
	if observation.Status == AvailabilityObserved && isFinite(observation.Value) &&
		observation.Value >= 0 && observation.Source != "" && observation.Source != SourceUnavailable {
		return observation
	}
	return unavailableScalar()
}

func unavailableString() StringObservation {
	return StringObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func preferObservation(preferred, fallback StringObservation) StringObservation {
	if preferred.Status == AvailabilityObserved {
		return preferred
	}
	return fallback
}

func normalizePlayer(player PlayerRef) PlayerRef {
	if player.Status == AvailabilityObserved && player.ID != 0 &&
		player.Source != "" && player.Source != SourceUnavailable {
		return player
	}
	return PlayerRef{Status: AvailabilityUnavailable, Source: SourceUnavailable}
}

func playerWithSource(player PlayerRef, source string) PlayerRef {
	player = normalizePlayer(player)
	player.Source = source
	return player
}

func availability(observed bool) Availability {
	if observed {
		return AvailabilityObserved
	}
	return AvailabilityUnavailable
}

func normalizeRelation(value Relation) Relation {
	switch value {
	case RelationSelf, RelationTeammate, RelationEnemy, RelationUnknown:
		return value
	default:
		return RelationUnknown
	}
}

func normalizedType(value Type) Type {
	switch value {
	case TypeFlashbang, TypeSmoke, TypeHE, TypeMolotov, TypeIncendiary, TypeDecoy:
		return value
	default:
		return TypeUnknown
	}
}

func compatibleType(left, right Type) bool {
	if right == TypeUnknown || left == right {
		return true
	}
	return isFire(left) && isFire(right)
}

func compatibleHintType(candidate Type, hint CallbackHint) bool {
	if hint.TypeFamily == TypeFamilyFire {
		return isFire(candidate)
	}
	return compatibleType(candidate, hint.Type)
}

func isFire(value Type) bool {
	return value == TypeMolotov || value == TypeIncendiary
}

func sameObservation(observation TickPositionObservation, tick int, source string) bool {
	return observation.Status == AvailabilityObserved && observation.Tick == tick && observation.Source == source
}

func vectorDistance(left, right Vector) float64 {
	dx, dy, dz := left.X-right.X, left.Y-right.Y, left.Z-right.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func compareVectors(left, right Vector) int {
	for _, pair := range [][2]float64{{left.X, right.X}, {left.Y, right.Y}, {left.Z, right.Z}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

func compareStrings(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func comparePlayerRefs(left, right PlayerRef) int {
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	for _, pair := range [][2]string{
		{left.Name, right.Name},
		{left.Side, right.Side},
		{string(left.Status), string(right.Status)},
		{left.Source, right.Source},
	} {
		if compared := compareStrings(pair[0], pair[1]); compared != 0 {
			return compared
		}
	}
	return 0
}

func isFiniteVector(vector Vector) bool {
	return isFinite(vector.X) && isFinite(vector.Y) && isFinite(vector.Z)
}

func vectorMagnitude(vector Vector) float64 {
	return math.Sqrt(vector.X*vector.X + vector.Y*vector.Y + vector.Z*vector.Z)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func ticksForSeconds(tickRate, seconds float64) int {
	if tickRate <= 0 {
		tickRate = 64
	}
	return int(math.Ceil(tickRate * seconds))
}
