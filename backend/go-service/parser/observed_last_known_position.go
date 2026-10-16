package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cs2-demo-service/models"
)

const (
	lastKnownPositionArtifact = "replay_data_in_memory"

	lastKnownReasonPositionUnavailable = "last_known_position_unavailable"
	lastKnownReasonAgeUnavailable      = "information_age_unavailable"
	lastKnownReasonNoPriorContact      = "no_prior_confirmed_visual_contact"
	lastKnownReasonIdentityUnavailable = "participant_identity_unavailable"
	lastKnownReasonIdentityAmbiguous   = "participant_identity_ambiguous"
	lastKnownReasonPositionNonFinite   = "contact_subject_position_non_finite"
)

var (
	lastKnownParticipantAliasPattern = regexp.MustCompile(`^participant-[0-9]{3,}$`)
	lastKnownSourceAliasPattern      = regexp.MustCompile(`^demo-[0-9a-f]{20}$|^[a-z0-9][a-z0-9._-]*$`)
	lastKnownInputFields             = []string{
		"replay.subject_pawn_origin",
		"visual_contact.availability_tick",
		"visual_contact.provenance.source_frame_tick",
		"visual_contact.status",
		"visual_contact.value",
	}
)

type lastKnownFrameParticipant struct {
	alive        bool
	lifeSequence int
	position     *models.ObservedLastKnownPositionVector
	ambiguous    bool
}

type lastKnownPairState struct {
	position             models.ObservedLastKnownPositionVector
	observedTick         int
	availabilityTick     int
	observerLifeSequence int
	subjectLifeSequence  int
}

type lastKnownLifeTracker struct {
	seen         bool
	present      bool
	alive        bool
	lifeSequence int
}

// BuildObservedLastKnownPositionLedger projects the last legally visible pawn
// origin for every directed visual-contact evaluation. aliases must map the
// visual ledger's internal join IDs to local opaque participant aliases.
func BuildObservedLastKnownPositionLedger(
	replay *models.ReplayData,
	visualLedger models.ObservedVisualContactLedger,
	aliases map[string]string,
) (models.ObservedLastKnownPositionLedger, error) {
	if replay == nil {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: replay data is nil")
	}
	tickRate := replay.Metadata.TickRate
	if math.IsNaN(tickRate) || math.IsInf(tickRate, 0) || tickRate <= 0 {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: tick rate is invalid")
	}
	if len(replay.Rounds) != 1 {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: exactly one round is required")
	}
	if visualLedger.SchemaID != models.ObservedVisualContactSchemaID ||
		visualLedger.ContractVersion != models.ObservedVisualContactContractVersion {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: visual contact contract mismatch")
	}
	if err := ValidateObservedVisualContactLedger(visualLedger); err != nil {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: invalid visual contact ledger: %w", err)
	}

	frameIndex, err := buildLastKnownFrameIndex(replay.Rounds[0])
	if err != nil {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: %w", err)
	}
	evaluations := append([]models.ObservedVisualContactEvaluation(nil), visualLedger.Evaluations...)
	sort.Slice(evaluations, func(i, j int) bool {
		left, right := evaluations[i], evaluations[j]
		if left.RoundNumber != right.RoundNumber {
			return left.RoundNumber < right.RoundNumber
		}
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		leftObserver, rightObserver := aliases[left.JoinKeys.ObserverID], aliases[right.JoinKeys.ObserverID]
		if leftObserver != rightObserver {
			return leftObserver < rightObserver
		}
		return aliases[left.JoinKeys.SubjectID] < aliases[right.JoinKeys.SubjectID]
	})

	ledger := models.ObservedLastKnownPositionLedger{
		SchemaID:          models.ObservedLastKnownPositionSchemaID,
		ContractVersion:   models.ObservedLastKnownPositionContractVersion,
		IdentitySemantics: models.ObservedLastKnownPositionIdentity,
		CausalRole:        models.ObservedLastKnownPositionCausalRole,
		TickRateHz:        tickRate,
		Rows:              make([]models.ObservedLastKnownPositionRow, 0, len(evaluations)),
	}
	states := make(map[string]lastKnownPairState)
	seenSamples := make(map[string]struct{}, len(evaluations))
	for index, evaluation := range evaluations {
		row, nextState, keepState, err := deriveObservedLastKnownPositionRow(
			evaluation,
			replay.Rounds[0].Round,
			tickRate,
			aliases,
			frameIndex,
			states,
			index+1,
		)
		if err != nil {
			return models.ObservedLastKnownPositionLedger{}, err
		}
		sampleKey := fmt.Sprintf(
			"%d:%d:%s:%s",
			row.RoundNumber,
			row.T0Tick,
			row.JoinKeys.ObserverAlias,
			row.JoinKeys.SubjectAlias,
		)
		if _, exists := seenSamples[sampleKey]; exists {
			return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: duplicate directed pair sample %s", sampleKey)
		}
		seenSamples[sampleKey] = struct{}{}
		pairKey := row.JoinKeys.ObserverAlias + "\x00" + row.JoinKeys.SubjectAlias
		if keepState {
			states[pairKey] = nextState
		} else {
			delete(states, pairKey)
		}
		ledger.Rows = append(ledger.Rows, row)
	}
	if err := ValidateObservedLastKnownPositionLedger(ledger); err != nil {
		return models.ObservedLastKnownPositionLedger{}, fmt.Errorf("build last-known position: %w", err)
	}
	return ledger, nil
}

