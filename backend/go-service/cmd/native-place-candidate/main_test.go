package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func ptr[T any](v T) *T { return &v }
func fixtureConfig() config {
	return config{Schema: targetSchema, SourceSHA: strings.Repeat("a", 64), SourceAlias: "demo-opaque", MapName: "de_test", Targets: []target{{
		ID: "target-opaque", ContextSHA: strings.Repeat("b", 64), Cut: cut{3, 103, 1, 0, 1, "playing"},
		Actors: []actorTarget{{Alias: "T1", Handle: ptr(uint64(16385)), Side: "T", Connected: ptr(true), Alive: ptr(true), Position: &position{1, 3, 4}}},
	}}}
}

func TestTargetContractRejectsForgedMetadata(t *testing.T) {
	good := fixtureConfig()
	raw, _ := json.Marshal(good)
	if _, err := decodeConfig(raw, good.SourceSHA); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*config){
		"source hash":  func(c *config) { c.SourceSHA = strings.Repeat("c", 64) },
		"context hash": func(c *config) { c.Targets[0].ContextSHA = "not-a-hash" },
		"future inconsistent frame": func(c *config) {
			c.Targets = append(c.Targets, c.Targets[0])
			c.Targets[1].ID = "second"
			c.Targets[1].Cut.Tick++
		},
		"duplicate ID": func(c *config) { c.Targets = append(c.Targets, c.Targets[0]) },
		"duplicate handle": func(c *config) {
			a := c.Targets[0].Actors[0]
			a.Alias = "T2"
			c.Targets[0].Actors = append(c.Targets[0].Actors, a)
		},
		"duplicate alias": func(c *config) {
			a := c.Targets[0].Actors[0]
			a.Handle = ptr(uint64(16386))
			c.Targets[0].Actors = append(c.Targets[0].Actors, a)
		},
		"invalid pawn handle":        func(c *config) { c.Targets[0].Actors[0].Handle = ptr(uint64(math.MaxUint32)) },
		"identity absent with facts": func(c *config) { c.Targets[0].Actors[0].Handle = nil },
		"negative tick":              func(c *config) { c.Targets[0].Cut.Tick = -1 },
		"zero instance":              func(c *config) { c.Targets[0].Cut.Instance = 0 },
		"unknown side":               func(c *config) { c.Targets[0].Actors[0].Side = "unknown" },
		"missing phase":              func(c *config) { c.Targets[0].Cut.Phase = "" },
		"nonfinite position":         func(c *config) { c.Targets[0].Actors[0].Position.X = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			c := fixtureConfig()
			mutate(&c)
			if c.validate(good.SourceSHA) == nil {
				t.Fatal("forged metadata accepted")
			}
		})
	}
	for name, altered := range map[string]string{
		"duplicate key":        strings.Replace(string(raw), `"epoch":0`, `"epoch":1,"epoch":0`, 1),
		"unknown field":        strings.Replace(string(raw), `"phase":"playing"`, `"phase":"playing","tactical_label":"A"`, 1),
		"trailing JSON":        string(raw) + "{}",
		"missing coordinate":   strings.Replace(string(raw), `"x":1,`, "", 1),
		"null coordinate":      strings.Replace(string(raw), `"x":1`, `"x":null`, 1),
		"null epoch":           strings.Replace(string(raw), `"epoch":0`, `"epoch":null`, 1),
		"missing availability": strings.Replace(string(raw), `"alive":true,`, "", 1),
		"string handle":        strings.Replace(string(raw), `"handle":16385`, `"handle":"16385"`, 1),
		"fractional handle":    strings.Replace(string(raw), `"handle":16385`, `"handle":16385.5`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeConfig([]byte(altered), good.SourceSHA); err == nil {
				t.Fatal("malformed document accepted")
			}
		})
	}
}

func TestNativeStringAvailability(t *testing.T) {
	for _, c := range []struct {
		value  any
		status string
	}{{nil, "unset"}, {"", "empty_string"}, {"arbitrary native token", "observed"}, {false, "unsupported_type"}, {123, "unsupported_type"}} {
		if got := stringValue(c.value); got.Status != c.status {
			t.Fatalf("%v: %s", c.value, got.Status)
		}
	}
	for _, c := range []struct {
		entity st.Entity
		status string
	}{{nil, "entity_unavailable"}, {&fakeEntity{}, "absent_property"}, {&fakeEntity{lookupPanic: true}, "lookup_error"}, {&fakeEntity{props: map[string]*fakeProperty{placeField: {panicRead: true}}}, "read_error"}} {
		if got := read(c.entity, placeField); got.Status != c.status {
			t.Fatalf("status %s, wanted %s", got.Status, c.status)
		}
	}
}

