package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

const (
	weaponAbstentionActiveUnavailable = "active_weapon_unavailable"
	weaponAbstentionNonAmmunition     = "non_ammunition_weapon"
	weaponAbstentionAmmoSentinel      = "invalid_ammo_sentinel"
	weaponAbstentionAvailability      = "availability_after_t0"
	weaponAbstentionEventAmbiguity    = "event_reconciliation_ambiguous"
)

type observedWeaponFrameSample struct {
	tick    int
	players []observedWeaponPlayerSample
	key     string
}

type observedWeaponPlayerSample struct {
	steamID     uint64
	observation models.ReplayWeaponStateObservation
}

type observedWeaponEventMatch struct {
	fireEvents   []combat.Event
	reloadEvents []combat.Event
}

type observedWeaponEventKey struct {
	round  int
	tick   int
	actor  uint64
	weapon string
}

type observedWeaponEventIndex map[observedWeaponEventKey]observedWeaponEventMatch

// BuildObservedWeaponStateLedger publishes only candidate facts available at
// or before each 16 Hz T0. Events can confirm a same-tick snapshot, but never
// fill a missing value. The canonical 3.8.0 export is not modified.
func BuildObservedWeaponStateLedger(
	replay *models.ReplayData,
	events []combat.Event,
	cutoffTick int,
) (models.ObservedWeaponStateLedger, error) {
	ledger := newObservedWeaponStateLedger()
	if replay == nil {
		return ledger, fmt.Errorf("build observed weapon state: replay data is nil")
	}
	matchID := strings.TrimSpace(replay.Metadata.MatchID)
	if matchID == "" {
		return ledger, fmt.Errorf("build observed weapon state: replay metadata match_id is empty")
	}
	if cutoffTick < 0 {
		return ledger, fmt.Errorf("build observed weapon state: cutoff tick must be non-negative")
	}
	tickRate := replay.Metadata.TickRate
	if math.IsNaN(tickRate) || math.IsInf(tickRate, 0) || tickRate < models.TacticalTargetHz {
		return ledger, fmt.Errorf(
			"build observed weapon state: tick_rate must be finite and at least %d, got %v",
			models.TacticalTargetHz,
			tickRate,
		)
	}

	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		if rounds[i].StartTick != rounds[j].StartTick {
			return rounds[i].StartTick < rounds[j].StartTick
		}
		return rounds[i].EndTick < rounds[j].EndTick
	})

	periodTicks := tickRate / float64(models.TacticalTargetHz)
	playerAliases := observedWeaponPlayerAliases(rounds, cutoffTick)
	eventIndex := indexObservedWeaponEvents(events)
	seenRounds := make(map[int]struct{}, len(rounds))
	for _, round := range rounds {
		if round.Round <= 0 {
			return ledger, fmt.Errorf("build observed weapon state: round number must be positive, got %d", round.Round)
		}
		if _, exists := seenRounds[round.Round]; exists {
			return ledger, fmt.Errorf("build observed weapon state: duplicate round number %d", round.Round)
		}
		seenRounds[round.Round] = struct{}{}
		if round.EndTick < round.StartTick {
			return ledger, fmt.Errorf("build observed weapon state: round %d has reversed tick bounds", round.Round)
		}

		frames, err := canonicalObservedWeaponFrames(round)
		if err != nil {
			return ledger, fmt.Errorf("build observed weapon state: round %d: %w", round.Round, err)
		}
		targetTicks := tacticalTargetTicks(round.StartTick, round.EndTick, periodTicks)
		roundID := fmt.Sprintf("%s:round:%03d", matchID, round.Round)
		frameIndex := 0
		previousTarget := round.StartTick
		for targetIndex, targetTick := range targetTicks {
			if targetTick > cutoffTick {
				break
			}
			var selected *observedWeaponFrameSample
			for frameIndex < len(frames) && frames[frameIndex].tick <= targetTick {
				candidate := &frames[frameIndex]
				if targetIndex == 0 || candidate.tick > previousTarget {
					selected = candidate
				}
				frameIndex++
			}
			if selected == nil {
				previousTarget = targetTick
				continue
			}

			for _, player := range selected.players {
				playerID, exists := playerAliases[player.steamID]
				if !exists {
					continue
				}
				row := observedWeaponStateRow(
					matchID,
					roundID,
					round.Round,
					targetTick,
					selected.tick,
					player,
					playerID,
				)
				reconcileObservedWeaponEvents(&row, player.steamID, selected.tick, eventIndex)
				canonicalizeWeaponAbstentions(&row)
				ledger.Rows = append(ledger.Rows, row)
			}
			previousTarget = targetTick
		}
	}

	sort.Slice(ledger.Rows, func(i, j int) bool {
		left, right := ledger.Rows[i], ledger.Rows[j]
		if left.RoundNumber != right.RoundNumber {
			return left.RoundNumber < right.RoundNumber
		}
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		return left.JoinKeys.PlayerID < right.JoinKeys.PlayerID
	})
	if err := ValidateObservedWeaponStateLedger(ledger); err != nil {
		return models.ObservedWeaponStateLedger{}, err
	}
	return ledger, nil
}