func deriveObservedLastKnownPositionRow(
	evaluation models.ObservedVisualContactEvaluation,
	roundNumber int,
	tickRate float64,
	aliases map[string]string,
	frameIndex map[int]map[string]lastKnownFrameParticipant,
	states map[string]lastKnownPairState,
	rowSequence int,
) (models.ObservedLastKnownPositionRow, lastKnownPairState, bool, error) {
	if evaluation.RoundNumber != roundNumber || evaluation.Provenance.SourceRound != roundNumber {
		return models.ObservedLastKnownPositionRow{}, lastKnownPairState{}, false, fmt.Errorf("build last-known position: round join mismatch")
	}
	if evaluation.Tick < 0 || evaluation.Tick > evaluation.T0Tick || evaluation.AvailabilityTick == nil ||
		*evaluation.AvailabilityTick < 0 || evaluation.Provenance.SourceFrameTick < 0 ||
		*evaluation.AvailabilityTick != evaluation.Provenance.SourceFrameTick ||
		*evaluation.AvailabilityTick > evaluation.Tick {
		return models.ObservedLastKnownPositionRow{}, lastKnownPairState{}, false, fmt.Errorf("build last-known position: causal tick violation at evaluation tick %d", evaluation.Tick)
	}
	observerAlias, observerOK := aliases[evaluation.JoinKeys.ObserverID]
	subjectAlias, subjectOK := aliases[evaluation.JoinKeys.SubjectID]
	if !observerOK || !subjectOK || !lastKnownParticipantAliasPattern.MatchString(observerAlias) ||
		!lastKnownParticipantAliasPattern.MatchString(subjectAlias) || observerAlias == subjectAlias {
		return models.ObservedLastKnownPositionRow{}, lastKnownPairState{}, false, fmt.Errorf("build last-known position: opaque participant alias is unavailable")
	}
	sourceAlias := strings.TrimSpace(evaluation.JoinKeys.MatchID)
	if !lastKnownSourceAliasPattern.MatchString(sourceAlias) {
		return models.ObservedLastKnownPositionRow{}, lastKnownPairState{}, false, fmt.Errorf("build last-known position: source alias is invalid")
	}

	row := models.ObservedLastKnownPositionRow{
		RowID:       fmt.Sprintf("last-known-%09d", rowSequence),
		RoundNumber: evaluation.RoundNumber,
		T0Tick:      evaluation.Tick,
		JoinKeys: models.ObservedLastKnownPositionJoinKeys{
			SourceAlias:   sourceAlias,
			RoundID:       fmt.Sprintf("%s:round:%03d", sourceAlias, evaluation.RoundNumber),
			ObserverAlias: observerAlias,
			SubjectAlias:  subjectAlias,
		},
		ContactObservation: models.ObservedLastKnownPositionContactObservation{
			Status:           models.ObservedLastKnownPositionStatusUnavailable,
			AvailabilityTick: copyLastKnownInt(evaluation.AvailabilityTick),
			SourceFrameTick:  lastKnownIntPointer(evaluation.Provenance.SourceFrameTick),
			Source:           models.ObservedLastKnownPositionContactSource,
		},
		LastKnownPosition: unavailableObservedLastKnownPositionFact(),
		InformationAge:    unavailableObservedInformationAgeFact(),
		Derivation:        models.ObservedLastKnownPositionDerivationUnavailable,
		AbstentionReasons: []string{},
		Provenance: models.ObservedLastKnownPositionProvenance{
			VisualContactSchemaID:        models.ObservedVisualContactSchemaID,
			VisualContactContractVersion: models.ObservedVisualContactContractVersion,
			PositionSourceArtifact:       lastKnownPositionArtifact,
			InputFields:                  append([]string(nil), lastKnownInputFields...),
		},
	}

	framePlayers, frameOK := frameIndex[evaluation.Provenance.SourceFrameTick]
	observerFrame, observerOK := framePlayers[evaluation.JoinKeys.ObserverID]
	subjectFrame, subjectOK := framePlayers[evaluation.JoinKeys.SubjectID]
	identityExact := frameOK && observerOK && subjectOK && observerFrame.alive && subjectFrame.alive &&
		!observerFrame.ambiguous && !subjectFrame.ambiguous && observerFrame.lifeSequence > 0 && subjectFrame.lifeSequence > 0
	if identityExact {
		row.ContactObservation.ObserverLifeSequence = lastKnownIntPointer(observerFrame.lifeSequence)
		row.ContactObservation.SubjectLifeSequence = lastKnownIntPointer(subjectFrame.lifeSequence)
	}

	pairKey := observerAlias + "\x00" + subjectAlias
	state, hadState := states[pairKey]
	if !identityExact {
		row.AbstentionReasons = append(row.AbstentionReasons, lastKnownReasonIdentityUnavailable)
		if (observerOK && observerFrame.ambiguous) || (subjectOK && subjectFrame.ambiguous) {
			row.AbstentionReasons = append(row.AbstentionReasons, lastKnownReasonIdentityAmbiguous)
		}
		hadState = false
	} else if hadState && (state.observerLifeSequence != observerFrame.lifeSequence ||
		state.subjectLifeSequence != subjectFrame.lifeSequence) {
		hadState = false
	}

	contactObserved := evaluation.Status == models.TacticalStatusObserved &&
		evaluation.VisualContact.Status == models.VisualContactFactDerived &&
		evaluation.VisualContact.Value != nil
	if contactObserved {
		row.ContactObservation.Status = models.ObservedLastKnownPositionStatusObserved
		row.ContactObservation.VisualContact = copyLastKnownBool(evaluation.VisualContact.Value)
	}
	if contactObserved && *evaluation.VisualContact.Value && identityExact {
		if subjectFrame.position == nil || !finiteLastKnownVector(*subjectFrame.position) {
			row.AbstentionReasons = append(row.AbstentionReasons, lastKnownReasonPositionNonFinite)
			hadState = false
		} else {
			position := *subjectFrame.position
			row.ContactObservation.SubjectPositionAtContact = copyLastKnownVector(position)
			state = lastKnownPairState{
				position:             position,
				observedTick:         evaluation.Provenance.SourceFrameTick,
				availabilityTick:     *evaluation.AvailabilityTick,
				observerLifeSequence: observerFrame.lifeSequence,
				subjectLifeSequence:  subjectFrame.lifeSequence,
			}
			hadState = true
			row.Derivation = models.ObservedLastKnownPositionDerivationUpdate
		}
	}

	if hadState {
		ageMS := (float64(evaluation.Tick-state.observedTick) / tickRate) * 1000
		if state.observedTick > evaluation.Tick || math.IsNaN(ageMS) || math.IsInf(ageMS, 0) || ageMS < 0 {
			return models.ObservedLastKnownPositionRow{}, lastKnownPairState{}, false, fmt.Errorf("build last-known position: invalid information age at tick %d", evaluation.Tick)
		}
		if row.Derivation != models.ObservedLastKnownPositionDerivationUpdate {
			row.Derivation = models.ObservedLastKnownPositionDerivationCarry
		}
		row.LastKnownPosition = models.ObservedLastKnownPositionFact{
			Status:           models.ObservedLastKnownPositionStatusObserved,
			Value:            copyLastKnownVector(state.position),
			ObservedTick:     lastKnownIntPointer(state.observedTick),
			AvailabilityTick: lastKnownIntPointer(state.availabilityTick),
			CoordinateFrame:  models.ObservedLastKnownPositionCoordinateFrame,
			SpatialSemantics: models.ObservedLastKnownPositionSpatialSemantics,
			Source:           models.ObservedLastKnownPositionSourceContact,
		}
		row.InformationAge = models.ObservedInformationAgeFact{
			Status:           models.ObservedLastKnownPositionStatusObserved,
			ValueMS:          lastKnownFloatPointer(ageMS),
			AvailabilityTick: lastKnownIntPointer(evaluation.Tick),
			Units:            models.ObservedInformationAgeUnits,
			Source:           models.ObservedLastKnownPositionSourceAge,
		}
		row.AbstentionReasons = []string{}
		return row, state, true, nil
	}

	row.AbstentionReasons = append(
		row.AbstentionReasons,
		lastKnownReasonPositionUnavailable,
		lastKnownReasonAgeUnavailable,
		lastKnownReasonNoPriorContact,
	)
	sort.Strings(row.AbstentionReasons)
	row.AbstentionReasons = uniqueLastKnownStrings(row.AbstentionReasons)
	return row, lastKnownPairState{}, false, nil
}