func TestHandleTypesAndSerialBinding(t *testing.T) {
	for _, raw := range []any{uint64(16385), uint32(16385), int(16385), int32(16385), int64(16385)} {
		h, ok := unsigned(raw)
		if !ok || !handleMatches(h, 1, 1) || handleMatches(h, 1, 2) || handleMatches(h, 2, 1) {
			t.Fatal("identity or serial binding invalid")
		}
	}
	for _, raw := range []any{-1, int64(-2), uint64(math.MaxUint32) + 1, "16385", 1.0, nil, true} {
		if _, ok := unsigned(raw); ok {
			t.Fatal("non-native integer accepted")
		}
	}
	if handleMatches(math.MaxUint32, 16383, 262143) || handleMatches(0, 0, 0) {
		t.Fatal("invalid source handle accepted")
	}
}

func TestCausalBoundary(t *testing.T) {
	limit := fixtureConfig().Targets[0].Cut
	if !within(limit, limit) {
		t.Fatal("exact cut rejected")
	}
	for _, field := range []string{"frame", "tick", "instance", "epoch", "round"} {
		c := limit
		switch field {
		case "frame":
			c.Frame++
		case "tick":
			c.Tick++
		case "instance":
			c.Instance--
		case "epoch":
			c.Epoch++
		case "round":
			c.Round++
		}
		if within(c, limit) {
			t.Fatalf("%s crosses boundary", field)
		}
	}
}

type fakeProperty struct {
	st.Property
	raw       any
	panicRead bool
	handler   st.PropertyUpdateHandler
}

func (p *fakeProperty) Value() st.PropertyValue {
	if p.panicRead {
		panic("read")
	}
	return st.PropertyValue{Any: p.raw}
}
func (p *fakeProperty) OnUpdate(h st.PropertyUpdateHandler) { p.handler = h; h(p.Value()) }

type fakeEntity struct {
	st.Entity
	index, serial int
	props         map[string]*fakeProperty
	lookupPanic   bool
	pos           r3.Vector
}

func (e *fakeEntity) ID() int        { return e.index }
func (e *fakeEntity) SerialNum() int { return e.serial }
func (e *fakeEntity) Property(n string) st.Property {
	if e.lookupPanic {
		panic("lookup")
	}
	p := e.props[n]
	if p == nil {
		return nil
	}
	return p
}
func (e *fakeEntity) PropertyValue(n string) (st.PropertyValue, bool) {
	p := e.props[n]
	if p == nil {
		return st.PropertyValue{}, false
	}
	return p.Value(), true
}
func (e *fakeEntity) PropertyValueMust(n string) st.PropertyValue {
	v, ok := e.PropertyValue(n)
	if !ok {
		panic("absent")
	}
	return v
}
func (e *fakeEntity) Position() r3.Vector { return e.pos }

type fakeProvider struct{ entity st.Entity }

func (d fakeProvider) IngameTick() int                              { return 1 }
func (d fakeProvider) TickRate() float64                            { return 64 }
func (d fakeProvider) FindPlayerByHandle(uint64) *common.Player     { return nil }
func (d fakeProvider) FindPlayerByPawnHandle(uint64) *common.Player { return nil }
func (d fakeProvider) FindWeaponByEntityID(int) *common.Equipment   { return nil }
func (d fakeProvider) FindEntityByHandle(uint64) st.Entity          { return d.entity }
func playerFixture() (*common.Player, *fakeEntity) {
	pawn := &fakeEntity{index: 1, serial: 1, pos: r3.Vector{X: 1.4, Y: 2.6, Z: 4}, props: map[string]*fakeProperty{placeField: {raw: "ArbitraryPlace"}, "m_iTeamNum": {raw: uint64(2)}, "m_iHealth": {raw: int32(100)}, "m_lifeState": {raw: uint64(0)}}}
	for _, n := range []string{"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ", "CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ"} {
		pawn.props[n] = &fakeProperty{raw: uint64(0)}
	}
	p := common.NewPlayer(fakeProvider{pawn})
	p.Entity = &fakeEntity{props: map[string]*fakeProperty{"m_hPawn": {raw: uint64(16385)}, "m_hPlayerPawn": {raw: uint64(16385)}}}
	p.Team = common.TeamTerrorists
	p.IsConnected = true
	return p, pawn
}