func newObservedWeaponStateLedger() models.ObservedWeaponStateLedger {
	return models.ObservedWeaponStateLedger{
		SchemaID:          models.ObservedWeaponStateSchemaID,
		ContractVersion:   models.ObservedWeaponStateContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		CausalRole:        models.ObservedWeaponStateCausalRole,
		Sampling: models.ObservedWeaponSampling{
			TargetHz: models.TacticalTargetHz,
			Strategy: models.TacticalSamplingStrategy,
		},
		Rows: make([]models.ObservedWeaponStateRow, 0),
	}
}

func canonicalObservedWeaponFrames(round models.ReplayRound) ([]observedWeaponFrameSample, error) {
	byTick := make(map[int]observedWeaponFrameSample, len(round.Frames))
	for _, frame := range round.Frames {
		if frame.Tick < round.StartTick || frame.Tick > round.EndTick {
			continue
		}
		players := canonicalObservedWeaponPlayers(frame.Players)
		keyBytes, err := json.Marshal(observedWeaponPlayersKey(players))
		if err != nil {
			return nil, fmt.Errorf("frame tick %d: canonicalize players: %w", frame.Tick, err)
		}
		candidate := observedWeaponFrameSample{tick: frame.Tick, players: players, key: string(keyBytes)}
		current, exists := byTick[frame.Tick]
		if !exists || len(candidate.players) > len(current.players) ||
			(len(candidate.players) == len(current.players) && candidate.key < current.key) {
			byTick[frame.Tick] = candidate
		}
	}

	frames := make([]observedWeaponFrameSample, 0, len(byTick))
	for _, frame := range byTick {
		frames = append(frames, frame)
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].tick < frames[j].tick })
	return frames, nil
}

func canonicalObservedWeaponPlayers(players []models.ReplayPlayerState) []observedWeaponPlayerSample {
	byPlayer := make(map[uint64]observedWeaponPlayerSample, len(players))
	keys := make(map[uint64]string, len(players))
	scores := make(map[uint64]int, len(players))
	for _, player := range players {
		if player.SteamID == 0 {
			continue
		}
		candidate := observedWeaponPlayerSample{
			steamID:     player.SteamID,
			observation: cloneReplayWeaponObservation(player.ObservedWeaponState),
		}
		keyBytes, _ := json.Marshal(observedWeaponObservationKey(candidate.observation))
		key := string(keyBytes)
		score := observedWeaponObservationScore(candidate.observation)
		if currentKey, exists := keys[player.SteamID]; !exists || score > scores[player.SteamID] ||
			(score == scores[player.SteamID] && key < currentKey) {
			byPlayer[player.SteamID] = candidate
			keys[player.SteamID] = key
			scores[player.SteamID] = score
		}
	}
	result := make([]observedWeaponPlayerSample, 0, len(byPlayer))
	for _, player := range byPlayer {
		result = append(result, player)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].steamID < result[j].steamID })
	return result
}

type observedWeaponObservationKeyValue struct {
	ActiveWeapon                 *string
	ActiveWeaponStatus           string
	ActiveWeaponAvailabilityTick *int
	AmmoInMagazine               *int
	AmmoInMagazineStatus         string
	AmmoInMagazineTick           *int
	AmmoReserve                  *int
	AmmoReserveStatus            string
	AmmoReserveTick              *int
	IsReloading                  *bool
	IsReloadingStatus            string
	IsReloadingTick              *int
	NonAmmunitionWeapon          bool
	AbstentionReasons            []string
}

type observedWeaponPlayerKeyValue struct {
	SteamID     uint64
	Observation observedWeaponObservationKeyValue
}

func observedWeaponPlayersKey(players []observedWeaponPlayerSample) []observedWeaponPlayerKeyValue {
	result := make([]observedWeaponPlayerKeyValue, 0, len(players))
	for _, player := range players {
		result = append(result, observedWeaponPlayerKeyValue{
			SteamID:     player.steamID,
			Observation: observedWeaponObservationKey(player.observation),
		})
	}
	return result
}

func observedWeaponObservationKey(value models.ReplayWeaponStateObservation) observedWeaponObservationKeyValue {
	return observedWeaponObservationKeyValue{
		ActiveWeapon:                 value.ActiveWeapon,
		ActiveWeaponStatus:           value.ActiveWeaponStatus,
		ActiveWeaponAvailabilityTick: value.ActiveWeaponAvailabilityTick,
		AmmoInMagazine:               value.AmmoInMagazine,
		AmmoInMagazineStatus:         value.AmmoInMagazineStatus,
		AmmoInMagazineTick:           value.AmmoInMagazineAvailabilityTick,
		AmmoReserve:                  value.AmmoReserve,
		AmmoReserveStatus:            value.AmmoReserveStatus,
		AmmoReserveTick:              value.AmmoReserveAvailabilityTick,
		IsReloading:                  value.IsReloading,
		IsReloadingStatus:            value.IsReloadingStatus,
		IsReloadingTick:              value.IsReloadingAvailabilityTick,
		NonAmmunitionWeapon:          value.NonAmmunitionWeapon,
		AbstentionReasons:            append([]string(nil), value.AbstentionReasons...),
	}
}

