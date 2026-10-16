package parser

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"cs2-demo-service/models"
)

const EconomyCandidateV2SchemaID = "stratai.observed_economy_candidate@2"
const EconomyCandidateV2ManifestSchemaID = "stratai.economy_candidate_manifest@2"
const EconomyCandidateV2Version = "1.0.0-candidate.2"

type EconomyCanonicalActor struct {
	RoundID     string `json:"round_id"`
	RoundNumber int    `json:"round_number"`
	PlayerID    string `json:"player_id"`
	Side        string `json:"side"`
}

type EconomyRoundLink struct {
	Status           string  `json:"status"`
	Reason           string  `json:"reason"`
	CanonicalRoundID *string `json:"canonical_round_id"`
	Source           string  `json:"source"`
}

type EconomyCandidateFrontier struct {
	models.EconomyInitialFrontier
	RoundLink EconomyRoundLink `json:"round_link"`
}

type EconomyCandidateV2 struct {
	EconomyCandidate
	InitialFrontiers []EconomyCandidateFrontier `json:"initial_frontiers"`
}

// Only identity and boundaries are read from the canonical bundle. Winners,
// future balances and scores never complete or qualify the observed snapshot.
func BuildEconomyCandidateV2(ctx *models.DemoContext, alias, checksum, build string, rounds []models.CanonicalRound, actors []EconomyCanonicalActor) (EconomyCandidateV2, error) {
	base, err := BuildEconomyCandidate(ctx, alias, checksum, build)
	if err != nil {
		return EconomyCandidateV2{}, err
	}
	if len(ctx.EconomyObservations.InitialFrontiers) != len(base.Rounds) {
		return EconomyCandidateV2{}, errors.New("initial frontier observer unavailable")
	}
	result := EconomyCandidateV2{EconomyCandidate: base, InitialFrontiers: []EconomyCandidateFrontier{}}
	result.SchemaID, result.ContractVersion = EconomyCandidateV2SchemaID, EconomyCandidateV2Version
	for i, frontier := range ctx.EconomyObservations.InitialFrontiers {
		link := economyRoundLink(base, i, frontier, rounds, actors)
		result.InitialFrontiers = append(result.InitialFrontiers, EconomyCandidateFrontier{EconomyInitialFrontier: frontier, RoundLink: link})
	}
	// One canonical row must never certify two observer instances.
	uses := map[string]int{}
	for _, frontier := range result.InitialFrontiers {
		if frontier.RoundLink.CanonicalRoundID != nil {
			uses[*frontier.RoundLink.CanonicalRoundID]++
		}
	}
	for i := range result.InitialFrontiers {
		link := &result.InitialFrontiers[i].RoundLink
		if link.CanonicalRoundID != nil && uses[*link.CanonicalRoundID] != 1 {
			link.Status, link.Reason, link.CanonicalRoundID = "excluded", "ambiguous_instance", nil
		}
	}
	return result, nil
}

func economyRoundLink(base EconomyCandidate, index int, frontier models.EconomyInitialFrontier, canonical []models.CanonicalRound, actors []EconomyCanonicalActor) EconomyRoundLink {
	link := EconomyRoundLink{Status: "excluded", Reason: "native_competitive_state_unavailable", Source: "canonical/core/rounds.json+derived/economy_players.json"}
	for key, expected := range map[string]bool{"match_started": true, "warmup": false, "freeze": true} {
		fact := frontier.NativeRules[key]
		if fact.Status != "observed" || fact.Value != expected {
			return link
		}
	}
	round := base.Rounds[index]
	link.Reason = "canonical_boundaries_unmatched"
	if frontier.OfficialEndTick == nil {
		link.Reason = "official_end_unavailable"
		return link
	}
	matched := []models.CanonicalRound{}
	for _, row := range canonical {
		if row.RoundNumber == round.RoundNumber && row.StartTick != nil && *row.StartTick == round.StartTick && row.EndTick != nil && *row.EndTick == *frontier.OfficialEndTick {
			matched = append(matched, row)
		}
	}
	if len(matched) != 1 {
		if len(matched) > 1 {
			link.Reason = "ambiguous_canonical_round"
		}
		return link
	}
	id := matched[0].RoundID
	if id != fmt.Sprintf("%s:round:%03d", base.SourceAlias, round.RoundNumber) {
		link.Reason = "canonical_identity_mismatch"
		return link
	}
	if frontier.RoundStartProbe == nil {
		link.Reason = "initial_roster_unavailable"
		return link
	}
	bridge := map[string]string{}
	for _, p := range base.Participants {
		bridge[p.ActorKey] = p.Alias
	}
	expected := map[string]string{}
	for _, row := range actors {
		if row.RoundID != id {
			continue
		}
		key := fmt.Sprintf("%x", sha256.Sum256([]byte(base.SourceSHA256+"\n"+row.PlayerID)))
		alias, exists := bridge[key]
		if !exists || expected[alias] != "" || row.RoundNumber != round.RoundNumber || (row.Side != "ct" && row.Side != "t") {
			link.Reason = "canonical_participants_mismatch"
			return link
		}
		expected[alias] = strings.ToUpper(row.Side)
	}
	if len(expected) != 10 || len(frontier.RoundStartProbe.Actors) != 10 {
		link.Reason = "initial_roster_unavailable"
		return link
	}
	counts := map[string]int{}
	for _, actor := range frontier.RoundStartProbe.Actors {
		if expected[actor.ActorAlias] != actor.Side {
			link.Reason = "canonical_participants_mismatch"
			return link
		}
		delete(expected, actor.ActorAlias)
		counts[actor.Side]++
	}
	if len(expected) != 0 || counts["CT"] != 5 || counts["T"] != 5 {
		link.Reason = "canonical_participants_mismatch"
		return link
	}
	link.Status, link.Reason, link.CanonicalRoundID = "verified", "unique_identity_start_official_end_and_roster", &id
	return link
}

func EconomyCandidateV2Summary(candidate EconomyCandidateV2) map[string]int {
	result := EconomyCandidateSummary(candidate.EconomyCandidate)
	result["round_start_probes"], result["competitive_round_links"], result["excluded_round_links"] = 0, 0, 0
	for _, frontier := range candidate.InitialFrontiers {
		if frontier.RoundStartProbe != nil {
			result["round_start_probes"]++
		}
		if frontier.RoundLink.Status == "verified" {
			result["competitive_round_links"]++
		} else {
			result["excluded_round_links"]++
		}
	}
	return result
}
