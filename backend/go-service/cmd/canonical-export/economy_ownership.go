package main

import (
	"errors"
	"path/filepath"

	"cs2-demo-service/models"
)

func writeEconomyOwnership(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyOwnershipObservations == nil {
		return errors.New("native economic ownership observations unavailable")
	}
	source, err := economyFileReference(root, "economy_native_identity.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_ownership_observations.json"), map[string]any{
		"schema_id": "stratai.economy_ownership_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_identity": source, "cut_contract": "fresh_FrameDone_after_queued_messages_and_frame_events_native_freeze_strict_instance_window",
		"partition":          "execution_evidence_only_no_initial_features_or_player_knowledge",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyOwnershipObservations,
	})
}
