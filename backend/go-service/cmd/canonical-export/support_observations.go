package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strconv"

	"cs2-demo-service/models"
)

func writeSupportObservations(ctx *models.DemoContext, opts options, root, bundleName string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportObservations == nil || len(ctx.SupportObservations.Frames) == 0 {
		return errors.New("support observations unavailable")
	}
	base, err := economyFileReference(root, filepath.Join(bundleName, "canonical", "manifest.json"))
	if err != nil {
		return err
	}
	for fi := range ctx.SupportObservations.Frames {
		for pi := range ctx.SupportObservations.Frames[fi].Players {
			p := &ctx.SupportObservations.Frames[fi].Players[pi]
			digest := sha256.Sum256([]byte(opts.checksum + "\nsteam:" + strconv.FormatUint(p.NativeActor, 10)))
			p.ActorAlias = "support-actor-" + hex.EncodeToString(digest[:])[:20]
		}
	}
	return writeCompactJSONReport(filepath.Join(root, "support_observations.json"), map[string]any{
		"schema_id": "stratai.support_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"base_manifest": base, "capture_contract": "fresh_replay_16hz_FrameDone_with_full_pawn_life@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"observations": ctx.SupportObservations,
	})
}
