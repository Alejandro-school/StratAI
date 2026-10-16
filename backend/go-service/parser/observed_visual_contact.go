package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/visualcontact"

	"github.com/golang/geo/r3"
)

const (
	visualContactSourceArtifact          = "replay_data_in_memory"
	visualContactGeometryLoaded          = "loaded"
	visualContactGeometryMissing         = "not_loaded"
	visualContactGeometryPositionMissing = "position_unavailable"
	visualContactUnscopedFOV             = 90.0
)

var visualContactInputFields = []string{
	"active_smoke_effects",
	"map_geometry",
	"observer_eye_position",
	"observer_scope_and_zoom",
	"observer_view_angles",
	"subject_eye_position",
}

type visualContactPlayerSample struct {
	steamID            uint64
	team               string
	alive              bool
	eyes               *r3.Vector
	eyePositionSource  string
	yaw                *float64
	pitch              *float64
	weapon             string
	scoped             bool
	zoomLevel          int
	zoomLevelAvailable bool
	key                string
}

type visualContactFrameSample struct {
	tick           int
	players        []visualContactPlayerSample
	activeEffects  []visualContactSmokeSample
	smokeAvailable bool
	key            string
}

type visualContactSmokeSample struct {
	x         float64
	y         float64
	radius    float64
	startTick int
	key       string
}

type visualContactPairKey struct {
	roundNumber int
	observerID  string
	subjectID   string
}

