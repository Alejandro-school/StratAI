// Package navigation derives deterministic, factual path measurements from a
// directed navigation-area graph. It deliberately knows nothing about tactical
// labels, player intent, outcomes, raycasts, or future replay state.
package navigation

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/golang/geo/r3"
)

const (
	GraphContractVersion             = "nav_area_connections_with_polygons@2"
	ShortestPathAlgorithm            = "directed_dijkstra_stable_ties@1"
	EndpointAssociationMethod        = "containing_nav_polygon_xy_then_nearest_center_3d@2"
	EdgeWeightMethod                 = "nav_area_center_euclidean_world_units@1"
	DefaultEndpointTolerance         = 96.0
	DefaultEndpointVerticalTolerance = 128.0
)

// Node is an internal navigation-area vertex. Its ID is never emitted as a
// model feature; it exists only to join connections while calculating a path.
type Node struct {
	ID       uint32
	Position r3.Vector
	Corners  []r3.Vector
}

// Edge is a directed, positively weighted navigation-area connection.
type Edge struct {
	FromID uint32
	ToID   uint32
	Weight float64
}

// Graph is an immutable-by-convention snapshot of one loaded NAV asset.
type Graph struct {
	Version          string
	AssetReference   string
	AssetSHA256      string
	AvailabilityTick int
	Nodes            []Node
	Edges            []Edge
}

// Config versions the physical assumptions used by Evaluate.
type Config struct {
	EndpointTolerance         float64
	EndpointVerticalTolerance float64
}

// Measurement keeps direct distance separate from graph reachability and path
// distance. Timing scenarios belong to the causal producer, not the graph.
type Measurement struct {
	DirectDistance            float64
	Reachable                 *bool
	PathDistance              *float64
	OriginAssociationDistance *float64
	TargetAssociationDistance *float64
	UnavailableReason         string
}

// PathTrace is a reviewer-facing geometric trace of the same directed path
// measured by Evaluate. Points contain the observed origin, the stable
// shortest-path NAV area centres, and the requested target. It does not carry
// tactical meaning, safety, timing, or a recommendation.
type PathTrace struct {
	Measurement Measurement
	Points      []r3.Vector
}

type shortestPathCacheKey struct {
	originID uint32
	targetID uint32
}

type shortestPathCacheValue struct {
	distance float64
	found    bool
}

// PreparedEvaluator is an immutable graph index plus a directed shortest-path
// cache. It is safe for concurrent use. Preparing once avoids rebuilding and
// re-sorting the same NAV graph for every replay-frame query.
type PreparedEvaluator struct {
	graphAvailable bool
	nodes          map[uint32]Node
	nodeIDs        []uint32
	adjacency      map[uint32][]adjacencyEdge
	config         Config

	cacheMu           sync.RWMutex
	shortestPathCache map[shortestPathCacheKey]shortestPathCacheValue
}

type adjacencyEdge struct {
	to     uint32
	weight float64
}

type queueItem struct {
	id       uint32
	distance float64
	index    int
}

type priorityQueue []*queueItem

func (q priorityQueue) Len() int { return len(q) }
func (q priorityQueue) Less(i, j int) bool {
	if q[i].distance != q[j].distance {
		return q[i].distance < q[j].distance
	}
	return q[i].id < q[j].id
}
func (q priorityQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}
func (q *priorityQueue) Push(value any) {
	item := value.(*queueItem)
	item.index = len(*q)
	*q = append(*q, item)
}
func (q *priorityQueue) Pop() any {
	old := *q
	last := len(old) - 1
	item := old[last]
	old[last] = nil
	item.index = -1
	*q = old[:last]
	return item
}

// DefaultConfig returns the explicit endpoint-association assumption.
func DefaultConfig() Config {
	return Config{
		EndpointTolerance:         DefaultEndpointTolerance,
		EndpointVerticalTolerance: DefaultEndpointVerticalTolerance,
	}
}

// NewPreparedEvaluator validates and defensively indexes a graph once. A nil
// or empty graph is a valid prepared evaluator that returns the same factual
// unavailable measurement as Evaluate.
func NewPreparedEvaluator(graph *Graph, config Config) (*PreparedEvaluator, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	prepared := &PreparedEvaluator{
		config:            config,
		shortestPathCache: make(map[shortestPathCacheKey]shortestPathCacheValue),
	}
	if graph == nil || len(graph.Nodes) == 0 {
		return prepared, nil
	}

	nodes, adjacency, err := validateAndIndex(graph)
	if err != nil {
		return nil, err
	}
	prepared.graphAvailable = true
	prepared.nodes = nodes
	prepared.adjacency = adjacency
	prepared.nodeIDs = sortedNodeIDs(nodes)
	return prepared, nil
}

