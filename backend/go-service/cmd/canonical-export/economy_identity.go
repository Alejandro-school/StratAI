package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyIdentity(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyNativeIdentity == nil {
		return errors.New("native economic identity unavailable")
	}
	source, err := economyFileReference(root, "economy_transaction_trace.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_native_identity.json"), map[string]any{
		"schema_id": "stratai.economy_native_identity@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_trace": source, "sdk_version": "demoinfocs-golang/v5@v5.2.0",
		"order_contract":     "sdk_delivery_after_priority_sort_not_wire_or_server_subtick",
		"partition":          "execution_evidence_only_no_initial_features_or_player_knowledge",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": ctx.EconomyNativeIdentity,
	})
}