func TestNativeSamplingPreservesDeadAndUnknown(t *testing.T) {
	p, pawn := playerFixture()
	target := fixtureConfig().Targets[0].Actors[0]
	o := samplePlayer(p, 16385)
	if o.Availability != "observed" || o.RawAPIEqual == nil || !*o.RawAPIEqual || validateKnownFacts(target, o) != nil {
		t.Fatalf("valid sample rejected: %+v", o)
	}
	pawn.props["m_iHealth"].raw = int32(0)
	pawn.props["m_lifeState"].raw = uint64(2)
	o = samplePlayer(p, 16385)
	if o.Alive == nil || *o.Alive || o.Place.Value != "ArbitraryPlace" {
		t.Fatal("dead state or raw native string lost")
	}
	if validateKnownFacts(target, o) == nil {
		t.Fatal("known life mismatch accepted")
	}
	target.Alive = ptr(false)
	if validateKnownFacts(target, o) != nil {
		t.Fatal("dead actor wrongly rejected")
	}
	delete(pawn.props, "m_lifeState")
	o = samplePlayer(p, 16385)
	if o.Alive != nil {
		t.Fatal("missing life data fabricated")
	}
	pawn.serial = 2
	o = samplePlayer(p, 16385)
	if o.Availability != "pawn_reference_unresolved" || o.Place.Value != nil {
		t.Fatal("stale identity leaked native place")
	}
	if o = samplePlayer(nil, 16385); o.Availability != "actor_absent" || o.Alive != nil {
		t.Fatal("absent actor inferred dead")
	}
}

func TestPropertyRegistrationIsNotNativeDelivery(t *testing.T) {
	_, pawn := playerFixture()
	r := &recorder{requestedHandles: map[uint64]bool{16385: true}}
	r.attach(pawn)
	if len(r.pending) != 0 {
		t.Fatal("registration callback mislabelled as a new native delivery")
	}
	p := pawn.props[placeField]
	p.raw = "AnotherRawValue"
	p.handler(p.Value())
	if len(r.pending) != 1 || r.pending[0].value.Value != "AnotherRawValue" {
		t.Fatal("native update absent")
	}
}

func TestKnownPositionSideConnectionAndSerialMismatch(t *testing.T) {
	p, _ := playerFixture()
	baseline := samplePlayer(p, 16385)
	target := fixtureConfig().Targets[0].Actors[0]
	for name, mutate := range map[string]func(*observation){
		"position":   func(o *observation) { o.RoundedPosition = &position{2, 3, 4} },
		"side":       func(o *observation) { o.Side = ptr("CT") },
		"connected":  func(o *observation) { o.Connected = ptr(false) },
		"serial":     func(o *observation) { o.Serial = ptr(2) },
		"identity":   func(o *observation) { o.Handle = ptr(uint64(32769)) },
		"native API": func(o *observation) { o.Availability = "raw_api_mismatch" },
	} {
		t.Run(name, func(t *testing.T) {
			o := baseline
			mutate(&o)
			if validateKnownFacts(target, o) == nil {
				t.Fatal("forged fact accepted")
			}
		})
	}
}

type fakeParticipants struct {
	dem.Participants
	players []*common.Player
}

func (p fakeParticipants) All() []*common.Player { return p.players }

type fakeState struct {
	dem.GameState
	tick         int
	participants fakeParticipants
}

func (g *fakeState) IngameTick() int                { return g.tick }
func (g *fakeState) Participants() dem.Participants { return g.participants }

type fakeParser struct {
	dem.Parser
	state *fakeState
	frame int
}

func (p *fakeParser) GameState() dem.GameState { return p.state }
func (p *fakeParser) CurrentFrame() int        { return p.frame }

