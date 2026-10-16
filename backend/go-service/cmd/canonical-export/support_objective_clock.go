package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportObjectiveClock(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportObjectiveClockObservations == nil || ctx.SupportC4Observations == nil || ctx.SupportObservations == nil || ctx.SupportObjectiveObservations == nil {
		return errors.New("support objective clock observations unavailable")
	}
	data := ctx.SupportObjectiveClockObservations
	if len(data.Frames) == 0 || len(data.Frames) != len(ctx.SupportObservations.Frames) || len(data.Frames) != len(ctx.SupportObjectiveObservations.Frames) || len(data.Frames) != len(ctx.SupportC4Observations.Cuts) {
		return errors.New("support objective clock cuts unavailable")
	}
	for i := range data.Frames {
		cut := &data.Frames[i]
		base, objective, c4 := ctx.SupportObservations.Frames[i], ctx.SupportObjectiveObservations.Frames[i], ctx.SupportC4Observations.Cuts[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ObjectiveOrder != objective.Order || cut.C4StateSequence != c4.StateSequence || len(cut.Players) != len(base.Players) {
			return errors.New("support objective clock exact cut mismatch")
		}
		for j := range cut.Players {
			p := &cut.Players[j]
			if p.NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" {
				return errors.New("support objective clock actor mismatch")
			}
			p.ActorAlias = base.Players[j].ActorAlias
		}
	}
	refs := map[string]economyArtifactReference{}
	if ctx.SupportRoundContextObservations != nil {
		if err := writeSupportRoundContext(ctx, opts, root); err != nil {
			return err
		}
	}
	for key, file := range map[string]string{"support_source": "support_observations.json", "objective_source": "support_objective_observations.json", "c4_source": "support_c4_observations.json"} {
		ref, err := economyFileReference(root, file)
		if err != nil {
			return err
		}
		refs[key] = ref
	}
	return writeCompactJSONReport(filepath.Join(root, "support_objective_clock_observations.json"), map[string]any{
		"schema_id": "stratai.support_objective_clock_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"references": refs, "capture_contract": "same_support_FrameDone_native_game_clocks@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data,
	})
}
