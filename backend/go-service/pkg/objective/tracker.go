package objective

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// Tracker is the single source of truth for C4 lifecycle state during parsing.
// It is safe to read from exporter code while parser callbacks are running.
type Tracker struct {
	mu sync.RWMutex

	snapshot  Snapshot
	events    []Event
	attempts  []Attempt
	summaries map[int]RoundSummary

	eventSequence   uint64
	attemptSequence uint64
	activePlant     int
	activeDefuse    int
	recordedCarrier uint64
	plantEndTick    int
}

func NewTracker() *Tracker {
	return &Tracker{
		snapshot: Snapshot{
			State:    StateUnknown,
			Phase:    PhasePreplant,
			Position: UnavailablePosition(SourceDemoinfocsNativeSnapshot),
		},
		events:       make([]Event, 0, 32),
		attempts:     make([]Attempt, 0, 8),
		summaries:    make(map[int]RoundSummary),
		activePlant:  -1,
		activeDefuse: -1,
		plantEndTick: -1,
	}
}

// BeginRound resets current state but retains the immutable ledger and prior
// round summaries. Calling it twice for the same round is harmless.
func (t *Tracker) BeginRound(round, tick int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.beginRoundLocked(round, tick)
}

// EndRound makes the objective inactive when the game resolves by elimination.
// Attempts without an explicit terminal callback remain incomplete in the ledger.
func (t *Tracker) EndRound(round, tick int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if round <= 0 || t.snapshot.Round != round {
		return
	}
	finalState := t.snapshot.State
	t.snapshot.Tick = tick
	if t.snapshot.Resolution == "" {
		t.snapshot.State = StateResolved
	}
	t.snapshot.Phase = PhaseResolved
	t.snapshot.IsPlantedNow = false
	t.snapshot.Carrier = Actor{}
	t.snapshot.PlantingPlayer = Actor{}
	t.snapshot.Defuser = Actor{}
	t.snapshot.ActivePlantAttemptID = ""
	t.snapshot.ActiveDefuseAttemptID = ""
	t.activePlant = -1
	t.activeDefuse = -1

	summary := t.summaries[round]
	summary.Round = round
	summary.WasPlanted = t.snapshot.WasPlantedThisRound
	summary.Site = t.snapshot.Site
	summary.PlantTick = t.snapshot.PlantTick
	summary.Resolution = t.snapshot.Resolution
	summary.ResolutionTick = t.snapshot.ResolutionTick
	summary.ResolvedAtRoundEnd = t.snapshot.Resolution == ""
	summary.RoundEndTick = tick
	summary.FinalState = finalState
	t.summaries[round] = summary
}

func (t *Tracker) beginRoundLocked(round, tick int) {
	if round <= 0 {
		return
	}
	if t.snapshot.Round == round {
		if t.snapshot.Phase != PhaseResolved {
			return
		}
		// Some FACEIT demos resolve a preliminary round and then reuse round 1
		// for the first competitive round. Replace the resolved observation for
		// that number so state samples and the objective ledger share one epoch.
		events := t.events[:0]
		for _, event := range t.events {
			if event.Round != round {
				events = append(events, event)
			}
		}
		t.events = events
		attempts := t.attempts[:0]
		for _, attempt := range t.attempts {
			if attempt.Round != round {
				attempts = append(attempts, attempt)
			}
		}
		t.attempts = attempts
		delete(t.summaries, round)
	}
	t.snapshot = Snapshot{
		Round:    round,
		Tick:     tick,
		State:    StateUnknown,
		Phase:    PhasePreplant,
		Position: UnavailablePosition(SourceDemoinfocsNativeSnapshot),
	}
	t.activePlant = -1
	t.activeDefuse = -1
	t.recordedCarrier = 0
	t.plantEndTick = -1
	t.summaries[round] = RoundSummary{Round: round, RoundStartTick: tick, FinalState: StateUnknown}
}

