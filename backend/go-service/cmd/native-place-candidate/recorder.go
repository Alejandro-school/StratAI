package main

import (
	"encoding/json"
	"fmt"
	"sort"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type delivery struct {
	Sequence int    `json:"sequence"`
	Cut      cut    `json:"cut"`
	Handle   uint64 `json:"pawn_handle"`
	Value    scalar `json:"value"`
}
type pendingDelivery struct {
	handle uint64
	value  scalar
}
type trackedState struct {
	signature string
	first     *cut
	changed   *cut
	delivery  *delivery
}
type scope struct {
	limit   cut
	handles []uint64
	states  map[uint64]*trackedState
}

type actorSnapshot struct {
	Alias string `json:"alias"`
	observation
	LastDelivery  *delivery `json:"last_native_delivery"`
	FirstObserved *cut      `json:"first_observed_in_instance"`
	LastChange    *cut      `json:"last_observed_state_change"`
}
type snapshot struct {
	Schema     string          `json:"schema_id"`
	TargetID   string          `json:"target_id"`
	ContextSHA string          `json:"candidate_context_sha256"`
	Cut        cut             `json:"cut"`
	SDKFrame   int             `json:"sdk_frame"`
	Actors     []actorSnapshot `json:"actors"`
}

type recorder struct {
	p                                                            dem.Parser
	cfg                                                          config
	frame, instance, round, epoch, lastTick                      int
	hasTick                                                      bool
	phase                                                        string
	err                                                          error
	header                                                       map[string]any
	snapshots, deliveries, changes, framesSampled, actorsSampled int
	outputSnapshots, outputDeliveries, outputChanges             *json.Encoder
	scopes                                                       map[string]*scope
	targetFrames                                                 map[int][]target
	requestedHandles                                             map[uint64]bool
	pending                                                      []pendingDelivery
}

func scopeKey(c cut) string { return fmt.Sprintf("%d/%d/%d", c.Epoch, c.Instance, c.Round) }
func (r *recorder) now() cut {
	return cut{r.frame, r.p.GameState().IngameTick(), r.instance, r.epoch, r.round, r.phase}
}

func (r *recorder) configure() {
	r.scopes, r.targetFrames, r.requestedHandles = map[string]*scope{}, map[int][]target{}, map[uint64]bool{}
	for _, t := range r.cfg.Targets {
		r.targetFrames[t.Cut.Frame] = append(r.targetFrames[t.Cut.Frame], t)
		key := scopeKey(t.Cut)
		s := r.scopes[key]
		if s == nil {
			s = &scope{limit: t.Cut, states: map[uint64]*trackedState{}}
			r.scopes[key] = s
		}
		if t.Cut.Frame > s.limit.Frame {
			s.limit = t.Cut
		}
		for _, a := range t.Actors {
			if a.Handle == nil {
				continue
			}
			h := *a.Handle
			r.requestedHandles[h] = true
			if s.states[h] == nil {
				s.states[h] = &trackedState{}
				s.handles = append(s.handles, h)
			}
		}
	}
	for _, s := range r.scopes {
		sort.Slice(s.handles, func(i, j int) bool { return s.handles[i] < s.handles[j] })
	}
}

func (r *recorder) attach(e st.Entity) {
	if e.ID() < 0 || e.SerialNum() < 0 {
		return
	}
	h := uint64(e.SerialNum())<<14 | uint64(e.ID())
	if !handleMatches(h, e.ID(), e.SerialNum()) || !r.requestedHandles[h] {
		return
	}
	p, _ := property(e, placeField)
	if p == nil {
		return
	}
	registering := true
	p.OnUpdate(func(v st.PropertyValue) {
		// OnUpdate synchronously invokes the handler with its current value.
		// That registration callback is not a newly delivered property update.
		if registering {
			return
		}
		// SDK entity callbacks run inside a frame. Resolve its final clock only
		// at FrameDone, after round/epoch events, never from a future frame.
		r.pending = append(r.pending, pendingDelivery{h, stringValue(v.Any)})
	})
	registering = false
}

func playersByRequestedHandle(players []*common.Player, handles []uint64) (map[uint64]*common.Player, error) {
	wanted := map[uint64]bool{}
	for _, h := range handles {
		wanted[h] = true
	}
	result := map[uint64]*common.Player{}
	for _, p := range players {
		if p == nil || p.Entity == nil {
			continue
		}
		h := nativeInteger(p.Entity, "m_hPlayerPawn")
		if h == nil || !wanted[*h] {
			continue
		}
		if previous := result[*h]; previous != nil && previous != p {
			return nil, fmt.Errorf("ambiguous pawn handle across controllers")
		}
		result[*h] = p
	}
	return result, nil
}

func (r *recorder) encode(enc *json.Encoder, value any) {
	if r.err == nil {
		r.err = enc.Encode(value)
	}
}

func (r *recorder) frameDone() {
	r.frame++
	tick := r.p.GameState().IngameTick()
	if r.hasTick && tick < r.lastTick {
		r.epoch++
	}
	r.lastTick, r.hasTick = tick, true
	limit := r.cfg.Targets[len(r.cfg.Targets)-1].Cut
	if r.frame > limit.Frame {
		r.err = fmt.Errorf("future frame processed")
		return
	}
	now := r.now()
	s := r.scopes[scopeKey(now)]
	pending := r.pending
	r.pending = nil
	if s == nil || !within(now, s.limit) {
		if len(r.targetFrames[r.frame]) != 0 {
			r.err = fmt.Errorf("target instance, epoch or round differs")
		}
		return
	}
	for _, item := range pending {
		state := s.states[item.handle]
		if state == nil {
			continue
		}
		r.deliveries++
		d := &delivery{r.deliveries, now, item.handle, item.value}
		state.delivery = d
		r.encode(r.outputDeliveries, d)
	}
	matches, err := playersByRequestedHandle(r.p.GameState().Participants().All(), s.handles)
	if err != nil {
		r.err = err
		return
	}
	observations := map[uint64]observation{}
	r.framesSampled++
	for _, h := range s.handles {
		o := samplePlayer(matches[h], h)
		r.actorsSampled++
		observations[h] = o
		state := s.states[h]
		if o.Availability == "observed" && state.first == nil {
			c := now
			state.first = &c
		}
		sig, err := json.Marshal([]any{o.Availability, o.Place, o.Alive, o.Side, o.Connected})
		if err != nil {
			r.err = err
			return
		}
		if string(sig) != state.signature {
			state.signature = string(sig)
			c := now
			state.changed = &c
			r.changes++
			r.encode(r.outputChanges, struct {
				Sequence int `json:"sequence"`
				Cut      cut `json:"cut"`
				observation
			}{r.changes, now, o})
		}
	}
	for _, t := range r.targetFrames[r.frame] {
		if now != t.Cut || r.p.CurrentFrame() != r.frame {
			r.err = fmt.Errorf("exact target clock or phase differs")
			return
		}
		out := snapshot{Schema: "stratai.native_place_snapshot@1", TargetID: t.ID, ContextSHA: t.ContextSHA, Cut: now, SDKFrame: r.p.CurrentFrame(), Actors: []actorSnapshot{}}
		for _, a := range t.Actors {
			o := observation{Availability: "identity_unavailable", Place: scalar{Status: "entity_unavailable"}}
			var state *trackedState
			if a.Handle != nil {
				o, state = observations[*a.Handle], s.states[*a.Handle]
			}
			if err := validateKnownFacts(a, o); err != nil {
				r.err = fmt.Errorf("target %s actor %s: %w", t.ID, a.Alias, err)
				return
			}
			row := actorSnapshot{Alias: a.Alias, observation: o}
			if state != nil {
				row.FirstObserved, row.LastChange, row.LastDelivery = state.first, state.changed, state.delivery
			}
			if row.LastDelivery != nil && !within(row.LastDelivery.Cut, now) {
				r.err = fmt.Errorf("future or cross-instance property delivery")
				return
			}
			out.Actors = append(out.Actors, row)
		}
		r.encode(r.outputSnapshots, out)
		r.snapshots++
	}
}

func (r *recorder) register() {
	r.configure()
	r.p.RegisterNetMessageHandler(func(m *msg.CDemoFileHeader) {
		r.header = map[string]any{"map_name": m.GetMapName(), "build": m.GetBuildNum(), "patch": m.GetPatchVersion()}
	})
	r.p.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, c := range r.p.ServerClasses().All() {
			if c.Name() == "CCSPlayerPawn" {
				c.OnEntityCreated(r.attach)
			}
		}
	})
	r.p.RegisterEventHandler(func(events.RoundStart) {
		g := r.p.GameState()
		if g.IsWarmupPeriod() {
			return
		}
		r.instance++
		r.round = g.TotalRoundsPlayed() + 1
		r.phase = "freezetime"
	})
	r.p.RegisterEventHandler(func(events.RoundFreezetimeEnd) { r.phase = "playing" })
	r.p.RegisterEventHandler(func(events.RoundEnd) { r.phase = "postround" })
	r.p.RegisterEventHandler(func(events.FrameDone) { r.frameDone() })
}