func observedWeaponObservationScore(value models.ReplayWeaponStateObservation) int {
	score := 0
	for _, status := range []string{
		value.ActiveWeaponStatus,
		value.AmmoInMagazineStatus,
		value.AmmoReserveStatus,
		value.IsReloadingStatus,
	} {
		if status == models.ObservedWeaponFactObserved {
			score++
		}
	}
	return score
}

func cloneReplayWeaponObservation(value models.ReplayWeaponStateObservation) models.ReplayWeaponStateObservation {
	value.ActiveWeapon = cloneWeaponString(value.ActiveWeapon)
	value.ActiveWeaponAvailabilityTick = cloneWeaponInt(value.ActiveWeaponAvailabilityTick)
	value.AmmoInMagazine = cloneWeaponInt(value.AmmoInMagazine)
	value.AmmoInMagazineAvailabilityTick = cloneWeaponInt(value.AmmoInMagazineAvailabilityTick)
	value.AmmoReserve = cloneWeaponInt(value.AmmoReserve)
	value.AmmoReserveAvailabilityTick = cloneWeaponInt(value.AmmoReserveAvailabilityTick)
	value.IsReloading = cloneWeaponBool(value.IsReloading)
	value.IsReloadingAvailabilityTick = cloneWeaponInt(value.IsReloadingAvailabilityTick)
	value.AbstentionReasons = append([]string(nil), value.AbstentionReasons...)
	sort.Strings(value.AbstentionReasons)
	return value
}

func observedWeaponStateRow(
	matchID, roundID string,
	roundNumber, targetTick, sourceTick int,
	player observedWeaponPlayerSample,
	playerID string,
) models.ObservedWeaponStateRow {
	row := models.ObservedWeaponStateRow{
		SchemaID:          models.ObservedWeaponStateSchemaID,
		ContractVersion:   models.ObservedWeaponStateContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		CausalRole:        models.ObservedWeaponStateCausalRole,
		JoinKeys: models.ObservedWeaponStateJoinKeys{
			MatchID:       matchID,
			RoundID:       roundID,
			PlayerID:      playerID,
			ObservationID: fmt.Sprintf("%s:weapon-state:%d:%s", roundID, targetTick, playerID),
		},
		RoundNumber:       roundNumber,
		Tick:              targetTick,
		T0Tick:            targetTick,
		ActiveWeapon:      unavailableWeaponStringFact(models.ObservedWeaponReconciliationUnavailable),
		AmmoInMagazine:    unavailableWeaponIntFact(models.ObservedWeaponReconciliationUnavailable),
		AmmoReserve:       unavailableWeaponIntFact(models.ObservedWeaponReconciliationUnavailable),
		IsReloading:       unavailableWeaponBoolFact(models.ObservedWeaponReconciliationUnavailable),
		AbstentionReasons: append([]string(nil), player.observation.AbstentionReasons...),
	}

	observation := player.observation
	row.ActiveWeapon = observedWeaponStringFact(
		observation.ActiveWeapon,
		observation.ActiveWeaponStatus,
		observation.ActiveWeaponAvailabilityTick,
		sourceTick,
		targetTick,
		models.ObservedWeaponSourceActiveWeapon,
		&row.AbstentionReasons,
	)
	if row.ActiveWeapon.Status != models.ObservedWeaponFactObserved {
		row.AbstentionReasons = append(row.AbstentionReasons, weaponAbstentionActiveUnavailable)
		return row
	}
	if observation.NonAmmunitionWeapon || knownNonAmmunitionWeapon(*row.ActiveWeapon.Value) {
		row.AmmoInMagazine = unavailableWeaponIntFact(models.ObservedWeaponReconciliationNotApplicable)
		row.AmmoReserve = unavailableWeaponIntFact(models.ObservedWeaponReconciliationNotApplicable)
		row.IsReloading = unavailableWeaponBoolFact(models.ObservedWeaponReconciliationNotApplicable)
		row.AbstentionReasons = append(row.AbstentionReasons, weaponAbstentionNonAmmunition)
		return row
	}

	row.AmmoInMagazine = observedWeaponIntFact(
		observation.AmmoInMagazine,
		observation.AmmoInMagazineStatus,
		observation.AmmoInMagazineAvailabilityTick,
		sourceTick,
		targetTick,
		models.ObservedWeaponSourceAmmo,
		"magazine",
		&row.AbstentionReasons,
	)
	row.AmmoReserve = observedWeaponIntFact(
		observation.AmmoReserve,
		observation.AmmoReserveStatus,
		observation.AmmoReserveAvailabilityTick,
		sourceTick,
		targetTick,
		models.ObservedWeaponSourceAmmo,
		"reserve",
		&row.AbstentionReasons,
	)
	row.IsReloading = observedWeaponBoolFact(
		observation.IsReloading,
		observation.IsReloadingStatus,
		observation.IsReloadingAvailabilityTick,
		sourceTick,
		targetTick,
		models.ObservedWeaponSourceReload,
		&row.AbstentionReasons,
	)
	if row.AmmoInMagazine.Status == models.ObservedWeaponFactUnavailable {
		row.AbstentionReasons = append(row.AbstentionReasons, "ammo_in_magazine_snapshot_unavailable")
	}
	if row.AmmoReserve.Status == models.ObservedWeaponFactUnavailable {
		row.AbstentionReasons = append(row.AbstentionReasons, "ammo_reserve_snapshot_unavailable")
	}
	if row.IsReloading.Status == models.ObservedWeaponFactUnavailable {
		row.AbstentionReasons = append(row.AbstentionReasons, "reload_snapshot_unavailable")
	}
	return row
}

