package sound

import (
	"github.com/golang/geo/r3"
	"testing"
)

func TestNativeSoundReceptionIsPossibleAnonymousAndThreeDimensional(t *testing.T) {
	ledger := NewInformation()
	p, _ := LookupProfile("Base.Footstep")
	pos := r3.Vector{}
	ledger.Observe(Candidate{Tick: 120, Source: "net.CMsgSosStartSoundEvent", Profile: p.Event, Position: &pos, EmitterID: 20, EmitterSide: "CT"}, []Observer{
		{ID: 10, Side: "T", Alive: true, Position: r3.Vector{Z: 1099}},
		{ID: 11, Side: "T", Alive: true, Position: r3.Vector{Z: 1100}},
		{ID: 12, Side: "T", Alive: false},
		{ID: 13, Side: "CT", Alive: true},
	})
	if len(ledger.Observations) != 1 || ledger.Observations[0].ObserverID != 10 || ledger.Observations[0].Distance != 1099 {
		t.Fatalf("wrong 3D reception: %+v", ledger.Observations)
	}
	if ledger.Observations[0].Status != "estimated_possible_reception" || len(ledger.Emissions) != 1 {
		t.Fatal("certainty or provenance lost")
	}
}

func TestZeroVolumeLocalOnlyAndMissingPositionCannotGrantReception(t *testing.T) {
	for _, name := range []string{"HEGrenade.PullPin_Grenade", "unknown"} {
		ledger := NewInformation()
		p := r3.Vector{}
		ledger.Observe(Candidate{Tick: 1, Source: "net.CMsgSosStartSoundEvent", Profile: name, Position: &p}, []Observer{{ID: 1, Side: "T", Alive: true}})
		if len(ledger.Observations) != 0 {
			t.Fatal("ineligible profile granted reception")
		}
	}
	ledger := NewInformation()
	zero := 0.0
	p := r3.Vector{}
	ledger.Observe(Candidate{Tick: 1, Profile: "Base.Footstep", Position: &p, Volume: &zero}, []Observer{{ID: 1, Side: "T", Alive: true}})
	ledger.Observe(Candidate{Tick: 1, Profile: "Base.Footstep"}, []Observer{{ID: 1, Side: "T", Alive: true}})
	if len(ledger.Observations) != 0 || len(ledger.Abstentions) != 2 {
		t.Fatal("zero or absent evidence became hearing")
	}
}

func TestSoundLifeReconciliationIsCausalAndRoundBound(t *testing.T) {
	ledger := NewInformation()
	p := r3.Vector{}
	for _, tick := range []int{90, 149, 150, 151, 201} {
		ledger.Observe(Candidate{Tick: tick, Profile: "Base.Footstep", Position: &p}, []Observer{{ID: 1, Side: "T", Alive: true}})
	}
	ledger.Reconcile(100, 200, map[uint64]int{1: 150})
	if len(ledger.Observations) != 1 || ledger.Observations[0].Tick != 149 {
		t.Fatalf("death or round boundary leaked: %+v", ledger.Observations)
	}
	if len(ledger.Emissions) != 3 {
		t.Fatal("causal emission audit lost or outside-round evidence retained")
	}
}

func TestFireRadiusUsesNativeItemAndSoundType(t *testing.T) {
	normal, ok := FireProfile(61, 2)
	if !ok || normal.Radius != 9000 {
		t.Fatal("native unsilenced USP profile missing")
	}
	silenced, ok := FireProfile(61, 9)
	if !ok || silenced.Radius != 1400 {
		t.Fatal("silencer state guessed or wrong native map")
	}
	if _, ok := FireProfile(61, 12345); ok {
		t.Fatal("unknown sound type received a firing fallback")
	}
}

func TestUnknownAffiliationCannotMakeTheEmitterAnEnemyListener(t *testing.T) {
	ledger := NewInformation()
	point := r3.Vector{}
	ledger.Observe(Candidate{Tick: 1, Profile: "Base.Footstep", Position: &point, EmitterID: 1}, []Observer{{ID: 1, Side: "T", Alive: true}})
	if len(ledger.Observations) != 0 {
		t.Fatal("emitter became its own possible enemy sound")
	}
}
