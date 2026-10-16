package analyzers

import (
	"cs2-demo-service/models"
	"math"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestUtilityPointVisibilityDoesNotBypassPerceptionGuards(t *testing.T) {
	for _, c := range []struct {
		name         string
		wall, loaded bool
		yaw, flash   float32
		smokes       map[int64]r3.Vector
		visible      bool
	}{
		{name: "visible", wall: true, loaded: true, visible: true},
		{name: "wall", loaded: true},
		{name: "missing_geometry", wall: true},
		{name: "behind_view", wall: true, loaded: true, yaw: 180},
		{name: "flashed", wall: true, loaded: true, flash: 2},
		{name: "smoke", wall: true, loaded: true, smokes: map[int64]r3.Vector{1: {X: 200, Z: 64}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			observer := contactProbePlayer(11, common.TeamTerrorists, 0, c.yaw, c.flash)
			ctx := &models.DemoContext{MapManager: contactProbeGeometry{head: c.wall, chest: c.wall, loaded: c.loaded}}
			result := EstimateUtilityPointVisibility(ctx, observer, r3.Vector{X: 1000, Z: 64}, c.smokes)
			if result.Visible != c.visible {
				t.Fatalf("got %+v, expected visible=%v", result, c.visible)
			}
			if !c.loaded && result.Available {
				t.Fatal("missing geometry became available")
			}
		})
	}
}

func TestUtilityPointVisibilityAbstainsFromMissingNativeViewAndNonfinitePoints(t *testing.T) {
	ctx := &models.DemoContext{MapManager: contactProbeGeometry{head: true, chest: true, loaded: true}}
	observer := contactProbePlayer(11, common.TeamTerrorists, 0, 0, 0)
	pawn := observer.PlayerPawnEntity().(*contactProbeEntity)
	delete(pawn.properties, "m_angEyeAngles")
	if EstimateUtilityPointVisibility(ctx, observer, r3.Vector{X: 1000, Z: 64}, nil).Available {
		t.Fatal("missing native view became a positive source")
	}
	observer = contactProbePlayer(11, common.TeamTerrorists, 0, 0, 0)
	if EstimateUtilityPointVisibility(ctx, observer, r3.Vector{X: math.NaN(), Z: 64}, nil).Available {
		t.Fatal("nonfinite utility point became available")
	}
	if EstimateUtilityPointVisibility(ctx, nil, r3.Vector{X: 1000, Z: 64}, nil).Available {
		t.Fatal("missing observer became available")
	}
}
