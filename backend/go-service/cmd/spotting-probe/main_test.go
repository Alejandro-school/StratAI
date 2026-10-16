package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestMaskWordsBoundariesAndInvalid(t *testing.T) {
	cases := []struct {
		raw  any
		word int
		want []int
		ok   bool
	}{
		{uint64(0), 0, []int{}, true}, {uint64(1), 0, []int{0}, true},
		{uint64(0x80000001), 0, []int{0, 31}, true}, {uint32(0x80000001), 1, []int{32, 63}, true},
		{int64(3), 1, []int{32, 33}, true}, {nil, 0, nil, false}, {-1, 0, nil, false},
		{uint64(1) << 32, 0, nil, false}, {true, 0, nil, false}, {float64(1), 0, nil, false}, {uint64(1), 2, nil, false},
	}
	for _, c := range cases {
		got, ok := bits(c.raw, c.word)
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%v/%d: %v %v", c.raw, c.word, got, ok)
		}
	}
}
func TestScalarPreservesMissingZeroFalseAndUnsupported(t *testing.T) {
	for _, c := range []struct {
		raw    any
		status string
	}{{nil, "unset"}, {uint64(0), "observed"}, {false, "observed"}, {"private-player-name", "unsupported_type"}} {
		s := value(c.raw)
		if s.Status != c.status {
			t.Fatal(s)
		}
		if c.status == "observed" && !reflect.DeepEqual(s.Value, c.raw) {
			t.Fatal(s)
		}
		if c.status != "observed" && s.Value != nil {
			t.Fatal(s)
		}
	}
}

type propStub struct {
	st.Property
	raw any
}

func (p propStub) Value() st.PropertyValue { return st.PropertyValue{Any: p.raw} }

type entityStub struct {
	st.Entity
	index, serial int
	props         map[string]st.Property
	panicPath     string
}

func (e *entityStub) Property(n string) st.Property {
	if n == e.panicPath {
		panic("invalid path")
	}
	return e.props[n]
}
func (e *entityStub) ID() int                     { return e.index }
func (e *entityStub) SerialNum() int              { return e.serial }
func (e *entityStub) ServerClass() st.ServerClass { return classStub{} }

type classStub struct{ st.ServerClass }

func (classStub) Name() string { return "CCSPlayerPawn" }
func TestPropertiesMissingUnsetAndLookupFailure(t *testing.T) {
	e := &entityStub{props: map[string]st.Property{"zero": propStub{raw: uint64(0)}, "unset": propStub{}}, panicPath: "bad"}
	for name, want := range map[string]string{"zero": "observed", "unset": "unset", "missing": "absent_property", "bad": "lookup_error"} {
		if s := read(e, name); s.Status != want {
			t.Fatalf("%s: %+v", name, s)
		}
	}
	if read(nil, "zero").Status != "absent_property" {
		t.Fatal("nil entity became zero")
	}
}
func TestEntityReuseDoesNotReuseLife(t *testing.T) {
	r := recorder{objects: map[st.Entity]*entityRef{}}
	a := &entityStub{index: 7, serial: 1}
	b := &entityStub{index: 7, serial: 2}
	c := &entityStub{index: 7, serial: 2}
	first := *r.ref(a)
	if r.ref(b).Life == first.Life || r.ref(c).Life == r.ref(b).Life {
		t.Fatal("reused entity index/object incarnation conflated")
	}
	if *r.ref(a) != first {
		t.Fatal("later entity retroactively changed earlier reference")
	}
	if r.ref(nil) != nil {
		t.Fatal("missing entity received identity")
	}
}
func TestRadarMessagePreservesPresenceAndRawCoordinates(t *testing.T) {
	zero := int32(0)
	negative := int32(-123)
	no := false
	m := &msg.CCSUsrMsg_ProcessSpottedEntityUpdate{NewUpdate: &no, EntityUpdates: []*msg.CCSUsrMsg_ProcessSpottedEntityUpdate_SpottedEntityUpdate{{EntityIdx: &zero, OriginX: &negative, OriginY: &zero}}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	row := d["entity_updates"].([]any)[0].(map[string]any)
	if _, ok := row["origin_z"]; ok {
		t.Fatal("missing coordinate became zero")
	}
	if row["origin_x"] != float64(-123) || row["entity_idx"] != float64(0) || d["new_update"] != false {
		t.Fatal(string(b))
	}
	for _, word := range []string{"recipient", "visible", "hud_state", "position"} {
		if strings.Contains(string(b), word) {
			t.Fatal("invented field", word)
		}
	}
}
func TestOutputGuardAndBeforeInputOpen(t *testing.T) {
	empty := t.TempDir()
	if err := validateOutput(empty); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(empty, "retained"), []byte("do not overwrite"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{empty, os.TempDir(), "", filepath.Join(t.TempDir(), "missing")} {
		err := run("input-that-does-not-exist", p, strings.Repeat("0", 64))
		if err == nil || strings.Contains(err.Error(), "input-that-does-not-exist") {
			t.Fatalf("input opened before guard: %q %v", p, err)
		}
	}
	outside, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if validateOutput(outside) == nil {
		t.Fatal("workspace accepted")
	}
	if data, _ := os.ReadFile(filepath.Join(empty, "retained")); string(data) != "do not overwrite" {
		t.Fatal("overwrote retained evidence")
	}
}
func TestOutputGuardRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if validateOutput(link) == nil {
		t.Fatal("linked target accepted")
	}
}
func TestChecksumMismatchDoesNotStartParsingOrWrite(t *testing.T) {
	input := filepath.Join(t.TempDir(), "fake.dem")
	if err := os.WriteFile(input, []byte("not a demo"), 0600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	err := run(input, out, strings.Repeat("0", 64))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Fatal("wrote before checksum validation")
	}
}
func TestDeclarationFramingRejectsTruncation(t *testing.T) {
	for _, raw := range [][]byte{nil, {0x80}, {5, 1}, {1, 0xff}} {
		if _, err := declarations(raw); err == nil {
			t.Fatalf("accepted %v", raw)
		}
	}
}
