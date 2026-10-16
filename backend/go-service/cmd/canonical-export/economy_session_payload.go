package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomySessionPayloads(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomySessionPayloads == nil {
		return errors.New("native session payloads unavailable")
	}
	source, err := economyFileReference(root, "economy_rule_context_manifest.json")
	if err != nil {
		return err
	}
	_, probe := models.LoadEconomySessionPayloadProbe()
	_, economicProbe := models.LoadEconomyRuleProbe()
	data := map[string]any{
		"schema_id": "stratai.economy_session_payload@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_context": source, "probe_sha256": probe, "economic_probe_sha256": economicProbe,
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"exact_build_asset_binding": "not_accredited", "observations": ctx.EconomySessionPayloads,
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_session_payload.json"), data)
}
