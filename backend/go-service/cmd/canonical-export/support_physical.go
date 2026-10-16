package main

import (
	"cs2-demo-service/models"
	"errors"
	"path/filepath"
)

func writeSupportPhysical(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportPhysicalObservations == nil || ctx.SupportObservations == nil || len(ctx.SupportPhysicalObservations.Frames) == 0 || len(ctx.SupportPhysicalObservations.Frames) != len(ctx.SupportObservations.Frames) {
		return errors.New("support physical observations unavailable")
	}
	source, err := economyFileReference(root, "support_observations.json")
	if err != nil {
		return err
	}
	for i := range ctx.SupportPhysicalObservations.Frames {
		f := &ctx.SupportPhysicalObservations.Frames[i]
		base := ctx.SupportObservations.Frames[i]
		if f.SourceSequence != base.Sequence || len(f.Players) != len(base.Players) {
			return errors.New("support physical frame mismatch")
		}
		for j := range f.Players {
			if f.Players[j].NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" {
				return errors.New("support physical actor mismatch")
			}
			f.Players[j].ActorAlias = base.Players[j].ActorAlias
		}
	}
	return writeCompactJSONReport(filepath.Join(root, "support_physical_observations.json"), map[string]any{
		"schema_id": "stratai.support_physical_observations@1", "contract_version": "1.0.0-candidate.1",
		"source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue,
		"support_source": source, "capture_contract": "same_support_FrameDone_native_physical@1",
		"comparison_allowed": false, "training_allowed": false, "product_supported": false, "observations": ctx.SupportPhysicalObservations,
	})
}
