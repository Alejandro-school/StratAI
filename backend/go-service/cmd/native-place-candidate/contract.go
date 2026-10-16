// Command native-place-candidate captures native, unclassified place properties.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
)

const producerVersion = "stratai.native_place_candidate@1"
const targetSchema = "stratai.native_place_targets@1"
const placeField = "m_szLastPlaceName"

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var opaquePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
var mapPattern = regexp.MustCompile(`^[a-z0-9_/-]{1,100}$`)

type position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type cut struct {
	Frame    int    `json:"frame"`
	Tick     int    `json:"tick"`
	Instance int    `json:"instance"`
	Epoch    int    `json:"epoch"`
	Round    int    `json:"round_number"`
	Phase    string `json:"phase"`
}

type actorTarget struct {
	Alias     string    `json:"alias"`
	Handle    *uint64   `json:"handle"`
	Side      string    `json:"side"`
	Connected *bool     `json:"connected"`
	Alive     *bool     `json:"alive"`
	Position  *position `json:"position"`
}

type target struct {
	ID         string        `json:"target_id"`
	ContextSHA string        `json:"candidate_context_sha256"`
	Cut        cut           `json:"cut"`
	Actors     []actorTarget `json:"actors"`
}

type config struct {
	Schema      string   `json:"schema_id"`
	SourceSHA   string   `json:"source_sha256"`
	SourceAlias string   `json:"source_alias"`
	MapName     string   `json:"map_name"`
	Targets     []target `json:"targets"`
}

func finite(p *position) bool {
	return p != nil && !math.IsNaN(p.X) && !math.IsInf(p.X, 0) &&
		!math.IsNaN(p.Y) && !math.IsInf(p.Y, 0) && !math.IsNaN(p.Z) && !math.IsInf(p.Z, 0)
}

// A causal cut must belong to the same round instance AND tick epoch. Frame
// ordering alone cannot establish tick availability across demo resets.
func within(observation, limit cut) bool {
	return observation.Epoch == limit.Epoch && observation.Instance == limit.Instance &&
		observation.Round == limit.Round && observation.Frame <= limit.Frame && observation.Tick <= limit.Tick
}

func (c config) validate(expectedSHA string) error {
	if c.Schema != targetSchema || !hashPattern.MatchString(c.SourceSHA) || c.SourceSHA != expectedSHA {
		return errors.New("target schema or expected source hash differs")
	}
	if !opaquePattern.MatchString(c.SourceAlias) || !mapPattern.MatchString(c.MapName) {
		return errors.New("opaque source alias and explicit map name required")
	}
	if len(c.Targets) == 0 || len(c.Targets) > 100000 {
		return errors.New("target count outside supported capture scope")
	}
	ids := map[string]bool{}
	frames := map[int]cut{}
	previousFrame := -1
	for _, t := range c.Targets {
		if !opaquePattern.MatchString(t.ID) || ids[t.ID] || !hashPattern.MatchString(t.ContextSHA) {
			return errors.New("target ID or context hash invalid or duplicate")
		}
		ids[t.ID] = true
		if t.Cut.Frame < 1 || t.Cut.Tick < 0 || t.Cut.Instance < 1 || t.Cut.Epoch < 0 || t.Cut.Round < 1 ||
			(t.Cut.Phase != "playing" && t.Cut.Phase != "freezetime" && t.Cut.Phase != "postround") {
			return errors.New("target cut is incomplete or invalid")
		}
		if t.Cut.Frame < previousFrame {
			return errors.New("targets must be ordered by nondecreasing frame")
		}
		previousFrame = t.Cut.Frame
		if before, exists := frames[t.Cut.Frame]; exists && before != t.Cut {
			return errors.New("conflicting metadata for a target frame")
		}
		frames[t.Cut.Frame] = t.Cut
		if len(t.Actors) == 0 || len(t.Actors) > 10 {
			return errors.New("target must explicitly retain one to ten actors")
		}
		aliases, handles := map[string]bool{}, map[uint64]bool{}
		for _, a := range t.Actors {
			if !opaquePattern.MatchString(a.Alias) || aliases[a.Alias] || (a.Side != "T" && a.Side != "CT") {
				return errors.New("actor alias or side invalid or duplicate")
			}
			aliases[a.Alias] = true
			if a.Handle != nil {
				if *a.Handle > math.MaxUint32 || *a.Handle == math.MaxUint32 || *a.Handle == 0 || handles[*a.Handle] {
					return errors.New("pawn handle invalid or duplicated within target")
				}
				handles[*a.Handle] = true
			}
			if a.Position != nil && !finite(a.Position) {
				return errors.New("nonfinite target position")
			}
			if a.Handle == nil && (a.Alive != nil || a.Position != nil) {
				return errors.New("known actor facts require a resolved pawn handle")
			}
		}
	}
	return nil
}

// A second token pass rejects duplicate object keys, which encoding/json would
// otherwise silently replace. Unknown fields and trailing documents also fail.
func rejectDuplicateKeys(d *json.Decoder) error {
	tok, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return errors.New("duplicate JSON key")
			}
			seen[s] = true
			if err = rejectDuplicateKeys(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := rejectDuplicateKeys(d); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}

func decodeConfig(raw []byte, expectedSHA string) (config, error) {
	var c config
	if err := rejectDuplicateKeys(json.NewDecoder(bytes.NewReader(raw))); err != nil {
		return c, err
	}
	if err := validateShape(raw); err != nil {
		return c, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("invalid target document: %w", err)
	}
	var next any
	if err := d.Decode(&next); err != io.EOF {
		return c, errors.New("trailing target document")
	}
	return c, c.validate(expectedSHA)
}

func objectShape(raw json.RawMessage, required []string, nullable map[string]bool) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return nil, errors.New("required JSON object absent")
	}
	for _, k := range required {
		v, ok := m[k]
		if !ok || (bytes.Equal(bytes.TrimSpace(v), []byte("null")) && !nullable[k]) {
			return nil, fmt.Errorf("required non-null field missing: %s", k)
		}
	}
	return m, nil
}

func validateShape(raw []byte) error {
	m, err := objectShape(raw, []string{"schema_id", "source_sha256", "source_alias", "map_name", "targets"}, nil)
	if err != nil {
		return err
	}
	var targets []json.RawMessage
	if err = json.Unmarshal(m["targets"], &targets); err != nil {
		return err
	}
	for _, rawTarget := range targets {
		t, err := objectShape(rawTarget, []string{"target_id", "candidate_context_sha256", "cut", "actors"}, nil)
		if err != nil {
			return err
		}
		if _, err = objectShape(t["cut"], []string{"frame", "tick", "instance", "epoch", "round_number", "phase"}, nil); err != nil {
			return err
		}
		var actors []json.RawMessage
		if err = json.Unmarshal(t["actors"], &actors); err != nil {
			return err
		}
		for _, rawActor := range actors {
			a, err := objectShape(rawActor, []string{"alias", "handle", "side", "connected", "alive", "position"}, map[string]bool{"handle": true, "connected": true, "alive": true, "position": true})
			if err != nil {
				return err
			}
			if !bytes.Equal(bytes.TrimSpace(a["position"]), []byte("null")) {
				if _, err = objectShape(a["position"], []string{"x", "y", "z"}, nil); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
