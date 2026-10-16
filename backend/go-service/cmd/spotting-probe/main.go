// spotting-probe is an isolated diagnostic. Its observations are never player
// knowledge, HUD states, coordinates, or tactical features.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"google.golang.org/protobuf/proto"
)

const version = "stratai.spotting_probe@1"

var fields = []string{"m_bSpotted", "m_bSpottedByMask.0000", "m_bSpottedByMask.0001", "m_entitySpottedState.m_bSpotted", "m_entitySpottedState.m_bSpottedByMask.0000", "m_entitySpottedState.m_bSpottedByMask.0001"}

type scalar struct {
	Status string `json:"status"`
	Type   string `json:"native_type"`
	Value  any    `json:"value"`
}

func value(raw any) scalar {
	if raw == nil {
		return scalar{Status: "unset"}
	}
	s := scalar{Status: "observed", Type: fmt.Sprintf("%T", raw)}
	switch raw.(type) {
	case bool, int, int32, int64, uint32, uint64:
		s.Value = raw
	default:
		s.Status = "unsupported_type"
	}
	return s
}

// A missing declared path and a present-but-unset value remain distinct.
func property(e st.Entity, name string) (p st.Property, status string) {
	status = "absent_property"
	defer func() {
		if recover() != nil {
			p = nil
			status = "lookup_error"
		}
	}()
	if e == nil {
		return
	}
	p = e.Property(name)
	if p != nil {
		status = "present_property"
	}
	return
}
func read(e st.Entity, name string) scalar {
	p, status := property(e, name)
	if p == nil {
		return scalar{Status: status}
	}
	return value(p.Value().Any)
}

// Slots are the SDK's candidate bit indices, not certified HUD recipients.
func bits(raw any, word int) ([]int, bool) {
	var n uint64
	switch x := raw.(type) {
	case uint64:
		n = x
	case uint32:
		n = uint64(x)
	case int:
		if x < 0 {
			return nil, false
		}
		n = uint64(x)
	case int32:
		if x < 0 {
			return nil, false
		}
		n = uint64(x)
	case int64:
		if x < 0 {
			return nil, false
		}
		n = uint64(x)
	default:
		return nil, false
	}
	if n > 0xffffffff || word < 0 || word > 1 {
		return nil, false
	}
	out := []int{}
	for i := 0; i < 32; i++ {
		if n&(uint64(1)<<i) != 0 {
			out = append(out, word*32+i)
		}
	}
	return out, true
}

type entityRef struct {
	Life   int    `json:"entity_life"`
	Index  int    `json:"index"`
	Serial int    `json:"serial"`
	Class  string `json:"class"`
}
type actor struct {
	Alias            string            `json:"alias"`
	SDKEntityID      int               `json:"sdk_entity_id"`
	Connected        bool              `json:"connected"`
	Team             int               `json:"sdk_team"`
	Controller       *entityRef        `json:"controller"`
	Pawn             *entityRef        `json:"pawn"`
	ControllerFields map[string]scalar `json:"controller_fields"`
	PawnFields       map[string]scalar `json:"pawn_fields"`
	Slots            map[string][]int  `json:"candidate_mask_slots"`
}
type recorder struct {
	p            dem.Parser
	encoder      *json.Encoder
	err          error
	seq          int
	frame        int
	epoch        int
	lastTick     int
	haveTick     bool
	counts       map[string]int
	objects      map[st.Entity]*entityRef
	actors       map[*common.Player]string
	lastSnapshot []byte
}