// NativeSnapshot projects a directly observed native bomb snapshot into the
// current state. It never turns a missing carrier into a fabricated drop.
func (t *Tracker) NativeSnapshot(observation NativeObservation) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.ensureRoundLocked(observation.Round, observation.Tick)
	t.snapshot.Tick = observation.Tick
	if observation.Position.Available() {
		t.snapshot.Position = normalizedPosition(observation.Position, SourceDemoinfocsNativeSnapshot)
	}

	if t.snapshot.Phase == PhaseResolved || !isPreplantState(t.snapshot.State) || observation.Carrier.SteamID == 0 {
		return t.snapshot
	}

	actor := normalizeActor(observation.Carrier)
	t.snapshot.State = StateCarried
	t.snapshot.Phase = PhasePreplant
	t.snapshot.Carrier = actor
	if t.recordedCarrier != actor.SteamID {
		input := EventInput{
			Round:    observation.Round,
			Tick:     observation.Tick,
			Actor:    actor,
			Position: observation.Position,
			Source:   SourceDemoinfocsNativeSnapshot,
		}
		event := t.newEventLocked(EventCarrierSnapshot, input)
		event.StateAfter = t.snapshot.State
		event.PhaseAfter = t.snapshot.Phase
		t.appendEventLocked(event)
		t.recordedCarrier = actor.SteamID
	}
	return t.snapshot
}

func (t *Tracker) Drop(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)
	if t.snapshot.Phase == PhasePlanted || t.snapshot.Phase == PhaseDefusing || t.snapshot.Phase == PhaseResolved {
		return Event{}
	}
	if t.snapshot.State == StateDropped {
		return Event{}
	}

	t.snapshot.State = StateDropped
	t.snapshot.Phase = PhasePreplant
	t.snapshot.Carrier = Actor{}
	t.snapshot.PlantingPlayer = Actor{}
	t.snapshot.IsPlantedNow = false
	t.applyObservedPositionLocked(input.Position)
	t.recordedCarrier = 0
	return t.recordLocked(EventDrop, input)
}

func (t *Tracker) Pickup(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)
	if t.snapshot.Phase == PhasePlanting || t.snapshot.Phase == PhasePlanted ||
		t.snapshot.Phase == PhaseDefusing || t.snapshot.Phase == PhaseResolved || input.Tick == t.plantEndTick ||
		input.Actor.SteamID == 0 {
		return Event{}
	}

	t.snapshot.State = StateCarried
	t.snapshot.Phase = PhasePreplant
	t.snapshot.Carrier = input.Actor
	t.snapshot.PlantingPlayer = Actor{}
	t.snapshot.IsPlantedNow = false
	t.applyObservedPositionLocked(input.Position)
	t.recordedCarrier = input.Actor.SteamID
	return t.recordLocked(EventPickup, input)
}

