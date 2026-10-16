package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyCashflow(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyCashflowObservations == nil {
		return errors.New("native economic cashflow observations unavailable")
	}
	entries, err := economyFileReference(root, "economy_entry_observations.json")
	if err != nil {
		return err
	}
	trace, err := economyFileReference(root, "economy_transaction_trace.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_cashflow_observations.json"), map[string]any{
		"schema_id": "stratai.economy_cashflow_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_entries": entries, "source_trace": trace,
		"cut_contract":       "fresh_accounts_in_entry_ownership_FrameDone_trace_next_frame_equals_cut_frame_plus_one",
		"partition":          "execution_evidence_only_no_initial_features_or_player_knowledge",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyCashflowObservations,
	})
}
