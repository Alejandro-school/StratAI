package engagement

import (
	"math"

	"cs2-demo-service/models"
)

const (
	AlgorithmVersion             = "engagement_causal@2"
	TradeAlgorithmVersion        = "trade_response@2"
	PairContinuationWindowMS     = 1500
	MultiTargetWindowMS          = 750
	MaxEngagementDurationMS      = 5000
	AggressorPreludeWindowMS     = 500
	TradeWindowMS                = 5000
	TradeMaxDistanceWorldUnits   = 1250.0
	TradeAssumedMovementSpeedUPS = 250.0
	TradeMaxFacingDeltaDeg       = 100.0
	PeekVelocityThresholdUPS     = 100.0
)

func ticksForMilliseconds(milliseconds int, tickRate float64) int {
	if milliseconds <= 0 || tickRate <= 0 {
		return 0
	}
	return int(math.Ceil(float64(milliseconds) * tickRate / 1000.0))
}

func engagementConfig(tickRate float64) models.CanonicalEngagementConfig {
	return models.CanonicalEngagementConfig{
		AlgorithmVersion:          AlgorithmVersion,
		TickRateHz:                tickRate,
		PairContinuationWindowMS:  PairContinuationWindowMS,
		PairContinuationTicks:     ticksForMilliseconds(PairContinuationWindowMS, tickRate),
		MultiTargetWindowMS:       MultiTargetWindowMS,
		MultiTargetWindowTicks:    ticksForMilliseconds(MultiTargetWindowMS, tickRate),
		MaxEngagementDurationMS:   MaxEngagementDurationMS,
		MaxEngagementDurationTick: ticksForMilliseconds(MaxEngagementDurationMS, tickRate),
		AggressorPreludeWindowMS:  AggressorPreludeWindowMS,
		AggressorPreludeTicks:     ticksForMilliseconds(AggressorPreludeWindowMS, tickRate),
	}
}

func tradeConfig(tickRate float64) models.CanonicalTradeConfig {
	return models.CanonicalTradeConfig{
		AlgorithmVersion:             TradeAlgorithmVersion,
		TickRateHz:                   tickRate,
		TradeWindowMS:                TradeWindowMS,
		TradeWindowTicks:             ticksForMilliseconds(TradeWindowMS, tickRate),
		MaxDistanceWorldUnits:        TradeMaxDistanceWorldUnits,
		AssumedMovementSpeedWorldUPS: TradeAssumedMovementSpeedUPS,
		MaxFacingDeltaDeg:            TradeMaxFacingDeltaDeg,
		PhysicalEvidenceRequirement:  "alive+distance+connection_time+physics_mesh_los+orientation",
	}
}