func (t *Tracker) PlantStart(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)

	event := t.newEventLocked(EventPlantStart, input)
	attemptIndex := t.startAttemptLocked(AttemptPlant, input, event.ID, true)
	attempt := &t.attempts[attemptIndex]

	t.snapshot.State = StatePlanting
	t.snapshot.Phase = PhasePlanting
	t.snapshot.Carrier = input.Actor
	t.snapshot.PlantingPlayer = input.Actor
	t.snapshot.ActivePlantAttemptID = attempt.ID
	t.applySiteLocked(input.Site)
	t.applyObservedPositionLocked(input.Position)

	t.attachAttemptLocked(&event, attempt, AttemptInProgress)
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) PlantAbort(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)
	t.discardPickupAtTickLocked(input.Round, input.Tick, input.Actor.SteamID)
	t.plantEndTick = input.Tick

	event := t.newEventLocked(EventPlantAbort, input)
	attemptIndex := t.ensureAttemptLocked(AttemptPlant, input)
	attempt := &t.attempts[attemptIndex]
	if input.Actor.SteamID == 0 {
		input.Actor = attempt.Actor
		event.Actor = attempt.Actor
	}
	t.finishAttemptLocked(AttemptPlant, AttemptAborted, input.Tick, event.ID)

	wasDropped := t.snapshot.State == StateDropped
	t.snapshot.State = StateCarried
	t.snapshot.Phase = PhasePreplant
	t.snapshot.Carrier = input.Actor
	if input.Actor.SteamID == 0 {
		t.snapshot.State = StateUnknown
		t.snapshot.Carrier = Actor{}
	}
	if wasDropped {
		t.snapshot.State = StateDropped
		t.snapshot.Carrier = Actor{}
	}
	t.snapshot.PlantingPlayer = Actor{}
	t.snapshot.ActivePlantAttemptID = ""
	t.applyObservedPositionLocked(input.Position)

	t.attachAttemptLocked(&event, attempt, AttemptAborted)
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) Plant(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)
	t.discardPickupAtTickLocked(input.Round, input.Tick, input.Actor.SteamID)
	t.plantEndTick = input.Tick

	event := t.newEventLocked(EventPlant, input)
	attemptIndex := t.ensureAttemptLocked(AttemptPlant, input)
	attempt := &t.attempts[attemptIndex]
	if input.Actor.SteamID == 0 {
		input.Actor = attempt.Actor
		event.Actor = attempt.Actor
	}
	t.finishAttemptLocked(AttemptPlant, AttemptCompleted, input.Tick, event.ID)

	t.snapshot.State = StatePlanted
	t.snapshot.Phase = PhasePlanted
	t.snapshot.IsPlantedNow = true
	t.snapshot.WasPlantedThisRound = true
	t.snapshot.Carrier = Actor{}
	t.snapshot.PlantingPlayer = Actor{}
	t.snapshot.ActivePlantAttemptID = ""
	t.snapshot.PlantTick = input.Tick
	t.applySiteLocked(input.Site)
	t.applyObservedPositionLocked(input.Position)
	t.recordedCarrier = 0

	t.attachAttemptLocked(&event, attempt, AttemptCompleted)
	event.Site = t.snapshot.Site
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) DefuseStart(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)

	event := t.newEventLocked(EventDefuseStart, input)
	attemptIndex := t.startAttemptLocked(AttemptDefuse, input, event.ID, true)
	attempt := &t.attempts[attemptIndex]

	t.snapshot.State = StateDefusing
	t.snapshot.Phase = PhaseDefusing
	t.snapshot.IsPlantedNow = true
	t.snapshot.WasPlantedThisRound = true
	t.snapshot.Defuser = input.Actor
	t.snapshot.ActiveDefuseAttemptID = attempt.ID
	t.applySiteLocked(input.Site)
	t.applyObservedPositionLocked(input.Position)

	t.attachAttemptLocked(&event, attempt, AttemptInProgress)
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) DefuseAbort(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)

	event := t.newEventLocked(EventDefuseAbort, input)
	attemptIndex := t.ensureAttemptLocked(AttemptDefuse, input)
	attempt := &t.attempts[attemptIndex]
	if input.Actor.SteamID == 0 {
		input.Actor = attempt.Actor
		event.Actor = attempt.Actor
	}
	t.finishAttemptLocked(AttemptDefuse, AttemptAborted, input.Tick, event.ID)

	t.snapshot.State = StatePlanted
	t.snapshot.Phase = PhasePlanted
	t.snapshot.IsPlantedNow = true
	t.snapshot.Defuser = Actor{}
	t.snapshot.ActiveDefuseAttemptID = ""
	t.applyObservedPositionLocked(input.Position)

	t.attachAttemptLocked(&event, attempt, AttemptAborted)
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) Defuse(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)

	event := t.newEventLocked(EventDefuse, input)
	attemptIndex := t.ensureAttemptLocked(AttemptDefuse, input)
	attempt := &t.attempts[attemptIndex]
	if input.Actor.SteamID == 0 {
		input.Actor = attempt.Actor
		event.Actor = attempt.Actor
	}
	t.finishAttemptLocked(AttemptDefuse, AttemptCompleted, input.Tick, event.ID)

	t.snapshot.State = StateDefused
	t.snapshot.Phase = PhaseResolved
	t.snapshot.IsPlantedNow = false
	t.snapshot.WasPlantedThisRound = true
	t.snapshot.Carrier = Actor{}
	t.snapshot.Defuser = input.Actor
	t.snapshot.ActiveDefuseAttemptID = ""
	t.snapshot.Resolution = EventDefuse
	t.snapshot.ResolutionTick = input.Tick
	t.applySiteLocked(input.Site)
	t.applyObservedPositionLocked(input.Position)

	t.attachAttemptLocked(&event, attempt, AttemptCompleted)
	event.Site = t.snapshot.Site
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) Explode(input EventInput) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prepareEventLocked(&input)

	t.snapshot.State = StateExploded
	t.snapshot.Phase = PhaseResolved
	t.snapshot.IsPlantedNow = false
	t.snapshot.WasPlantedThisRound = true
	t.snapshot.Carrier = Actor{}
	t.snapshot.Defuser = Actor{}
	t.snapshot.ActiveDefuseAttemptID = ""
	t.activeDefuse = -1
	t.snapshot.Resolution = EventExplode
	t.snapshot.ResolutionTick = input.Tick
	t.applySiteLocked(input.Site)
	t.applyObservedPositionLocked(input.Position)
	return t.recordLocked(EventExplode, input)
}

