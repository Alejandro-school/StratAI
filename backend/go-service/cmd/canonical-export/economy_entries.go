package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyEntries(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyEntryObservations == nil {
		return errors.New("native economic entry observations unavailable")
	}
	source, err := economyFileReference(root, "economy_ownership_observations.json")
	if err != nil {
		return err
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_entry_observations.json"), map[string]any{
		"schema_id": "stratai.economy_entry_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_ownership": source, "cut_contract": "fresh_sellback_read_in_same_ownership_FrameDone_no_parser_advance",
		"source_property":    "m_pBuyServices.m_vecSellbackPurchaseEntries",
		"partition":          "execution_evidence_only_no_initial_features_or_player_knowledge",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyEntryObservations,
	})
}