// Config returns the endpoint-association assumptions captured at prepare
// time. Config contains only values and cannot mutate the evaluator.
func (evaluator *PreparedEvaluator) Config() Config {
	if evaluator == nil {
		return Config{}
	}
	return evaluator.config
}

// Evaluate derives one measurement from the prepared immutable graph index.
func (evaluator *PreparedEvaluator) Evaluate(origin, target r3.Vector) (Measurement, error) {
	if evaluator == nil {
		return Measurement{}, fmt.Errorf("navigation prepared evaluator is nil")
	}
	if err := validateVector(origin, "origin"); err != nil {
		return Measurement{}, err
	}
	if err := validateVector(target, "target"); err != nil {
		return Measurement{}, err
	}

	measurement := Measurement{DirectDistance: distance(origin, target)}
	if !evaluator.graphAvailable {
		measurement.UnavailableReason = "nav_graph_unavailable"
		return measurement, nil
	}

	originID, originDistance, originFound := associateNode(
		evaluator.nodes,
		evaluator.nodeIDs,
		origin,
		evaluator.config,
	)
	targetID, targetDistance, targetFound := associateNode(
		evaluator.nodes,
		evaluator.nodeIDs,
		target,
		evaluator.config,
	)
	if !originFound || !targetFound {
		switch {
		case !originFound && !targetFound:
			measurement.UnavailableReason = "both_endpoints_outside_nav_tolerance"
		case !originFound:
			measurement.UnavailableReason = "origin_outside_nav_tolerance"
		default:
			measurement.UnavailableReason = "target_outside_nav_tolerance"
		}
		return measurement, nil
	}
	measurement.OriginAssociationDistance = float64Pointer(originDistance)
	measurement.TargetAssociationDistance = float64Pointer(targetDistance)

	graphDistance, found := evaluator.shortestDistance(originID, targetID)
	if !found {
		measurement.Reachable = boolPointer(false)
		measurement.UnavailableReason = "no_navigable_connection"
		return measurement, nil
	}

	pathDistance := originDistance + graphDistance + targetDistance
	measurement.Reachable = boolPointer(true)
	measurement.PathDistance = float64Pointer(pathDistance)
	return measurement, nil
}

// EvaluatePath returns the factual measurement plus a deterministic polyline
// suitable for displaying the route that produced the distance. Unavailable
// or unreachable measurements return an empty point list.
func (evaluator *PreparedEvaluator) EvaluatePath(origin, target r3.Vector) (PathTrace, error) {
	measurement, err := evaluator.Evaluate(origin, target)
	if err != nil {
		return PathTrace{}, err
	}
	if measurement.Reachable == nil || !*measurement.Reachable {
		return PathTrace{Measurement: measurement, Points: []r3.Vector{}}, nil
	}

	originID, _, originFound := associateNode(
		evaluator.nodes,
		evaluator.nodeIDs,
		origin,
		evaluator.config,
	)
	targetID, _, targetFound := associateNode(
		evaluator.nodes,
		evaluator.nodeIDs,
		target,
		evaluator.config,
	)
	if !originFound || !targetFound {
		return PathTrace{}, errors.New("navigation trace association disagrees with measurement")
	}
	_, nodeIDs, found := shortestPath(originID, targetID, evaluator.adjacency)
	if !found {
		return PathTrace{}, errors.New("navigation trace reachability disagrees with measurement")
	}
	points := make([]r3.Vector, 0, len(nodeIDs)+2)
	points = append(points, origin)
	for _, id := range nodeIDs {
		points = append(points, evaluator.nodes[id].Position)
	}
	points = append(points, target)
	return PathTrace{Measurement: measurement, Points: compactConsecutivePoints(points)}, nil
}

