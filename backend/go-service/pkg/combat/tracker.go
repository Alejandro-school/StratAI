package combat

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

const maxShotCorrelationTicks = 2

type Tracker struct {
	mu sync.RWMutex

	observations []Event
	roundEnds    map[int]int
	diagnostics  Diagnostics
}

func NewTracker() *Tracker {
	return &Tracker{
		observations: make([]Event, 0, 2048),
		roundEnds:    make(map[int]int),
		diagnostics: Diagnostics{
			ObservedByType:    make(map[EventType]int),
			RecordedByType:    make(map[EventType]int),
			DiscardedByType:   make(map[EventType]int),
			DiscardedByReason: make(map[string]int),
		},
	}
}

func (tracker *Tracker) RecordDiscardedCallback(eventType EventType, reason string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.ObservedByType[eventType]++
	tracker.diagnostics.DiscardedByType[eventType]++
	tracker.diagnostics.DiscardedByReason[reason]++
}

func (tracker *Tracker) RecordWeaponFire(input FireInput) {
	tracker.record(Event{
		Round: input.Round, Tick: input.Tick, Type: EventWeaponFire, Source: SourceWeaponFire,
		Actor: input.Actor, Weapon: input.Weapon, ViewYaw: cloneFloat(input.ViewYaw),
		ViewPitch: cloneFloat(input.ViewPitch), Ammo: cloneAmmo(input.Ammo),
	})
}

func (tracker *Tracker) RecordBulletDamage(input BulletDamageInput) {
	distance := input.Distance
	penetrations := input.PenetratedObjects
	noScope := input.NoScope
	inAir := input.AttackerInAir
	direction := input.Direction
	tracker.record(Event{
		Round: input.Round, Tick: input.Tick, Type: EventBulletDamage, Source: SourceBulletDamage,
		Actor: input.Actor, Target: input.Target, Relation: relation(input.Actor, input.Target),
		ImpactPositionStatus: AvailabilityUnavailable, ImpactPositionSource: SourceUnavailable,
		BulletDistance: &distance, DamageDirection: &direction, PenetratedObjects: &penetrations,
		NoScope: &noScope, AttackerInAir: &inAir,
	})
}

func (tracker *Tracker) RecordPlayerHurt(input HurtInput) {
	healthDamage := max(0, input.HealthDamage)
	healthTaken := max(0, input.HealthDamageTaken)
	armorDamage := max(0, input.ArmorDamage)
	armorTaken := max(0, input.ArmorDamageTaken)
	healthAfter := max(0, input.HealthAfter)
	armorAfter := max(0, input.ArmorAfter)
	healthBefore := min(100, healthAfter+healthTaken)
	armorBefore := armorAfter + armorTaken
	hitgroup := strings.ToLower(strings.TrimSpace(input.Hitgroup))
	headshot := hitgroup == "head"
	status := AvailabilityObserved
	if hitgroup == "" {
		status = AvailabilityUnavailable
	}
	event := Event{
		Round: input.Round, Tick: input.Tick, Type: EventPlayerHurt, Source: SourcePlayerHurt,
		Actor: input.Actor, Target: input.Target, Relation: relation(input.Actor, input.Target),
		Weapon: input.Weapon, HealthDamage: &healthDamage, HealthDamageTaken: &healthTaken,
		ArmorDamage: &armorDamage, ArmorDamageTaken: &armorTaken, HealthBefore: &healthBefore,
		HealthAfter: &healthAfter, ArmorBefore: &armorBefore, ArmorAfter: &armorAfter,
		DamageStatus: AvailabilityObserved, DamageSource: SourcePlayerHurt,
		HitgroupStatus: status, HitgroupSource: SourcePlayerHurt, IsHeadshot: &headshot,
	}
	if status == AvailabilityObserved {
		event.Hitgroup = &hitgroup
	} else {
		event.HitgroupSource = SourceUnavailable
	}
	tracker.record(event)
}

