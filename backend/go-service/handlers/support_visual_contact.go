package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/visualcontact"
	"encoding/json"
	"fmt"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportVisualFields = []string{
	"m_vecX", "m_vecY", "m_vecZ", "m_pCameraServices.m_iFOV", "m_pCameraServices.m_iFOVStart",
	"m_pCameraServices.m_flFOVTime", "m_pCameraServices.m_flFOVRate",
	"m_flFlashDuration", "m_flFlashMaxAlpha", "m_bIsScoped",
}
var supportVisualCoordinates = []string{"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ", "CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ"}

type supportVisualTracker struct {
	objects map[st.Entity]*supportMotionLife
}

func (h *ReplayHandler) registerSupportVisualContact() {
	if h.ctx.SupportVisualObservations == nil {
		return
	}
	h.supportVisual = &supportVisualTracker{objects: map[st.Entity]*supportMotionLife{}}
	h.ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range h.ctx.Parser.ServerClasses().All() {
			if class.Name() == "CCSPlayerPawn" {
				class.OnEntityCreated(func(e st.Entity) { h.bindSupportVisual(e) })
			}
		}
	})
}

func (h *ReplayHandler) visualDelivery(life *supportMotionLife, kind, name string, raw any) {
	data := h.ctx.SupportVisualObservations
	declared := h.supportObjectiveClock.declared("CCSPlayerPawn", name)
	value := positionScalar(raw, declared)
	previous, ok := life.previous[name]
	if !ok {
		previous = positionScalar(nil, declared)
	}
	d := models.SupportObjectivePositionDelivery{Sequence: len(data.Deliveries) + 1, Frame: h.attemptNextFrame(), Tick: h.ctx.Parser.GameState().IngameTick(), Life: life.life, Kind: kind, Property: name, Previous: previous, Value: value}
	data.Deliveries = append(data.Deliveries, d)
	if name != "" {
		life.previous[name], life.sequences[name] = value, d.Sequence
	}
}

func (h *ReplayHandler) bindSupportVisual(e st.Entity) {
	if h.supportVisual == nil {
		return
	}
	life := &supportMotionLife{life: fmt.Sprintf("support-visual-pawn-%06d", len(h.supportVisual.objects)+1), createdFrame: h.attemptNextFrame(), alive: true, previous: map[string]models.SupportClockScalar{}, sequences: map[string]int{}}
	h.supportVisual.objects[e] = life
	for _, name := range supportVisualFields {
		h.visualDelivery(life, "created", name, h.positionValue(e, "CCSPlayerPawn", name))
		if prop := h.positionProperty(e, "CCSPlayerPawn", name); prop != nil {
			prop.OnUpdate(func(v st.PropertyValue) {
				kind := "update"
				if h.attemptNextFrame() == life.createdFrame {
					kind = "creation_replay"
				}
				h.visualDelivery(life, kind, name, v.Any)
			})
		}
	}
	e.OnDestroy(func() { h.visualDelivery(life, "destroyed", "", nil); life.alive = false })
}

// supportVisualEye is only the endpoint of a static-mesh diagnostic. It does
// not certify camera interpolation, parent transforms or visibility to a person.
func supportVisualEye(coords, fields map[string]models.SupportClockScalar) *r3.Vector {
	values := [3]float64{}
	for i, axis := range []string{"X", "Y", "Z"} {
		cell, offset, eye := coords["CBodyComponent.m_cell"+axis], coords["CBodyComponent.m_vec"+axis], fields["m_vec"+axis]
		if cell.Status != "observed" || cell.DeclaredType == nil || *cell.DeclaredType != "uint16" {
			return nil
		}
		n := supportHandle(cell.Value)
		if n == nil || *n > 65535 {
			return nil
		}
		v, vok := offset.Value.(float64)
		e, eok := eye.Value.(float64)
		if offset.Status != "observed" || eye.Status != "observed" || !vok || !eok || offset.DeclaredType == nil || eye.DeclaredType == nil || *offset.DeclaredType != "CNetworkedQuantizedFloat" || *eye.DeclaredType != "CNetworkedQuantizedFloat" {
			return nil
		}
		values[i] = float64(*n)*512 - 16384 + v + e
	}
	v := r3.Vector{X: values[0], Y: values[1], Z: values[2]}
	if !visualcontact.IsFiniteVector(v) {
		return nil
	}
	return &v
}