func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.snapshot
}

func (t *Tracker) Events() []Event {
	t.mu.RLock()
	defer t.mu.RUnlock()
	events := make([]Event, len(t.events))
	for index := range t.events {
		events[index] = cloneEvent(t.events[index])
	}
	return events
}

func (t *Tracker) Attempts() []Attempt {
	t.mu.RLock()
	defer t.mu.RUnlock()
	attempts := make([]Attempt, len(t.attempts))
	for index := range t.attempts {
		attempts[index] = cloneAttempt(t.attempts[index])
	}
	return attempts
}

func (t *Tracker) RoundSummary(round int) (RoundSummary, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	summary, ok := t.summaries[round]
	return summary, ok
}

func (t *Tracker) prepareEventLocked(input *EventInput) {
	t.ensureRoundLocked(input.Round, input.Tick)
	input.Round = t.snapshot.Round
	input.Actor = normalizeActor(input.Actor)
	input.Site = normalizeSite(input.Site)
	if input.Source == "" {
		input.Source = SourceDemoinfocsEvent
	}
	input.Position = normalizedPosition(input.Position, input.Source)
	t.snapshot.Tick = input.Tick
}

func (t *Tracker) ensureRoundLocked(round, tick int) {
	if round <= 0 {
		round = t.snapshot.Round
	}
	if round > 0 && round != t.snapshot.Round {
		t.beginRoundLocked(round, tick)
	}
}

func (t *Tracker) newEventLocked(eventType EventType, input EventInput) Event {
	t.eventSequence++
	return Event{
		ID:       fmt.Sprintf("objective-r%03d-t%09d-%s-%06d", input.Round, input.Tick, eventType, t.eventSequence),
		Sequence: t.eventSequence,
		Type:     eventType,
		Round:    input.Round,
		Tick:     input.Tick,
		Actor:    input.Actor,
		Site:     input.Site,
		Position: normalizedPosition(input.Position, input.Source),
		EntityID: cloneInt(input.EntityID),
		HasKit:   cloneBool(input.HasKit),
		Source:   input.Source,
	}
}

func (t *Tracker) recordLocked(eventType EventType, input EventInput) Event {
	event := t.newEventLocked(eventType, input)
	event.StateAfter = t.snapshot.State
	event.PhaseAfter = t.snapshot.Phase
	if eventType == EventExplode {
		event.Site = firstNonEmpty(event.Site, t.snapshot.Site)
	}
	t.appendEventLocked(event)
	return cloneEvent(event)
}

func (t *Tracker) appendEventLocked(event Event) {
	t.events = append(t.events, cloneEvent(event))
	summary := t.summaries[event.Round]
	summary.Round = event.Round
	summary.WasPlanted = t.snapshot.WasPlantedThisRound
	summary.Site = t.snapshot.Site
	summary.PlantTick = t.snapshot.PlantTick
	summary.Resolution = t.snapshot.Resolution
	summary.ResolutionTick = t.snapshot.ResolutionTick
	summary.FinalState = t.snapshot.State
	summary.EventCount++
	t.summaries[event.Round] = summary
}

func (t *Tracker) discardPickupAtTickLocked(round, tick int, actorID uint64) {
	for index := len(t.events) - 1; index >= 0; index-- {
		event := t.events[index]
		if event.Round != round || event.Tick != tick {
			if event.Round < round || event.Round == round && event.Tick < tick {
				break
			}
			continue
		}
		if event.Type != EventPickup || actorID != 0 && event.Actor.SteamID != actorID {
			continue
		}
		t.events = append(t.events[:index], t.events[index+1:]...)
		summary := t.summaries[round]
		if summary.EventCount > 0 {
			summary.EventCount--
			t.summaries[round] = summary
		}
	}
}

func (t *Tracker) startAttemptLocked(kind AttemptKind, input EventInput, startEventID string, observed bool) int {
	active := t.activeAttemptIndexLocked(kind)
	if active >= 0 {
		if kind == AttemptPlant {
			t.activePlant = -1
		} else {
			t.activeDefuse = -1
		}
	}

	t.attemptSequence++
	attempt := Attempt{
		ID:            fmt.Sprintf("objective-r%03d-%s-%06d", input.Round, kind, t.attemptSequence),
		Sequence:      t.attemptSequence,
		Kind:          kind,
		Round:         input.Round,
		Actor:         input.Actor,
		Site:          input.Site,
		HasKit:        cloneBool(input.HasKit),
		StartTick:     input.Tick,
		Outcome:       AttemptInProgress,
		StartEventID:  startEventID,
		StartObserved: observed,
	}
	t.attempts = append(t.attempts, attempt)
	index := len(t.attempts) - 1
	if kind == AttemptPlant {
		t.activePlant = index
	} else {
		t.activeDefuse = index
	}
	summary := t.summaries[input.Round]
	summary.Round = input.Round
	summary.AttemptCount++
	t.summaries[input.Round] = summary
	return index
}