// BuildObservedVisualContactLedger derives only information available at or
// before t0Tick. The returned @1 contract is intentionally separate from the
// canonical 3.8.0 bundle so old bundles cannot silently gain new semantics.
func BuildObservedVisualContactLedger(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	t0Tick int,
) (models.ObservedVisualContactLedger, error) {
	if replay == nil {
		return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: replay data is nil")
	}
	matchID := strings.TrimSpace(replay.Metadata.MatchID)
	if matchID == "" {
		return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: replay metadata match_id is empty")
	}
	if t0Tick < 0 {
		return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: t0_tick must be non-negative")
	}
	tickRate := replay.Metadata.TickRate
	if math.IsNaN(tickRate) || math.IsInf(tickRate, 0) || tickRate < models.TacticalTargetHz {
		return models.ObservedVisualContactLedger{}, fmt.Errorf(
			"build observed visual contact: tick_rate must be finite and at least %d, got %v",
			models.TacticalTargetHz,
			tickRate,
		)
	}

	periodTicks := tickRate / float64(models.TacticalTargetHz)
	ledger := models.ObservedVisualContactLedger{
		SchemaID:          models.ObservedVisualContactSchemaID,
		ContractVersion:   models.ObservedVisualContactContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		CausalRole:        models.VisualContactCausalRole,
		JoinKeys:          models.TacticalExportJoinKeys{MatchID: matchID},
		T0Tick:            t0Tick,
		Sampling: models.TacticalSampling{
			TargetHz:    models.TacticalTargetHz,
			TickRate:    tickRate,
			PeriodTicks: periodTicks,
			Strategy:    models.TacticalSamplingStrategy,
		},
		Evaluations: make([]models.ObservedVisualContactEvaluation, 0),
		Contacts:    make([]models.ObservedVisualContactEpisode, 0),
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

	activeContacts := make(map[visualContactPairKey]int)
	seenRounds := make(map[int]struct{}, len(rounds))
	for _, round := range rounds {
		if round.StartTick > t0Tick {
			continue
		}
		if round.Round <= 0 {
			return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: round number must be positive, got %d", round.Round)
		}
		if round.EndTick < round.StartTick {
			return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: round %d ends before it starts", round.Round)
		}
		if _, exists := seenRounds[round.Round]; exists {
			return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: duplicate round number %d", round.Round)
		}
		seenRounds[round.Round] = struct{}{}
		endTick := round.EndTick
		if endTick > t0Tick {
			endTick = t0Tick
		}
		frames, err := canonicalVisualContactFrames(round, endTick)
		if err != nil {
			return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: round %d: %w", round.Round, err)
		}
		targetTicks := tacticalTargetTicks(round.StartTick, endTick, periodTicks)
		roundID := fmt.Sprintf("%s:round:%03d", matchID, round.Round)
		frameIndex := 0
		previousTarget := round.StartTick
		for targetIndex, targetTick := range targetTicks {
			var selected *visualContactFrameSample
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

			for _, observer := range selected.players {
				if observer.team == "" || !observer.alive {
					continue
				}
				for _, subject := range selected.players {
					if subject.team == "" || !subject.alive || observer.team == subject.team {
						continue
					}
					evaluation, err := evaluateObservedVisualContact(
						replay,
						geometry,
						matchID,
						roundID,
						round.Round,
						targetTick,
						t0Tick,
						*selected,
						observer,
						subject,
					)
					if err != nil {
						return models.ObservedVisualContactLedger{}, err
					}
					applyVisualContactBoundary(&ledger, &evaluation, activeContacts)
					ledger.Evaluations = append(ledger.Evaluations, evaluation)
				}
			}
			previousTarget = targetTick
		}
	}

	sort.Slice(ledger.Contacts, func(i, j int) bool {
		left, right := ledger.Contacts[i], ledger.Contacts[j]
		if left.RoundNumber != right.RoundNumber {
			return left.RoundNumber < right.RoundNumber
		}
		if left.StartTick != right.StartTick {
			return left.StartTick < right.StartTick
		}
		if left.JoinKeys.ObserverID != right.JoinKeys.ObserverID {
			return left.JoinKeys.ObserverID < right.JoinKeys.ObserverID
		}
		return left.JoinKeys.SubjectID < right.JoinKeys.SubjectID
	})
	if err := ValidateObservedVisualContactLedger(ledger); err != nil {
		return models.ObservedVisualContactLedger{}, fmt.Errorf("build observed visual contact: %w", err)
	}
	return ledger, nil
}

func canonicalVisualContactFrames(round models.ReplayRound, maxTick int) ([]visualContactFrameSample, error) {
	byTick := make(map[int]visualContactFrameSample, len(round.Frames))
	for _, frame := range round.Frames {
		if frame.Tick < round.StartTick || frame.Tick > round.EndTick || frame.Tick > maxTick {
			continue
		}
		players, err := canonicalVisualContactPlayers(frame.Players)
		if err != nil {
			return nil, fmt.Errorf("frame tick %d: %w", frame.Tick, err)
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
			return nil, fmt.Errorf("frame tick %d: canonicalize contact inputs: %w", frame.Tick, err)
		}
		candidate := visualContactFrameSample{
			tick:           frame.Tick,
			players:        players,
			activeEffects:  effects,
			smokeAvailable: frame.ActiveEffects != nil,
			key:            string(keyBytes),
		}
		current, exists := byTick[frame.Tick]
		if !exists || len(candidate.players) > len(current.players) ||
			(len(candidate.players) == len(current.players) && candidate.key < current.key) {
			byTick[frame.Tick] = candidate
		}
	}

	frames := make([]visualContactFrameSample, 0, len(byTick))
	for _, frame := range byTick {
		frames = append(frames, frame)
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].tick < frames[j].tick })
	return frames, nil
}

