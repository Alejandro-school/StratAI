package models

// Trace sequences are local observer delivery order, never server subtick order.
// Null actor/object fields are not filled later from end-of-match knowledge.
type EconomyTraceRow struct {
	Sequence    int     `json:"sequence"`
	Tick        int     `json:"tick"`
	Frame       int     `json:"frame"`
	Channel     string  `json:"channel"`
	Source      string  `json:"source"`
	ActorAlias  *string `json:"actor_alias"`
	Side        *string `json:"side"`
	EntityAlias *string `json:"entity_alias"`
	Instance    *int    `json:"instance"`
	Value       any     `json:"value"`
}

type EconomyTraceField struct {
	Ordinal        int     `json:"ordinal"`
	Name           *string `json:"name"`
	DescriptorType *int32  `json:"descriptor_type"`
	WireType       *int32  `json:"wire_type"`
	Status         string  `json:"status"`
	Value          any     `json:"value"`
}

type EconomyTransactionTrace struct {
	Rows                  []EconomyTraceRow `json:"rows"`
	UnmappedNativeEvents  int               `json:"unmapped_native_events"`
	NativeDescriptorLists int               `json:"native_descriptor_lists"`
}