func (tracker *Tracker) RecordKill(input KillInput) {
	headshot := input.IsHeadshot
	assistedFlash := input.AssistedFlash
	penetrations := input.PenetratedObjects
	noScope := input.NoScope
	throughSmoke := input.ThroughSmoke
	attackerBlind := input.AttackerBlind
	distance := input.Distance
	event := Event{
		Round: input.Round, Tick: input.Tick, Type: EventKill, Source: SourceKill,
		Actor: input.Actor, Target: input.Target, Assister: input.Assister,
		Relation: relation(input.Actor, input.Target), Weapon: input.Weapon,
		IsHeadshot: &headshot, IsKill: true, AssistedFlash: &assistedFlash,
		PenetratedObjects: &penetrations, NoScope: &noScope,
		ThroughSmoke: &throughSmoke, AttackerBlind: &attackerBlind, KillDistance: &distance,
		DamageStatus: AvailabilityUnavailable, DamageSource: SourceUnavailable,
		ImpactPositionStatus: AvailabilityUnavailable, ImpactPositionSource: SourceUnavailable,
	}
	if headshot {
		hitgroup := "head"
		event.Hitgroup = &hitgroup
		event.HitgroupStatus = AvailabilityObserved
		event.HitgroupSource = SourceKill
	} else {
		event.HitgroupStatus = AvailabilityUnavailable
		event.HitgroupSource = SourceUnavailable
	}
	tracker.record(event)
}

func (tracker *Tracker) RecordWeaponReload(input ReloadInput) {
	phase := "start"
	tracker.record(Event{
		Round: input.Round, Tick: input.Tick, Type: EventWeaponReload, Source: SourceWeaponReload,
		Actor: input.Actor, Weapon: input.Weapon, Ammo: cloneAmmo(input.Ammo), ReloadPhase: &phase,
		ReloadEndStatus: AvailabilityUnavailable,
	})
}

func (tracker *Tracker) RecordWeaponEquip(input EquipInput) {
	tracker.record(Event{
		Round: input.Round, Tick: input.Tick, Type: EventWeaponEquip, Source: SourceItemEquip,
		Actor: input.Actor, Weapon: input.Weapon, Ammo: cloneAmmo(input.Ammo),
		PreviousWeaponStatus: AvailabilityUnavailable,
	})
}

func (tracker *Tracker) EndRound(round, tick int) {
	if round <= 0 || tick < 0 {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if prior, exists := tracker.roundEnds[round]; !exists || tick > prior {
		tracker.roundEnds[round] = tick
	}
}

func (tracker *Tracker) Snapshot() []Event {
	tracker.mu.RLock()
	observations := cloneEvents(tracker.observations)
	roundEnds := make(map[int]int, len(tracker.roundEnds))
	for round, tick := range tracker.roundEnds {
		roundEnds[round] = tick
	}
	tracker.mu.RUnlock()

	sort.Slice(observations, func(left, right int) bool {
		return eventLess(observations[left], observations[right])
	})
	assignIdentities(observations)
	correlate(observations, roundEnds)
	observeWeaponLifecycle(observations)
	return observations
}

func (tracker *Tracker) Diagnostics() Diagnostics {
	events := tracker.Snapshot()
	tracker.mu.RLock()
	diagnostics := cloneDiagnostics(tracker.diagnostics)
	tracker.mu.RUnlock()
	diagnostics.InvalidLinks, diagnostics.FutureLinks,
		diagnostics.DuplicateLocalIDs, diagnostics.DuplicateShotIDs = validateLinks(events)
	return diagnostics
}

func (tracker *Tracker) record(event Event) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.diagnostics.ObservedByType[event.Type]++
	if event.Round <= 0 || event.Tick < 0 {
		tracker.diagnostics.DiscardedByType[event.Type]++
		tracker.diagnostics.DiscardedByReason[DiscardInvalidObservation]++
		return
	}
	event.Actor = normalizePlayer(event.Actor)
	event.Target = normalizePlayer(event.Target)
	event.Assister = normalizePlayer(event.Assister)
	event.Weapon = normalizeWeapon(event.Weapon)
	event.Ammo = normalizeAmmo(event.Ammo)
	if event.Relation == "" {
		event.Relation = relation(event.Actor, event.Target)
	}
	event.CorrelationStatus = CorrelationUnavailable
	event.CorrelationSource = SourceUnavailable
	event.ShotResult = ShotUnavailable
	event.ShotResultStatus = AvailabilityUnavailable
	event.ShotResultSource = SourceUnavailable
	if event.ImpactPositionStatus == "" {
		event.ImpactPositionStatus = AvailabilityUnavailable
		event.ImpactPositionSource = SourceUnavailable
	}
	if event.DamageStatus == "" {
		event.DamageStatus = AvailabilityUnavailable
		event.DamageSource = SourceUnavailable
	}
	if event.HitgroupStatus == "" {
		event.HitgroupStatus = AvailabilityUnavailable
		event.HitgroupSource = SourceUnavailable
	}
	if event.ReloadEndStatus == "" {
		event.ReloadEndStatus = AvailabilityUnavailable
	}
	if event.PreviousWeaponStatus == "" {
		event.PreviousWeaponStatus = AvailabilityUnavailable
	}
	tracker.observations = append(tracker.observations, event)
	tracker.diagnostics.RecordedByType[event.Type]++
}