func observedWeaponPlayerAliases(rounds []models.ReplayRound, cutoffTick int) map[uint64]string {
	type firstAppearance struct {
		steamID uint64
		round   int
		tick    int
	}
	firstByPlayer := make(map[uint64]firstAppearance)
	for _, round := range rounds {
		for _, frame := range round.Frames {
			if frame.Tick < round.StartTick || frame.Tick > round.EndTick || frame.Tick > cutoffTick {
				continue
			}
			for _, player := range frame.Players {
				if player.SteamID == 0 {
					continue
				}
				candidate := firstAppearance{steamID: player.SteamID, round: round.Round, tick: frame.Tick}
				current, exists := firstByPlayer[player.SteamID]
				if !exists || candidate.round < current.round ||
					(candidate.round == current.round && candidate.tick < current.tick) {
					firstByPlayer[player.SteamID] = candidate
				}
			}
		}
	}
	appearances := make([]firstAppearance, 0, len(firstByPlayer))
	for _, appearance := range firstByPlayer {
		appearances = append(appearances, appearance)
	}
	sort.Slice(appearances, func(i, j int) bool {
		if appearances[i].round != appearances[j].round {
			return appearances[i].round < appearances[j].round
		}
		if appearances[i].tick != appearances[j].tick {
			return appearances[i].tick < appearances[j].tick
		}
		return appearances[i].steamID < appearances[j].steamID
	})
	aliases := make(map[uint64]string, len(appearances))
	for index, appearance := range appearances {
		aliases[appearance.steamID] = fmt.Sprintf("participant-%03d", index+1)
	}
	return aliases
}

func observedWeaponStringFact(
	value *string,
	status string,
	availability *int,
	sourceTick, t0Tick int,
	source string,
	reasons *[]string,
) models.ObservedWeaponStringFact {
	if status != models.ObservedWeaponFactObserved || value == nil || strings.TrimSpace(*value) == "" {
		return unavailableWeaponStringFact(models.ObservedWeaponReconciliationUnavailable)
	}
	if availability == nil || *availability != sourceTick || *availability > t0Tick {
		*reasons = append(*reasons, weaponAbstentionAvailability+":active_weapon")
		return unavailableWeaponStringFact(models.ObservedWeaponReconciliationUnavailable)
	}
	clean := strings.TrimSpace(*value)
	tick := *availability
	return models.ObservedWeaponStringFact{
		Status:           models.ObservedWeaponFactObserved,
		Value:            &clean,
		AvailabilityTick: &tick,
		Provenance: models.ObservedWeaponFactProvenance{
			Source: source, SourceTick: cloneWeaponInt(&tick),
			Reconciliation: models.ObservedWeaponReconciliationNotApplicable,
		},
	}
}

func observedWeaponIntFact(
	value *int,
	status string,
	availability *int,
	sourceTick, t0Tick int,
	source string,
	field string,
	reasons *[]string,
) models.ObservedWeaponIntFact {
	if status != models.ObservedWeaponFactObserved || value == nil {
		return unavailableWeaponIntFact(models.ObservedWeaponReconciliationUnavailable)
	}
	if *value < 0 {
		*reasons = append(*reasons, weaponAbstentionAmmoSentinel+":"+field)
		return unavailableWeaponIntFact(models.ObservedWeaponReconciliationUnavailable)
	}
	if availability == nil || *availability != sourceTick || *availability > t0Tick {
		*reasons = append(*reasons, weaponAbstentionAvailability+":"+field)
		return unavailableWeaponIntFact(models.ObservedWeaponReconciliationUnavailable)
	}
	copyValue, tick := *value, *availability
	return models.ObservedWeaponIntFact{
		Status: models.ObservedWeaponFactObserved,
		Value:  &copyValue, AvailabilityTick: &tick,
		Provenance: models.ObservedWeaponFactProvenance{
			Source: source, SourceTick: cloneWeaponInt(&tick),
			Reconciliation: models.ObservedWeaponReconciliationNotApplicable,
		},
	}
}

