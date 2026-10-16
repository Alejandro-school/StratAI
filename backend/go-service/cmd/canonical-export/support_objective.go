package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportObjective(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportObjectiveObservations == nil || ctx.SupportObservations == nil || len(ctx.SupportObjectiveObservations.Frames) == 0 || len(ctx.SupportObjectiveObservations.Frames) != len(ctx.SupportObservations.Frames) {
		return errors.New("support objective observations unavailable")
	}
	source, err := economyFileReference(root, "support_observations.json")
	if err != nil {
		return err
	}
	for i := range ctx.SupportObjectiveObservations.Frames {
		f := &ctx.SupportObjectiveObservations.Frames[i]
		base := ctx.SupportObservations.Frames[i]
		if f.SourceSequence != base.Sequence || f.Frame != base.Frame || f.Tick != base.Tick || f.Instance != base.Instance || f.Epoch != base.Epoch || len(f.Players) != len(base.Players) {
			return errors.New("support objective frame mismatch")
		}
		for j := range f.Players {
			if f.Players[j].NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" {
				return errors.New("support objective actor mismatch")
			}
			f.Players[j].ActorAlias = base.Players[j].ActorAlias
		}
	}
	return writeCompactJSONReport(filepath.Join(root, "support_objective_observations.json"), map[string]any{
		"schema_id": "stratai.support_objective_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"support_source": source, "capture_contract": "same_support_FrameDone_public_objective_prefix@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": ctx.SupportObjectiveObservations,
	})
}