func assignIdentities(events []Event) {
	roundSequence := 0
	tickSequence := 0
	lastRound := -1
	lastTick := -1
	shotSequenceByRound := make(map[int]int)
	for index := range events {
		event := &events[index]
		if event.Round != lastRound {
			roundSequence = 0
		}
		if event.Round != lastRound || event.Tick != lastTick {
			tickSequence = 0
		}
		roundSequence++
		tickSequence++
		event.SequenceInRound = roundSequence
		event.SequenceInTick = tickSequence
		event.LocalID = fmt.Sprintf("r%03d-e%06d", event.Round, roundSequence)
		if event.Type == EventWeaponFire {
			shotSequenceByRound[event.Round]++
			event.ShotID = fmt.Sprintf("r%03d-s%06d", event.Round, shotSequenceByRound[event.Round])
		}
		lastRound = event.Round
		lastTick = event.Tick
	}
}

func correlate(events []Event, roundEnds map[int]int) {
	firesByActor := make(map[uint64][]int)
	bulletsByPair := make(map[eventPair][]int)
	hitByShot := make(map[string]int)
	for index := range events {
		event := &events[index]
		switch event.Type {
		case EventWeaponFire:
			if event.Actor.Status == AvailabilityObserved {
				firesByActor[event.Actor.ID] = append(firesByActor[event.Actor.ID], index)
			}
		case EventBulletDamage:
			linkToFire(events, index, firesByActor[event.Actor.ID], hitByShot)
			if event.Actor.Status == AvailabilityObserved && event.Target.Status == AvailabilityObserved {
				pair := eventPair{round: event.Round, tick: event.Tick, actor: event.Actor.ID, target: event.Target.ID}
				bulletsByPair[pair] = append(bulletsByPair[pair], index)
			}
		case EventPlayerHurt:
			linkToFire(events, index, firesByActor[event.Actor.ID], hitByShot)
			linkToBullet(events, index, bulletsByPair)
		case EventKill:
			linkKill(events, index, firesByActor[event.Actor.ID])
		}
	}
	for index := range events {
		event := &events[index]
		if event.Type != EventWeaponFire {
			continue
		}
		if tick, hit := hitByShot[event.ShotID]; hit {
			event.ShotResult = ShotHit
			event.ShotResultStatus = AvailabilityDerived
			event.ShotResultSource = SourcePlayerHurt
			event.ShotResultAvailabilityTick = intPointer(tick)
			continue
		}
		endTick, closed := roundEnds[event.Round]
		if closed && endTick >= event.Tick && !weaponIsUtility(event.Weapon) {
			event.ShotResult = ShotMiss
			event.ShotResultStatus = AvailabilityDerived
			event.ShotResultSource = SourceRoundClosure
			event.ShotResultAvailabilityTick = intPointer(endTick)
		}
	}
}

