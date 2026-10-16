package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	mapspkg "cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

const (
	requestSchemaID = "stratai.spatial_review_route_trace_request@1"
	outputSchemaID  = "stratai.spatial_review_route_traces@1"
)

var traceIDPattern = regexp.MustCompile(`^trace-[0-9a-f]{24}$`)

type vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type routeRequest struct {
	TraceID     string `json:"trace_id"`
	Origin      vector `json:"origin_world"`
	Destination vector `json:"destination_world"`
}

type request struct {
	SchemaID string         `json:"schema_id"`
	MapID    string         `json:"map_id"`
	Routes   []routeRequest `json:"routes"`
}

type measurement struct {
	Reachable                        *bool    `json:"reachable"`
	DirectDistanceWorldUnits         float64  `json:"direct_distance_world_units"`
	PathDistanceWorldUnits           *float64 `json:"path_distance_world_units"`
	OriginAssociationWorldUnits      *float64 `json:"origin_association_world_units"`
	DestinationAssociationWorldUnits *float64 `json:"destination_association_world_units"`
	UnavailableReason                string   `json:"unavailable_reason,omitempty"`
}

type routeTrace struct {
	TraceID     string      `json:"trace_id"`
	Origin      vector      `json:"origin_world"`
	Destination vector      `json:"destination_world"`
	Measurement measurement `json:"measurement"`
	PathPoints  []vector    `json:"path_points_world"`
}

type output struct {
	SchemaID              string       `json:"schema_id"`
	ContractVersion       string       `json:"contract_version"`
	MapID                 string       `json:"map_id"`
	GraphContractVersion  string       `json:"graph_contract_version"`
	ShortestPathAlgorithm string       `json:"shortest_path_algorithm"`
	NavAssetSHA256        string       `json:"nav_asset_sha256"`
	Routes                []routeTrace `json:"routes"`
}

func main() {
	var requestPath, mapsRoot, outputPath string
	flag.StringVar(&requestPath, "request", "", "route trace request JSON")
	flag.StringVar(&mapsRoot, "maps-root", "", "read-only map asset root")
	flag.StringVar(&outputPath, "output", "", "new route trace JSON path under system temp")
	flag.Parse()
	if flag.NArg() != 0 || requestPath == "" || mapsRoot == "" || outputPath == "" {
		fmt.Fprintln(os.Stderr, "--request, --maps-root and --output are required; positional arguments are forbidden")
		os.Exit(2)
	}
	if err := run(requestPath, mapsRoot, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "build spatial review paths: %v\n", err)
		os.Exit(1)
	}
}

func run(requestPath, mapsRoot, outputPath string) error {
	if err := validateNewTemporaryOutput(outputPath); err != nil {
		return err
	}
	payload, err := os.ReadFile(requestPath)
	if err != nil {
		return err
	}
	var input request
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request contains trailing JSON")
	}
	if input.SchemaID != requestSchemaID || input.MapID == "" || len(input.Routes) == 0 {
		return errors.New("request identity or routes are invalid")
	}
	seen := map[string]struct{}{}
	for _, route := range input.Routes {
		if !traceIDPattern.MatchString(route.TraceID) {
			return errors.New("request contains invalid trace id")
		}
		if _, exists := seen[route.TraceID]; exists {
			return errors.New("request contains duplicate trace id")
		}
		seen[route.TraceID] = struct{}{}
	}

	manager := mapspkg.NewMapManager(mapsRoot)
	if err := manager.LoadMap(input.MapID); err != nil {
		return fmt.Errorf("load map: %w", err)
	}
	graph, err := manager.NavigationGraphSnapshot()
	if err != nil || graph == nil {
		return errors.New("navigation graph unavailable")
	}
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return err
	}
	result := output{
		SchemaID:              outputSchemaID,
		ContractVersion:       "1.0.0",
		MapID:                 input.MapID,
		GraphContractVersion:  navigation.GraphContractVersion,
		ShortestPathAlgorithm: navigation.ShortestPathAlgorithm,
		NavAssetSHA256:        graph.AssetSHA256,
		Routes:                make([]routeTrace, 0, len(input.Routes)),
	}
	for _, route := range input.Routes {
		trace, err := prepared.EvaluatePath(toR3(route.Origin), toR3(route.Destination))
		if err != nil {
			return fmt.Errorf("trace %s: %w", route.TraceID, err)
		}
		points := make([]vector, 0, len(trace.Points))
		for _, point := range trace.Points {
			points = append(points, fromR3(point))
		}
		result.Routes = append(result.Routes, routeTrace{
			TraceID:     route.TraceID,
			Origin:      route.Origin,
			Destination: route.Destination,
			Measurement: measurement{
				Reachable:                        trace.Measurement.Reachable,
				DirectDistanceWorldUnits:         trace.Measurement.DirectDistance,
				PathDistanceWorldUnits:           trace.Measurement.PathDistance,
				OriginAssociationWorldUnits:      trace.Measurement.OriginAssociationDistance,
				DestinationAssociationWorldUnits: trace.Measurement.TargetAssociationDistance,
				UnavailableReason:                trace.Measurement.UnavailableReason,
			},
			PathPoints: points,
		})
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func validateNewTemporaryOutput(outputPath string) error {
	if _, err := os.Lstat(outputPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return errors.New("output must not already exist")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(outputPath))
	if err != nil {
		return err
	}
	temporaryRoot, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(temporaryRoot, parent)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("output parent must be a dedicated directory under system temp")
	}
	return nil
}

func toR3(value vector) r3.Vector   { return r3.Vector{X: value.X, Y: value.Y, Z: value.Z} }
func fromR3(value r3.Vector) vector { return vector{X: value.X, Y: value.Y, Z: value.Z} }
