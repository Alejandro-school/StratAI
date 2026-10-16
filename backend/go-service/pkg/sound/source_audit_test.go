package sound

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

func TestNativeRadiusNeverBecomesReceptionWithoutSemantics(t *testing.T) {
	audit := NewSourceAudit()
	audit.record(Signal{Tick: 123, Source: "events.PlayerSound", Radius: intPointer(800), Family: "unknown"})
	if audit.Counts["events.PlayerSound"] != 1 || len(audit.Signals) != 1 {
		t.Fatal("positive native emission was lost")
	}
	data, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "reception_estimated") || audit.ReceptionStatus != "unavailable" {
		t.Fatal("an unverified native radius granted hearing")
	}
	if audit.Signals[0].RangeStatus != "native_radius_semantics_unverified" || audit.Signals[0].Family != "unknown" {
		t.Fatal("generic sound was reclassified or its range invented")
	}
}

func TestFootstepMissingRangeAndNoEventsRemainExplicit(t *testing.T) {
	audit := NewSourceAudit()
	if audit.Counts["events.PlayerSound"] != 0 || audit.Counts["events.Footstep"] != 0 || audit.Signals == nil {
		t.Fatal("zero coverage must be explicit, not an omitted collection")
	}
	audit.record(Signal{Tick: 42, Source: "events.Footstep", Family: "footstep"})
	if audit.Signals[0].RangeStatus != "unavailable" || audit.Signals[0].Radius != nil {
		t.Fatal("footstep received a default radius")
	}
	audit.Actions["weapon_fire"]++
	if len(audit.Signals) != 1 || audit.ReceptionStatus != "unavailable" {
		t.Fatal("action manufactured sound or hearing")
	}
}

func TestSourceSignalKeepsNativeTickAndInvalidRangeIsNotPositive(t *testing.T) {
	for _, radius := range []int{0, -1} {
		audit := NewSourceAudit()
		audit.record(Signal{Tick: 27, Source: "events.PlayerSound", Radius: intPointer(radius), Family: "unknown"})
		if audit.Signals[0].Tick != 27 || audit.Signals[0].AvailabilityTick != 27 || audit.Signals[0].RangeStatus != "invalid_native_radius" {
			t.Fatal("tick or range availability changed")
		}
	}
}

func TestRawCountsAndDescriptorsRemainIndependentOfTypedCallbacks(t *testing.T) {
	audit := NewSourceAudit()
	audit.RawEvents["player_sound"] = 2
	audit.Descriptors["player_sound"] = []string{"userid", "radius", "duration"}
	if audit.Counts["events.PlayerSound"] != 0 || len(audit.Signals) != 0 {
		t.Fatal("raw descriptors or counters synthesized a typed sound")
	}
}

func TestInternalEventListFallbackDoesNotBecomeFalseSoundAbsence(t *testing.T) {
	audit := NewSourceAudit()
	audit.RawEvents["unknown_descriptor"] = 1
	audit.recordResolved(events.GenericGameEvent{Name: "player_sound", Data: map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"radius": {}, "duration": {}, "step": {},
	}})
	if audit.ResolvedEvents["player_sound"] != 1 || audit.RawEvents["player_sound"] != 0 {
		t.Fatal("fallback resolution was lost or raw evidence was rewritten")
	}
	if len(audit.ResolvedSoundKeys["player_sound"]) != 3 || len(audit.Signals) != 0 {
		t.Fatal("resolved fields lost or counter manufactured a sound callback")
	}
}
