package sound

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"github.com/golang/geo/r3"
	"strconv"
	"strings"
)

// Pinned, generated game data; the generator and all primary-source hashes are
// retained in outputs/stratai-sound-ranges-20261015. No universal radius fallback.
//
//go:embed cs2_profiles.json
var profileJSON []byte

type Profile struct {
	Event        string  `json:"event"`
	Hash         uint32  `json:"hash"`
	Family       string  `json:"family"`
	Radius       float64 `json:"radius_units"`
	RadiusBasis  string  `json:"radius_basis"`
	Usable       bool    `json:"usable"`
	LocalOnly    bool    `json:"local_only"`
	Silent       bool    `json:"silent"`
	Broadcast    bool    `json:"broadcast"`
	SourceFile   string  `json:"source_file"`
	SourceSHA256 string  `json:"source_sha256"`
	Volume       float64 `json:"volume"`
}

type profileCatalog struct {
	SchemaID   string                    `json:"schema_id"`
	ID         string                    `json:"catalog_id"`
	Commit     string                    `json:"game_source_commit"`
	Events     []Profile                 `json:"events"`
	FireSounds map[string]map[int]string `json:"fire_sounds_by_item_definition"`
}

var catalog profileCatalog
var profilesByName = map[string]Profile{}
var profilesByHash = map[uint32]Profile{}

func init() {
	if err := json.Unmarshal(profileJSON, &catalog); err != nil {
		panic(err)
	}
	if catalog.SchemaID != "stratai.cs2_sound_profiles@1" || len(catalog.Events) == 0 {
		panic("invalid sound profile catalog")
	}
	for _, profile := range catalog.Events {
		key := strings.ToLower(profile.Event)
		if _, duplicate := profilesByName[key]; duplicate {
			panic("duplicate sound profile")
		}
		if _, collision := profilesByHash[profile.Hash]; collision {
			panic("sound hash collision")
		}
		if profile.Hash != SoundNameHash(profile.Event) || !finite(profile.Radius) || profile.Radius < 0 || (profile.Usable && (profile.Radius <= 0 || profile.LocalOnly || profile.Silent || !profile.Broadcast)) {
			panic("invalid sound radius or hash")
		}
		profilesByName[key] = profile
		profilesByHash[profile.Hash] = profile
	}
}

func CatalogID() string     { return catalog.ID }
func CatalogSHA256() string { hash := sha256.Sum256(profileJSON); return hex.EncodeToString(hash[:]) }
func LookupProfile(name string) (Profile, bool) {
	p, ok := profilesByName[strings.ToLower(name)]
	return p, ok
}
func LookupHash(hash uint32) (Profile, bool) { p, ok := profilesByHash[hash]; return p, ok }
func FireProfile(definition uint32, kind int32) (Profile, bool) {
	// Item definitions and WeaponSound_t keys come from weapons.vdata and the
	// native enum, not from the player's current weapon or a silencer guess.
	name := catalog.FireSounds[strconv.FormatUint(uint64(definition), 10)][int(kind)]
	return LookupProfile(name)
}

// The endpoint has zero gain; it is excluded. Orientation, walls and smoke are
// deliberately not input parameters to this agreed geometric possibility model.
func WithinRange(source, observer r3.Vector, radius float64) bool {
	return finite(source.X) && finite(source.Y) && finite(source.Z) && finite(observer.X) && finite(observer.Y) && finite(observer.Z) && finite(radius) && radius > 0 && source.Sub(observer).Norm() < radius
}
