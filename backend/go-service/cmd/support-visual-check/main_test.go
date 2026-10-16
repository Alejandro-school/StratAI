package main

import (
	"cs2-demo-service/models"
	"encoding/json"
	"testing"

	"github.com/golang/geo/r3"
)

func TestEndpointRecomputationPreservesZeroAndRejectsMissingOrWrongTypedInputs(t *testing.T) {
	cellType, floatType, nativeInt, nativeFloat := "uint16", "CNetworkedQuantizedFloat", "uint64", "float32"
	p := models.SupportVisualPlayer{Coordinates: map[string]models.SupportClockScalar{}}
	s := models.SupportVisualState{Fields: map[string]models.SupportClockScalar{}}
	for _, axis := range []string{"X", "Y", "Z"} {
		p.Coordinates["CBodyComponent.m_cell"+axis] = models.SupportClockScalar{Status: "observed", NativeType: &nativeInt, DeclaredType: &cellType, Value: json.Number("32")}
		p.Coordinates["CBodyComponent.m_vec"+axis] = models.SupportClockScalar{Status: "observed", NativeType: &nativeFloat, DeclaredType: &floatType, Value: json.Number("0")}
		s.Fields["m_vec"+axis] = models.SupportClockScalar{Status: "observed", NativeType: &nativeFloat, DeclaredType: &floatType, Value: json.Number("0")}
	}
	v, ok := endpoint(p, s)
	if !ok || v != (r3.Vector{}) {
		t.Fatal("zero lost")
	}
	cell := p.Coordinates["CBodyComponent.m_cellX"]
	badCell := cell
	badCell.NativeType = &nativeFloat
	p.Coordinates["CBodyComponent.m_cellX"] = badCell
	if _, ok = endpoint(p, s); ok {
		t.Fatal("floating native cell accepted")
	}
	p.Coordinates["CBodyComponent.m_cellX"] = cell
	bad := s.Fields["m_vecZ"]
	bad.NativeType = &nativeInt
	s.Fields["m_vecZ"] = bad
	if _, ok = endpoint(p, s); ok {
		t.Fatal("integer eye coordinate accepted")
	}
	delete(s.Fields, "m_vecZ")
	if _, ok = endpoint(p, s); ok {
		t.Fatal("missing eye replaced by zero")
	}
}
