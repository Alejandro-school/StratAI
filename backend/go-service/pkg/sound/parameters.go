package sound

import (
	"encoding/binary"
	"errors"
	"github.com/golang/geo/r3"
	"math"
)

// Separate hashes: sound event names use SRC2; parameter names use the string
// token seed. Primary reverse-engineering report and native fixture examples:
// https://bbs.csgocn.net/thread-1026.htm (samyyc, 2025-06-08).
func nameHash(name string, seed uint32) uint32 {
	raw := []byte(name)
	for i, b := range raw {
		if b >= 'A' && b <= 'Z' {
			raw[i] = b + ('a' - 'A')
		}
	}
	h := seed ^ uint32(len(raw))
	const m uint32 = 0x5bd1e995
	for len(raw) >= 4 {
		k := binary.LittleEndian.Uint32(raw)
		k *= m
		k ^= k >> 24
		k *= m
		h = h*m ^ k
		raw = raw[4:]
	}
	for i, b := range raw {
		h ^= uint32(b) << uint(8*i)
	}
	if len(raw) > 0 {
		h *= m
	}
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}
func SoundNameHash(name string) uint32 { return nameHash(name, 0x53524332) }
func ParameterHash(name string) uint32 { return nameHash(name, 0x31415926) }

type Parameters struct {
	Position         *r3.Vector
	Volume           *float64
	Pitch            *float64
	SourceSoundscape *int32
	ImpactSpeed      *float64
}

// Decode only the published float32/float3 cases with exact typed lengths.
// Unknown keys/types, duplicate controls, truncation and nonfinite data abstain;
// bytes are never scanned heuristically for plausible coordinates or radii.
func DecodeParameters(raw []byte) (Parameters, error) {
	p := Parameters{}
	seen := map[uint32]bool{}
	for len(raw) > 0 {
		if len(raw) < 7 {
			return Parameters{}, errors.New("truncated sound parameter header")
		}
		key := binary.LittleEndian.Uint32(raw)
		kind := raw[4]
		size := int(binary.LittleEndian.Uint16(raw[5:7]))
		raw = raw[7:]
		if len(raw) < size || seen[key] {
			return Parameters{}, errors.New("truncated or duplicate sound parameter")
		}
		seen[key] = true
		data := raw[:size]
		raw = raw[size:]
		get := func(offset int) float64 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])))
		}
		switch key {
		case ParameterHash("public.position"):
			if kind != 0x0a || size != 12 {
				return Parameters{}, errors.New("unsupported sound position type")
			}
			pos := r3.Vector{X: get(0), Y: get(4), Z: get(8)}
			if !finite(pos.X) || !finite(pos.Y) || !finite(pos.Z) {
				return Parameters{}, errors.New("nonfinite sound position")
			}
			p.Position = &pos
		case ParameterHash("public.source_soundscape"):
			// Pinned native soundstack declares this as int. It is not volume,
			// radius, entity identity or evidence that a player heard the sound.
			if kind != 2 || size != 4 {
				return Parameters{}, errors.New("unsupported soundscape type")
			}
			value := int32(binary.LittleEndian.Uint32(data))
			p.SourceSoundscape = &value
		case ParameterHash("public.volume"), ParameterHash("public.pitch"), ParameterHash("public.impact_speed_input"):
			if kind != 8 || size != 4 {
				return Parameters{}, errors.New("unsupported scalar sound type")
			}
			value := get(0)
			if !finite(value) || value < 0 {
				return Parameters{}, errors.New("invalid sound scalar")
			}
			if key == ParameterHash("public.volume") {
				p.Volume = &value
			} else if key == ParameterHash("public.pitch") {
				p.Pitch = &value
			} else {
				p.ImpactSpeed = &value
			}
		default:
			return Parameters{}, errors.New("unsupported native sound parameter")
		}
	}
	return p, nil
}
