package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportInventory(ctx *models.DemoContext, opts options, root string) error {
	data := ctx.SupportInventoryObservations
	if data == nil || len(data.Cuts) == 0 || len(data.Cuts) != len(ctx.SupportC4Observations.Cuts) || len(data.States) == 0 {
		return errors.New("inventory cuts unavailable")
	}
	for i, cut := range data.Cuts {
		base := ctx.SupportC4Observations.Cuts[i]
		if cut.SourceSequence != base.SourceSequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.C4StateSequence != base.StateSequence || cut.StateSequence < 1 || cut.StateSequence > len(data.States) || data.States[cut.StateSequence-1].FirstCut > cut.SourceSequence {
			return errors.New("inventory exact cut mismatch")
		}
	}
	ref, err := economyFileReference(root, "support_c4_observations.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "support_inventory_observations.json"), map[string]any{
		"schema_id": "stratai.support_inventory_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"c4_source": ref, "capture_contract": "same_support_FrameDone_native_reserve@1",
		"training_allowed": false, "product_supported": false, "comparison_allowed": false, "observations": data,
	})
}