func observedWeaponBoolFact(
	value *bool,
	status string,
	availability *int,
	sourceTick, t0Tick int,
	source string,
	reasons *[]string,
) models.ObservedWeaponBoolFact {
	if status != models.ObservedWeaponFactObserved || value == nil {
		return unavailableWeaponBoolFact(models.ObservedWeaponReconciliationUnavailable)
	}
	if availability == nil || *availability != sourceTick || *availability > t0Tick {
		*reasons = append(*reasons, weaponAbstentionAvailability+":is_reloading")
		return unavailableWeaponBoolFact(models.ObservedWeaponReconciliationUnavailable)
	}
	copyValue, tick := *value, *availability
	return models.ObservedWeaponBoolFact{
		Status: models.ObservedWeaponFactObserved,
		Value:  &copyValue, AvailabilityTick: &tick,
		Provenance: models.ObservedWeaponFactProvenance{
			Source: source, SourceTick: cloneWeaponInt(&tick),
			Reconciliation: models.ObservedWeaponReconciliationNotApplicable,
		},
	}
}

func reconcileObservedWeaponEvents(
	row *models.ObservedWeaponStateRow,
	steamID uint64,
	sourceTick int,
	events observedWeaponEventIndex,
) {
	if row == nil || row.ActiveWeapon.Status != models.ObservedWeaponFactObserved || row.ActiveWeapon.Value == nil {
		return
	}
	match := events[observedWeaponEventKey{
		round:  row.RoundNumber,
		tick:   sourceTick,
		actor:  steamID,
		weapon: canonicalObservedWeaponName(*row.ActiveWeapon.Value),
	}]
	if len(match.fireEvents) == 0 && len(match.reloadEvents) == 0 {
		return
	}
	if len(match.fireEvents) > 0 && len(match.reloadEvents) > 0 {
		row.AmmoInMagazine = unavailableWeaponIntFact(models.ObservedWeaponReconciliationAmbiguous)
		row.AmmoReserve = unavailableWeaponIntFact(models.ObservedWeaponReconciliationAmbiguous)
		row.IsReloading = unavailableWeaponBoolFact(models.ObservedWeaponReconciliationAmbiguous)
		row.AbstentionReasons = append(row.AbstentionReasons, weaponAbstentionEventAmbiguity)
		return
	}

	if len(match.fireEvents) > 0 {
		row.ActiveWeapon.Provenance.Reconciliation = models.ObservedWeaponReconciliationFire
		reconcileObservedWeaponAmmo(&row.AmmoInMagazine, match.fireEvents, true, models.ObservedWeaponReconciliationFire, &row.AbstentionReasons)
		reconcileObservedWeaponAmmo(&row.AmmoReserve, match.fireEvents, false, models.ObservedWeaponReconciliationFire, &row.AbstentionReasons)
		return
	}

	row.ActiveWeapon.Provenance.Reconciliation = models.ObservedWeaponReconciliationReload
	reconcileObservedWeaponAmmo(&row.AmmoInMagazine, match.reloadEvents, true, models.ObservedWeaponReconciliationReload, &row.AbstentionReasons)
	reconcileObservedWeaponAmmo(&row.AmmoReserve, match.reloadEvents, false, models.ObservedWeaponReconciliationReload, &row.AbstentionReasons)
	if row.IsReloading.Status == models.ObservedWeaponFactObserved && row.IsReloading.Value != nil && *row.IsReloading.Value {
		row.IsReloading.Provenance.Reconciliation = models.ObservedWeaponReconciliationReload
	} else if row.IsReloading.Status == models.ObservedWeaponFactObserved {
		row.IsReloading = unavailableWeaponBoolFact(models.ObservedWeaponReconciliationAmbiguous)
		row.AbstentionReasons = append(row.AbstentionReasons, weaponAbstentionEventAmbiguity+":is_reloading")
	}
}

func indexObservedWeaponEvents(events []combat.Event) observedWeaponEventIndex {
	result := make(observedWeaponEventIndex)
	for _, event := range events {
		if event.Round <= 0 || event.Tick < 0 || event.Actor.Status != combat.AvailabilityObserved ||
			event.Actor.ID == 0 || event.Weapon.Status != combat.AvailabilityObserved {
			continue
		}
		weapon := canonicalObservedWeaponName(event.Weapon.Name)
		if weapon == "" {
			continue
		}
		key := observedWeaponEventKey{round: event.Round, tick: event.Tick, actor: event.Actor.ID, weapon: weapon}
		match := result[key]
		switch event.Type {
		case combat.EventWeaponFire:
			match.fireEvents = append(match.fireEvents, event)
		case combat.EventWeaponReload:
			match.reloadEvents = append(match.reloadEvents, event)
		default:
			continue
		}
		result[key] = match
	}
	for key, match := range result {
		sort.Slice(match.fireEvents, func(i, j int) bool {
			return observedWeaponEventLess(match.fireEvents[i], match.fireEvents[j])
		})
		sort.Slice(match.reloadEvents, func(i, j int) bool {
			return observedWeaponEventLess(match.reloadEvents[i], match.reloadEvents[j])
		})
		result[key] = match
	}
	return result
}

