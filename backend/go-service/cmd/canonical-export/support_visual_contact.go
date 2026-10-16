package main

import (
	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
	"errors"
	"path/filepath"
)

func writeSupportVisualContact(ctx *models.DemoContext, opts options, root string) error {
	if ctx == nil || !ctx.ParseCompleted || ctx.SupportVisualObservations == nil || ctx.SupportWeaponTimingObservations == nil || ctx.SupportObservations == nil {
		return errors.New("support visual observations unavailable")
	}
	data := ctx.SupportVisualObservations
	if len(data.Cuts) == 0 || len(data.States) == 0 || len(data.Deliveries) == 0 || len(data.Cuts) != len(ctx.SupportObservations.Frames) || len(data.Cuts) != len(ctx.SupportWeaponTimingObservations.Cuts) {
		return errors.New("support visual capture empty or incomplete")
	}
	for i := range data.Cuts {
		cut := &data.Cuts[i]
		base, prior := ctx.SupportObservations.Frames[i], ctx.SupportWeaponTimingObservations.Cuts[i]
		if cut.SourceSequence != base.Sequence || cut.Frame != base.Frame || cut.Tick != base.Tick || cut.Instance != base.Instance || cut.Epoch != base.Epoch || cut.ClockOrder != prior.ClockOrder || len(cut.Players) != len(base.Players) || cut.StateSequence > len(data.States) || cut.DeliverySequence > len(data.Deliveries) {
			return errors.New("support visual exact cut mismatch")
		}
		for j := range cut.Players {
			if cut.Players[j].NativeActor != base.Players[j].NativeActor || base.Players[j].ActorAlias == "" {
				return errors.New("support visual actor mismatch")
			}
			cut.Players[j].ActorAlias = base.Players[j].ActorAlias
		}
	}
	assets := []parserpkg.ObservedVisualContactSmokeAsset{}
	loaded := ctx.MapManager != nil && ctx.MapManager.IsLoaded()
	if loaded {
		var err error
		assets, err = visualContactGeometryAssets(ctx.MapManager, opts.mapsRoot)
		if err != nil {
			return err
		}
	}
	refs := map[string]economyArtifactReference{}
	for key, file := range map[string]string{"support_source": "support_observations.json", "objective_source": "support_objective_observations.json", "c4_source": "support_c4_observations.json", "clock_source": "support_objective_clock_observations.json", "attempt_source": "support_attempt_origin_observations.json", "position_source": "support_objective_position_observations.json", "motion_source": "support_native_motion_observations.json", "physical_source": "support_physical_observations.json", "timing_source": "support_weapon_timing_observations.json"} {
		ref, err := economyFileReference(root, file)
		if err != nil {
			return err
		}
		refs[key] = ref
	}
	return writeCompactJSONReport(filepath.Join(root, "support_visual_contact_observations.json"), map[string]any{
		"schema_id": "stratai.support_visual_contact_observations@1", "contract_version": "1.0.0-candidate.1", "source_alias": opts.sourceAlias, "source_sha256": opts.checksum, "build_identifier": opts.buildIDValue, "references": refs,
		"capture_contract": "native_visual_inputs_and_exact_support_cuts@1", "comparison_allowed": false, "training_allowed": false, "product_supported": false,
		"geometry":     map[string]any{"map_name": ctx.MapName, "assets": assets, "raycast_method": "map_mesh_segment_raycast@1", "semantic_scope": "static_physics_mesh_eye_segment_diagnostic_only", "dynamic_occluders_accredited": false, "smoke_volume_accredited": false, "rendered_flash_accredited": false},
		"observations": data,
	})
}
