package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportWeaponTiming(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportWeaponTimingObservations == nil || ctx.SupportNativeMotionObservations == nil || ctx.SupportObservations == nil {
		return errors.New("support weapon timing observations unavailable")
	}
	data := ctx.SupportWeaponTimingObservations
	if len(data.Cuts) == 0 || len(data.States) == 0 || len(data.Deliveries) == 0 || len(data.Cuts) != len(ctx.SupportObservations.Frames) || len(data.Cuts) != len(ctx.SupportNativeMotionObservations.Cuts) {
		return errors.New("support weapon timing capture empty or incomplete")
	}
	for i := range data.Cuts {
		cut := &data.Cuts[i]
		base, previous := ctx.SupportObservations.Frames[i], ctx.SupportNativeMotionObservations.Cuts[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ClockOrder != previous.ClockOrder || len(cut.Players) != len(base.Players) || cut.StateSequence > len(data.States) || cut.DeliverySequence > len(data.Deliveries) {
			return errors.New("support weapon timing exact cut mismatch")
		}
		for j := range cut.Players {
			if cut.Players[j].NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" {
				return errors.New("support weapon timing actor mismatch")
			}
			cut.Players[j].ActorAlias = base.Players[j].ActorAlias
		}
	}
	refs := map[string]economyArtifactReference{}
	for key, file := range map[string]string{"support_source": "support_observations.json", "objective_source": "support_objective_observations.json", "c4_source": "support_c4_observations.json", "clock_source": "support_objective_clock_observations.json", "attempt_source": "support_attempt_origin_observations.json", "position_source": "support_objective_position_observations.json", "motion_source": "support_native_motion_observations.json", "physical_source": "support_physical_observations.json"} {
		ref, err := economyFileReference(root, file)
		if err != nil {
			return err
		}
		refs[key] = ref
	}
	return writeCompactJSONReport(filepath.Join(root, "support_weapon_timing_observations.json"), map[string]any{"schema_id": "stratai.support_weapon_timing_observations@1", "contract_version": "1.0.0-candidate.1", "source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue, "references": refs, "capture_contract": "native_weapon_timing_deliveries_and_exact_support_cuts@1", "comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data})
}