func (h *ReplayHandler) captureSupportVisualContact(base models.SupportFrame, players map[uint64]*common.Player) {
	if h.supportVisual == nil {
		return
	}
	data, current := h.ctx.SupportVisualObservations, h.ctx.Parser.GameState().Entities()
	cut := models.SupportVisualCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ClockOrder: h.supportObjectiveClock.order, DeliverySequence: len(data.Deliveries), Players: []models.SupportVisualPlayer{}, Rays: []models.SupportVisualRay{}}
	cut.GeometryLoaded = h.ctx.MapManager != nil && h.ctx.MapManager.IsLoaded()
	eyes := []*r3.Vector{}
	for _, actor := range base.Players {
		row := models.SupportVisualPlayer{SupportNativeMotionPlayer: models.SupportNativeMotionPlayer{NativeActor: actor.NativeActor}, Coordinates: map[string]models.SupportClockScalar{}}
		var eye *r3.Vector
		p := players[actor.NativeActor]
		proof := objectivePlayer(p, actor)
		if proof.PawnLife != nil {
			e := p.PlayerPawnEntity()
			life := h.supportVisual.objects[e]
			if life != nil && life.alive && current[e.ID()] == e && h.support.lives[e] == *proof.PawnLife {
				state := models.SupportVisualState{Instance: base.Instance, Epoch: base.Epoch, Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: "CCSPlayerPawn", CreatedFrame: life.createdFrame, Fields: map[string]models.SupportClockScalar{}, FieldSequences: map[string]int{}}
				for _, name := range supportVisualFields {
					state.Fields[name] = positionScalar(h.positionValue(e, "CCSPlayerPawn", name), h.supportObjectiveClock.declared("CCSPlayerPawn", name))
					state.FieldSequences[name] = life.sequences[name]
				}
				raw, _ := json.Marshal(state)
				if string(raw) != life.fingerprint {
					state.Sequence, state.FirstCut = len(data.States)+1, base.Sequence
					data.States = append(data.States, state)
					life.fingerprint, life.stateSequence = string(raw), state.Sequence
				}
				seq := life.stateSequence
				row.PawnLife, row.PawnHandle, row.StateSequence = proof.PawnLife, actor.PawnHandle, &seq
				for _, name := range supportVisualCoordinates {
					row.Coordinates[name] = positionScalar(h.positionValue(e, "CCSPlayerPawn", name), h.supportObjectiveClock.declared("CCSPlayerPawn", name))
				}
				eye = supportVisualEye(row.Coordinates, state.Fields)
			}
		}
		cut.Players = append(cut.Players, row)
		eyes = append(eyes, eye)
	}
	// One directed diagnostic per current usable opponent pair; no historic
	// contact, smoke snapshot, spotted flag, death or outcome enters this cut.
	for i, actor := range base.Players {
		if cut.Players[i].StateSequence == nil || (actor.Side != "T" && actor.Side != "CT") {
			continue
		}
		for j, target := range base.Players {
			if cut.Players[j].StateSequence == nil || actor.Side == target.Side || (target.Side != "T" && target.Side != "CT") {
				continue
			}
			ray := models.SupportVisualRay{Observer: i, Subject: j}
			if cut.GeometryLoaded && eyes[i] != nil && eyes[j] != nil {
				clear := h.ctx.MapManager.IsVisible(*eyes[i], *eyes[j])
				ray.Clear = &clear
			}
			cut.Rays = append(cut.Rays, ray)
		}
	}
	cut.StateSequence = len(data.States)
	data.Cuts = append(data.Cuts, cut)
}