func recorderFixture() (*recorder, *fakeParser, *bytes.Buffer, *bytes.Buffer) {
	p, _ := playerFixture()
	parser := &fakeParser{state: &fakeState{participants: fakeParticipants{players: []*common.Player{p}}}}
	var snapshots, deliveries, changes bytes.Buffer
	r := &recorder{p: parser, cfg: fixtureConfig(), instance: 1, round: 1, phase: "playing", outputSnapshots: json.NewEncoder(&snapshots), outputDeliveries: json.NewEncoder(&deliveries), outputChanges: json.NewEncoder(&changes)}
	r.configure()
	return r, parser, &snapshots, &deliveries
}
func TestMultipleExactTargetsAndCompletedFrameDelivery(t *testing.T) {
	r, p, snapshots, deliveries := recorderFixture()
	early := r.cfg.Targets[0]
	early.ID = "early"
	early.Cut.Frame = 1
	early.Cut.Tick = 101
	unknown := actorTarget{Alias: "T2", Side: "T"}
	early.Actors = append(early.Actors, unknown)
	r.cfg.Targets = append([]target{early}, r.cfg.Targets...)
	r.configure()
	for i := 1; i <= 3; i++ {
		p.frame = i
		p.state.tick = 100 + i
		r.pending = append(r.pending, pendingDelivery{16385, stringValue("ArbitraryPlace")})
		r.frameDone()
		if r.err != nil {
			t.Fatal(r.err)
		}
	}
	if r.snapshots != 2 || r.deliveries != 3 {
		t.Fatal("missing targets or deliveries")
	}
	var first snapshot
	if err := json.NewDecoder(bytes.NewReader(snapshots.Bytes())).Decode(&first); err != nil {
		t.Fatal(err)
	}
	if first.Cut.Frame != 1 || first.Actors[0].LastDelivery.Cut.Frame != 1 || first.Actors[1].Availability != "identity_unavailable" || first.Actors[1].Handle != nil {
		t.Fatal("future/absent identity contaminated snapshot")
	}
	var d delivery
	if err := json.NewDecoder(deliveries).Decode(&d); err != nil {
		t.Fatal(err)
	}
	if d.Cut.Frame != 1 || d.Cut.Tick != 101 {
		t.Fatal("delivery assigned before final frame clock")
	}
	p.frame = 4
	p.state.tick = 104
	r.frameDone()
	if r.err == nil {
		t.Fatal("future frame accepted")
	}
}
func TestExactCutRejectsTickAndEpochMismatch(t *testing.T) {
	for _, wrongEpoch := range []bool{false, true} {
		r, p, _, _ := recorderFixture()
		r.frame = 2
		p.frame = 3
		p.state.tick = 104
		if wrongEpoch {
			r.epoch = 1
			p.state.tick = 103
		}
		r.frameDone()
		if r.err == nil {
			t.Fatal("mismatched exact cut accepted")
		}
	}
}

func TestDuplicateControllersFailClosed(t *testing.T) {
	p, _ := playerFixture()
	q, _ := playerFixture()
	if _, err := playersByRequestedHandle([]*common.Player{p, q}, []uint64{16385}); err == nil {
		t.Fatal("ambiguous identity accepted")
	}
}

func TestOutputGuardBeforeSourcesAndExclusiveFiles(t *testing.T) {
	empty := t.TempDir()
	if err := validateOutput(empty); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(empty, "retained")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(runOptions{input: "absent-demo", targets: "absent-targets", output: empty}); err == nil || !strings.Contains(err.Error(), "must be empty") {
		t.Fatalf("output not checked first: %v", err)
	}
	if err := exclusiveJSON(file, map[string]bool{"overwrite": true}); err == nil {
		t.Fatal("overwrite permitted")
	}
	raw, _ := os.ReadFile(file)
	if string(raw) != "keep" {
		t.Fatal("original mutated")
	}
	if validateOutput(os.TempDir()) == nil {
		t.Fatal("TEMP root accepted")
	}
	cwd, _ := os.Getwd()
	if validateOutput(cwd) == nil {
		t.Fatal("repository destination accepted")
	}
	if validateOutput(filepath.Join(os.TempDir(), "..")) == nil {
		t.Fatal("escaped TEMP accepted")
	}
}
func TestSymlinkDestinationRejected(t *testing.T) {
	base := t.TempDir()
	link := filepath.Join(base, "linked")
	target := t.TempDir()
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink creation unavailable on this host")
	}
	if validateOutput(link) == nil {
		t.Fatal("symlink/reparse output accepted")
	}
}
func TestTargetHashVerifiedBeforeDemoParsing(t *testing.T) {
	base := t.TempDir()
	targets := filepath.Join(base, "targets.json")
	raw, _ := json.Marshal(fixtureConfig())
	if err := os.WriteFile(targets, raw, 0600); err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	err := run(runOptions{input: "absent-demo", targets: targets, output: output, sourceSHA: strings.Repeat("a", 64), targetsSHA: strings.Repeat("c", 64)})
	if err == nil || !strings.Contains(err.Error(), "targets file hash differs") {
		t.Fatalf("target provenance not checked: %v", err)
	}
	entries, _ := os.ReadDir(output)
	if len(entries) != 0 {
		t.Fatal("invalid targets created artifacts")
	}
}
