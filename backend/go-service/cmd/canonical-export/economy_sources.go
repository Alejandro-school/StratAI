package main

import (
	"errors"
	"path/filepath"

	"cs2-demo-service/models"
)

func writeEconomyNativeSources(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyNativeSources == nil {
		return errors.New("native economic source ledger unavailable")
	}
	source, err := economyFileReference(root, "economy_candidate_manifest.json")
	if err != nil {
		return err
	}
	data := map[string]any{
		"schema_id": "stratai.economy_native_sources@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_candidate_manifest_sha256": source.SHA256,
		"sampling_policy":                  "every_freeze_frame_retain_changes_and_boundaries_no_intermediate_transaction_guarantee",
		"partition":                        "execution_evidence_only_not_initial_features",
		"comparison_allowed":               false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyNativeSources,
	}
	if err = writeCompactJSONReport(filepath.Join(root, "economy_native_sources.json"), data); err != nil {
		return err
	}
	artifact, err := economyFileReference(root, "economy_native_sources.json")
	if err != nil {
		return err
	}
	summary := map[string]int{"snapshots": len(ctx.EconomyNativeSources.Snapshots), "snapshot_checks": ctx.EconomyNativeSources.SnapshotChecks, "version_observations": len(ctx.EconomyNativeSources.VersionObservations), "sellback_observed": 0, "round_purchases_observed": 0, "match_purchases_observed": 0, "native_purchase_events": ctx.EconomyNativeSources.GenericEventCounts["item_purchase"]}
	for _, row := range ctx.EconomyNativeSources.Snapshots {
		if row.Sellback.Status == "observed" {
			summary["sellback_observed"]++
		}
		if row.PurchasesRound.Status == "observed" {
			summary["round_purchases_observed"]++
		}
		if row.PurchasesMatch.Status == "observed" {
			summary["match_purchases_observed"]++
		}
	}
	manifest := map[string]any{
		"schema_id": "stratai.economy_native_sources_manifest@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"source_candidate": source, "artifact": artifact, "summary": summary,
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_native_sources_manifest.json"), manifest)
}
