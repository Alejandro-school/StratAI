package combat

import "sort"

func Summaries(events []Event) map[uint64]PlayerSummary {
	summaries := make(map[uint64]PlayerSummary)
	for _, event := range events {
		ensureSummary(summaries, event.Actor)
		ensureSummary(summaries, event.Target)
		ensureSummary(summaries, event.Assister)
	}

	for _, event := range events {
		switch event.Type {
		case EventWeaponFire:
			applyFireSummary(summaries, event)
		case EventKill:
			applyKillSummary(summaries, event)
		}
	}
	applyDamageSummaries(summaries, events)
	return summaries
}

func applyFireSummary(summaries map[uint64]PlayerSummary, event Event) {
	if event.Actor.Status != AvailabilityObserved || weaponIsUtility(event.Weapon) {
		return
	}
	summary := summaries[event.Actor.ID]
	summary.ShotsFired++
	weapon := summary.WeaponStats[event.Weapon.Name]
	weapon.ShotsFired++
	switch event.ShotResult {
	case ShotHit:
		summary.ShotsHit++
		weapon.ShotsHit++
	case ShotMiss:
		summary.ShotsMissed++
		weapon.ShotsMissed++
	}
	summary.WeaponStats[event.Weapon.Name] = weapon
	summaries[event.Actor.ID] = summary
}

func applyKillSummary(summaries map[uint64]PlayerSummary, event Event) {
	if event.Target.Status == AvailabilityObserved {
		target := summaries[event.Target.ID]
		target.Deaths++
		summaries[event.Target.ID] = target
	}
	if event.Relation != RelationEnemy || event.Actor.Status != AvailabilityObserved {
		return
	}
	actor := summaries[event.Actor.ID]
	actor.Kills++
	if event.IsHeadshot != nil && *event.IsHeadshot {
		actor.Headshots++
	}
	if event.Weapon.Status == AvailabilityObserved && !weaponIsUtility(event.Weapon) {
		weapon := actor.WeaponStats[event.Weapon.Name]
		weapon.Kills++
		if event.IsHeadshot != nil && *event.IsHeadshot {
			weapon.Headshots++
		}
		actor.WeaponStats[event.Weapon.Name] = weapon
	}
	summaries[event.Actor.ID] = actor

	if event.Assister.Status != AvailabilityObserved || event.Assister.ID == event.Actor.ID ||
		event.Assister.ID == event.Target.ID || event.Assister.Side == "" || event.Assister.Side != event.Actor.Side {
		return
	}
	assister := summaries[event.Assister.ID]
	assister.Assists++
	if event.AssistedFlash != nil && *event.AssistedFlash {
		assister.FlashAssists++
	}
	summaries[event.Assister.ID] = assister
}

func applyDamageSummaries(summaries map[uint64]PlayerSummary, events []Event) {
	groups := make(map[damageGroupKey]*damageGroup)
	for _, event := range events {
		if event.Type != EventPlayerHurt || event.Actor.Status != AvailabilityObserved {
			continue
		}
		key := damageGroupKey{
			round: event.Round, tick: event.Tick, actor: event.Actor.ID, target: event.Target.ID,
			weapon: event.Weapon.Name, relation: event.Relation,
		}
		group := groups[key]
		if group == nil {
			group = &damageGroup{minAfter: 101}
			groups[key] = group
		}
		if event.HealthBefore != nil && (!group.hasHealth || *event.HealthBefore > group.maxBefore) {
			group.maxBefore = *event.HealthBefore
			group.hasHealth = true
		}
		if event.HealthAfter != nil && (!group.hasAfter || *event.HealthAfter < group.minAfter) {
			group.minAfter = *event.HealthAfter
			group.hasAfter = true
		}
		if event.HealthDamageTaken != nil {
			group.fallbackDamage += *event.HealthDamageTaken
		}
		group.weapon = event.Weapon
		if event.Relation == RelationEnemy && event.Hitgroup != nil &&
			event.HealthDamageTaken != nil && *event.HealthDamageTaken > 0 {
			summary := summaries[event.Actor.ID]
			summary.BodyPartHits[*event.Hitgroup]++
			summaries[event.Actor.ID] = summary
		}
	}

	keys := make([]damageGroupKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].round != keys[right].round {
			return keys[left].round < keys[right].round
		}
		if keys[left].tick != keys[right].tick {
			return keys[left].tick < keys[right].tick
		}
		if keys[left].actor != keys[right].actor {
			return keys[left].actor < keys[right].actor
		}
		if keys[left].target != keys[right].target {
			return keys[left].target < keys[right].target
		}
		return keys[left].weapon < keys[right].weapon
	})
	for _, key := range keys {
		group := groups[key]
		damage := group.fallbackDamage
		if group.hasHealth && group.hasAfter {
			damage = max(0, group.maxBefore-group.minAfter)
		}
		summary := summaries[key.actor]
		switch key.relation {
		case RelationEnemy:
			summary.EnemyDamage += damage
			if group.weapon.Status == AvailabilityObserved && !weaponIsUtility(group.weapon) {
				weapon := summary.WeaponStats[group.weapon.Name]
				weapon.Damage += damage
				summary.WeaponStats[group.weapon.Name] = weapon
			}
		case RelationFriendly:
			summary.FriendlyDamage += damage
		case RelationSelf:
			summary.SelfDamage += damage
		}
		summaries[key.actor] = summary
	}
}

func ensureSummary(summaries map[uint64]PlayerSummary, player PlayerRef) {
	if player.Status != AvailabilityObserved {
		return
	}
	if _, exists := summaries[player.ID]; exists {
		return
	}
	summaries[player.ID] = PlayerSummary{
		BodyPartHits: make(map[string]int),
		WeaponStats:  make(map[string]WeaponSummary),
	}
}

type damageGroupKey struct {
	round    int
	tick     int
	actor    uint64
	target   uint64
	weapon   string
	relation Relation
}

type damageGroup struct {
	maxBefore      int
	minAfter       int
	fallbackDamage int
	hasHealth      bool
	hasAfter       bool
	weapon         WeaponRef
}

func validateLinks(events []Event) (invalid, future, duplicateEvents, duplicateShots int) {
	byID := make(map[string]Event, len(events))
	shotIDs := make(map[string]struct{})
	for _, event := range events {
		if _, exists := byID[event.LocalID]; exists {
			duplicateEvents++
		}
		byID[event.LocalID] = event
		if event.Type == EventWeaponFire {
			if _, exists := shotIDs[event.ShotID]; exists {
				duplicateShots++
			}
			shotIDs[event.ShotID] = struct{}{}
		}
	}
	for _, event := range events {
		seen := make(map[string]struct{}, len(event.SourceEventIDs))
		for _, sourceID := range event.SourceEventIDs {
			source, exists := byID[sourceID]
			if !exists {
				invalid++
				continue
			}
			if _, exists := seen[sourceID]; exists {
				invalid++
			}
			seen[sourceID] = struct{}{}
			if source.Round > event.Round || source.Round == event.Round && source.Tick > event.Tick {
				future++
			}
		}
		if event.Type == EventWeaponFire && event.ShotID == "" {
			invalid++
		}
		if event.Type != EventWeaponFire && event.ShotID != "" {
			if _, exists := shotIDs[event.ShotID]; !exists {
				invalid++
			}
		}
	}
	return invalid, future, duplicateEvents, duplicateShots
}
