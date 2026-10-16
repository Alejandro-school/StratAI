package models

// Public server configuration only. Empty, absent and null values stay distinct.
type SupportRoundConvarEntry struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
}

type SupportRoundConvarMessage struct {
	Sequence int                       `json:"sequence"`
	Order    int                       `json:"order"`
	Frame    int                       `json:"frame"`
	Tick     int                       `json:"tick"`
	Session  int                       `json:"session"`
	Entries  []SupportRoundConvarEntry `json:"entries"`
}

type SupportRoundContextFrame struct {
	SourceSequence  int                           `json:"source_sequence"`
	Frame           int                           `json:"frame"`
	Tick            int                           `json:"tick"`
	Instance        int                           `json:"instance"`
	Epoch           int                           `json:"epoch"`
	Order           int                           `json:"order"`
	Session         int                           `json:"session"`
	MessageSequence int                           `json:"message_sequence"`
	Rules           map[string]SupportClockScalar `json:"rules"`
}

type SupportRoundContextObservations struct {
	Messages []SupportRoundConvarMessage `json:"messages"`
	Frames   []SupportRoundContextFrame  `json:"frames"`
}
