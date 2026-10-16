package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportRoundContext(ctx *models.DemoContext, opts options, root string) error {
	data := ctx.SupportRoundContextObservations
	if data == nil || !ctx.ParseCompleted || ctx.SupportObservations == nil || len(data.Frames) == 0 || len(data.Frames) != len(ctx.SupportObservations.Frames) {
		return errors.New("round context cuts unavailable")
	}
	for i, row := range data.Frames {
		base := ctx.SupportObservations.Frames[i]
		if row.SourceSequence != base.Sequence || row.Frame != base.Frame || row.Tick != base.Tick || row.Instance != base.Instance || row.Epoch != base.Epoch {
			return errors.New("round context exact cut mismatch")
		}
	}
	ref, err := economyFileReference(root, "support_observations.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "support_round_context_observations.json"), map[string]any{
		"schema_id": "stratai.support_round_context_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"support_source": ref, "capture_contract": "same_support_FrameDone_public_round_configuration@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": data,
	})
}
