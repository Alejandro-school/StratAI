package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportC4(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportC4Observations == nil || ctx.SupportObservations == nil || ctx.SupportObjectiveObservations == nil {
		return errors.New("support C4 observations unavailable")
	}
	data := ctx.SupportC4Observations
	if len(data.Cuts) == 0 || len(data.Cuts) != len(ctx.SupportObservations.Frames) || len(data.Cuts) != len(ctx.SupportObjectiveObservations.Frames) || len(data.States) == 0 {
		return errors.New("support C4 cuts unavailable")
	}
	for i, cut := range data.Cuts {
		base, objective := ctx.SupportObservations.Frames[i], ctx.SupportObjectiveObservations.Frames[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ObjectiveOrder != objective.Order || cut.StateSequence < 1 || cut.StateSequence > len(data.States) {
			return errors.New("support C4 exact cut mismatch")
		}
		state := &data.States[cut.StateSequence-1]
		if state.FirstCut > cut.SourceSequence || len(state.Players) != len(base.Players) {
			return errors.New("support C4 state mismatch")
		}
		for j := range state.Players {
			p := &state.Players[j]
			if p.NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" || (p.ActorAlias != "" && p.ActorAlias != base.Players[j].ActorAlias) {
				return errors.New("support C4 actor mismatch")
			}
			p.ActorAlias = base.Players[j].ActorAlias
		}
	}
	support, err := economyFileReference(root, "support_observations.json")
	if err != nil {
		return err
	}
	objective, err := economyFileReference(root, "support_objective_observations.json")
	if err != nil {
		return err
	}
	if err := writeCompactJSONReport(filepath.Join(root, "support_c4_observations.json"), map[string]any{
		"schema_id": "stratai.support_c4_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"support_source": support, "objective_source": objective, "capture_contract": "same_support_FrameDone_native_inventory_owner@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data,
	}); err != nil {
		return err
	}
	if ctx.SupportInventoryObservations != nil {
		return writeSupportInventory(ctx, opts, root)
	}
	return nil
}
