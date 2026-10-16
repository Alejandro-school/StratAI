package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyTrace(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyTransactionTrace == nil {
		return errors.New("economic transaction trace unavailable")
	}
	source, err := economyFileReference(root, "economy_candidate_manifest.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_transaction_trace.json"), map[string]any{
		"schema_id": "stratai.economy_transaction_trace@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_candidate": source, "sdk_version": "demoinfocs-golang/v5@v5.2.0",
		"order_contract":     "observer_delivery_sequence_not_server_subtick",
		"partition":          "execution_evidence_only_no_player_knowledge_or_initial_features",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyTransactionTrace,
	})
}
