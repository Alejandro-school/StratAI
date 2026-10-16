package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeEconomyManifest(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.EconomyManifestSessions == nil {
		return errors.New("native resource manifests unavailable")
	}
	source, err := economyFileReference(root, "economy_rule_context_manifest.json")
	if err != nil {
		return err
	}
	data := map[string]any{
		"schema_id": "stratai.economy_resource_manifest@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"source_context": source, "codec_reference": "clarity-7fb3f1d07564a12efa99194d45cfbf5762ba5910-bounded-v1",
		"native_format_version": nil, "exact_build_asset_binding": "not_accredited", "economic_rules": "not_accredited",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "sessions": *ctx.EconomyManifestSessions,
	}
	return writeCompactJSONReport(filepath.Join(root, "economy_resource_manifest.json"), data)
}