func linkToFire(events []Event, eventIndex int, candidates []int, hitByShot map[string]int) {
	event := &events[eventIndex]
	for candidateIndex := len(candidates) - 1; candidateIndex >= 0; candidateIndex-- {
		fire := &events[candidates[candidateIndex]]
		if fire.Round != event.Round || fire.Tick > event.Tick || event.Tick-fire.Tick > maxShotCorrelationTicks {
			continue
		}
		if event.Weapon.Status == AvailabilityObserved && fire.Weapon.Status == AvailabilityObserved &&
			event.Weapon.Name != fire.Weapon.Name {
			continue
		}
		event.ShotID = fire.ShotID
		event.SourceEventIDs = appendUnique(event.SourceEventIDs, fire.LocalID)
		if fire.Tick == event.Tick {
			event.CorrelationStatus = CorrelationExact
		} else {
			event.CorrelationStatus = CorrelationInferred
		}
		event.CorrelationSource = SourceFireCorrelation
		if event.Type == EventPlayerHurt && event.HealthDamageTaken != nil && *event.HealthDamageTaken > 0 {
			if prior, exists := hitByShot[fire.ShotID]; !exists || event.Tick > prior {
				hitByShot[fire.ShotID] = event.Tick
			}
		}
		return
	}
}

func linkToBullet(events []Event, eventIndex int, bullets map[eventPair][]int) {
	event := &events[eventIndex]
	if event.Actor.Status != AvailabilityObserved || event.Target.Status != AvailabilityObserved {
		return
	}
	pair := eventPair{round: event.Round, tick: event.Tick, actor: event.Actor.ID, target: event.Target.ID}
	candidates := bullets[pair]
	if len(candidates) == 0 {
		return
	}
	bulletIndex := candidates[0]
	bullets[pair] = candidates[1:]
	bullet := &events[bulletIndex]
	event.SourceEventIDs = appendUnique(event.SourceEventIDs, bullet.LocalID)
	if event.ShotID == "" && bullet.ShotID != "" {
		event.ShotID = bullet.ShotID
	}
	if event.CorrelationStatus == CorrelationUnavailable {
		event.CorrelationStatus = CorrelationExact
		event.CorrelationSource = SourceBulletCorrelation
	}
}

func linkKill(events []Event, killIndex int, fireCandidates []int) {
	kill := &events[killIndex]
	for index := killIndex - 1; index >= 0; index-- {
		hurt := &events[index]
		if hurt.Round != kill.Round || hurt.Tick != kill.Tick {
			if hurt.Round < kill.Round || hurt.Tick < kill.Tick {
				break
			}
			continue
		}
		if hurt.Type != EventPlayerHurt || !sameObservedPlayer(hurt.Actor, kill.Actor) ||
			!sameObservedPlayer(hurt.Target, kill.Target) || hurt.HealthAfter == nil || *hurt.HealthAfter != 0 {
			continue
		}
		kill.SourceEventIDs = appendUnique(kill.SourceEventIDs, hurt.LocalID)
		kill.ShotID = hurt.ShotID
		kill.CorrelationStatus = CorrelationExact
		kill.CorrelationSource = SourceFatalHurt
		return
	}
	linkToFire(events, killIndex, fireCandidates, map[string]int{})
}

func observeWeaponLifecycle(events []Event) {
	lastWeapon := make(map[uint64]string)
	for index := range events {
		event := &events[index]
		if event.Actor.Status != AvailabilityObserved {
			continue
		}
		previous, hasPrevious := lastWeapon[event.Actor.ID]
		if event.Type == EventWeaponEquip {
			if hasPrevious {
				event.PreviousWeapon = &previous
				event.PreviousWeaponStatus = AvailabilityDerived
				switchValue := event.Weapon.Status == AvailabilityObserved && previous != event.Weapon.Name
				event.IsWeaponSwitch = &switchValue
			}
		}
		if event.Weapon.Status == AvailabilityObserved {
			lastWeapon[event.Actor.ID] = event.Weapon.Name
		}
	}
}

func eventLess(left, right Event) bool {
	leftKey := eventSortKey(left)
	rightKey := eventSortKey(right)
	for index := range leftKey {
		if leftKey[index] != rightKey[index] {
			return leftKey[index] < rightKey[index]
		}
	}
	return false
}

