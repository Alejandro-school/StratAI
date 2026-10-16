package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
)

type economyArtifactReference struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type economyCandidateManifest struct {
	SchemaID          string                   `json:"schema_id"`
	ContractVersion   string                   `json:"contract_version"`
	SourceAlias       string                   `json:"source_alias"`
	SourceSHA256      string                   `json:"source_sha256"`
	BuildIdentifier   string                   `json:"build_identifier"`
	TrainingAllowed   bool                     `json:"training_allowed"`
	ProductSupported  bool                     `json:"product_supported"`
	BaseExportVersion string                   `json:"base_export_version"`
	Artifact          economyArtifactReference `json:"artifact"`
	BaseManifest      economyArtifactReference `json:"base_manifest"`
	Summary           map[string]int           `json:"summary"`
}

func economyFileReference(root, relative string) (economyArtifactReference, error) {
	path := filepath.Join(root, relative)
	info, err := os.Stat(path)
	if err != nil {
		return economyArtifactReference{}, err
	}
	digest, err := sha256File(path)
	return economyArtifactReference{Path: filepath.ToSlash(relative), Bytes: info.Size(), SHA256: digest}, err
}

func writeEconomyCandidate(ctx *models.DemoContext, opts options, root, bundleName string) error {
	if opts.economyCandidateVersion >= 2 {
		if err := writeEconomyCandidateV2(ctx, opts, root, bundleName); err != nil {
			return err
		}
		if opts.economyCandidateVersion >= 3 {
			if err := writeEconomyNativeSources(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 4 {
			if err := writeEconomyRuleContext(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 5 {
			if err := writeEconomySessionPayloads(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 6 {
			if err := writeEconomyManifest(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 7 {
			if err := writeEconomyTrace(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 8 {
			if err := writeEconomyIdentity(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 9 {
			if err := writeEconomyOwnership(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion >= 10 {
			if err := writeEconomyEntries(ctx, opts, root); err != nil {
				return err
			}
		}
		if opts.economyCandidateVersion == 11 {
			return writeEconomyCashflow(ctx, opts, root)
		}
		return nil
	}
	candidate, err := parserpkg.BuildEconomyCandidate(ctx, opts.sourceAlias, opts.checksum, opts.buildIDValue)
	if err != nil {
		return err
	}
	if err = writeCompactJSONReport(filepath.Join(root, parserpkg.EconomyCandidateFilename), candidate); err != nil {
		return err
	}
	artifact, err := economyFileReference(root, parserpkg.EconomyCandidateFilename)
	if err != nil {
		return err
	}
	base, err := economyFileReference(root, filepath.Join(bundleName, "canonical", "manifest.json"))
	if err != nil {
		return err
	}
	manifest := economyCandidateManifest{
		SchemaID: parserpkg.EconomyCandidateManifestSchemaID, ContractVersion: parserpkg.EconomyCandidateVersion,
		SourceAlias: opts.sourceAlias, SourceSHA256: opts.checksum, BuildIdentifier: opts.buildIDValue,
		BaseExportVersion: "3.8.0", Artifact: artifact, BaseManifest: base, Summary: parserpkg.EconomyCandidateSummary(candidate),
	}
	return writeCompactJSONReport(filepath.Join(root, parserpkg.EconomyCandidateManifestFilename), manifest)
}

func writeEconomyCandidateV2(ctx *models.DemoContext, opts options, root, bundleName string) error {
	canonicalRoot := filepath.Join(root, bundleName, "canonical")
	roundBytes, err := os.ReadFile(filepath.Join(canonicalRoot, "core", "rounds.json"))
	if err != nil {
		return err
	}
	var rounds models.CanonicalRoundsExport
	if err = json.Unmarshal(roundBytes, &rounds); err != nil {
		return err
	}
	actorBytes, err := os.ReadFile(filepath.Join(canonicalRoot, "derived", "economy_players.json"))
	if err != nil {
		return err
	}
	var actors struct {
		MatchID string                            `json:"match_id"`
		Players []parserpkg.EconomyCanonicalActor `json:"players"`
	}
	if err = json.Unmarshal(actorBytes, &actors); err != nil {
		return err
	}
	if rounds.MatchID != opts.sourceAlias || actors.MatchID != opts.sourceAlias {
		return errors.New("canonical identity mismatch")
	}
	candidate, err := parserpkg.BuildEconomyCandidateV2(ctx, opts.sourceAlias, opts.checksum, opts.buildIDValue, rounds.Rounds, actors.Players)
	if err != nil {
		return err
	}
	if err = writeCompactJSONReport(filepath.Join(root, parserpkg.EconomyCandidateFilename), candidate); err != nil {
		return err
	}
	artifact, err := economyFileReference(root, parserpkg.EconomyCandidateFilename)
	if err != nil {
		return err
	}
	base, err := economyFileReference(root, filepath.Join(bundleName, "canonical", "manifest.json"))
	if err != nil {
		return err
	}
	manifest := economyCandidateManifest{
		SchemaID: parserpkg.EconomyCandidateV2ManifestSchemaID, ContractVersion: parserpkg.EconomyCandidateV2Version,
		SourceAlias: opts.sourceAlias, SourceSHA256: opts.checksum, BuildIdentifier: opts.buildIDValue,
		BaseExportVersion: "3.8.0", Artifact: artifact, BaseManifest: base, Summary: parserpkg.EconomyCandidateV2Summary(candidate),
	}
	return writeCompactJSONReport(filepath.Join(root, parserpkg.EconomyCandidateManifestFilename), manifest)
}