func (t *Tracker) ensureAttemptLocked(kind AttemptKind, input EventInput) int {
	if active := t.activeAttemptIndexLocked(kind); active >= 0 {
		return active
	}
	return t.startAttemptLocked(kind, input, "", false)
}

func (t *Tracker) finishAttemptLocked(kind AttemptKind, outcome AttemptOutcome, endTick int, endEventID string) {
	index := t.activeAttemptIndexLocked(kind)
	if index < 0 {
		return
	}
	attempt := &t.attempts[index]
	duration := endTick - attempt.StartTick
	if duration < 0 {
		duration = 0
	}
	attempt.EndTick = intPointer(endTick)
	if attempt.StartObserved {
		attempt.DurationTicks = intPointer(duration)
	} else {
		attempt.DurationTicks = nil
	}
	attempt.Outcome = outcome
	attempt.EndEventID = endEventID
	if kind == AttemptPlant {
		t.activePlant = -1
	} else {
		t.activeDefuse = -1
	}
}

func (t *Tracker) activeAttemptIndexLocked(kind AttemptKind) int {
	if kind == AttemptPlant {
		return t.activePlant
	}
	return t.activeDefuse
}

func (t *Tracker) attachAttemptLocked(event *Event, attempt *Attempt, outcome AttemptOutcome) {
	event.AttemptID = attempt.ID
	event.AttemptSequence = attempt.Sequence
	event.AttemptOutcome = outcome
	event.AttemptStartObserved = attempt.StartObserved
	if outcome != AttemptInProgress {
		event.RelatedEventID = attempt.StartEventID
	}
	event.DurationTicks = cloneInt(attempt.DurationTicks)
	event.HasKit = cloneBool(attempt.HasKit)
	if event.Site == "" {
		event.Site = firstNonEmpty(attempt.Site, t.snapshot.Site)
	}
}

func (t *Tracker) applySiteLocked(site string) {
	if site = normalizeSite(site); site != "" {
		t.snapshot.Site = site
	}
}

func (t *Tracker) applyObservedPositionLocked(position Position) {
	if position.Available() {
		t.snapshot.Position = position
	}
}

func isPreplantState(state State) bool {
	switch state {
	case StateUnknown, StateCarried, StateDropped:
		return true
	default:
		return false
	}
}

func normalizeActor(actor Actor) Actor {
	actor.Side = strings.ToUpper(strings.TrimSpace(actor.Side))
	if actor.Side != "T" && actor.Side != "CT" {
		actor.Side = ""
	}
	return actor
}

func normalizeSite(site string) string {
	site = strings.ToUpper(strings.TrimSpace(site))
	if site == "A" || site == "B" {
		return site
	}
	return ""
}

func normalizedPosition(position Position, fallbackSource string) Position {
	if position.Status == "" {
		return UnavailablePosition(fallbackSource)
	}
	if position.Source == "" {
		position.Source = fallbackSource
	}
	if position.Status != PositionObserved ||
		math.IsNaN(position.X) || math.IsInf(position.X, 0) ||
		math.IsNaN(position.Y) || math.IsInf(position.Y, 0) ||
		math.IsNaN(position.Z) || math.IsInf(position.Z, 0) {
		position.Status = PositionUnavailable
		position.X = 0
		position.Y = 0
		position.Z = 0
	}
	return position
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cloneEvent(event Event) Event {
	event.EntityID = cloneInt(event.EntityID)
	event.HasKit = cloneBool(event.HasKit)
	event.DurationTicks = cloneInt(event.DurationTicks)
	return event
}

func cloneAttempt(attempt Attempt) Attempt {
	attempt.HasKit = cloneBool(attempt.HasKit)
	attempt.EndTick = cloneInt(attempt.EndTick)
	attempt.DurationTicks = cloneInt(attempt.DurationTicks)
	return attempt
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func intPointer(value int) *int {
	return &value
}
