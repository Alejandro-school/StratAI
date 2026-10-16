package parser

const (
	// SpatialFactContractVersion versions the sidecar boundary independently from
	// the legacy v1 map packages that still contain deprecated review placeholders.
	SpatialFactContractVersion = "2.0.0"

	// SpatialTacticalJudgmentUnavailableReason is intentionally independent from
	// geometry/NAV availability. Tactical labels require a complete T0 decision
	// and physically valid alternatives, neither of which these factual sidecars
	// claim to provide.
	SpatialTacticalJudgmentUnavailableReason = "tactical_decision_context_required"
)

type SpatialTacticalJudgment struct {
	Status      string   `json:"status"`
	ReasonCodes []string `json:"reason_codes"`
}