func buildLastKnownFrameIndex(round models.ReplayRound) (map[int]map[string]lastKnownFrameParticipant, error) {
	frames, err := canonicalVisualContactFrames(round, round.EndTick)
	if err != nil {
		return nil, err
	}
	rawByTick := make(map[int][]models.ReplayFrame)
	for _, frame := range round.Frames {
		if frame.Tick >= round.StartTick && frame.Tick <= round.EndTick {
			rawByTick[frame.Tick] = append(rawByTick[frame.Tick], frame)
		}
	}
	trackers := make(map[string]lastKnownLifeTracker)
	result := make(map[int]map[string]lastKnownFrameParticipant, len(frames))
	for _, frame := range frames {
		present := make(map[string]visualContactPlayerSample, len(frame.players))
		for _, player := range frame.players {
			present[tacticalPlayerID(player.steamID)] = player
		}
		for id, tracker := range trackers {
			if _, exists := present[id]; !exists {
				tracker.present = false
				tracker.alive = false
				trackers[id] = tracker
			}
		}
		frameParticipants := make(map[string]lastKnownFrameParticipant, len(present))
		for id, player := range present {
			tracker := trackers[id]
			if !tracker.seen {
				tracker.seen = true
				if player.alive {
					tracker.lifeSequence = 1
				}
			} else if player.alive && (!tracker.present || !tracker.alive) {
				tracker.lifeSequence++
			}
			tracker.present = true
			tracker.alive = player.alive
			trackers[id] = tracker
			position, ambiguous := resolveLastKnownPawnOrigin(rawByTick[frame.tick], frame.key, player.steamID)
			frameParticipants[id] = lastKnownFrameParticipant{
				alive:        player.alive,
				lifeSequence: tracker.lifeSequence,
				position:     position,
				ambiguous:    ambiguous,
			}
		}
		result[frame.tick] = frameParticipants
	}
	return result, nil
}

