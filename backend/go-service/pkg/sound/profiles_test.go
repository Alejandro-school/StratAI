package sound

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/golang/geo/r3"
	"math"
	"testing"
)

func TestPinnedGameSoundRadiiAndBroadcastOverride(t *testing.T) {
	for name, want := range map[string]float64{"Base.Footstep": 1100, "Base.Land": 1100, "Weapon_AK47.Clipout": 1100,
		"Weapon_AWP.Zoom": 597.142883, "C4.PlantSound": 1300, "Weapon_AK47.Single": 9000} {
		p, ok := LookupProfile(name)
		if !ok || p.Radius != want || !p.Usable {
			t.Fatalf("%s: %+v, %v; want %v", name, p, ok, want)
		}
	}
	if p, _ := LookupProfile("Weapon_AK47.Single"); p.RadiusBasis != "broadcast_distance_override" {
		t.Fatal("gunshot override lost")
	}
	if p, _ := LookupProfile("HEGrenade.PullPin_Grenade"); p.Usable {
		t.Fatal("local-only pin sound granted enemy information")
	}
	if _, ok := LookupProfile("not-a-native-event"); ok {
		t.Fatal("unknown sound received a default radius")
	}
}

func TestSpatialRangeUsesHeightAllDirectionsAndStrictBoundary(t *testing.T) {
	for _, p := range []r3.Vector{{X: 99}, {X: -99}, {Y: 99}, {Y: -99}, {Z: 99}, {Z: -99}} {
		if !WithinRange(r3.Vector{}, p, 100) {
			t.Fatalf("direction rejected: %+v", p)
		}
	}
	if WithinRange(r3.Vector{}, r3.Vector{X: 60, Y: 80, Z: 1}, 100) || WithinRange(r3.Vector{}, r3.Vector{X: 100}, 100) {
		t.Fatal("height ignored or zero-gain endpoint included")
	}
	if WithinRange(r3.Vector{Z: math.NaN()}, r3.Vector{}, 100) || WithinRange(r3.Vector{}, r3.Vector{}, 0) {
		t.Fatal("invalid geometry accepted")
	}
}

func TestSilentFootstepDefinitionsDoNotGrantRemoteReception(t *testing.T) {
	for _, name := range []string{"CT_Silent.StepLeft", "T_Silent.StepLeft", "Land_Silent.StepLeft"} {
		p, ok := LookupProfile(name)
		if !ok || p.Usable {
			t.Fatalf("silent definition became audible: %s %+v", name, p)
		}
	}
}

func TestPackedSoundParametersFollowPublishedTypedLengths(t *testing.T) {
	// Researcher's example: public.volume, float32, 4-byte payload, padding 0.
	data := []byte{0xe9, 0x54, 0x60, 0xbd, 8, 4, 0, 0, 0, 0, 0x3f}
	p, err := DecodeParameters(data)
	if err != nil || p.Volume == nil || *p.Volume != 0.5 {
		t.Fatalf("float sample differs: %+v, %v", p, err)
	}
	for _, bad := range [][]byte{data[:10], append(append([]byte{}, data...), 1), {1, 2, 3, 4, 8, 4, 0, 0, 0, 0, 0}} {
		if _, err := DecodeParameters(bad); err == nil {
			t.Fatal("malformed or unsupported parameters accepted")
		}
	}
	bits := math.Float32bits(float32(math.Inf(1)))
	bad := append([]byte{}, data...)
	binary.LittleEndian.PutUint32(bad[7:], bits)
	if _, err := DecodeParameters(bad); err == nil {
		t.Fatal("nonfinite native parameter accepted")
	}
}

func TestRealFootstepParametersIncludeSourceSoundscapeNotAnUnknownRadius(t *testing.T) {
	// Same approved demo: position followed by a native source_soundscape int.
	raw, _ := hex.DecodeString("4dce7c5a0a0c00197cc8c45d6cfdc4857c83c341e11bd9020400c4e73251")
	p, err := DecodeParameters(raw)
	if err != nil || p.Position == nil {
		t.Fatalf("real footstep rejected: %v", err)
	}
	if math.Abs(p.Position.X-(-1603.8780517578125)) > 1e-6 {
		t.Fatalf("position decoded incorrectly: %+v", p.Position)
	}
}

func TestSoundHashDoesNotUseGenericStringTokenSeed(t *testing.T) {
	if ParameterHash("public.volume") != 0xbd6054e9 {
		t.Fatal("published parameter hash differs")
	}
	if SoundNameHash("Weapon_AWP.Zoom") != SoundNameHash("weapon_awp.zoom") {
		t.Fatal("sound names must ignore ASCII case")
	}
	if SoundNameHash("public.volume") == ParameterHash("public.volume") {
		t.Fatal("sound and parameter seeds were conflated")
	}
}
