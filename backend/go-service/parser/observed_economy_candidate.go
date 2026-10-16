package parser

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/economy"
)

const EconomyCandidateSchemaID = "stratai.observed_economy_candidate@1"
const EconomyCandidateManifestSchemaID = "stratai.economy_candidate_manifest@1"
const EconomyCandidateVersion = "1.0.0-candidate.1"
const EconomyCandidateFilename = "observed_economy_candidate.json"
const EconomyCandidateManifestFilename = "economy_candidate_manifest.json"

type EconomyExecution struct {
	FreezeSnapshot *models.EconomySnapshot       `json:"freeze_snapshot"`
	Events         []models.EconomyCallbackEvent `json:"events"`
}

type EconomyComparison struct {
	ActorAlias string `json:"actor_alias"`
	Metric     string `json:"metric"`
	Delta      *int   `json:"delta"`
	Status     string `json:"status"`
}

type EconomyReconciliation struct {
	RoundEndSnapshot *models.EconomySnapshot       `json:"round_end_snapshot"`
	OtherEvents      []models.EconomyCallbackEvent `json:"other_events"`
	Comparisons      []EconomyComparison           `json:"comparisons"`
}

type EconomyCandidateRound struct {
	Instance       int                     `json:"instance"`
	RoundNumber    int                     `json:"round_number"`
	StartTick      int                     `json:"start_tick"`
	FreezeEndTick  *int                    `json:"freeze_end_tick"`
	EndTick        *int                    `json:"end_tick"`
	InitialState   *models.EconomySnapshot `json:"initial_state"`
	Execution      EconomyExecution        `json:"execution"`
	Reconciliation EconomyReconciliation   `json:"reconciliation"`
}

type EconomyConfigReference struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Use     string `json:"use"`
}

type EconomyCandidate struct {
	SchemaID           string                            `json:"schema_id"`
	ContractVersion    string                            `json:"contract_version"`
	SourceAlias        string                            `json:"source_alias"`
	SourceSHA256       string                            `json:"source_sha256"`
	MapID              string                            `json:"map_id"`
	TickRate           float64                           `json:"tick_rate"`
	BuildIdentifier    string                            `json:"build_identifier"`
	VisibilityContract string                            `json:"visibility_contract"`
	OrderContract      string                            `json:"order_contract"`
	TrainingAllowed    bool                              `json:"training_allowed"`
	ProductSupported   bool                              `json:"product_supported"`
	RulesCompatibility string                            `json:"rules_compatibility"`
	Configs            map[string]EconomyConfigReference `json:"configs"`
	Participants       []SpatialParticipantBridgeEntry   `json:"participants"`
	Rounds             []EconomyCandidateRound           `json:"rounds"`
}

