package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyRuleContext(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyRuleContext == nil {
		return errors.New("native economic rule context unavailable")
	}
	source, err := economyFileReference(root, "economy_native_sources_manifest.json")
	if err != nil {
		return err
	}
	_, probeSHA := models.LoadEconomyRuleProbe()
	data := map[string]any{
		"schema_id": "stratai.economy_rule_context@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_native_manifest_sha256": source.SHA256, "probe_sha256": probeSHA,
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.EconomyRuleContext,
	}
	if err = writeCompactJSONReport(filepath.Join(root, "economy_rule_context.json"), data); err != nil {
		return err
	}
	artifact, err := economyFileReference(root, "economy_rule_context.json")
	if err != nil {
		return err
	}
	manifest := map[string]any{
		"schema_id": "stratai.economy_rule_context_manifest@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"source_native": source, "artifact": artifact, "probe_sha256": probeSHA,
		"summary": map[string]int{"messages": len(ctx.EconomyRuleContext.Messages), "boundaries": len(ctx.EconomyRuleContext.Boundaries), "sessions": len(ctx.EconomyRuleContext.Sessions)},
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_rule_context_manifest.json"), manifest)
}
