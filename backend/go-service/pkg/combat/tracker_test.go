package combat

import (
	"reflect"
	"testing"
)

func TestSnapshotIsIndependentOfCallbackOrder(t *testing.T) {
	record := func(tracker *Tracker, reverse bool) {
		operations := []func(){
			func() { tracker.RecordWeaponFire(testFire(100, "AK-47")) },
			func() { tracker.RecordBulletDamage(testBullet(100)) },
			func() { tracker.RecordPlayerHurt(testHurt(100, 0, "AK-47")) },
			func() { tracker.RecordKill(testKill(100, "AK-47")) },
		}
		if reverse {
			for index := len(operations) - 1; index >= 0; index-- {
				operations[index]()
			}
			return
		}
		for _, operation := range operations {
			operation()
		}
	}

	ordered := NewTracker()
	record(ordered, false)
	ordered.EndRound(1, 200)
	reversed := NewTracker()
	record(reversed, true)
	reversed.EndRound(1, 200)

	left := ordered.Snapshot()
	right := reversed.Snapshot()
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("callback order changed snapshot:\n%+v\n%+v", left, right)
	}
	if len(left) != 4 || left[0].Type != EventWeaponFire || left[1].Type != EventBulletDamage ||
		left[2].Type != EventPlayerHurt || left[3].Type != EventKill {
		t.Fatalf("unexpected causal order: %+v", left)
	}
	if left[0].ShotResult != ShotHit || left[2].ShotID != left[0].ShotID || left[3].ShotID != left[0].ShotID {
		t.Fatalf("shot chain was not reconciled: %+v", left)
	}
	if !reflect.DeepEqual(left[2].SourceEventIDs, []string{left[0].LocalID, left[1].LocalID}) {
		t.Fatalf("hurt provenance = %v", left[2].SourceEventIDs)
	}
	if !reflect.DeepEqual(left[3].SourceEventIDs, []string{left[2].LocalID}) {
		t.Fatalf("kill provenance = %v", left[3].SourceEventIDs)
	}
}

func TestMissRequiresObservedRoundClosure(t *testing.T) {
	open := NewTracker()
	open.RecordWeaponFire(testFire(100, "M4A4"))
	if got := open.Snapshot()[0].ShotResult; got != ShotUnavailable {
		t.Fatalf("open-round result = %q", got)
	}

	closed := NewTracker()
	closed.RecordWeaponFire(testFire(100, "M4A4"))
	closed.EndRound(1, 150)
	event := closed.Snapshot()[0]
	if event.ShotResult != ShotMiss || event.ShotResultAvailabilityTick == nil || *event.ShotResultAvailabilityTick != 150 {
		t.Fatalf("closed-round result = %+v", event)
	}
}

func TestWorldFriendlyAndSelfDamageArePreserved(t *testing.T) {
	tracker := NewTracker()
	world := testHurt(10, 90, "World")
	world.Actor = PlayerRef{Status: AvailabilityUnavailable, Source: SourceUnavailable}
	tracker.RecordPlayerHurt(world)
	friendly := testHurt(11, 80, "AK-47")
	friendly.Actor.Side = friendly.Target.Side
	tracker.RecordPlayerHurt(friendly)
	self := testHurt(12, 70, "HE Grenade")
	self.Target = self.Actor
	tracker.RecordPlayerHurt(self)

	events := tracker.Snapshot()
	if len(events) != 3 || events[0].Relation != RelationWorld || events[1].Relation != RelationFriendly || events[2].Relation != RelationSelf {
		t.Fatalf("relations were not preserved: %+v", events)
	}
}

func TestCorrelationNeverUsesFutureFire(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordPlayerHurt(testHurt(99, 90, "AK-47"))
	tracker.RecordWeaponFire(testFire(100, "AK-47"))
	events := tracker.Snapshot()
	if events[0].Type != EventPlayerHurt || events[0].ShotID != "" || events[0].CorrelationStatus != CorrelationUnavailable {
		t.Fatalf("hurt used future fire: %+v", events)
	}
}

