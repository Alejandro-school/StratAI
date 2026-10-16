package parser

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	"cs2-demo-service/models"
)

type SpatialParticipantBridgeEntry struct {
	ActorKey string `json:"actor_key"`
	Alias    string `json:"participant_alias"`
}

type SpatialParticipantBridge struct {
	SchemaID        string                          `json:"schema_id"`
	SourceAlias     string                          `json:"source_alias"`
	SourceSHA256    string                          `json:"source_sha256"`
	MapID           string                          `json:"map_id"`
	KeyContract     string                          `json:"key_contract"`
	TrainingAllowed bool                            `json:"training_allowed"`
	Participants    []SpatialParticipantBridgeEntry `json:"participants"`
}

// BuildSpatialParticipantBridge binds the existing local aliases to canonical
// actors without retaining Steam identities. Keys are match-scoped, join-only.
func BuildSpatialParticipantBridge(replay *models.ReplayData, alias, checksum string) (SpatialParticipantBridge, error) {
	if replay == nil || !validObservedVisualContactSmokeIdentity(alias, checksum) || replay.Metadata.MapName == "" {
		return SpatialParticipantBridge{}, errors.New("invalid spatial bridge source")
	}
	aliases, err := exposureParticipantAliases(replay.Rounds)
	if err != nil || len(aliases) == 0 {
		return SpatialParticipantBridge{}, errors.New("spatial bridge has no reconciled participants")
	}
	bridge := SpatialParticipantBridge{
		SchemaID: "stratai.spatial_participant_bridge@1", SourceAlias: alias,
		SourceSHA256: checksum, MapID: replay.Metadata.MapName,
		KeyContract: "sha256(source_sha256 + LF + canonical_actor_id)@1",
	}
	for steamID, participant := range aliases {
		key := sha256.Sum256([]byte(checksum + "\n" + canonicalPlayerID(steamID)))
		bridge.Participants = append(bridge.Participants, SpatialParticipantBridgeEntry{ActorKey: fmt.Sprintf("%x", key), Alias: participant})
	}
	sort.Slice(bridge.Participants, func(i, j int) bool { return bridge.Participants[i].Alias < bridge.Participants[j].Alias })
	return bridge, nil
}
