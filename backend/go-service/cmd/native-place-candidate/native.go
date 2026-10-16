package main

import (
	"fmt"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type scalar struct {
	Status string `json:"status"`
	Type   string `json:"native_type,omitempty"`
	Value  any    `json:"value"`
}

func stringValue(raw any) scalar {
	if raw == nil {
		return scalar{Status: "unset"}
	}
	s := scalar{Status: "unsupported_type", Type: fmt.Sprintf("%T", raw)}
	if v, ok := raw.(string); ok {
		s.Status, s.Value = "observed", v
		if v == "" {
			s.Status = "empty_string"
		}
	}
	return s
}

func property(e st.Entity, name string) (p st.Property, status string) {
	status = "absent_property"
	defer func() {
		if recover() != nil {
			p, status = nil, "lookup_error"
		}
	}()
	if e == nil {
		return nil, "entity_unavailable"
	}
	p = e.Property(name)
	if p != nil {
		status = "present_property"
	}
	return
}

func read(e st.Entity, name string) (s scalar) {
	defer func() {
		if recover() != nil {
			s = scalar{Status: "read_error"}
		}
	}()
	p, status := property(e, name)
	if p == nil {
		return scalar{Status: status}
	}
	return stringValue(p.Value().Any)
}

func unsigned(raw any) (uint64, bool) {
	var n uint64
	switch v := raw.(type) {
	case uint64:
		n = v
	case uint32:
		n = uint64(v)
	case int:
		if v < 0 {
			return 0, false
		}
		n = uint64(v)
	case int32:
		if v < 0 {
			return 0, false
		}
		n = uint64(v)
	case int64:
		if v < 0 {
			return 0, false
		}
		n = uint64(v)
	default:
		return 0, false
	}
	return n, n <= math.MaxUint32
}

func handleMatches(h uint64, index, serial int) bool {
	return h != 0 && h != math.MaxUint32 && h <= math.MaxUint32 && index >= 0 && serial >= 0 &&
		h&16383 == uint64(index) && h>>14 == uint64(serial)
}

func nativeInteger(e st.Entity, name string) (n *uint64) {
	defer func() {
		if recover() != nil {
			n = nil
		}
	}()
	p, _ := property(e, name)
	if p == nil {
		return nil
	}
	v, ok := unsigned(p.Value().Any)
	if !ok {
		return nil
	}
	return &v
}

func finitePosition(p *common.Player, pawn st.Entity) (pos *position) {
	defer func() {
		if recover() != nil {
			pos = nil
		}
	}()
	for _, name := range []string{"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ", "CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ"} {
		v, _ := property(pawn, name)
		if v == nil || v.Value().Any == nil {
			return nil
		}
	}
	v := p.Position()
	pos = &position{v.X, v.Y, v.Z}
	if !finite(pos) {
		return nil
	}
	return pos
}

type observation struct {
	Handle          *uint64   `json:"pawn_handle"`
	EntityID        *int      `json:"entity_id"`
	Serial          *int      `json:"serial"`
	Availability    string    `json:"availability"`
	Connected       *bool     `json:"connected"`
	Side            *string   `json:"side"`
	Alive           *bool     `json:"alive"`
	Health          *uint64   `json:"health"`
	LifeState       *uint64   `json:"life_state"`
	RawPosition     *position `json:"raw_position"`
	RoundedPosition *position `json:"rounded_xy_position"`
	Place           scalar    `json:"place"`
	SDKPlace        *string   `json:"sdk_last_place_name"`
	RawAPIEqual     *bool     `json:"raw_api_equal"`
}

func samplePlayer(p *common.Player, h uint64) (o observation) {
	o = observation{Handle: &h, Availability: "actor_absent", Place: scalar{Status: "entity_unavailable"}}
	defer func() {
		if recover() != nil {
			o.Availability = "sample_error"
			o.Place = scalar{Status: "read_error"}
		}
	}()
	if p == nil || p.Entity == nil {
		return
	}
	actual := nativeInteger(p.Entity, "m_hPlayerPawn")
	if actual == nil || *actual != h {
		o.Availability = "identity_mismatch"
		return
	}
	pawn := p.PlayerPawnEntity()
	if pawn == nil || !handleMatches(h, pawn.ID(), pawn.SerialNum()) {
		o.Availability = "pawn_reference_unresolved"
		return
	}
	index, serial, connected := pawn.ID(), pawn.SerialNum(), p.IsConnected
	o.EntityID, o.Serial, o.Connected = &index, &serial, &connected
	o.Availability = "observed"
	team := nativeInteger(pawn, "m_iTeamNum")
	if team != nil {
		var side string
		if *team == 2 {
			side = "T"
		} else if *team == 3 {
			side = "CT"
		}
		if side != "" {
			o.Side = &side
		}
	}
	o.Health, o.LifeState = nativeInteger(pawn, "m_iHealth"), nativeInteger(pawn, "m_lifeState")
	if o.Health != nil && o.LifeState != nil {
		alive := *o.Health > 0 || *o.LifeState == 0
		o.Alive = &alive
	}
	o.RawPosition = finitePosition(p, pawn)
	if o.RawPosition != nil {
		o.RoundedPosition = &position{math.Round(o.RawPosition.X), math.Round(o.RawPosition.Y), o.RawPosition.Z}
	}
	o.Place = read(pawn, placeField)
	if o.Place.Status == "observed" || o.Place.Status == "empty_string" {
		api := p.LastPlaceName()
		equal := api == o.Place.Value.(string)
		o.SDKPlace, o.RawAPIEqual = &api, &equal
		if !equal {
			o.Availability = "raw_api_mismatch"
		}
	}
	return
}

func validateKnownFacts(a actorTarget, o observation) error {
	if a.Handle == nil {
		return nil
	}
	if o.Handle == nil || *o.Handle != *a.Handle {
		return fmt.Errorf("target pawn identity differs")
	}
	if o.Availability == "observed" {
		if o.EntityID == nil || o.Serial == nil || !handleMatches(*a.Handle, *o.EntityID, *o.Serial) {
			return fmt.Errorf("target pawn serial differs")
		}
		if o.Side == nil || *o.Side != a.Side {
			return fmt.Errorf("target side differs or is unavailable")
		}
	}
	if a.Alive != nil && (o.Alive == nil || *o.Alive != *a.Alive) {
		return fmt.Errorf("target life differs or is unavailable")
	}
	if a.Connected != nil && (o.Connected == nil || *o.Connected != *a.Connected) {
		return fmt.Errorf("target connection differs or is unavailable")
	}
	if a.Position != nil && (o.RoundedPosition == nil || *o.RoundedPosition != *a.Position) {
		return fmt.Errorf("target position differs or is unavailable")
	}
	if o.Availability == "identity_mismatch" || o.Availability == "raw_api_mismatch" {
		return fmt.Errorf("native identity or API conflict")
	}
	return nil
}
