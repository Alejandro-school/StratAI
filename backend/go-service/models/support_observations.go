package models

// Support snapshots are opt-in sidecar data, never part of replay/3.8 JSON.
type SupportPlayer struct {
	NativeActor    uint64           `json:"-"`
	ActorAlias     string           `json:"actor_alias"`
	Side           string           `json:"side"`
	Connected      bool             `json:"connected"`
	PawnHandle     *uint64          `json:"pawn_handle"`
	PawnLife       *string          `json:"pawn_life"`
	IdentityStatus string           `json:"identity_status"`
	Position       *SupportPosition `json:"position"`
	PositionStatus string           `json:"position_status"`
	Alive          *bool            `json:"alive"`
	AliveStatus    string           `json:"alive_status"`
}

type SupportPosition struct {
	X int     `json:"x"`
	Y int     `json:"y"`
	Z float64 `json:"z"`
}

type SupportFrame struct {
	Sequence    int             `json:"sequence"`
	Frame       int             `json:"frame"`
	Instance    int             `json:"instance"`
	RoundNumber int             `json:"round_number"`
	Tick        int             `json:"tick"`
	Epoch       int             `json:"epoch"`
	SampleStep  int             `json:"sample_step"`
	TickRate    *float64        `json:"tick_rate"`
	Phase       string          `json:"phase"`
	Players     []SupportPlayer `json:"players"`
}

type SupportObservations struct {
	Frames []SupportFrame `json:"frames"`
}