func (evaluator *PreparedEvaluator) shortestDistance(originID, targetID uint32) (float64, bool) {
	key := shortestPathCacheKey{originID: originID, targetID: targetID}
	evaluator.cacheMu.RLock()
	cached, exists := evaluator.shortestPathCache[key]
	evaluator.cacheMu.RUnlock()
	if exists {
		return cached.distance, cached.found
	}

	distance, found := shortestDistance(originID, targetID, evaluator.adjacency)
	evaluator.cacheMu.Lock()
	if concurrent, alreadyCached := evaluator.shortestPathCache[key]; alreadyCached {
		distance, found = concurrent.distance, concurrent.found
	} else {
		evaluator.shortestPathCache[key] = shortestPathCacheValue{distance: distance, found: found}
	}
	evaluator.cacheMu.Unlock()
	return distance, found
}

// Evaluate computes the shortest directed graph path with stable tie-breaking.
// A nil/empty graph or an endpoint outside tolerance is unavailable. A valid
// graph with no connection produces reachable=false and nil path/ETA values.
func Evaluate(graph *Graph, origin, target r3.Vector, config Config) (Measurement, error) {
	if err := validateVector(origin, "origin"); err != nil {
		return Measurement{}, err
	}
	if err := validateVector(target, "target"); err != nil {
		return Measurement{}, err
	}
	if err := validateConfig(config); err != nil {
		return Measurement{}, err
	}
	prepared, err := NewPreparedEvaluator(graph, config)
	if err != nil {
		return Measurement{}, err
	}
	return prepared.Evaluate(origin, target)
}

func validateConfig(config Config) error {
	if !finitePositive(config.EndpointTolerance) {
		return fmt.Errorf("navigation endpoint tolerance must be finite and positive")
	}
	if !finitePositive(config.EndpointVerticalTolerance) {
		return fmt.Errorf("navigation endpoint vertical tolerance must be finite and positive")
	}
	return nil
}

func validateAndIndex(graph *Graph) (map[uint32]Node, map[uint32][]adjacencyEdge, error) {
	nodes := make(map[uint32]Node, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node.ID == 0 {
			return nil, nil, fmt.Errorf("navigation graph contains zero node id")
		}
		if _, exists := nodes[node.ID]; exists {
			return nil, nil, fmt.Errorf("navigation graph contains duplicate node id %d", node.ID)
		}
		if err := validateVector(node.Position, fmt.Sprintf("node %d", node.ID)); err != nil {
			return nil, nil, err
		}
		for index, corner := range node.Corners {
			if err := validateVector(corner, fmt.Sprintf("node %d corner %d", node.ID, index)); err != nil {
				return nil, nil, err
			}
		}
		node.Corners = append([]r3.Vector(nil), node.Corners...)
		nodes[node.ID] = node
	}

	adjacency := make(map[uint32][]adjacencyEdge, len(nodes))
	for id := range nodes {
		adjacency[id] = []adjacencyEdge{}
	}
	seenEdges := make(map[[2]uint32]struct{}, len(graph.Edges))
	for _, edge := range graph.Edges {
		if _, ok := nodes[edge.FromID]; !ok {
			return nil, nil, fmt.Errorf("navigation edge references missing from node %d", edge.FromID)
		}
		if _, ok := nodes[edge.ToID]; !ok {
			return nil, nil, fmt.Errorf("navigation edge references missing to node %d", edge.ToID)
		}
		if !finitePositive(edge.Weight) {
			return nil, nil, fmt.Errorf("navigation edge %d -> %d weight must be finite and positive", edge.FromID, edge.ToID)
		}
		key := [2]uint32{edge.FromID, edge.ToID}
		if _, exists := seenEdges[key]; exists {
			return nil, nil, fmt.Errorf("navigation graph contains duplicate edge %d -> %d", edge.FromID, edge.ToID)
		}
		seenEdges[key] = struct{}{}
		adjacency[edge.FromID] = append(adjacency[edge.FromID], adjacencyEdge{to: edge.ToID, weight: edge.Weight})
	}
	for id := range adjacency {
		sort.Slice(adjacency[id], func(i, j int) bool {
			if adjacency[id][i].to != adjacency[id][j].to {
				return adjacency[id][i].to < adjacency[id][j].to
			}
			return adjacency[id][i].weight < adjacency[id][j].weight
		})
	}
	return nodes, adjacency, nil
}