func eventSortKey(event Event) []string {
	return []string{
		fmt.Sprintf("%09d", event.Round), fmt.Sprintf("%012d", event.Tick),
		fmt.Sprintf("%02d", eventTypePriority(event.Type)), fmt.Sprintf("%020d", event.Actor.ID),
		fmt.Sprintf("%020d", event.Target.ID), fmt.Sprintf("%020d", event.Assister.ID),
		event.Weapon.Name, fmt.Sprintf("%09d", pointerValue(event.HealthDamageTaken)),
		fmt.Sprintf("%09d", pointerValue(event.ArmorDamageTaken)), pointerString(event.Hitgroup),
		fmt.Sprintf("%t", pointerBoolValue(event.IsHeadshot)), event.Source,
	}
}

func eventTypePriority(eventType EventType) int {
	switch eventType {
	case EventWeaponEquip:
		return 0
	case EventWeaponReload:
		return 1
	case EventWeaponFire:
		return 2
	case EventBulletDamage:
		return 3
	case EventPlayerHurt:
		return 4
	case EventKill:
		return 5
	default:
		return 99
	}
}

type eventPair struct {
	round  int
	tick   int
	actor  uint64
	target uint64
}

func relation(actor, target PlayerRef) Relation {
	if target.Status != AvailabilityObserved {
		return RelationUnknown
	}
	if actor.Status != AvailabilityObserved {
		return RelationWorld
	}
	if actor.ID == target.ID {
		return RelationSelf
	}
	if actor.Side != "" && target.Side != "" && actor.Side == target.Side {
		return RelationFriendly
	}
	if actor.Side != "" && target.Side != "" {
		return RelationEnemy
	}
	return RelationUnknown
}

func normalizePlayer(player PlayerRef) PlayerRef {
	if player.Status != AvailabilityObserved || player.ID == 0 {
		return PlayerRef{Status: AvailabilityUnavailable, Source: SourceUnavailable, PositionStatus: AvailabilityUnavailable, PositionSource: SourceUnavailable}
	}
	if player.Source == "" {
		player.Source = SourceCallbackPlayer
	}
	if player.Position == nil || !finiteVector(*player.Position) {
		player.Position = nil
		player.PositionStatus = AvailabilityUnavailable
		player.PositionSource = SourceUnavailable
	} else {
		player.PositionStatus = AvailabilityObserved
		if player.PositionSource == "" {
			player.PositionSource = SourceCallbackPosition
		}
	}
	return player
}

func normalizeWeapon(weapon WeaponRef) WeaponRef {
	weapon.Name = strings.TrimSpace(weapon.Name)
	if weapon.Status != AvailabilityObserved || weapon.Name == "" {
		return WeaponRef{Status: AvailabilityUnavailable, Source: SourceUnavailable}
	}
	if weapon.Source == "" {
		weapon.Source = SourceCallbackWeapon
	}
	weapon.IsUtility = cloneBool(weapon.IsUtility)
	return weapon
}

func normalizeAmmo(ammo AmmoObservation) AmmoObservation {
	if ammo.Status != AvailabilityObserved || (ammo.InMagazine == nil && ammo.Reserve == nil) {
		return AmmoObservation{Status: AvailabilityUnavailable, Source: SourceUnavailable}
	}
	if ammo.Source == "" {
		ammo.Source = SourceActiveWeaponAmmo
	}
	ammo.InMagazine = cloneInt(ammo.InMagazine)
	ammo.Reserve = cloneInt(ammo.Reserve)
	return ammo
}

func weaponIsUtility(weapon WeaponRef) bool {
	return weapon.IsUtility != nil && *weapon.IsUtility
}

func finiteVector(vector Vector) bool {
	return !math.IsNaN(vector.X) && !math.IsInf(vector.X, 0) &&
		!math.IsNaN(vector.Y) && !math.IsInf(vector.Y, 0) &&
		!math.IsNaN(vector.Z) && !math.IsInf(vector.Z, 0)
}