func resolveLastKnownPawnOrigin(
	frames []models.ReplayFrame,
	selectedKey string,
	steamID uint64,
) (*models.ObservedLastKnownPositionVector, bool) {
	matches := make([]models.ReplayPlayerState, 0, 1)
	for _, frame := range frames {
		candidate, err := lastKnownVisualFrameCandidate(frame)
		if err != nil || candidate.key != selectedKey {
			continue
		}
		for _, player := range frame.Players {
			if player.SteamID == steamID {
				matches = append(matches, player)
			}
		}
	}
	if len(matches) != 1 {
		return nil, true
	}
	position := models.ObservedLastKnownPositionVector{
		X: float64(matches[0].X),
		Y: float64(matches[0].Y),
		Z: matches[0].Z,
	}
	if !finiteLastKnownVector(position) {
		return nil, false
	}
	return &position, false
}

func lastKnownVisualFrameCandidate(frame models.ReplayFrame) (visualContactFrameSample, error) {
	players, err := canonicalVisualContactPlayers(frame.Players)
	if err != nil {
		return visualContactFrameSample{}, err
	}
	effects := canonicalVisualContactEffects(frame.ActiveEffects)
	playerKeys := make([]string, 0, len(players))
	for _, player := range players {
		playerKeys = append(playerKeys, player.key)
	}
	effectKeys := make([]string, 0, len(effects))
	for _, effect := range effects {
		effectKeys = append(effectKeys, effect.key)
	}
	keyBytes, err := json.Marshal(struct {
		Players        []string `json:"players"`
		ActiveEffects  []string `json:"active_effects"`
		SmokeAvailable bool     `json:"smoke_available"`
	}{playerKeys, effectKeys, frame.ActiveEffects != nil})
	if err != nil {
		return visualContactFrameSample{}, err
	}
	return visualContactFrameSample{
		tick:           frame.Tick,
		players:        players,
		activeEffects:  effects,
		smokeAvailable: frame.ActiveEffects != nil,
		key:            string(keyBytes),
	}, nil
}

