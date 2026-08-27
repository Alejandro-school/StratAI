package objective

// State is the current lifecycle state of the C4.
type State string

const (
	StateUnknown  State = "unknown"
	StateCarried  State = "carried"
	StateDropped  State = "dropped"
	StatePlanting State = "planting"
	StatePlanted  State = "planted"
	StateDefusing State = "defusing"
	StateDefused  State = "defused"
	StateExploded State = "exploded"
	StateResolved State = "resolved"
)

// Phase is a compact objective phase suitable for downstream features.
type Phase string

const (
	PhasePreplant Phase = "preplant"
	PhasePlanting Phase = "planting"
	PhasePlanted  Phase = "planted"
	PhaseDefusing Phase = "defusing"
	PhaseResolved Phase = "resolved"
)

// EventType is a causal lifecycle transition emitted by demoinfocs.
type EventType string

const (
	EventCarrierSnapshot EventType = "carrier_snapshot"
	EventDrop            EventType = "drop"
	EventPickup          EventType = "pickup"
	EventPlantStart      EventType = "plant_start"
	EventPlantAbort      EventType = "plant_abort"
	EventPlant           EventType = "plant"
	EventDefuseStart     EventType = "defuse_start"
	EventDefuseAbort     EventType = "defuse_abort"
	EventDefuse          EventType = "defuse"
	EventExplode         EventType = "explode"
)

type AttemptKind string

const (
	AttemptPlant  AttemptKind = "plant"
	AttemptDefuse AttemptKind = "defuse"
)

type AttemptOutcome string

const (
	AttemptInProgress AttemptOutcome = "in_progress"
	AttemptCompleted  AttemptOutcome = "completed"
	AttemptAborted    AttemptOutcome = "aborted"
)

const (
	SourceDemoinfocsEvent          = "demoinfocs_event"
	SourceDemoinfocsNativeSnapshot = "demoinfocs_native_snapshot"

	PositionObserved    = "observed"
	PositionUnavailable = "unavailable"
)

// Position preserves whether coordinates were directly observed and where
// that observation came from. Zero coordinates are valid when Status is observed.
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
	Status string  `json:"status"`
	Source string  `json:"source"`
}

func ObservedPosition(x, y, z float64, source string) Position {
	return Position{X: x, Y: y, Z: z, Status: PositionObserved, Source: source}
}

func UnavailablePosition(source string) Position {
	return Position{Status: PositionUnavailable, Source: source}
}

func (p Position) Available() bool {
	return p.Status == PositionObserved
}

type Actor struct {
	SteamID uint64 `json:"steam_id,string"`
	Name    string `json:"name"`
	Side    string `json:"side"`
}

// EventInput contains only facts observed at the event callback.
type EventInput struct {
	Round    int
	Tick     int
	Actor    Actor
	Site     string
	Position Position
	EntityID *int
	HasKit   *bool
	Source   string
}

// NativeObservation is a demoinfocs GameState.Bomb() observation. It enriches
// current carrier/position without inventing lifecycle transitions.
type NativeObservation struct {
	Round    int
	Tick     int
	Carrier  Actor
	Position Position
}

// Event is the immutable objective ledger record exposed to exporters.
type Event struct {
	ID                   string         `json:"id"`
	Sequence             uint64         `json:"sequence"`
	Type                 EventType      `json:"event_type"`
	Round                int            `json:"round"`
	Tick                 int            `json:"tick"`
	Actor                Actor          `json:"actor"`
	Site                 string         `json:"site"`
	Position             Position       `json:"position"`
	EntityID             *int           `json:"entity_id"`
	HasKit               *bool          `json:"has_kit"`
	Source               string         `json:"source"`
	StateAfter           State          `json:"state_after"`
	PhaseAfter           Phase          `json:"phase_after"`
	AttemptID            string         `json:"attempt_id"`
	AttemptSequence      uint64         `json:"attempt_sequence"`
	AttemptOutcome       AttemptOutcome `json:"attempt_outcome"`
	AttemptStartObserved bool           `json:"attempt_start_observed"`
	RelatedEventID       string         `json:"related_event_id"`
	DurationTicks        *int           `json:"duration_ticks"`
}

// Attempt links begin/abort/complete events without inferring unobserved time.
type Attempt struct {
	ID            string         `json:"id"`
	Sequence      uint64         `json:"sequence"`
	Kind          AttemptKind    `json:"kind"`
	Round         int            `json:"round"`
	Actor         Actor          `json:"actor"`
	Site          string         `json:"site"`
	HasKit        *bool          `json:"has_kit"`
	StartTick     int            `json:"start_tick"`
	EndTick       *int           `json:"end_tick"`
	DurationTicks *int           `json:"duration_ticks"`
	Outcome       AttemptOutcome `json:"outcome"`
	StartEventID  string         `json:"start_event_id"`
	EndEventID    string         `json:"end_event_id"`
	StartObserved bool           `json:"start_observed"`
}

// Snapshot separates current state from historical round facts.
type Snapshot struct {
	Round                 int       `json:"round"`
	Tick                  int       `json:"tick"`
	State                 State     `json:"state"`
	Phase                 Phase     `json:"objective_phase"`
	IsPlantedNow          bool      `json:"is_planted_now"`
	WasPlantedThisRound   bool      `json:"was_planted_this_round"`
	Carrier               Actor     `json:"carrier"`
	PlantingPlayer        Actor     `json:"planting_player"`
	Defuser               Actor     `json:"defuser"`
	Site                  string    `json:"site"`
	PlantTick             int       `json:"plant_tick"`
	Position              Position  `json:"position"`
	ActivePlantAttemptID  string    `json:"active_plant_attempt_id"`
	ActiveDefuseAttemptID string    `json:"active_defuse_attempt_id"`
	Resolution            EventType `json:"resolution"`
	ResolutionTick        int       `json:"resolution_tick"`
}

type RoundSummary struct {
	Round              int       `json:"round"`
	RoundStartTick     int       `json:"round_start_tick"`
	WasPlanted         bool      `json:"was_planted"`
	Site               string    `json:"site"`
	PlantTick          int       `json:"plant_tick"`
	Resolution         EventType `json:"resolution"`
	ResolutionTick     int       `json:"resolution_tick"`
	ResolvedAtRoundEnd bool      `json:"resolved_at_round_end"`
	RoundEndTick       int       `json:"round_end_tick"`
	FinalState         State     `json:"final_state"`
	EventCount         int       `json:"event_count"`
	AttemptCount       int       `json:"attempt_count"`
}