func observedWeaponEventLess(left, right combat.Event) bool {
	leftMagazine, rightMagazine := observedWeaponEventAmmoSortValue(left.Ammo.InMagazine), observedWeaponEventAmmoSortValue(right.Ammo.InMagazine)
	if leftMagazine != rightMagazine {
		return leftMagazine < rightMagazine
	}
	leftReserve, rightReserve := observedWeaponEventAmmoSortValue(left.Ammo.Reserve), observedWeaponEventAmmoSortValue(right.Ammo.Reserve)
	if leftReserve != rightReserve {
		return leftReserve < rightReserve
	}
	if left.SequenceInTick != right.SequenceInTick {
		return left.SequenceInTick < right.SequenceInTick
	}
	return left.LocalID < right.LocalID
}

func observedWeaponEventAmmoSortValue(value *int) int64 {
	if value == nil {
		return -1
	}
	return int64(*value)
}

func reconcileObservedWeaponAmmo(
	fact *models.ObservedWeaponIntFact,
	events []combat.Event,
	magazine bool,
	reconciliation string,
	reasons *[]string,
) {
	if fact == nil || fact.Status != models.ObservedWeaponFactObserved || fact.Value == nil {
		return
	}
	var eventValue *int
	for _, event := range events {
		if event.Ammo.Status != combat.AvailabilityObserved {
			continue
		}
		candidate := event.Ammo.Reserve
		if magazine {
			candidate = event.Ammo.InMagazine
		}
		if candidate == nil || *candidate < 0 {
			continue
		}
		if eventValue == nil {
			eventValue = cloneWeaponInt(candidate)
			continue
		}
		if *eventValue != *candidate {
			*fact = unavailableWeaponIntFact(models.ObservedWeaponReconciliationAmbiguous)
			*reasons = append(*reasons, weaponAbstentionEventAmbiguity+":ammo")
			return
		}
	}
	if eventValue == nil {
		return
	}
	if *eventValue != *fact.Value {
		*fact = unavailableWeaponIntFact(models.ObservedWeaponReconciliationAmbiguous)
		*reasons = append(*reasons, weaponAbstentionEventAmbiguity+":ammo")
		return
	}
	fact.Provenance.Reconciliation = reconciliation
}

func unavailableWeaponStringFact(reconciliation string) models.ObservedWeaponStringFact {
	return models.ObservedWeaponStringFact{
		Status: models.ObservedWeaponFactUnavailable,
		Provenance: models.ObservedWeaponFactProvenance{
			Source:         models.ObservedWeaponSourceUnavailable,
			Reconciliation: reconciliation,
		},
	}
}

func unavailableWeaponIntFact(reconciliation string) models.ObservedWeaponIntFact {
	return models.ObservedWeaponIntFact{
		Status: models.ObservedWeaponFactUnavailable,
		Provenance: models.ObservedWeaponFactProvenance{
			Source:         models.ObservedWeaponSourceUnavailable,
			Reconciliation: reconciliation,
		},
	}
}

func unavailableWeaponBoolFact(reconciliation string) models.ObservedWeaponBoolFact {
	return models.ObservedWeaponBoolFact{
		Status: models.ObservedWeaponFactUnavailable,
		Provenance: models.ObservedWeaponFactProvenance{
			Source:         models.ObservedWeaponSourceUnavailable,
			Reconciliation: reconciliation,
		},
	}
}

func canonicalizeWeaponAbstentions(row *models.ObservedWeaponStateRow) {
	if row == nil {
		return
	}
	set := make(map[string]struct{}, len(row.AbstentionReasons))
	for _, reason := range row.AbstentionReasons {
		if reason = strings.TrimSpace(reason); reason != "" {
			set[reason] = struct{}{}
		}
	}
	// Keep the required JSON shape stable even when there are no abstentions.
	// A nil slice would serialize as null and violate the candidate schema.
	row.AbstentionReasons = make([]string, 0, len(set))
	for reason := range set {
		row.AbstentionReasons = append(row.AbstentionReasons, reason)
	}
	sort.Strings(row.AbstentionReasons)
}

func sameObservedWeaponName(left, right string) bool {
	return canonicalObservedWeaponName(left) == canonicalObservedWeaponName(right) &&
		canonicalObservedWeaponName(left) != ""
}