func ValidateObservedLastKnownPositionLedger(ledger models.ObservedLastKnownPositionLedger) error {
	if ledger.SchemaID != models.ObservedLastKnownPositionSchemaID ||
		ledger.ContractVersion != models.ObservedLastKnownPositionContractVersion ||
		ledger.IdentitySemantics != models.ObservedLastKnownPositionIdentity ||
		ledger.CausalRole != models.ObservedLastKnownPositionCausalRole ||
		math.IsNaN(ledger.TickRateHz) || math.IsInf(ledger.TickRateHz, 0) || ledger.TickRateHz <= 0 {
		return fmt.Errorf("validate last-known position: contract metadata mismatch")
	}
	previousKey := ""
	seen := make(map[string]struct{}, len(ledger.Rows))
	for index, row := range ledger.Rows {
		if err := validateObservedLastKnownPositionRow(row, ledger.TickRateHz, index+1); err != nil {
			return fmt.Errorf("validate last-known position: row %d: %w", index, err)
		}
		key := fmt.Sprintf("%09d:%012d:%s:%s", row.RoundNumber, row.T0Tick, row.JoinKeys.ObserverAlias, row.JoinKeys.SubjectAlias)
		if previousKey != "" && key < previousKey {
			return fmt.Errorf("rows are not deterministically ordered")
		}
		previousKey = key
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate pair sample")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateObservedLastKnownPositionRow(
	row models.ObservedLastKnownPositionRow,
	tickRate float64,
	sequence int,
) error {
	if row.RowID != fmt.Sprintf("last-known-%09d", sequence) || row.RoundNumber <= 0 || row.T0Tick < 0 {
		return fmt.Errorf("invalid row identity or tick")
	}
	if !lastKnownSourceAliasPattern.MatchString(row.JoinKeys.SourceAlias) ||
		!lastKnownParticipantAliasPattern.MatchString(row.JoinKeys.ObserverAlias) ||
		!lastKnownParticipantAliasPattern.MatchString(row.JoinKeys.SubjectAlias) ||
		row.JoinKeys.ObserverAlias == row.JoinKeys.SubjectAlias ||
		row.JoinKeys.RoundID != fmt.Sprintf("%s:round:%03d", row.JoinKeys.SourceAlias, row.RoundNumber) {
		return fmt.Errorf("invalid opaque joins")
	}
	contact := row.ContactObservation
	if contact.Source != models.ObservedLastKnownPositionContactSource || contact.AvailabilityTick == nil ||
		contact.SourceFrameTick == nil || *contact.AvailabilityTick != *contact.SourceFrameTick ||
		*contact.SourceFrameTick < 0 || *contact.SourceFrameTick > row.T0Tick {
		return fmt.Errorf("invalid visual-contact source ticks")
	}
	if contact.Status == models.ObservedLastKnownPositionStatusObserved {
		if contact.VisualContact == nil || (contact.ObserverLifeSequence == nil) != (contact.SubjectLifeSequence == nil) {
			return fmt.Errorf("observed contact has invalid causal identity")
		}
		if contact.ObserverLifeSequence != nil &&
			(*contact.ObserverLifeSequence <= 0 || *contact.SubjectLifeSequence <= 0) {
			return fmt.Errorf("observed contact has invalid life sequence")
		}
		if !*contact.VisualContact && contact.SubjectPositionAtContact != nil {
			return fmt.Errorf("negative visual contact retained current subject position")
		}
	} else if contact.Status != models.ObservedLastKnownPositionStatusUnavailable ||
		contact.VisualContact != nil || contact.SubjectPositionAtContact != nil {
		return fmt.Errorf("invalid unavailable contact observation")
	}
	if contact.SubjectPositionAtContact != nil && !finiteLastKnownVector(*contact.SubjectPositionAtContact) {
		return fmt.Errorf("non-finite contact position")
	}

	position := row.LastKnownPosition
	age := row.InformationAge
	if position.CoordinateFrame != models.ObservedLastKnownPositionCoordinateFrame ||
		position.SpatialSemantics != models.ObservedLastKnownPositionSpatialSemantics ||
		age.Units != models.ObservedInformationAgeUnits {
		return fmt.Errorf("coordinate or age units mismatch")
	}
	if position.Status == models.ObservedLastKnownPositionStatusUnavailable {
		if position.Value != nil || position.ObservedTick != nil || position.AvailabilityTick != nil ||
			position.Source != models.ObservedLastKnownPositionSourceUnavailable ||
			age.Status != models.ObservedLastKnownPositionStatusUnavailable || age.ValueMS != nil ||
			age.AvailabilityTick != nil || age.Source != models.ObservedLastKnownPositionSourceUnavailable ||
			row.Derivation != models.ObservedLastKnownPositionDerivationUnavailable || len(row.AbstentionReasons) == 0 {
			return fmt.Errorf("unavailable facts carry values or lack abstention")
		}
		if !sort.StringsAreSorted(row.AbstentionReasons) || !containsLastKnownString(row.AbstentionReasons, lastKnownReasonPositionUnavailable) ||
			!containsLastKnownString(row.AbstentionReasons, lastKnownReasonAgeUnavailable) {
			return fmt.Errorf("invalid unavailable reasons")
		}
		return validateLastKnownProvenance(row.Provenance)
	}
	if position.Status != models.ObservedLastKnownPositionStatusObserved || position.Value == nil ||
		position.ObservedTick == nil || position.AvailabilityTick == nil ||
		position.Source != models.ObservedLastKnownPositionSourceContact ||
		!finiteLastKnownVector(*position.Value) || *position.ObservedTick < 0 ||
		*position.AvailabilityTick < *position.ObservedTick || *position.AvailabilityTick > row.T0Tick {
		return fmt.Errorf("invalid observed last-known position")
	}
	if age.Status != models.ObservedLastKnownPositionStatusObserved || age.ValueMS == nil ||
		age.AvailabilityTick == nil || *age.AvailabilityTick != row.T0Tick ||
		age.Source != models.ObservedLastKnownPositionSourceAge || math.IsNaN(*age.ValueMS) ||
		math.IsInf(*age.ValueMS, 0) || *age.ValueMS < 0 || len(row.AbstentionReasons) != 0 {
		return fmt.Errorf("invalid observed information age")
	}
	expectedAge := float64(row.T0Tick-*position.ObservedTick) / tickRate * 1000
	if math.Abs(expectedAge-*age.ValueMS) > 1e-9 {
		return fmt.Errorf("information age mismatch")
	}
	if row.Derivation == models.ObservedLastKnownPositionDerivationUpdate {
		if contact.Status != models.ObservedLastKnownPositionStatusObserved || contact.VisualContact == nil ||
			!*contact.VisualContact || contact.SubjectPositionAtContact == nil ||
			contact.ObserverLifeSequence == nil || contact.SubjectLifeSequence == nil ||
			*position.ObservedTick != *contact.SourceFrameTick || *position.Value != *contact.SubjectPositionAtContact {
			return fmt.Errorf("contact update mismatch")
		}
	} else if row.Derivation != models.ObservedLastKnownPositionDerivationCarry || contact.SubjectPositionAtContact != nil {
		return fmt.Errorf("invalid carry derivation")
	}
	return validateLastKnownProvenance(row.Provenance)
}

func validateLastKnownProvenance(value models.ObservedLastKnownPositionProvenance) error {
	if value.VisualContactSchemaID != models.ObservedVisualContactSchemaID ||
		value.VisualContactContractVersion != models.ObservedVisualContactContractVersion ||
		value.PositionSourceArtifact != lastKnownPositionArtifact ||
		strings.Join(value.InputFields, "\n") != strings.Join(lastKnownInputFields, "\n") {
		return fmt.Errorf("provenance mismatch")
	}
	return nil
}

func unavailableObservedLastKnownPositionFact() models.ObservedLastKnownPositionFact {
	return models.ObservedLastKnownPositionFact{
		Status:           models.ObservedLastKnownPositionStatusUnavailable,
		CoordinateFrame:  models.ObservedLastKnownPositionCoordinateFrame,
		SpatialSemantics: models.ObservedLastKnownPositionSpatialSemantics,
		Source:           models.ObservedLastKnownPositionSourceUnavailable,
	}
}

func unavailableObservedInformationAgeFact() models.ObservedInformationAgeFact {
	return models.ObservedInformationAgeFact{
		Status: models.ObservedLastKnownPositionStatusUnavailable,
		Units:  models.ObservedInformationAgeUnits,
		Source: models.ObservedLastKnownPositionSourceUnavailable,
	}
}

func finiteLastKnownVector(value models.ObservedLastKnownPositionVector) bool {
	return !math.IsNaN(value.X) && !math.IsInf(value.X, 0) &&
		!math.IsNaN(value.Y) && !math.IsInf(value.Y, 0) &&
		!math.IsNaN(value.Z) && !math.IsInf(value.Z, 0)
}

func copyLastKnownVector(value models.ObservedLastKnownPositionVector) *models.ObservedLastKnownPositionVector {
	copy := value
	return &copy
}

func copyLastKnownInt(value *int) *int {
	if value == nil {
		return nil
	}
	return lastKnownIntPointer(*value)
}

func copyLastKnownBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func lastKnownIntPointer(value int) *int { return &value }

func lastKnownFloatPointer(value float64) *float64 { return &value }

func uniqueLastKnownStrings(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func containsLastKnownString(values []string, candidate string) bool {
	index := sort.SearchStrings(values, candidate)
	return index < len(values) && values[index] == candidate
}

func parseLastKnownTacticalPlayerID(value string) (uint64, bool) {
	if !strings.HasPrefix(value, "steam:") {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(value, "steam:"), 10, 64)
	return id, err == nil && id != 0
}
