// Offline geometric diagnostics for exact support-anchor endpoints. No demo
// parsing, output paths, clocks, identities, enemies, or tactical scores here.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

type query struct {
	Origin *point `json:"origin"`
	Target *point `json:"target"`
}
type point struct {
	X *float64
	Y *float64
	Z *float64
}

func (p *point) vector() r3.Vector {
	return r3.Vector{X: *p.X, Y: *p.Y, Z: *p.Z}
}

type request struct {
	MapsRoot    string  `json:"maps_root"`
	MapName     string  `json:"map_name"`
	AssetSHA256 string  `json:"asset_sha256"`
	Queries     []query `json:"queries"`
}

func run(input io.Reader, output io.Writer) error {
	var req request
	decoder := json.NewDecoder(io.LimitReader(input, 64*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing request data")
	}
	if !regexp.MustCompile(`^de_[a-z0-9_]+$`).MatchString(req.MapName) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(req.AssetSHA256) || len(req.Queries) > 100000 {
		return fmt.Errorf("invalid support navigation request")
	}
	for _, q := range req.Queries {
		for _, endpoint := range []*point{q.Origin, q.Target} {
			if endpoint == nil || endpoint.X == nil || endpoint.Y == nil || endpoint.Z == nil {
				return fmt.Errorf("missing exact endpoint component")
			}
		}
	}
	reference := req.MapName + "/" + req.MapName + ".nav"
	graph, err := maps.LoadNavigationCandidate(filepath.Join(req.MapsRoot, filepath.FromSlash(reference)), reference, req.AssetSHA256)
	if err != nil {
		return err
	}
	evaluator, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return err
	}
	rows := make([]navigation.PathTrace, 0, len(req.Queries))
	for _, q := range req.Queries {
		row, err := evaluator.EvaluatePath(q.Origin.vector(), q.Target.vector())
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return json.NewEncoder(output).Encode(map[string]any{"graph": graph, "results": rows})
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
