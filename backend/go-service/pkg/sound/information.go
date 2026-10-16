package sound

import (
	"fmt"
	"github.com/golang/geo/r3"
)

const InformationSchema = "stratai.calculated_sound_information@1"
const InformationProducer = "go.replay.native_sound_profile_3d_v1"

// These native coordinates and emitter IDs are audit inputs. Team projections
// must strip them; sound never grants a recognized enemy identity or exact point.
type Candidate struct {
	Tick           int        `json:"tick"`
	Source         string     `json:"source"`
	Profile        string     `json:"profile"`
	Position       *r3.Vector `json:"native_position"`
	PositionSource string     `json:"position_source"`
	EmitterID      uint64     `json:"emitter_id,string"`
	EmitterSide    string     `json:"emitter_side"`
	EntityIndex    int        `json:"native_entity_index"`
	ParamsHex      string     `json:"native_params_hex,omitempty"`
	Volume         *float64   `json:"native_volume,omitempty"`
	NativeHash     *uint32    `json:"native_sound_hash,omitempty"`
	ItemDefinition *uint32    `json:"native_item_definition,omitempty"`
	SoundType      *int32     `json:"native_sound_type,omitempty"`
}
type Observer struct {
	ID       uint64    `json:"player_id,string"`
	Side     string    `json:"side"`
	Alive    bool      `json:"alive"`
	Position r3.Vector `json:"native_position"`
}
type Emission struct {
	Candidate
	ID               string     `json:"id"`
	AvailabilityTick int        `json:"availability_tick"`
	Family           string     `json:"family"`
	Radius           float64    `json:"radius_units"`
	RadiusBasis      string     `json:"radius_basis"`
	NativeObservers  []Observer `json:"native_observers"`
}
type Observation struct {
	ID               string  `json:"id"`
	EmissionID       string  `json:"emission_id"`
	ObserverID       uint64  `json:"observer_id,string"`
	ObserverSide     string  `json:"observer_side"`
	Tick             int     `json:"tick"`
	AvailabilityTick int     `json:"availability_tick"`
	Distance         float64 `json:"distance_3d_units"`
	Status           string  `json:"status"`
}
type Information struct {
	SchemaID      string         `json:"schema_id"`
	Producer      string         `json:"producer"`
	CatalogID     string         `json:"catalog_id"`
	CatalogSHA256 string         `json:"catalog_sha256"`
	Emissions     []Emission     `json:"emissions"`
	Observations  []Observation  `json:"observations"`
	Abstentions   map[string]int `json:"abstentions"`
	sequence      int
}

func NewInformation() *Information {
	return &Information{SchemaID: InformationSchema, Producer: InformationProducer, CatalogID: CatalogID(), CatalogSHA256: CatalogSHA256(),
		Emissions: []Emission{}, Observations: []Observation{}, Abstentions: map[string]int{}}
}
func (v *Information) Abstain(reason string) { v.Abstentions[reason]++ }

func (v *Information) Observe(c Candidate, observers []Observer) {
	p, ok := LookupProfile(c.Profile)
	if !ok || !p.Usable {
		v.Abstain("sound_profile_unavailable_or_not_broadcast")
		return
	}
	if c.Position == nil || !finite(c.Position.X) || !finite(c.Position.Y) || !finite(c.Position.Z) {
		v.Abstain("native_sound_position_unavailable")
		return
	}
	volume := p.Volume
	if c.Volume != nil {
		volume = *c.Volume
	}
	if !finite(volume) || volume <= 0 {
		v.Abstain("native_sound_volume_zero_or_invalid")
		return
	}
	if c.Tick < 0 {
		v.Abstain("native_callback_tick_unavailable")
		return
	}
	if c.EmitterSide != "T" && c.EmitterSide != "CT" {
		c.EmitterSide = "unknown"
	}
	v.sequence++
	id := fmt.Sprintf("sound-emission:%d", v.sequence)
	v.Emissions = append(v.Emissions, Emission{Candidate: c, ID: id, AvailabilityTick: c.Tick, Family: p.Family, Radius: p.Radius, RadiusBasis: p.RadiusBasis, NativeObservers: observers})
	seen := map[uint64]bool{}
	for _, observer := range observers {
		if observer.ID == 0 || observer.ID == c.EmitterID || seen[observer.ID] || !observer.Alive || (observer.Side != "T" && observer.Side != "CT") || observer.Side == c.EmitterSide {
			continue
		}
		seen[observer.ID] = true
		if !WithinRange(*c.Position, observer.Position, p.Radius) {
			continue
		}
		v.Observations = append(v.Observations, Observation{ID: fmt.Sprintf("sound-observation:%d", len(v.Observations)+1), EmissionID: id,
			ObserverID: observer.ID, ObserverSide: observer.Side, Tick: c.Tick, AvailabilityTick: c.Tick,
			Distance: c.Position.Sub(observer.Position).Norm(), Status: "estimated_possible_reception"})
	}
}

// Native callback order can precede a death callback in the same tick. The
// completed causal round ledger removes those direct observers conservatively.
func (v *Information) Reconcile(start, end int, deaths map[uint64]int) {
	emissions := v.Emissions[:0]
	valid := map[string]bool{}
	for _, e := range v.Emissions {
		if e.Tick >= start && e.Tick <= end {
			emissions = append(emissions, e)
			valid[e.ID] = true
		}
	}
	v.Emissions = emissions
	observations := v.Observations[:0]
	for _, o := range v.Observations {
		death, dead := deaths[o.ObserverID]
		if !valid[o.EmissionID] {
			v.Abstain("outside_round_callback")
			continue
		}
		if dead && death >= start && death <= o.Tick {
			v.Abstain("observer_death_at_or_before_sound_tick")
			continue
		}
		observations = append(observations, o)
	}
	v.Observations = observations
}