func (r *recorder) emit(kind string, data any) {
	if r.err != nil {
		return
	}
	tick := r.p.GameState().IngameTick()
	if r.haveTick && tick < r.lastTick {
		r.epoch++
	}
	r.haveTick = true
	r.lastTick = tick
	r.seq++
	r.counts[kind]++
	r.err = r.encoder.Encode(map[string]any{"sequence": r.seq, "kind": kind, "frame_ordinal": r.frame + 1, "sdk_current_frame": r.p.CurrentFrame(), "tick": tick, "clock_epoch": r.epoch, "rounds_played": r.p.GameState().TotalRoundsPlayed(), "data": data})
}
func (r *recorder) ref(e st.Entity) *entityRef {
	if e == nil {
		return nil
	}
	if x := r.objects[e]; x != nil {
		return x
	}
	x := &entityRef{Life: len(r.objects) + 1, Index: e.ID(), Serial: e.SerialNum(), Class: e.ServerClass().Name()}
	r.objects[e] = x
	return x
}
func (r *recorder) snapshot() []actor {
	players := r.p.GameState().Participants().Connected()
	sort.Slice(players, func(i, j int) bool {
		if players[i].EntityID == players[j].EntityID {
			return players[i].UserID < players[j].UserID
		}
		return players[i].EntityID < players[j].EntityID
	})
	out := []actor{}
	for _, p := range players {
		if r.actors[p] == "" {
			r.actors[p] = fmt.Sprintf("actor-%03d", len(r.actors)+1)
		}
		pawn := p.PlayerPawnEntity()
		a := actor{Alias: r.actors[p], SDKEntityID: p.EntityID, Connected: p.IsConnected, Team: int(p.Team), Controller: r.ref(p.Entity), Pawn: r.ref(pawn), ControllerFields: map[string]scalar{}, PawnFields: map[string]scalar{}, Slots: map[string][]int{}}
		for _, name := range []string{"m_hPawn", "m_hPlayerPawn", "m_iTeamNum"} {
			a.ControllerFields[name] = read(p.Entity, name)
		}
		for _, name := range append(append([]string{}, fields...), "m_lifeState", "m_iHealth", "m_iTeamNum") {
			s := read(pawn, name)
			a.PawnFields[name] = s
			if strings.Contains(name, "ByMask.") && s.Status == "observed" {
				word := 0
				if strings.HasSuffix(name, ".0001") {
					word = 1
				}
				if b, ok := bits(s.Value, word); ok {
					a.Slots[name] = b
				}
			}
		}
		out = append(out, a)
	}
	return out
}
func (r *recorder) attach(e st.Entity) {
	ref := r.ref(e)
	r.emit("entity_created", ref)
	for _, name := range fields {
		p, status := property(e, name)
		r.emit("property_probe", map[string]any{"entity": ref, "property": name, "lookup_status": status})
		if p == nil {
			continue
		}
		registering := true
		p.OnUpdate(func(v st.PropertyValue) {
			r.emit("property_delivery", map[string]any{"entity": ref, "property": name, "registration_callback": registering, "value": value(v.Any)})
		})
		registering = false
	}
	e.OnCreateFinished(func() { r.emit("entity_create_finished", ref) })
	e.OnDestroy(func() { r.emit("entity_destroyed", ref) })
}
func declarations(raw []byte) (any, error) {
	n, offset := binary.Uvarint(raw)
	if offset <= 0 || n != uint64(len(raw)-offset) {
		return nil, fmt.Errorf("invalid sendtable framing")
	}
	m := &msg.CSVCMsg_FlattenedSerializer{}
	if err := proto.Unmarshal(raw[offset:], m); err != nil {
		return nil, err
	}
	symbol := func(p *int32) string {
		if p == nil || *p < 0 || int(*p) >= len(m.Symbols) {
			return ""
		}
		return m.Symbols[*p]
	}
	out := []map[string]any{}
	for _, s := range m.Serializers {
		for _, idx := range s.FieldsIndex {
			if idx < 0 || int(idx) >= len(m.Fields) {
				return nil, fmt.Errorf("bad serializer field index")
			}
			f := m.Fields[idx]
			name := symbol(f.VarNameSym)
			if strings.Contains(strings.ToLower(name), "spotted") || strings.Contains(strings.ToLower(symbol(s.SerializerNameSym)), "spotted") {
				out = append(out, map[string]any{"serializer": symbol(s.SerializerNameSym), "version": s.SerializerVersion, "name": name, "type": symbol(f.VarTypeSym), "nested_serializer": symbol(f.FieldSerializerNameSym)})
			}
		}
	}
	return out, nil
}
func (r *recorder) register() {
	r.p.RegisterNetMessageHandler(func(m *msg.CDemoSendTables) {
		d, err := declarations(m.Data)
		if err != nil {
			r.err = err
			return
		}
		r.emit("declarations", d)
	})
	r.p.RegisterNetMessageHandler(func(m *msg.CDemoFileHeader) {
		r.emit("header", map[string]any{"patch": m.PatchVersion, "build": m.BuildNum, "map": m.MapName, "allow_clientside_entities": m.AllowClientsideEntities})
	})
	r.p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		r.emit("server_info", map[string]any{"is_hltv": m.IsHltv, "player_slot": m.PlayerSlot, "tick_interval": m.TickInterval})
	})
	r.p.RegisterNetMessageHandler(func(m *msg.CCSUsrMsg_ProcessSpottedEntityUpdate) {
		r.emit("spotted_message", map[string]any{"message": m, "unknown_fields_bytes": len(m.ProtoReflect().GetUnknown()), "recipient": nil, "hud_state": nil})
	})
	r.p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_UserMessage) {
		if m.MsgType != nil && *m.MsgType == int32(msg.ECstrike15UserMessages_CS_UM_ProcessSpottedEntityUpdate) {
			h := sha256.Sum256(m.MsgData)
			r.emit("spotted_envelope", map[string]any{"msg_type": m.MsgType, "passthrough": m.Passthrough, "payload": m.MsgData, "payload_sha256": hex.EncodeToString(h[:]), "recipient": nil})
		}
	})
	r.p.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, c := range r.p.ServerClasses().All() {
			if c.Name() == "CCSPlayerPawn" {
				c.OnEntityCreated(r.attach)
			}
		}
	})
	r.p.RegisterEventHandler(func(e events.PlayerSpottersChanged) {
		var ref *entityRef
		if e.Spotted != nil {
			ref = r.ref(e.Spotted.PlayerPawnEntity())
		}
		r.emit("sdk_spotters_changed", ref)
	})
	r.p.RegisterEventHandler(func(events.FrameDone) {
		current := r.snapshot()
		b, err := json.Marshal(current)
		if err != nil {
			r.err = err
			return
		}
		if !reflect.DeepEqual(b, r.lastSnapshot) {
			r.emit("frame_snapshot", current)
			r.lastSnapshot = b
		}
		r.frame++
	})
}
func main() {
	demo := flag.String("demo", "", "read-only approved input")
	output := flag.String("output", "", "existing empty directory below system TEMP")
	checksum := flag.String("sha256", "", "required expected input checksum")
	flag.Parse()
	if err := run(*demo, *output, *checksum); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(input, output, checksum string) error {
	// Reject the destination before even opening the input.
	if err := validateOutput(output); err != nil {
		return err
	}
	expected, err := hex.DecodeString(checksum)
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf("required SHA-256 invalid")
	}
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), checksum) {
		return fmt.Errorf("input checksum mismatch")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	out, err := os.OpenFile(filepath.Join(output, "observations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer out.Close()
	buffer := bufio.NewWriterSize(out, 1<<20)
	defer buffer.Flush()
	p := dem.NewParser(f)
	defer p.Close()
	r := &recorder{p: p, encoder: json.NewEncoder(buffer), counts: map[string]int{}, objects: map[st.Entity]*entityRef{}, actors: map[*common.Player]string{}}
	r.register()
	started := time.Now()
	parseErr := p.ParseToEnd()
	if err = buffer.Flush(); err != nil {
		return err
	}
	if parseErr != nil {
		return parseErr
	}
	if r.err != nil {
		return r.err
	}
	summary := map[string]any{"schema_id": version, "sdk": "demoinfocs/v5@v5.2.0", "source_sha256": strings.ToLower(checksum), "demo_alias": "demo-" + strings.ToLower(checksum[:20]), "duration_seconds": time.Since(started).Seconds(), "frames": r.frame, "counts": r.counts, "entities": len(r.objects), "actor_aliases": len(r.actors), "records": r.seq, "tactical_features": 0, "hud_state_verified": false, "recipient_verified": false, "mask_slots_semantics": "candidate SDK indices only; not HUD recipients", "snapshot_semantics": "changed frame-end SDK state, not event-time retrospective attribution", "fields": fields}
	result, err := os.OpenFile(filepath.Join(output, "summary.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer result.Close()
	return json.NewEncoder(result).Encode(summary)
}