// BuildEconomyCandidate separates immutable parse-time observations from later
// execution labels and aggregate comparisons. Neither comparisons nor outcomes
// are consulted when copying the initial snapshot or its availability status.
func BuildEconomyCandidate(ctx *models.DemoContext, alias, checksum, build string) (EconomyCandidate, error) {
	if ctx == nil || ctx.EconomyObservations == nil || !validObservedVisualContactSmokeIdentity(alias, checksum) || ctx.MapName == "" || build == "" {
		return EconomyCandidate{}, errors.New("economy source/observer unavailable")
	}
	result := EconomyCandidate{
		SchemaID: EconomyCandidateSchemaID, ContractVersion: EconomyCandidateVersion,
		SourceAlias: alias, SourceSHA256: checksum, MapID: ctx.MapName, BuildIdentifier: build,
		VisibilityContract: "self_and_same_side_only_per_actor_no_opponent_economy",
		OrderContract:      "parser_callback_sequence_not_engine_subtick_order",
		RulesCompatibility: "unverified_no_price_or_reward_inference",
		Configs: map[string]EconomyConfigReference{
			"legacy_prices": {Version: economy.PriceTableVersion, SHA256: economy.PriceTableChecksum(), Use: "reference_only_not_applied"},
			"legacy_rules":  {Version: economy.RulesVersion, SHA256: economy.RulesChecksum(), Use: "reference_only_not_applied"},
		}, Participants: []SpatialParticipantBridgeEntry{}, Rounds: []EconomyCandidateRound{},
	}
	if ctx.Parser != nil {
		result.TickRate = ctx.Parser.TickRate()
	} else if ctx.ReplayData != nil {
		result.TickRate = ctx.ReplayData.Metadata.TickRate
	}
	for id, participant := range ctx.EconomyObservations.ActorAliases {
		key := sha256.Sum256([]byte(checksum + "\n" + canonicalPlayerID(id)))
		result.Participants = append(result.Participants, SpatialParticipantBridgeEntry{ActorKey: fmt.Sprintf("%x", key), Alias: participant})
	}
	sort.Slice(result.Participants, func(i, j int) bool { return result.Participants[i].Alias < result.Participants[j].Alias })
	instances := map[int]int{}
	for _, round := range ctx.EconomyObservations.Rounds {
		instances[round.RoundNumber]++
	}
	for _, source := range ctx.EconomyObservations.Rounds {
		round := EconomyCandidateRound{Instance: source.Instance, RoundNumber: source.RoundNumber, StartTick: source.StartTick,
			FreezeEndTick: source.FreezeEndTick, EndTick: source.EndTick,
			Execution:      EconomyExecution{Events: []models.EconomyCallbackEvent{}},
			Reconciliation: EconomyReconciliation{OtherEvents: []models.EconomyCallbackEvent{}, Comparisons: []EconomyComparison{}},
		}
		for _, snapshot := range source.Snapshots {
			copy := snapshot
			switch snapshot.Phase {
			case "initial_available":
				if round.InitialState != nil {
					return EconomyCandidate{}, errors.New("duplicate initial snapshot")
				}
				round.InitialState = &copy
			case "freeze_end":
				if round.Execution.FreezeSnapshot != nil {
					return EconomyCandidate{}, errors.New("duplicate freeze snapshot")
				}
				round.Execution.FreezeSnapshot = &copy
			case "round_end":
				if round.Reconciliation.RoundEndSnapshot != nil {
					return EconomyCandidate{}, errors.New("duplicate end snapshot")
				}
				round.Reconciliation.RoundEndSnapshot = &copy
			default:
				return EconomyCandidate{}, errors.New("unknown economy snapshot phase")
			}
		}
		for _, event := range source.Events {
			// The boundary tick is ambiguous, irrespective of callback order.
			// Unknown/missing freeze-end never accredits a complete buy window.
			if source.FreezeEndTick != nil && event.Tick > source.StartTick && event.Tick < *source.FreezeEndTick {
				round.Execution.Events = append(round.Execution.Events, event)
			} else {
				round.Reconciliation.OtherEvents = append(round.Reconciliation.OtherEvents, event)
			}
		}
		// Legacy aggregates do not retain restart instances. Never join an
		// ambiguous instance to an aggregate by the displayed round number.
		if instances[source.RoundNumber] == 1 {
			round.Reconciliation.Comparisons = economyAggregateComparisons(ctx, source.RoundNumber, round.InitialState, round.Reconciliation.RoundEndSnapshot)
		}
		result.Rounds = append(result.Rounds, round)
	}
	if err := ValidateEconomyCandidate(result); err != nil {
		return EconomyCandidate{}, err
	}
	return result, nil
}

func economyAggregateComparisons(ctx *models.DemoContext, number int, initial, end *models.EconomySnapshot) []EconomyComparison {
	result := []EconomyComparison{}
	var legacy *models.AI_EconomyRound
	for i := range ctx.AI_EconomyRounds {
		if ctx.AI_EconomyRounds[i].Round == number {
			if legacy != nil {
				return result
			}
			legacy = &ctx.AI_EconomyRounds[i]
		}
	}
	if legacy == nil {
		return result
	}
	for _, player := range legacy.Players {
		alias, ok := ctx.EconomyObservations.ActorAliases[player.SteamID]
		if !ok {
			continue
		}
		for _, pair := range []struct {
			name     string
			snapshot *models.EconomySnapshot
			value    int
		}{
			{"initial_minus_legacy_initial_money", initial, player.InitialMoney},
			{"end_frame_minus_legacy_round_end_money", end, player.MoneyAtRoundEnd},
		} {
			comparison := EconomyComparison{ActorAlias: alias, Metric: pair.name, Status: "unavailable"}
			if pair.snapshot != nil {
				for _, actor := range pair.snapshot.Actors {
					if actor.ActorAlias == alias && actor.Facts["money"].Status == "observed" {
						if value, ok := actor.Facts["money"].Value.(int); ok {
							delta := value - pair.value
							comparison.Delta = &delta
							comparison.Status = "diagnostic_different_observation_frontiers"
						}
					}
				}
			}
			result = append(result, comparison)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ActorAlias == result[j].ActorAlias {
			return result[i].Metric < result[j].Metric
		}
		return result[i].ActorAlias < result[j].ActorAlias
	})
	return result
}

