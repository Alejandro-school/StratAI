package handlers

import (
	"math"
	"testing"

	"cs2-demo-service/pkg/utility"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestObserveProjectileVelocityFailsClosedOnMissingPartialAndNonFiniteProperty(t *testing.T) {
	tests := []struct {
		name   string
		entity st.Entity
	}{
		{name: "missing entity"},
		{name: "missing property", entity: &initialVelocityTestEntity{}},
		{name: "nil property", entity: &initialVelocityTestEntity{present: true}},
		{name: "partial vector", entity: &initialVelocityTestEntity{
			present: true, value: st.PropertyValue{Any: []float32{1, 2}},
		}},
		{name: "non finite", entity: &initialVelocityTestEntity{
			present: true, value: st.PropertyValue{Any: [3]float32{1, float32(math.Inf(1)), 3}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := observeProjectileVelocity(&common.GrenadeProjectile{Entity: test.entity}, 100)
			if got.Status != utility.AvailabilityUnavailable || got.Source != utility.SourceUnavailable {
				t.Fatalf("invalid native property was observed: %+v", got)
			}
		})
	}
}

func TestObserveProjectileVelocityPreservesObservedZeroComponent(t *testing.T) {
	projectile := &common.GrenadeProjectile{Entity: &initialVelocityTestEntity{
		present: true,
		value:   st.PropertyValue{Any: [3]float32{0, 700, 180}},
	}}
	got := observeProjectileVelocity(projectile, 123)
	if got.Status != utility.AvailabilityObserved || got.Source != utility.SourceProjectileVelocity ||
		got.Vector.X != 0 || got.Vector.Y != 700 || got.Vector.Z != 180 || got.ObservedTick != 123 {
		t.Fatalf("observed native vector was not preserved: %+v", got)
	}
}

type initialVelocityTestEntity struct {
	present bool
	value   st.PropertyValue
	serial  int
}

func (entity *initialVelocityTestEntity) ServerClass() st.ServerClass { return nil }
func (entity *initialVelocityTestEntity) ID() int                     { return 17 }
func (entity *initialVelocityTestEntity) SerialNum() int              { return entity.serial }
func (entity *initialVelocityTestEntity) Properties() []st.Property   { return nil }
func (entity *initialVelocityTestEntity) Property(string) st.Property { return nil }
func (entity *initialVelocityTestEntity) BindProperty(string, any, st.PropertyValueType) {
}
func (entity *initialVelocityTestEntity) PropertyValue(string) (st.PropertyValue, bool) {
	return entity.value, entity.present
}
func (entity *initialVelocityTestEntity) PropertyValueMust(string) st.PropertyValue {
	if !entity.present {
		panic("missing property")
	}
	return entity.value
}
func (entity *initialVelocityTestEntity) Position() r3.Vector              { return r3.Vector{} }
func (entity *initialVelocityTestEntity) OnPositionUpdate(func(r3.Vector)) {}
func (entity *initialVelocityTestEntity) OnDestroy(func())                 {}
func (entity *initialVelocityTestEntity) Destroy()                         {}
func (entity *initialVelocityTestEntity) OnCreateFinished(func())          {}
