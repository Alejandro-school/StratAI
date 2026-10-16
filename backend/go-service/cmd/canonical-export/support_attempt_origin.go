package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportAttemptOrigin(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportAttemptOriginObservations == nil || ctx.SupportObjectiveClockObservations == nil || ctx.SupportObservations == nil {
		return errors.New("support attempt origin observations unavailable")
	}
	data := ctx.SupportAttemptOriginObservations
	if len(data.Cuts) == 0 || len(data.Cuts) != len(ctx.SupportObservations.Frames) || len(data.Frames) == 0 || len(data.Deliveries) == 0 {
		return errors.New("support attempt origin capture empty")
	}
	for i, cut := range data.Cuts {
		base, clock := ctx.SupportObservations.Frames[i], ctx.SupportObjectiveClockObservations.Frames[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ClockOrder != clock.Order || cut.FrameSequence > len(data.Frames) || cut.DeliverySequence > len(data.Deliveries) {
			return errors.New("support attempt origin exact cut mismatch")
		}
	}
	refs := map[string]economyArtifactReference{}
	for key, file := range map[string]string{"support_source": "support_observations.json", "objective_source": "support_objective_observations.json", "c4_source": "support_c4_observations.json", "clock_source": "support_objective_clock_observations.json"} {
		ref, err := economyFileReference(root, file)
		if err != nil {
			return err
		}
		refs[key] = ref
	}
	return writeCompactJSONReport(filepath.Join(root, "support_attempt_origin_observations.json"), map[string]any{
		"schema_id": "stratai.support_attempt_origin_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"references": refs, "capture_contract": "native_arming_delivery_and_every_active_FrameDone@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data,
	})
}