func ValidateEconomyCandidate(candidate EconomyCandidate) error {
	if candidate.SchemaID != EconomyCandidateSchemaID || candidate.ContractVersion != EconomyCandidateVersion || candidate.TickRate <= 0 || candidate.TrainingAllowed || candidate.ProductSupported || len(candidate.Rounds) == 0 || len(candidate.Participants) == 0 {
		return errors.New("invalid economic candidate metadata")
	}
	initialActors := 0
	for _, round := range candidate.Rounds {
		if round.RoundNumber < 1 || round.StartTick < 0 {
			return errors.New("invalid economic round")
		}
		for _, snapshot := range []*models.EconomySnapshot{round.InitialState, round.Execution.FreezeSnapshot, round.Reconciliation.RoundEndSnapshot} {
			if snapshot == nil {
				continue
			}
			if snapshot.Tick < round.StartTick {
				return errors.New("snapshot precedes round")
			}
			for _, actor := range snapshot.Actors {
				if snapshot.Phase == "initial_available" {
					initialActors++
				}
				for _, fact := range actor.Facts {
					if fact.Status == "observed" {
						if fact.Value == nil || fact.ObservationTick == nil || fact.AvailabilityTick == nil || *fact.ObservationTick != snapshot.Tick || *fact.AvailabilityTick != snapshot.Tick {
							return errors.New("economic fact has illegal availability")
						}
					} else if fact.Status != "unavailable" || fact.Value != nil || fact.ObservationTick != nil || fact.AvailabilityTick != nil {
						return errors.New("invalid missing economy fact")
					}
				}
			}
		}
	}
	if initialActors == 0 {
		return errors.New("no economic initial observations")
	}
	_, err := json.Marshal(candidate)
	return err
}

func EconomyCandidateSummary(candidate EconomyCandidate) map[string]int {
	counts := map[string]int{"round_instances": len(candidate.Rounds), "participants": len(candidate.Participants), "initial_snapshots": 0, "freeze_snapshots": 0, "end_snapshots": 0, "initial_actors": 0, "prepurchase_available": 0, "money_observed": 0, "money_zero": 0, "inventory_observed": 0, "execution_events": 0, "other_events": 0, "prior_allied_drop_links": 0, "unresolved_pickups": 0}
	for _, round := range candidate.Rounds {
		for _, item := range []struct {
			key      string
			snapshot *models.EconomySnapshot
		}{{"initial_snapshots", round.InitialState}, {"freeze_snapshots", round.Execution.FreezeSnapshot}, {"end_snapshots", round.Reconciliation.RoundEndSnapshot}} {
			if item.snapshot == nil {
				continue
			}
			counts[item.key]++
			for _, actor := range item.snapshot.Actors {
				if item.key == "initial_snapshots" {
					counts["initial_actors"]++
				}
				if actor.PrepurchaseStatus == "available" {
					counts["prepurchase_available"]++
				}
				if actor.Facts["money"].Status == "observed" {
					counts["money_observed"]++
					if actor.Facts["money"].Value == 0 {
						counts["money_zero"]++
					}
				}
				if actor.Facts["inventory"].Status == "observed" {
					counts["inventory_observed"]++
				}
			}
		}
		counts["execution_events"] += len(round.Execution.Events)
		counts["other_events"] += len(round.Reconciliation.OtherEvents)
		for _, events := range [][]models.EconomyCallbackEvent{round.Execution.Events, round.Reconciliation.OtherEvents} {
			for _, event := range events {
				if event.PriorDropEventID != nil {
					counts["prior_allied_drop_links"]++
				}
				if event.Attribution == "purchase_or_pickup_unresolved" {
					counts["unresolved_pickups"]++
				}
			}
		}
	}
	return counts
}