func canonicalVisualContactPlayers(players []models.ReplayPlayerState) ([]visualContactPlayerSample, error) {
	byID := make(map[uint64]visualContactPlayerSample, len(players))
	for _, player := range players {
		if player.SteamID == 0 {
			continue
		}
		sample := visualContactPlayerSample{
			steamID:            player.SteamID,
			team:               canonicalTacticalTeam(player.Team),
			alive:              player.Alive,
			eyePositionSource:  strings.TrimSpace(player.EyePositionSource),
			weapon:             strings.ToLower(strings.TrimSpace(player.Weapon)),
			scoped:             player.IsScoped,
			zoomLevel:          player.ZoomLevel,
			zoomLevelAvailable: player.ZoomLevelAvailable,
		}
		eyes := r3.Vector{X: player.EyeX, Y: player.EyeY, Z: player.EyeZ}
		if player.EyePositionAvailable && visualcontact.IsFiniteVector(eyes) {
			sample.eyes = &eyes
		}
		if player.ViewAnglesAvailable && visualcontact.IsFinite(float64(player.Yaw)) && visualcontact.IsFinite(float64(player.Pitch)) {
			yaw, pitch := float64(player.Yaw), float64(player.Pitch)
			sample.yaw, sample.pitch = &yaw, &pitch
		}
		keyBytes, err := json.Marshal(struct {
			SteamID            uint64     `json:"steam_id"`
			Team               string     `json:"team"`
			Alive              bool       `json:"alive"`
			Eyes               *r3.Vector `json:"eyes"`
			EyePositionSource  string     `json:"eye_position_source"`
			Yaw                *float64   `json:"yaw"`
			Pitch              *float64   `json:"pitch"`
			Weapon             string     `json:"weapon"`
			Scoped             bool       `json:"scoped"`
			ZoomLevel          int        `json:"zoom_level"`
			ZoomLevelAvailable bool       `json:"zoom_level_available"`
		}{
			sample.steamID, sample.team, sample.alive, sample.eyes,
			sample.eyePositionSource, sample.yaw, sample.pitch, sample.weapon,
			sample.scoped, sample.zoomLevel, sample.zoomLevelAvailable,
		})
		if err != nil {
			return nil, fmt.Errorf("canonicalize player steam:%d: %w", player.SteamID, err)
		}
		sample.key = string(keyBytes)
		current, exists := byID[player.SteamID]
		if !exists || sample.key < current.key {
			byID[player.SteamID] = sample
		}
	}
	result := make([]visualContactPlayerSample, 0, len(byID))
	for _, player := range byID {
		result = append(result, player)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].steamID < result[j].steamID })
	return result, nil
}

