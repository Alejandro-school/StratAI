package models

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
)

//go:embed economy_session_payload_probe.json
var sessionPayloadProbeJSON []byte

type EconomySessionPayloadProbe struct {
	Version string            `json:"version"`
	Fields  map[string]string `json:"fields"`
}

func LoadEconomySessionPayloadProbe() (EconomySessionPayloadProbe, string) {
	var probe EconomySessionPayloadProbe
	if err := json.Unmarshal(sessionPayloadProbeJSON, &probe); err != nil {
		panic(err)
	}
	digest := sha256.Sum256(sessionPayloadProbeJSON)
	return probe, hex.EncodeToString(digest[:])
}

type EconomySessionField struct {
	Occurrence          int     `json:"occurrence"`
	NodeIndex           *int    `json:"node_index"`
	Path                string  `json:"path"`
	Present             bool    `json:"present"`
	NativeType          *int    `json:"native_type"`
	Status              string  `json:"status"`
	Value               *int32  `json:"value"`
	ReferenceNameSHA256 *string `json:"reference_name_sha256"`
}

type EconomySessionBlock struct {
	SourcePath        string                `json:"source_path"`
	Source            EconomyOpaqueField    `json:"source"`
	Status            string                `json:"status"`
	Format            string                `json:"format"`
	Compression       string                `json:"compression"`
	ChecksumVerified  bool                  `json:"checksum_verified"`
	Nodes             int                   `json:"nodes"`
	Scalars           int                   `json:"scalars"`
	EconomicNamesSeen []string              `json:"economic_names_seen"`
	Fields            []EconomySessionField `json:"fields"`
}

type EconomySessionPayload struct {
	Sequence int                   `json:"sequence"`
	Tick     int                   `json:"tick"`
	Blocks   []EconomySessionBlock `json:"blocks"`
}

type EconomySessionPayloads struct {
	Sessions []EconomySessionPayload `json:"sessions"`
}