func TestWeaponLifecycleUsesOnlyPriorObservation(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordWeaponFire(testFire(10, "AK-47"))
	tracker.RecordWeaponEquip(EquipInput{
		Round: 1, Tick: 11, Actor: testPlayer(1, "T"), Weapon: testWeapon("Glock-18"),
	})
	events := tracker.Snapshot()
	equip := events[1]
	if equip.PreviousWeapon == nil || *equip.PreviousWeapon != "AK-47" || equip.IsWeaponSwitch == nil || !*equip.IsWeaponSwitch {
		t.Fatalf("equip lifecycle = %+v", equip)
	}
}

func TestSummariesUseShotLevelDamageAndAssists(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordWeaponFire(testFire(100, "XM1014"))
	first := testHurt(100, 60, "XM1014")
	first.HealthDamageTaken = 40
	tracker.RecordPlayerHurt(first)
	second := testHurt(100, 40, "XM1014")
	second.HealthDamageTaken = 20
	tracker.RecordPlayerHurt(second)
	kill := testKill(100, "XM1014")
	kill.Assister = testPlayer(3, "T")
	kill.AssistedFlash = true
	tracker.RecordKill(kill)
	tracker.EndRound(1, 200)

	summaries := Summaries(tracker.Snapshot())
	actor := summaries[1]
	if actor.ShotsFired != 1 || actor.ShotsHit != 1 || actor.EnemyDamage != 60 || actor.Kills != 1 {
		t.Fatalf("actor summary = %+v", actor)
	}
	if summaries[3].Assists != 1 || summaries[3].FlashAssists != 1 {
		t.Fatalf("assister summary = %+v", summaries[3])
	}
}

func TestDiscardedCallbacksRemainExplicitAndConserved(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordDiscardedCallback(EventWeaponFire, DiscardOutsideRound)
	diagnostics := tracker.Diagnostics()
	if diagnostics.ObservedByType[EventWeaponFire] != 1 ||
		diagnostics.RecordedByType[EventWeaponFire] != 0 ||
		diagnostics.DiscardedByType[EventWeaponFire] != 1 ||
		diagnostics.DiscardedByReason[DiscardOutsideRound] != 1 {
		t.Fatalf("discarded callback was not conserved: %+v", diagnostics)
	}
	if events := tracker.Snapshot(); len(events) != 0 {
		t.Fatalf("outside-round callback became a factual event: %+v", events)
	}
}

func testFire(tick int, weapon string) FireInput {
	return FireInput{Round: 1, Tick: tick, Actor: testPlayer(1, "T"), Weapon: testWeapon(weapon)}
}

func testBullet(tick int) BulletDamageInput {
	return BulletDamageInput{
		Round: 1, Tick: tick, Actor: testPlayer(1, "T"), Target: testPlayer(2, "CT"),
		Distance: 100, Direction: Vector{X: 1}, PenetratedObjects: 1,
	}
}

func testHurt(tick, healthAfter int, weapon string) HurtInput {
	return HurtInput{
		Round: 1, Tick: tick, Actor: testPlayer(1, "T"), Target: testPlayer(2, "CT"),
		Weapon: testWeapon(weapon), HealthDamage: 10, HealthDamageTaken: 10,
		HealthAfter: healthAfter, ArmorAfter: 50, Hitgroup: "chest",
	}
}

func testKill(tick int, weapon string) KillInput {
	return KillInput{
		Round: 1, Tick: tick, Actor: testPlayer(1, "T"), Target: testPlayer(2, "CT"),
		Weapon: testWeapon(weapon), IsHeadshot: false,
	}
}

func testPlayer(id uint64, side string) PlayerRef {
	position := Vector{X: float64(id)}
	return PlayerRef{
		ID: id, Side: side, Status: AvailabilityObserved, Source: SourceCallbackPlayer,
		Position: &position, PositionStatus: AvailabilityObserved, PositionSource: SourceCallbackPosition,
	}
}

func testWeapon(name string) WeaponRef {
	isUtility := false
	return WeaponRef{Name: name, Status: AvailabilityObserved, Source: SourceCallbackWeapon, IsUtility: &isUtility}
}
