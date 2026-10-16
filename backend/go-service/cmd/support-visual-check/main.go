// Recompute every retained static ray from the immutable source and map assets.
// This command never opens a demo and never writes to a source or output tree.
package main

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/golang/geo/r3"
)

func main() {
	source := flag.String("source", "", "visual observation source")
	root := flag.String("maps-root", "", "read-only map assets")
	flag.Parse()
	if err := run(*source, *root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func endpoint(player models.SupportVisualPlayer, state models.SupportVisualState) (r3.Vector, bool) {
	var xyz [3]float64
	for i, axis := range []string{"X", "Y", "Z"} {
		values := []models.SupportClockScalar{player.Coordinates["CBodyComponent.m_cell"+axis], player.Coordinates["CBodyComponent.m_vec"+axis], state.Fields["m_vec"+axis]}
		var numbers [3]float64
		for j, value := range values {
			if value.DeclaredType == nil || value.NativeType == nil {
				return r3.Vector{}, false
			}
			if j == 0 {
				if *value.DeclaredType != "uint16" {
					return r3.Vector{}, false
				}
				switch *value.NativeType {
				case "int", "int32", "int64", "uint32", "uint64":
				default:
					return r3.Vector{}, false
				}
			} else if *value.DeclaredType != "CNetworkedQuantizedFloat" || (*value.NativeType != "float32" && *value.NativeType != "float64") {
				return r3.Vector{}, false
			}
			n, ok := value.Value.(json.Number)
			if value.Status != "observed" || !ok {
				return r3.Vector{}, false
			}
			parsed, err := n.Float64()
			if err != nil {
				return r3.Vector{}, false
			}
			numbers[j] = parsed
			if j == 0 && (parsed < 0 || parsed > 65535 || float64(uint64(parsed)) != parsed) {
				return r3.Vector{}, false
			}
		}
		xyz[i] = (numbers[0]-32)*512 + numbers[1] + numbers[2]
	}
	return r3.Vector{X: xyz[0], Y: xyz[1], Z: xyz[2]}, true
}

func run(source, root string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	var data struct {
		SchemaID string `json:"schema_id"`
		Geometry struct {
			MapName string `json:"map_name"`
		} `json:"geometry"`
		Observations models.SupportVisualObservations `json:"observations"`
	}
	decoder := json.NewDecoder(f)
	decoder.UseNumber()
	if err = decoder.Decode(&data); err != nil {
		return err
	}
	if data.SchemaID != "stratai.support_visual_contact_observations@1" {
		return fmt.Errorf("wrong visual schema")
	}
	manager := maps.NewMapManager(root)
	stdout := os.Stdout
	os.Stdout = os.Stderr
	loadErr := manager.LoadMap(data.Geometry.MapName)
	os.Stdout = stdout
	_ = loadErr // Missing geometry is explicit, not a clear ray.
	checked, clear, blocked, unavailable := 0, 0, 0, 0
	for _, cut := range data.Observations.Cuts {
		if cut.GeometryLoaded != manager.IsLoaded() {
			return fmt.Errorf("visual geometry availability changed")
		}
		for _, ray := range cut.Rays {
			if ray.Observer < 0 || ray.Subject < 0 || ray.Observer >= len(cut.Players) || ray.Subject >= len(cut.Players) {
				return fmt.Errorf("ray index outside current cut")
			}
			a, b := cut.Players[ray.Observer], cut.Players[ray.Subject]
			if a.StateSequence == nil || b.StateSequence == nil || *a.StateSequence < 1 || *b.StateSequence < 1 || *a.StateSequence > len(data.Observations.States) || *b.StateSequence > len(data.Observations.States) {
				return fmt.Errorf("ray state unavailable")
			}
			start, okA := endpoint(a, data.Observations.States[*a.StateSequence-1])
			end, okB := endpoint(b, data.Observations.States[*b.StateSequence-1])
			if !cut.GeometryLoaded || !okA || !okB {
				if ray.Clear != nil {
					return fmt.Errorf("ray with missing input")
				}
				unavailable++
				continue
			}
			value := manager.IsVisible(start, end)
			if ray.Clear == nil || *ray.Clear != value {
				return fmt.Errorf("ray mismatch at source sequence %d", cut.SourceSequence)
			}
			checked++
			if value {
				clear++
			} else {
				blocked++
			}
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"checked": checked, "clear": clear, "blocked": blocked, "unavailable": unavailable, "mismatches": 0})
}
