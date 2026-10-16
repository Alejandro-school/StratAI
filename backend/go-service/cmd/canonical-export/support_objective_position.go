package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportObjectivePosition(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportObjectivePositionObservations == nil || ctx.SupportObservations == nil || ctx.SupportObjectiveClockObservations == nil {
		return errors.New("support objective position observations unavailable")
	}
	data := ctx.SupportObjectivePositionObservations
	if len(data.Cuts) == 0 || len(data.Cuts) != len(ctx.SupportObservations.Frames) || len(data.States) == 0 || len(data.Deliveries) == 0 || len(data.Births) == 0 {
		return errors.New("support objective position capture empty")
	}
	for i, cut := range data.Cuts {
		base, clock := ctx.SupportObservations.Frames[i], ctx.SupportObjectiveClockObservations.Frames[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ClockOrder != clock.Order || cut.StateSequence < 1 || cut.StateSequence > len(data.States) || cut.DeliverySequence > len(data.Deliveries) || cut.BirthSequence > len(data.Births) {
			return errors.New("support objective position exact cut mismatch")
		}
	}
	refs := map[string]economyArtifactReference{}
	for key, file := range map[string]string{"support_source": "support_observations.json", "objective_source": "support_objective_observations.json", "c4_source": "support_c4_observations.json", "clock_source": "support_objective_clock_observations.json", "attempt_source": "support_attempt_origin_observations.json"} {
		ref, err := economyFileReference(root, file)
		if err != nil {
			return err
		}
		refs[key] = ref
	}
	return writeCompactJSONReport(filepath.Join(root, "support_objective_position_observations.json"), map[string]any{
		"schema_id": "stratai.support_objective_position_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"references": refs, "capture_contract": "native_object_geometry_deliveries_births_and_support_cuts@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data,
	})
}