func sortedNodeIDs(nodes map[uint32]Node) []uint32 {
	ids := make([]uint32, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func associateNode(
	nodes map[uint32]Node,
	ids []uint32,
	point r3.Vector,
	config Config,
) (uint32, float64, bool) {
	containedID := uint32(0)
	containedZDistance := math.Inf(1)
	for _, id := range ids {
		node := nodes[id]
		if len(node.Corners) < 3 || !pointInPolygonXY(point, node.Corners) {
			continue
		}
		zDistance := math.Abs(point.Z - node.Position.Z)
		if zDistance <= config.EndpointVerticalTolerance &&
			(zDistance < containedZDistance || (zDistance == containedZDistance && id < containedID)) {
			containedID = id
			containedZDistance = zDistance
		}
	}
	if containedID != 0 {
		return containedID, distance(point, nodes[containedID].Position), true
	}

	bestID := uint32(0)
	bestDistance := math.Inf(1)
	for _, id := range ids {
		candidate := distance(point, nodes[id].Position)
		if candidate < bestDistance {
			bestID = id
			bestDistance = candidate
		}
	}
	return bestID, bestDistance, bestDistance <= config.EndpointTolerance
}

func pointInPolygonXY(point r3.Vector, polygon []r3.Vector) bool {
	inside := false
	previous := len(polygon) - 1
	for current := 0; current < len(polygon); current++ {
		left, right := polygon[current], polygon[previous]
		if pointOnSegmentXY(point, left, right) {
			return true
		}
		if (left.Y > point.Y) != (right.Y > point.Y) &&
			point.X < (right.X-left.X)*(point.Y-left.Y)/(right.Y-left.Y)+left.X {
			inside = !inside
		}
		previous = current
	}
	return inside
}

func pointOnSegmentXY(point, left, right r3.Vector) bool {
	cross := (point.Y-left.Y)*(right.X-left.X) - (point.X-left.X)*(right.Y-left.Y)
	if math.Abs(cross) > 1e-9 {
		return false
	}
	return point.X >= math.Min(left.X, right.X)-1e-9 &&
		point.X <= math.Max(left.X, right.X)+1e-9 &&
		point.Y >= math.Min(left.Y, right.Y)-1e-9 &&
		point.Y <= math.Max(left.Y, right.Y)+1e-9
}

func shortestDistance(originID, targetID uint32, adjacency map[uint32][]adjacencyEdge) (float64, bool) {
	distance, _, found := shortestPath(originID, targetID, adjacency)
	return distance, found
}

func shortestPath(originID, targetID uint32, adjacency map[uint32][]adjacencyEdge) (float64, []uint32, bool) {
	if originID == targetID {
		return 0, []uint32{originID}, true
	}
	distances := map[uint32]float64{originID: 0}
	previous := map[uint32]uint32{}
	queue := priorityQueue{&queueItem{id: originID, distance: 0}}
	heap.Init(&queue)
	for queue.Len() > 0 {
		current := heap.Pop(&queue).(*queueItem)
		known, ok := distances[current.id]
		if !ok || current.distance != known {
			continue
		}
		if current.id == targetID {
			path := []uint32{targetID}
			for path[len(path)-1] != originID {
				parent, ok := previous[path[len(path)-1]]
				if !ok {
					return 0, nil, false
				}
				path = append(path, parent)
			}
			for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
				path[left], path[right] = path[right], path[left]
			}
			return current.distance, path, true
		}
		for _, edge := range adjacency[current.id] {
			candidate := current.distance + edge.weight
			knownTarget, seen := distances[edge.to]
			if !seen || candidate < knownTarget {
				distances[edge.to] = candidate
				previous[edge.to] = current.id
				heap.Push(&queue, &queueItem{id: edge.to, distance: candidate})
			}
		}
	}
	return 0, nil, false
}

func compactConsecutivePoints(points []r3.Vector) []r3.Vector {
	result := make([]r3.Vector, 0, len(points))
	for _, point := range points {
		if len(result) == 0 || result[len(result)-1] != point {
			result = append(result, point)
		}
	}
	return result
}

func validateVector(value r3.Vector, name string) error {
	if !finite(value.X) || !finite(value.Y) || !finite(value.Z) {
		return fmt.Errorf("navigation %s position must be finite", name)
	}
	return nil
}

func distance(left, right r3.Vector) float64 {
	delta := left.Sub(right)
	return math.Sqrt(delta.Dot(delta))
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finitePositive(value float64) bool {
	return finite(value) && value > 0
}

func boolPointer(value bool) *bool          { return &value }
func float64Pointer(value float64) *float64 { return &value }
