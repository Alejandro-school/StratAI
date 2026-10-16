package models

// Delivery time is retained separately from the sampled support cut. Creation
// replays are not native starts, even when their arming value is true.
type SupportAttemptDelivery struct {
	Sequence int                `json:"sequence"`
	Frame    int                `json:"frame"`
	Tick     int                `json:"tick"`
	Life     string             `json:"entity_life"`
	Kind     string             `json:"kind"`
	Property string             `json:"property"`
	Previous SupportClockScalar `json:"previous"`
	Value    SupportClockScalar `json:"value"`
}

type SupportAttemptItem struct {
	Life            string                        `json:"entity_life"`
	Handle          uint64                        `json:"handle"`
	CreatedFrame    int                           `json:"created_frame"`
	Current         bool                          `json:"current"`
	Fields          map[string]SupportClockScalar `json:"fields"`
	Definition      EconomyEntryScalar            `json:"definition"`
	Owner           SupportC4Reference            `json:"owner"`
	InventoryStatus string                        `json:"inventory_status"`
	InventorySlots  []SupportC4Reference          `json:"inventory_slots"`
}

type SupportAttemptFrame struct {
	Sequence         int                               `json:"sequence"`
	Frame            int                               `json:"frame"`
	Tick             int                               `json:"tick"`
	Instance         int                               `json:"instance"`
	Epoch            int                               `json:"epoch"`
	Phase            string                            `json:"phase"`
	DeliverySequence int                               `json:"delivery_sequence"`
	Network          map[string][]*SupportClockMessage `json:"network"`
	Interval         *SupportClockMessage              `json:"interval"`
	Rules            *SupportClockObject               `json:"rules"`
	Items            []SupportAttemptItem              `json:"items"`
}

type SupportAttemptCut struct {
	SourceSequence   int `json:"source_sequence"`
	Frame            int `json:"frame"`
	Tick             int `json:"tick"`
	Instance         int `json:"instance"`
	Epoch            int `json:"epoch"`
	ClockOrder       int `json:"clock_order"`
	FrameSequence    int `json:"frame_sequence"`
	DeliverySequence int `json:"delivery_sequence"`
}

type SupportAttemptOriginObservations struct {
	Deliveries []SupportAttemptDelivery `json:"deliveries"`
	Frames     []SupportAttemptFrame    `json:"frames"`
	Cuts       []SupportAttemptCut      `json:"cuts"`
}