func sameObservedPlayer(left, right PlayerRef) bool {
	return left.Status == AvailabilityObserved && right.Status == AvailabilityObserved && left.ID == right.ID
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func cloneEvents(source []Event) []Event {
	result := make([]Event, len(source))
	for index := range source {
		result[index] = source[index]
		result[index].SourceEventIDs = append([]string(nil), source[index].SourceEventIDs...)
		result[index].Actor = clonePlayer(source[index].Actor)
		result[index].Target = clonePlayer(source[index].Target)
		result[index].Assister = clonePlayer(source[index].Assister)
		result[index].Weapon = normalizeWeapon(source[index].Weapon)
		result[index].ViewYaw = cloneFloat(source[index].ViewYaw)
		result[index].ViewPitch = cloneFloat(source[index].ViewPitch)
		result[index].Ammo = cloneAmmo(source[index].Ammo)
		result[index].ShotResultAvailabilityTick = cloneInt(source[index].ShotResultAvailabilityTick)
		result[index].HealthDamage = cloneInt(source[index].HealthDamage)
		result[index].HealthDamageTaken = cloneInt(source[index].HealthDamageTaken)
		result[index].ArmorDamage = cloneInt(source[index].ArmorDamage)
		result[index].ArmorDamageTaken = cloneInt(source[index].ArmorDamageTaken)
		result[index].HealthBefore = cloneInt(source[index].HealthBefore)
		result[index].HealthAfter = cloneInt(source[index].HealthAfter)
		result[index].ArmorBefore = cloneInt(source[index].ArmorBefore)
		result[index].ArmorAfter = cloneInt(source[index].ArmorAfter)
		result[index].Hitgroup = cloneString(source[index].Hitgroup)
		result[index].IsHeadshot = cloneBool(source[index].IsHeadshot)
		result[index].AssistedFlash = cloneBool(source[index].AssistedFlash)
		result[index].ImpactPosition = cloneVector(source[index].ImpactPosition)
		result[index].BulletDistance = cloneFloat(source[index].BulletDistance)
		result[index].DamageDirection = cloneVector(source[index].DamageDirection)
		result[index].PenetratedObjects = cloneInt(source[index].PenetratedObjects)
		result[index].NoScope = cloneBool(source[index].NoScope)
		result[index].AttackerInAir = cloneBool(source[index].AttackerInAir)
		result[index].ThroughSmoke = cloneBool(source[index].ThroughSmoke)
		result[index].AttackerBlind = cloneBool(source[index].AttackerBlind)
		result[index].KillDistance = cloneFloat(source[index].KillDistance)
		result[index].ReloadPhase = cloneString(source[index].ReloadPhase)
		result[index].ReloadEndTick = cloneInt(source[index].ReloadEndTick)
		result[index].PreviousWeapon = cloneString(source[index].PreviousWeapon)
		result[index].IsWeaponSwitch = cloneBool(source[index].IsWeaponSwitch)
	}
	return result
}

func clonePlayer(player PlayerRef) PlayerRef {
	player.Position = cloneVector(player.Position)
	return player
}

func cloneAmmo(ammo AmmoObservation) AmmoObservation {
	ammo.InMagazine = cloneInt(ammo.InMagazine)
	ammo.Reserve = cloneInt(ammo.Reserve)
	return ammo
}

func cloneDiagnostics(source Diagnostics) Diagnostics {
	result := source
	result.ObservedByType = cloneCounts(source.ObservedByType)
	result.RecordedByType = cloneCounts(source.RecordedByType)
	result.DiscardedByType = cloneCounts(source.DiscardedByType)
	result.DiscardedByReason = make(map[string]int, len(source.DiscardedByReason))
	for reason, count := range source.DiscardedByReason {
		result.DiscardedByReason[reason] = count
	}
	return result
}

func cloneCounts(source map[EventType]int) map[EventType]int {
	result := make(map[EventType]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneFloat(value *float64) *float64 {
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

func cloneVector(value *Vector) *Vector {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func intPointer(value int) *int {
	return &value
}

func pointerValue(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pointerBoolValue(value *bool) bool {
	return value != nil && *value
}
