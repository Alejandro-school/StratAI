package engagement

import (
	"sort"

	"cs2-demo-service/models"
)

var eventPriority = map[string]int{
	"weapon_fire":   1,
	"bullet_damage": 2,
	"player_hurt":   3,
	"kill":          4,
}

func sortedEvents(source []models.CanonicalCombatEvent) []models.CanonicalCombatEvent {
	events := append([]models.CanonicalCombatEvent(nil), source...)
	sort.Slice(events, func(i, j int) bool { return eventLess(events[i], events[j]) })
	return events
}

func eventLess(left, right models.CanonicalCombatEvent) bool {
	if left.RoundNumber != right.RoundNumber {
		return left.RoundNumber < right.RoundNumber
	}
	if left.Tick != right.Tick {
		return left.Tick < right.Tick
	}
	if left.SequenceInTick != right.SequenceInTick {
		return left.SequenceInTick < right.SequenceInTick
	}
	if eventPriority[left.EventType] != eventPriority[right.EventType] {
		return eventPriority[left.EventType] < eventPriority[right.EventType]
	}
	if compareOptionalString(left.ActorPlayerID, right.ActorPlayerID) != 0 {
		return compareOptionalString(left.ActorPlayerID, right.ActorPlayerID) < 0
	}
	if compareOptionalString(left.TargetPlayerID, right.TargetPlayerID) != 0 {
		return compareOptionalString(left.TargetPlayerID, right.TargetPlayerID) < 0
	}
	if compareOptionalString(left.ShotID, right.ShotID) != 0 {
		return compareOptionalString(left.ShotID, right.ShotID) < 0
	}
	return left.EventID < right.EventID
}

func eventBefore(left, right models.CanonicalCombatEvent) bool {
	return eventLess(left, right)
}

func eventAfter(left, right models.CanonicalCombatEvent) bool {
	return eventLess(right, left)
}

func compareOptionalString(left, right *string) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if *left < *right {
		return -1
	}
	if *left > *right {
		return 1
	}
	return 0
}

func eventClosure(
	event models.CanonicalCombatEvent,
	byID map[string]models.CanonicalCombatEvent,
) []models.CanonicalCombatEvent {
	seen := make(map[string]struct{})
	stack := []models.CanonicalCombatEvent{event}
	closure := make([]models.CanonicalCombatEvent, 0, len(event.SourceEventIDs)+1)
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, exists := seen[current.EventID]; exists {
			continue
		}
		seen[current.EventID] = struct{}{}
		closure = append(closure, current)
		for _, sourceID := range current.SourceEventIDs {
			if source, exists := byID[sourceID]; exists {
				stack = append(stack, source)
			}
		}
	}
	sort.Slice(closure, func(i, j int) bool { return eventLess(closure[i], closure[j]) })
	return closure
}

func sortedUniqueStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func stringPointer(value string) *string {
	copy := value
	return &copy
}

func intPointer(value int) *int {
	copy := value
	return &copy
}

func floatPointer(value float64) *float64 {
	copy := value
	return &copy
}