func canonicalObservedWeaponName(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func knownNonAmmunitionWeapon(value string) bool {
	name := canonicalObservedWeaponName(value)
	for _, token := range []string{
		"knife", "bayonet", "karambit", "dagger", "c4", "bomb", "grenade", "flashbang", "flash",
		"smoke", "molotov", "incendiary", "decoy", "hegrenade", "zeus", "taser",
	} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func cloneWeaponInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneWeaponBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneWeaponString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var observedWeaponParticipantAliasPattern = regexp.MustCompile(`^participant-[0-9]{3,}$`)

// ValidateObservedWeaponStateLedger enforces the causal/nullability boundary
// independently from the builder so callers can reject tampered artifacts.
func ValidateObservedWeaponStateLedger(ledger models.ObservedWeaponStateLedger) error {
	if ledger.SchemaID != models.ObservedWeaponStateSchemaID {
		return fmt.Errorf("validate observed weapon state: schema_id %q", ledger.SchemaID)
	}
	if ledger.ContractVersion != models.ObservedWeaponStateContractVersion {
		return fmt.Errorf("validate observed weapon state: contract_version %q", ledger.ContractVersion)
	}
	if ledger.IdentitySemantics != models.TacticalIdentityJoinOnly {
		return fmt.Errorf("validate observed weapon state: identity semantics must be join_only")
	}
	if ledger.CausalRole != models.ObservedWeaponStateCausalRole {
		return fmt.Errorf("validate observed weapon state: causal role %q", ledger.CausalRole)
	}
	if ledger.Sampling.TargetHz != models.TacticalTargetHz ||
		ledger.Sampling.Strategy != models.TacticalSamplingStrategy {
		return fmt.Errorf("validate observed weapon state: invalid sampling contract")
	}
	if ledger.Rows == nil {
		return fmt.Errorf("validate observed weapon state: rows must be an array")
	}

	seen := make(map[string]struct{}, len(ledger.Rows))
	for index := range ledger.Rows {
		row := ledger.Rows[index]
		location := fmt.Sprintf("row[%d]", index)
		if err := validateObservedWeaponStateRow(row, location); err != nil {
			return err
		}
		if _, exists := seen[row.JoinKeys.ObservationID]; exists {
			return fmt.Errorf("validate observed weapon state: %s duplicate observation_id", location)
		}
		seen[row.JoinKeys.ObservationID] = struct{}{}
		if index > 0 && !observedWeaponRowLess(ledger.Rows[index-1], row) {
			return fmt.Errorf("validate observed weapon state: rows are not in deterministic order at %s", location)
		}
	}
	return nil
}

func validateObservedWeaponStateRow(row models.ObservedWeaponStateRow, location string) error {
	if row.SchemaID != models.ObservedWeaponStateSchemaID ||
		row.ContractVersion != models.ObservedWeaponStateContractVersion ||
		row.IdentitySemantics != models.TacticalIdentityJoinOnly ||
		row.CausalRole != models.ObservedWeaponStateCausalRole {
		return fmt.Errorf("validate observed weapon state: %s contract metadata mismatch", location)
	}
	if strings.TrimSpace(row.JoinKeys.MatchID) == "" || strings.TrimSpace(row.JoinKeys.RoundID) == "" ||
		!observedWeaponParticipantAliasPattern.MatchString(row.JoinKeys.PlayerID) ||
		strings.TrimSpace(row.JoinKeys.ObservationID) == "" {
		return fmt.Errorf("validate observed weapon state: %s invalid join keys", location)
	}
	if row.RoundNumber <= 0 || row.Tick < 0 || row.Tick != row.T0Tick {
		return fmt.Errorf("validate observed weapon state: %s invalid round/tick", location)
	}
	expectedObservationID := fmt.Sprintf(
		"%s:weapon-state:%d:%s",
		row.JoinKeys.RoundID,
		row.Tick,
		row.JoinKeys.PlayerID,
	)
	if row.JoinKeys.ObservationID != expectedObservationID {
		return fmt.Errorf("validate observed weapon state: %s observation_id is not deterministic", location)
	}

	if err := validateObservedWeaponStringFact(row.ActiveWeapon, row.T0Tick, location+".active_weapon"); err != nil {
		return err
	}
	if err := validateObservedWeaponIntFact(row.AmmoInMagazine, row.T0Tick, location+".ammo_in_magazine"); err != nil {
		return err
	}
	if err := validateObservedWeaponIntFact(row.AmmoReserve, row.T0Tick, location+".ammo_reserve"); err != nil {
		return err
	}
	if err := validateObservedWeaponBoolFact(row.IsReloading, row.T0Tick, location+".is_reloading"); err != nil {
		return err
	}
	if row.ActiveWeapon.Status != models.ObservedWeaponFactObserved &&
		(row.AmmoInMagazine.Status == models.ObservedWeaponFactObserved ||
			row.AmmoReserve.Status == models.ObservedWeaponFactObserved ||
			row.IsReloading.Status == models.ObservedWeaponFactObserved) {
		return fmt.Errorf("validate observed weapon state: %s dependent fact observed without active weapon", location)
	}
	if row.ActiveWeapon.Status == models.ObservedWeaponFactObserved && row.ActiveWeapon.Value != nil &&
		knownNonAmmunitionWeapon(*row.ActiveWeapon.Value) &&
		(row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable ||
			row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable ||
			row.IsReloading.Status != models.ObservedWeaponFactUnavailable) {
		return fmt.Errorf("validate observed weapon state: %s non-ammunition weapon exposes ammo/reload", location)
	}
	if !sort.StringsAreSorted(row.AbstentionReasons) {
		return fmt.Errorf("validate observed weapon state: %s abstentions are not sorted", location)
	}
	for index, reason := range row.AbstentionReasons {
		if strings.TrimSpace(reason) == "" || (index > 0 && reason == row.AbstentionReasons[index-1]) {
			return fmt.Errorf("validate observed weapon state: %s invalid abstention reasons", location)
		}
	}
	return nil
}

func validateObservedWeaponStringFact(
	fact models.ObservedWeaponStringFact,
	t0Tick int,
	location string,
) error {
	if fact.Status == models.ObservedWeaponFactObserved {
		if fact.Value == nil || strings.TrimSpace(*fact.Value) == "" {
			return fmt.Errorf("validate observed weapon state: %s observed value missing", location)
		}
		return validateObservedWeaponProvenance(
			fact.Status,
			fact.AvailabilityTick,
			fact.Provenance,
			t0Tick,
			models.ObservedWeaponSourceActiveWeapon,
			map[string]bool{
				models.ObservedWeaponReconciliationNotApplicable: true,
				models.ObservedWeaponReconciliationFire:          true,
				models.ObservedWeaponReconciliationReload:        true,
			},
			location,
		)
	}
	if fact.Value != nil {
		return fmt.Errorf("validate observed weapon state: %s unavailable value must be null", location)
	}
	return validateUnavailableWeaponProvenance(fact.Status, fact.AvailabilityTick, fact.Provenance, location)
}

func validateObservedWeaponIntFact(
	fact models.ObservedWeaponIntFact,
	t0Tick int,
	location string,
) error {
	if fact.Status == models.ObservedWeaponFactObserved {
		if fact.Value == nil || *fact.Value < 0 {
			return fmt.Errorf("validate observed weapon state: %s observed ammo must be non-negative", location)
		}
		return validateObservedWeaponProvenance(
			fact.Status,
			fact.AvailabilityTick,
			fact.Provenance,
			t0Tick,
			models.ObservedWeaponSourceAmmo,
			map[string]bool{
				models.ObservedWeaponReconciliationNotApplicable: true,
				models.ObservedWeaponReconciliationFire:          true,
				models.ObservedWeaponReconciliationReload:        true,
			},
			location,
		)
	}
	if fact.Value != nil {
		return fmt.Errorf("validate observed weapon state: %s unavailable value must be null", location)
	}
	return validateUnavailableWeaponProvenance(fact.Status, fact.AvailabilityTick, fact.Provenance, location)
}

func validateObservedWeaponBoolFact(
	fact models.ObservedWeaponBoolFact,
	t0Tick int,
	location string,
) error {
	if fact.Status == models.ObservedWeaponFactObserved {
		if fact.Value == nil {
			return fmt.Errorf("validate observed weapon state: %s observed reload must be boolean", location)
		}
		return validateObservedWeaponProvenance(
			fact.Status,
			fact.AvailabilityTick,
			fact.Provenance,
			t0Tick,
			models.ObservedWeaponSourceReload,
			map[string]bool{
				models.ObservedWeaponReconciliationNotApplicable: true,
				models.ObservedWeaponReconciliationReload:        true,
			},
			location,
		)
	}
	if fact.Value != nil {
		return fmt.Errorf("validate observed weapon state: %s unavailable value must be null", location)
	}
	return validateUnavailableWeaponProvenance(fact.Status, fact.AvailabilityTick, fact.Provenance, location)
}

func validateObservedWeaponProvenance(
	status string,
	availability *int,
	provenance models.ObservedWeaponFactProvenance,
	t0Tick int,
	expectedSource string,
	allowedReconciliation map[string]bool,
	location string,
) error {
	if status != models.ObservedWeaponFactObserved || availability == nil || provenance.SourceTick == nil ||
		*availability < 0 || *provenance.SourceTick < 0 ||
		*availability != *provenance.SourceTick || *availability > t0Tick ||
		provenance.Source != expectedSource || !allowedReconciliation[provenance.Reconciliation] {
		return fmt.Errorf("validate observed weapon state: %s invalid observed provenance", location)
	}
	return nil
}

func validateUnavailableWeaponProvenance(
	status string,
	availability *int,
	provenance models.ObservedWeaponFactProvenance,
	location string,
) error {
	allowed := provenance.Reconciliation == models.ObservedWeaponReconciliationUnavailable ||
		provenance.Reconciliation == models.ObservedWeaponReconciliationAmbiguous ||
		provenance.Reconciliation == models.ObservedWeaponReconciliationNotApplicable
	if status != models.ObservedWeaponFactUnavailable || availability != nil || provenance.SourceTick != nil ||
		provenance.Source != models.ObservedWeaponSourceUnavailable || !allowed {
		return fmt.Errorf("validate observed weapon state: %s invalid unavailable provenance", location)
	}
	return nil
}

func observedWeaponRowLess(left, right models.ObservedWeaponStateRow) bool {
	if left.RoundNumber != right.RoundNumber {
		return left.RoundNumber < right.RoundNumber
	}
	if left.Tick != right.Tick {
		return left.Tick < right.Tick
	}
	return left.JoinKeys.PlayerID < right.JoinKeys.PlayerID
}