func canonicalVisualContactEffects(effects []models.ReplayActiveEffect) []visualContactSmokeSample {
	if effects == nil {
		return nil
	}
	result := make([]visualContactSmokeSample, 0)
	for _, effect := range effects {
		if strings.EqualFold(strings.TrimSpace(effect.Type), "smoke") {
			result = append(result, visualContactSmokeSample{
				x: effect.X, y: effect.Y, radius: effect.Radius, startTick: effect.StartTick,
				key: fmt.Sprintf(
					"%020d:%016x:%016x:%016x",
					effect.StartTick,
					math.Float64bits(effect.X),
					math.Float64bits(effect.Y),
					math.Float64bits(effect.Radius),
				),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].key < result[j].key })
	return result
}

func evaluateObservedVisualContact(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	matchID string,
	roundID string,
	roundNumber int,
	tick int,
	t0Tick int,
	frame visualContactFrameSample,
	observer visualContactPlayerSample,
	subject visualContactPlayerSample,
) (models.ObservedVisualContactEvaluation, error) {
	availabilityTick := frame.tick
	observerID, subjectID := tacticalPlayerID(observer.steamID), tacticalPlayerID(subject.steamID)
	evaluation := models.ObservedVisualContactEvaluation{
		SchemaID:          models.ObservedVisualContactSchemaID,
		ContractVersion:   models.ObservedVisualContactContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		JoinKeys: models.ObservedVisualContactJoinKeys{
			MatchID: matchID, RoundID: roundID, ObserverID: observerID, SubjectID: subjectID,
		},
		RoundNumber:      roundNumber,
		Tick:             tick,
		T0Tick:           t0Tick,
		AvailabilityTick: tacticalIntPointer(availabilityTick),
		Status:           models.TacticalStatusUnavailable,
		CausalRole:       models.VisualContactCausalRole,
		VisibilityScope:  models.VisualContactVisibilityScope,
		Source:           models.VisualContactSource,
		LineOfSight:      unavailableVisualContactFact(),
		FieldOfView: models.ObservedVisualContactFOVFact{
			Status: models.VisualContactFactUnavailable,
		},
		SmokeOcclusion: models.ObservedVisualContactSmokeFact{
			Status: models.VisualContactFactUnavailable,
		},
		VisualContact: unavailableVisualContactFact(),
		Boundary:      models.VisualContactBoundaryUnavailable,
		Provenance: models.ObservedVisualContactProvenance{
			SourceArtifact:      visualContactSourceArtifact,
			SourceSchemaVersion: optionalPositiveInt(replay.Metadata.SchemaVersion),
			SourceRound:         roundNumber,
			SourceFrameTick:     frame.tick,
			GeometryStatus:      visualContactGeometryMissing,
			EyePositionSource:   visualContactEyeSource(observer, subject),
			RaycastMethod:       visualcontact.RaycastMethodVersion,
			FOVMethod:           visualcontact.FOVMethodVersion,
			SmokeMethod:         visualcontact.SmokeMethodVersion,
			InputFields:         append([]string(nil), visualContactInputFields...),
		},
	}

	if availabilityTick > tick || availabilityTick > t0Tick {
		return models.ObservedVisualContactEvaluation{}, fmt.Errorf(
			"build observed visual contact: availability violation at round %d tick %d: source frame %d exceeds T0 %d",
			roundNumber, tick, availabilityTick, t0Tick,
		)
	}

	reasons := make([]string, 0, 3)
	var lineOfSight, insideFOV, smokeOccluded *bool
	if geometry == nil || !geometry.IsLoaded() {
		reasons = append(reasons, "geometry_unavailable")
	} else if observer.eyes == nil || subject.eyes == nil {
		evaluation.Provenance.GeometryStatus = visualContactGeometryPositionMissing
		reasons = append(reasons, "eye_position_unavailable")
	} else {
		evaluation.Provenance.GeometryStatus = visualContactGeometryLoaded
		lineOfSight = tacticalBoolPointer(geometry.IsVisible(*observer.eyes, *subject.eyes))
		evaluation.LineOfSight = derivedVisualContactFact(*lineOfSight)
	}

	referenceFOV, circular := visualContactReferenceFOV(observer)
	if observer.eyes == nil || observer.yaw == nil || observer.pitch == nil {
		reasons = append(reasons, "observer_orientation_unavailable")
	} else if referenceFOV == nil {
		reasons = append(reasons, "scoped_fov_unavailable")
	} else if subject.eyes == nil {
		reasons = append(reasons, "subject_position_unavailable_for_fov")
	} else {
		frustum := visualcontact.NewFrustum(*observer.pitch, *observer.yaw, *referenceFOV, circular)
		insideFOV = tacticalBoolPointer(frustum.Contains(*observer.eyes, *subject.eyes))
		evaluation.FieldOfView = models.ObservedVisualContactFOVFact{
			Status:                 models.VisualContactFactDerived,
			Inside:                 insideFOV,
			ReferenceHorizontalFOV: referenceFOV,
		}
	}

	if !frame.smokeAvailable {
		reasons = append(reasons, "smoke_snapshot_unavailable")
	} else if observer.eyes == nil || subject.eyes == nil {
		reasons = append(reasons, "position_unavailable_for_smoke")
	} else {
		occluded := false
		for _, smoke := range frame.activeEffects {
			if smoke.startTick > frame.tick || smoke.startTick > t0Tick {
				return models.ObservedVisualContactEvaluation{}, fmt.Errorf(
					"build observed visual contact: smoke availability tick %d exceeds frame/T0 %d/%d",
					smoke.startTick, frame.tick, t0Tick,
				)
			}
			if !visualcontact.IsFinite(smoke.x) || !visualcontact.IsFinite(smoke.y) ||
				!visualcontact.IsFinite(smoke.radius) || smoke.radius <= 0 {
				reasons = append(reasons, "smoke_geometry_unavailable")
				occluded = false
				break
			}
			center := r3.Vector{X: smoke.x, Y: smoke.y}
			if visualcontact.DistancePointToSegment2D(center, *observer.eyes, *subject.eyes) < smoke.radius {
				occluded = true
			}
		}
		if !containsVisualContactReason(reasons, "smoke_geometry_unavailable") {
			smokeOccluded = tacticalBoolPointer(occluded)
			count := len(frame.activeEffects)
			evaluation.SmokeOcclusion = models.ObservedVisualContactSmokeFact{
				Status:           models.VisualContactFactDerived,
				Occluded:         smokeOccluded,
				ActiveSmokeCount: &count,
			}
		}
	}

	if lineOfSight != nil && insideFOV != nil && smokeOccluded != nil {
		contact := *lineOfSight && *insideFOV && !*smokeOccluded
		evaluation.Status = models.TacticalStatusObserved
		evaluation.VisualContact = derivedVisualContactFact(contact)
		evaluation.AbstentionReasons = []string{}
	} else {
		sort.Strings(reasons)
		evaluation.AbstentionReasons = uniqueVisualContactStrings(reasons)
	}
	return evaluation, nil
}

func visualContactReferenceFOV(player visualContactPlayerSample) (*float64, bool) {
	if !player.scoped {
		value := visualContactUnscopedFOV
		return &value, false
	}
	if !player.zoomLevelAvailable {
		return nil, false
	}
	zoom := player.zoomLevel
	if zoom < 1 {
		zoom = 1
	}
	value := 40.0
	circular := false
	switch player.weapon {
	case "weapon_awp", "awp":
		circular = true
		if zoom >= 2 {
			value = 10
		}
	case "weapon_ssg08", "ssg 08", "ssg08", "weapon_g3sg1", "g3sg1", "weapon_scar20", "scar-20", "scar20":
		circular = true
		if zoom >= 2 {
			value = 15
		}
	case "weapon_aug", "aug", "weapon_sg556", "weapon_sg553", "sg 553", "sg553":
		value = 45
	}
	return &value, circular
}

func applyVisualContactBoundary(
	ledger *models.ObservedVisualContactLedger,
	evaluation *models.ObservedVisualContactEvaluation,
	active map[visualContactPairKey]int,
) {
	key := visualContactPairKey{
		roundNumber: evaluation.RoundNumber,
		observerID:  evaluation.JoinKeys.ObserverID,
		subjectID:   evaluation.JoinKeys.SubjectID,
	}
	activeIndex, hasActive := active[key]
	if evaluation.VisualContact.Value == nil {
		evaluation.Boundary = models.VisualContactBoundaryUnavailable
		if hasActive {
			evaluation.JoinKeys.ContactID = copyVisualContactString(ledger.Contacts[activeIndex].JoinKeys.ContactID)
		}
		return
	}
	if *evaluation.VisualContact.Value {
		if hasActive {
			evaluation.Boundary = models.VisualContactBoundaryContinued
			evaluation.JoinKeys.ContactID = copyVisualContactString(ledger.Contacts[activeIndex].JoinKeys.ContactID)
			ledger.Contacts[activeIndex].LastConfirmedTick = evaluation.Tick
			return
		}
		contactID := fmt.Sprintf(
			"contact:%03d:%s:%s:%d",
			evaluation.RoundNumber,
			evaluation.JoinKeys.ObserverID,
			evaluation.JoinKeys.SubjectID,
			evaluation.Tick,
		)
		evaluation.Boundary = models.VisualContactBoundaryStarted
		evaluation.JoinKeys.ContactID = &contactID
		ledger.Contacts = append(ledger.Contacts, models.ObservedVisualContactEpisode{
			SchemaID:          models.ObservedVisualContactSchemaID,
			ContractVersion:   models.ObservedVisualContactContractVersion,
			IdentitySemantics: models.TacticalIdentityJoinOnly,
			JoinKeys: models.ObservedVisualContactJoinKeys{
				MatchID: evaluation.JoinKeys.MatchID, RoundID: evaluation.JoinKeys.RoundID,
				ObserverID: evaluation.JoinKeys.ObserverID, SubjectID: evaluation.JoinKeys.SubjectID,
				ContactID: &contactID,
			},
			RoundNumber:           evaluation.RoundNumber,
			T0Tick:                evaluation.T0Tick,
			StartTick:             evaluation.Tick,
			LastConfirmedTick:     evaluation.Tick,
			StartAvailabilityTick: *evaluation.AvailabilityTick,
			Status:                models.VisualContactEpisodeOpen,
			CausalRole:            models.VisualContactCausalRole,
		})
		active[key] = len(ledger.Contacts) - 1
		return
	}

	if !hasActive {
		evaluation.Boundary = models.VisualContactBoundaryInactive
		return
	}
	evaluation.Boundary = models.VisualContactBoundaryEnded
	evaluation.JoinKeys.ContactID = copyVisualContactString(ledger.Contacts[activeIndex].JoinKeys.ContactID)
	endTick, endAvailability := evaluation.Tick, *evaluation.AvailabilityTick
	ledger.Contacts[activeIndex].EndTick = &endTick
	ledger.Contacts[activeIndex].EndAvailabilityTick = &endAvailability
	ledger.Contacts[activeIndex].Status = models.VisualContactEpisodeClosed
	delete(active, key)
}

// ValidateObservedVisualContactLedger enforces causal availability and the
// contact formula without consulting oracle or outcome partitions.
func ValidateObservedVisualContactLedger(ledger models.ObservedVisualContactLedger) error {
	if ledger.SchemaID != models.ObservedVisualContactSchemaID ||
		ledger.ContractVersion != models.ObservedVisualContactContractVersion {
		return fmt.Errorf("visual contact ledger: unsupported schema or contract version")
	}
	if ledger.IdentitySemantics != models.TacticalIdentityJoinOnly || strings.TrimSpace(ledger.JoinKeys.MatchID) == "" {
		return fmt.Errorf("visual contact ledger: identity must be non-empty and join_only")
	}
	if ledger.T0Tick < 0 {
		return fmt.Errorf("visual contact ledger: t0_tick must be non-negative")
	}
	seenEvaluations := make(map[string]struct{}, len(ledger.Evaluations))
	for index, evaluation := range ledger.Evaluations {
		location := fmt.Sprintf("visual contact evaluation[%d]", index)
		if evaluation.SchemaID != ledger.SchemaID || evaluation.ContractVersion != ledger.ContractVersion {
			return fmt.Errorf("%s: schema/version mismatch", location)
		}
		if evaluation.IdentitySemantics != models.TacticalIdentityJoinOnly ||
			evaluation.JoinKeys.MatchID != ledger.JoinKeys.MatchID ||
			strings.TrimSpace(evaluation.JoinKeys.RoundID) == "" ||
			strings.TrimSpace(evaluation.JoinKeys.ObserverID) == "" ||
			strings.TrimSpace(evaluation.JoinKeys.SubjectID) == "" {
			return fmt.Errorf("%s: identifiers must be isolated as non-empty join keys", location)
		}
		evaluationKey := fmt.Sprintf(
			"%d:%d:%s:%s", evaluation.RoundNumber, evaluation.Tick,
			evaluation.JoinKeys.ObserverID, evaluation.JoinKeys.SubjectID,
		)
		if _, exists := seenEvaluations[evaluationKey]; exists {
			return fmt.Errorf("%s: duplicate observer/subject/tick key", location)
		}
		seenEvaluations[evaluationKey] = struct{}{}
		if evaluation.T0Tick != ledger.T0Tick || evaluation.Tick > ledger.T0Tick ||
			evaluation.AvailabilityTick == nil || *evaluation.AvailabilityTick > evaluation.Tick ||
			*evaluation.AvailabilityTick > ledger.T0Tick {
			return fmt.Errorf("%s: availability_tick must be <= tick <= t0_tick", location)
		}
		if evaluation.CausalRole != models.VisualContactCausalRole ||
			evaluation.VisibilityScope != models.VisualContactVisibilityScope ||
			evaluation.Source != models.VisualContactSource ||
			evaluation.Provenance.SourceFrameTick > *evaluation.AvailabilityTick {
			return fmt.Errorf("%s: causal metadata or source frame is invalid", location)
		}
		if err := validateVisualContactFact(evaluation.LineOfSight, location+".line_of_sight"); err != nil {
			return err
		}
		if evaluation.FieldOfView.Inside == nil {
			if evaluation.FieldOfView.Status != models.VisualContactFactUnavailable ||
				evaluation.FieldOfView.ReferenceHorizontalFOV != nil {
				return fmt.Errorf("%s.field_of_view: missing FOV must be null/unavailable", location)
			}
		} else if evaluation.FieldOfView.Status != models.VisualContactFactDerived ||
			evaluation.FieldOfView.ReferenceHorizontalFOV == nil ||
			!visualcontact.IsFinite(*evaluation.FieldOfView.ReferenceHorizontalFOV) ||
			*evaluation.FieldOfView.ReferenceHorizontalFOV <= 0 {
			return fmt.Errorf("%s.field_of_view: derived FOV requires a finite positive reference", location)
		}
		if evaluation.SmokeOcclusion.Occluded == nil {
			if evaluation.SmokeOcclusion.Status != models.VisualContactFactUnavailable ||
				evaluation.SmokeOcclusion.ActiveSmokeCount != nil {
				return fmt.Errorf("%s.smoke_occlusion: missing smoke must be null/unavailable", location)
			}
		} else if evaluation.SmokeOcclusion.Status != models.VisualContactFactDerived ||
			evaluation.SmokeOcclusion.ActiveSmokeCount == nil ||
			*evaluation.SmokeOcclusion.ActiveSmokeCount < 0 {
			return fmt.Errorf("%s.smoke_occlusion: derived smoke requires a non-negative count", location)
		} else if *evaluation.SmokeOcclusion.Occluded && *evaluation.SmokeOcclusion.ActiveSmokeCount == 0 {
			return fmt.Errorf("%s.smoke_occlusion: occlusion requires at least one active smoke", location)
		}
		if err := validateVisualContactFact(evaluation.VisualContact, location+".visual_contact"); err != nil {
			return err
		}
		available := evaluation.LineOfSight.Value != nil && evaluation.FieldOfView.Inside != nil &&
			evaluation.SmokeOcclusion.Occluded != nil
		if available {
			expected := *evaluation.LineOfSight.Value && *evaluation.FieldOfView.Inside &&
				!*evaluation.SmokeOcclusion.Occluded
			if evaluation.Status != models.TacticalStatusObserved || evaluation.VisualContact.Value == nil ||
				*evaluation.VisualContact.Value != expected || len(evaluation.AbstentionReasons) != 0 {
				return fmt.Errorf("%s: visual contact must equal raycast && FOV && !smoke", location)
			}
		} else if evaluation.Status != models.TacticalStatusUnavailable ||
			evaluation.VisualContact.Value != nil || len(evaluation.AbstentionReasons) == 0 {
			return fmt.Errorf("%s: missing inputs require null/unavailable with abstention", location)
		}
		if err := validateVisualContactBoundary(evaluation, location); err != nil {
			return err
		}
		if !sort.StringsAreSorted(evaluation.AbstentionReasons) || !sort.StringsAreSorted(evaluation.Provenance.InputFields) {
			return fmt.Errorf("%s: reasons and provenance fields must be sorted", location)
		}
	}
	seenContacts := make(map[string]struct{}, len(ledger.Contacts))
	for index, episode := range ledger.Contacts {
		location := fmt.Sprintf("visual contact episode[%d]", index)
		if episode.SchemaID != ledger.SchemaID || episode.ContractVersion != ledger.ContractVersion ||
			episode.JoinKeys.ContactID == nil || strings.TrimSpace(*episode.JoinKeys.ContactID) == "" ||
			episode.IdentitySemantics != models.TacticalIdentityJoinOnly || episode.T0Tick != ledger.T0Tick ||
			episode.CausalRole != models.VisualContactCausalRole ||
			episode.JoinKeys.MatchID != ledger.JoinKeys.MatchID {
			return fmt.Errorf("%s: contact identity must be a join-only key", location)
		}
		if _, exists := seenContacts[*episode.JoinKeys.ContactID]; exists {
			return fmt.Errorf("%s: duplicate contact_id", location)
		}
		seenContacts[*episode.JoinKeys.ContactID] = struct{}{}
		if episode.StartAvailabilityTick > episode.StartTick || episode.StartTick > ledger.T0Tick ||
			episode.LastConfirmedTick < episode.StartTick || episode.LastConfirmedTick > ledger.T0Tick {
			return fmt.Errorf("%s: invalid causal start/confirmation ticks", location)
		}
		if episode.Status == models.VisualContactEpisodeClosed {
			if episode.EndTick == nil || episode.EndAvailabilityTick == nil ||
				*episode.EndTick < episode.LastConfirmedTick || *episode.EndTick > ledger.T0Tick ||
				*episode.EndAvailabilityTick > *episode.EndTick {
				return fmt.Errorf("%s: invalid causal end ticks", location)
			}
		} else if episode.Status != models.VisualContactEpisodeOpen || episode.EndTick != nil || episode.EndAvailabilityTick != nil {
			return fmt.Errorf("%s: invalid episode status/end", location)
		}
	}
	return nil
}

func validateVisualContactFact(fact models.ObservedVisualContactFact, location string) error {
	if fact.Value == nil {
		if fact.Status != models.VisualContactFactUnavailable {
			return fmt.Errorf("%s: null value must be unavailable", location)
		}
		return nil
	}
	if fact.Status != models.VisualContactFactDerived {
		return fmt.Errorf("%s: boolean value must be derived", location)
	}
	return nil
}

func validateVisualContactBoundary(
	evaluation models.ObservedVisualContactEvaluation,
	location string,
) error {
	contactIDPresent := evaluation.JoinKeys.ContactID != nil && strings.TrimSpace(*evaluation.JoinKeys.ContactID) != ""
	if evaluation.VisualContact.Value == nil {
		if evaluation.Boundary != models.VisualContactBoundaryUnavailable {
			return fmt.Errorf("%s: unavailable contact requires unavailable boundary", location)
		}
		return nil
	}
	if *evaluation.VisualContact.Value {
		if (evaluation.Boundary != models.VisualContactBoundaryStarted &&
			evaluation.Boundary != models.VisualContactBoundaryContinued) || !contactIDPresent {
			return fmt.Errorf("%s: visible contact requires started/continued boundary and contact join key", location)
		}
		return nil
	}
	if evaluation.Boundary == models.VisualContactBoundaryInactive {
		if contactIDPresent {
			return fmt.Errorf("%s: inactive contact cannot carry contact_id", location)
		}
		return nil
	}
	if evaluation.Boundary != models.VisualContactBoundaryEnded || !contactIDPresent {
		return fmt.Errorf("%s: negative contact requires inactive or ended boundary", location)
	}
	return nil
}

func derivedVisualContactFact(value bool) models.ObservedVisualContactFact {
	return models.ObservedVisualContactFact{Status: models.VisualContactFactDerived, Value: tacticalBoolPointer(value)}
}

func unavailableVisualContactFact() models.ObservedVisualContactFact {
	return models.ObservedVisualContactFact{Status: models.VisualContactFactUnavailable}
}

func visualContactEyeSource(observer, subject visualContactPlayerSample) string {
	observerSource := observer.eyePositionSource
	if observerSource == "" {
		observerSource = "unavailable"
	}
	subjectSource := subject.eyePositionSource
	if subjectSource == "" {
		subjectSource = "unavailable"
	}
	return "observer=" + observerSource + ";subject=" + subjectSource
}

func containsVisualContactReason(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func uniqueVisualContactStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func copyVisualContactString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
