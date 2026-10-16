package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
	mapspkg "cs2-demo-service/pkg/maps"

	"github.com/golang/geo/r3"
)

func TestRunRejectsMissingArgumentsWithoutLeakingPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, dependencies{})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stdout.String(), `"error_code":"invalid_arguments"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunRejectsChecksumMismatchBeforeParsing(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	called := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			called = true
			return nil, errors.New("must not be called")
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", filepath.Join(t.TempDir(), "out"),
		"--alias", "demo-" + checksum[:20],
		"--checksum", strings.Repeat("0", 64),
	}, &stdout, &stderr, deps)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3", code)
	}
	if called {
		t.Fatal("parser was called after checksum mismatch")
	}
	if strings.Contains(stdout.String(), demoPath) {
		t.Fatal("structured output leaked source path")
	}
}

func TestRunPublishesOnlyAfterSuccessfulExport(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := filepath.Join(t.TempDir(), "out")
	context := &models.DemoContext{}
	deps := dependencies{
		parse: func(path string) (*parserpkg.ParseDemoResult, error) {
			if path != demoPath {
				t.Fatalf("parse path = %q, want fixture", path)
			}
			return &parserpkg.ParseDemoResult{Context: context}, nil
		},
		export: func(
			actualContext *models.DemoContext,
			alias, stagingRoot, playedAt string,
			provenance models.CanonicalExportProvenance,
		) error {
			if actualContext != context || alias != "demo-"+checksum[:20] {
				t.Fatal("export received unexpected context or alias")
			}
			if playedAt != "" || provenance.DemoChecksum != checksum ||
				provenance.Source != "demo" || provenance.BuildIdentifier != buildID {
				t.Fatalf("unexpected provenance: %+v playedAt=%q", provenance, playedAt)
			}
			bundle := filepath.Join(stagingRoot, "match_"+alias)
			if err := os.MkdirAll(bundle, 0o750); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bundle, "marker"), []byte("ok"), 0o600)
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	finalBundle := filepath.Join(outputRoot, "match_demo-"+checksum[:20])
	if _, err := os.Stat(filepath.Join(finalBundle, "marker")); err != nil {
		t.Fatalf("published marker: %v", err)
	}
	if strings.Contains(stdout.String(), demoPath) || strings.Contains(stdout.String(), outputRoot) {
		t.Fatal("structured output leaked a private path")
	}
}

func TestRunDoesNotPublishFailedExport(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := filepath.Join(t.TempDir(), "out")
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{Context: &models.DemoContext{}}, nil
		},
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			return errors.New("quality gate failed")
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 5 {
		t.Fatalf("exit code = %d, want 5", code)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "match_demo-"+checksum[:20])); !os.IsNotExist(err) {
		t.Fatalf("failed export was published: %v", err)
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging was not cleaned: %v", entries)
	}
}

func TestRunPublishesCanonicalBundleAndNavigationSmokeAtomically(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	alias := "demo-" + checksum[:20]
	outputRoot := emptySmokeOutputRoot(t)
	context := &models.DemoContext{}
	replay := &models.ReplayData{}
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{Context: context, ReplayData: replay}, nil
		},
		navigationSmoke: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			queryLimit int,
		) (parserpkg.ObservedNavigationSmokeReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum || queryLimit != 7 {
				t.Fatalf("unexpected smoke inputs: parsed=%#v alias=%q checksum=%q queries=%d", parsed, sourceAlias, sourceSHA256, queryLimit)
			}
			return parserpkg.ObservedNavigationSmokeReport{
				SchemaID:     parserpkg.ObservedNavigationSmokeSchemaID,
				Status:       "passed",
				SourceAlias:  alias,
				SourceSHA256: checksum,
			}, nil
		},
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			bundle := filepath.Join(outputRoot, "unused")
			_ = bundle
			return nil
		},
	}
	// Replace the exporter with one that writes below the provided staging root.
	deps.export = func(
		_ *models.DemoContext,
		actualAlias, stagingRoot, _ string,
		_ models.CanonicalExportProvenance,
	) error {
		bundle := filepath.Join(stagingRoot, "match_"+actualAlias)
		if err := os.MkdirAll(bundle, 0o750); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(bundle, "marker"), []byte("ok"), 0o600)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", alias,
		"--checksum", checksum,
		"--build-id", "test-smoke-build",
		"--navigation-smoke",
		"--navigation-smoke-queries", "7",
	}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	runDir := filepath.Join(outputRoot, "smoke_run_"+alias)
	if _, err := os.Stat(filepath.Join(runDir, "match_"+alias, "marker")); err != nil {
		t.Fatalf("published smoke bundle: %v", err)
	}
	payload, err := os.ReadFile(filepath.Join(runDir, "observed_navigation_smoke.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"build_identifier": "test-smoke-build"`) ||
		!strings.Contains(string(payload), `"status": "passed"`) {
		t.Fatalf("unexpected smoke report: %s", payload)
	}
	if !strings.Contains(stdout.String(), `"smoke_report":"smoke_run_`) {
		t.Fatalf("structured output omitted smoke report: %s", stdout.String())
	}
}

func TestRunPublishesAllAICoachSmokeArtifactsFromOneParse(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	alias := "demo-" + checksum[:20]
	outputRoot := emptySmokeOutputRoot(t)
	mapsRoot := t.TempDir()
	context := &models.DemoContext{}
	replay := &models.ReplayData{}
	parseCalls := 0
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			parseCalls++
			return &parserpkg.ParseDemoResult{Context: context, ReplayData: replay}, nil
		},
		navigationSmoke: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			queryLimit int,
		) (parserpkg.ObservedNavigationSmokeReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum || queryLimit != 9 {
				t.Fatal("unexpected navigation smoke inputs")
			}
			return parserpkg.ObservedNavigationSmokeReport{
				SchemaID: parserpkg.ObservedNavigationSmokeSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		visualContactSmoke: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256, actualMapsRoot string,
			frameLimit int,
		) (parserpkg.ObservedVisualContactSmokeReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum ||
				actualMapsRoot != mapsRoot || frameLimit != 16 {
				t.Fatal("unexpected visual contact smoke inputs")
			}
			return parserpkg.ObservedVisualContactSmokeReport{
				SchemaID: parserpkg.ObservedVisualContactSmokeSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		export: func(
			_ *models.DemoContext,
			actualAlias, stagingRoot, _ string,
			_ models.CanonicalExportProvenance,
		) error {
			bundle := filepath.Join(stagingRoot, "match_"+actualAlias)
			if err := os.MkdirAll(bundle, 0o750); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bundle, "marker"), []byte("ok"), 0o600)
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", alias,
		"--checksum", checksum,
		"--maps-root", mapsRoot,
		"--build-id", "test-all-ai-coach-smoke",
		"--ai-coach-smoke",
		"--navigation-smoke-queries", "9",
		"--visual-contact-smoke-frames", "16",
	}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if parseCalls != 1 {
		t.Fatalf("parse calls = %d, want exactly one", parseCalls)
	}
	runDir := filepath.Join(outputRoot, "smoke_run_"+alias)
	for _, name := range []string{
		filepath.Join("match_"+alias, "marker"),
		"observed_navigation_smoke.json",
		"observed_visual_contact_smoke.json",
	} {
		if _, err := os.Stat(filepath.Join(runDir, name)); err != nil {
			t.Fatalf("published AI Coach smoke artifact %q: %v", name, err)
		}
	}
	if !strings.Contains(stdout.String(), `"visual_contact_smoke_report":"smoke_run_`) {
		t.Fatalf("structured output omitted visual contact report: %s", stdout.String())
	}
}

func TestRunPublishesFullDemoValidationArtifactsFromOneParse(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	alias := "demo-" + checksum[:20]
	outputRoot := emptySmokeOutputRoot(t)
	mapsRoot := t.TempDir()
	context := &models.DemoContext{}
	replay := &models.ReplayData{}
	parseCalls := 0
	navigationValidated := false
	visualValidated := false
	weaponValidated := false
	utilityVelocityValidated := false
	lastKnownValidated := false
	navigationLogical := []byte("{\"kind\":\"navigation\",\"round\":1}\n")
	visualLogical := []byte("{\"record_type\":\"evaluation\",\"round\":1,\"value\":{}}\n")
	weaponLogical := []byte("{\"schema_id\":\"stratai.observed_weapon_state@1\",\"round_number\":1}\n")
	utilityVelocityLogical := []byte("{\"row_id\":\"utility-velocity-000001\",\"round_number\":1}\n")
	lastKnownLogical := []byte("{\"row_id\":\"last-known-000000001\",\"round_number\":1}\n")

	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			parseCalls++
			return &parserpkg.ParseDemoResult{Context: context, ReplayData: replay}, nil
		},
		navigationSmoke: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			queryLimit int,
		) (parserpkg.ObservedNavigationSmokeReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum || queryLimit != 9 {
				t.Fatal("unexpected bounded navigation inputs")
			}
			return parserpkg.ObservedNavigationSmokeReport{
				SchemaID: parserpkg.ObservedNavigationSmokeSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		visualContactSmoke: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256, actualMapsRoot string,
			frameLimit int,
		) (parserpkg.ObservedVisualContactSmokeReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum ||
				actualMapsRoot != mapsRoot || frameLimit != 16 {
				t.Fatal("unexpected bounded visual-contact inputs")
			}
			return parserpkg.ObservedVisualContactSmokeReport{
				SchemaID: parserpkg.ObservedVisualContactSmokeSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		navigationFullDemo: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			writer io.Writer,
		) (parserpkg.ObservedNavigationFullDemoReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum {
				t.Fatal("unexpected full-demo navigation inputs")
			}
			if _, err := writer.Write(navigationLogical); err != nil {
				return parserpkg.ObservedNavigationFullDemoReport{}, err
			}
			return parserpkg.ObservedNavigationFullDemoReport{
				SchemaID: parserpkg.ObservedNavigationFullDemoSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		validateNavigationFullDemo: func(report parserpkg.ObservedNavigationFullDemoReport) error {
			navigationValidated = true
			if report.BuildIdentifier != "test-full-demo-build" ||
				report.ProcessingDurationMS < 0 || report.NavigationEvaluationDurationMS < 0 ||
				report.RowsArtifact.Path != parserpkg.ObservedNavigationFullDemoRowsFilename ||
				report.RowsArtifact.Bytes < 1 || !checksumPattern.MatchString(report.RowsArtifact.SHA256) {
				return errors.New("navigation physical metadata was not completed")
			}
			return nil
		},
		visualContactFullDemo: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256, actualMapsRoot string,
			writer io.Writer,
		) (parserpkg.ObservedVisualContactFullDemoReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum || actualMapsRoot != mapsRoot {
				t.Fatal("unexpected full-demo visual-contact inputs")
			}
			if _, err := writer.Write(visualLogical); err != nil {
				return parserpkg.ObservedVisualContactFullDemoReport{}, err
			}
			return parserpkg.ObservedVisualContactFullDemoReport{
				SchemaID: parserpkg.ObservedVisualContactFullDemoSchemaID,
				Status:   "passed", SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		validateVisualContactFullDemo: func(report parserpkg.ObservedVisualContactFullDemoReport) error {
			visualValidated = true
			if report.BuildIdentifier != "test-full-demo-build" || report.ProcessingDurationMS < 0 ||
				report.Audit.GzipRelativePath != parserpkg.ObservedVisualContactFullDemoAuditFilename ||
				report.Audit.GzipBytes < 1 || !checksumPattern.MatchString(report.Audit.GzipSHA256) {
				return errors.New("visual-contact physical metadata was not completed")
			}
			return nil
		},
		weaponStateFullDemo: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			writer io.Writer,
		) (parserpkg.ObservedWeaponStateFullDemoReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum {
				t.Fatal("unexpected full-demo weapon-state inputs")
			}
			if _, err := writer.Write(weaponLogical); err != nil {
				return parserpkg.ObservedWeaponStateFullDemoReport{}, err
			}
			return parserpkg.ObservedWeaponStateFullDemoReport{
				SchemaID:    parserpkg.ObservedWeaponStateFullDemoSchemaID,
				SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		validateWeaponStateFullDemo: func(report parserpkg.ObservedWeaponStateFullDemoReport) error {
			weaponValidated = true
			if report.BuildIdentifier != "test-full-demo-build" ||
				report.RowsArtifact.Path != parserpkg.ObservedWeaponStateFullDemoRowsFilename ||
				report.RowsArtifact.Bytes < 1 || !checksumPattern.MatchString(report.RowsArtifact.SHA256) {
				return errors.New("weapon-state physical metadata was not completed")
			}
			return nil
		},
		utilityInitialVelocityFullDemo: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			writer io.Writer,
		) (parserpkg.ObservedUtilityInitialVelocityFullDemoReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum {
				t.Fatal("unexpected full-demo utility initial-velocity inputs")
			}
			if _, err := writer.Write(utilityVelocityLogical); err != nil {
				return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{}, err
			}
			return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{
				SchemaID:    parserpkg.ObservedUtilityInitialVelocityFullDemoSchemaID,
				SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		validateUtilityInitialVelocityFullDemo: func(report parserpkg.ObservedUtilityInitialVelocityFullDemoReport) error {
			utilityVelocityValidated = true
			if report.BuildIdentifier != "test-full-demo-build" ||
				report.RowsArtifact.Path != parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename ||
				report.RowsArtifact.Bytes < 1 || !checksumPattern.MatchString(report.RowsArtifact.SHA256) {
				return errors.New("utility initial-velocity physical metadata was not completed")
			}
			return nil
		},
		lastKnownPositionFullDemo: func(
			parsed *parserpkg.ParseDemoResult,
			sourceAlias, sourceSHA256 string,
			writer io.Writer,
		) (parserpkg.ObservedLastKnownPositionFullDemoReport, error) {
			if parsed.ReplayData != replay || sourceAlias != alias || sourceSHA256 != checksum {
				t.Fatal("unexpected full-demo last-known position inputs")
			}
			if _, err := writer.Write(lastKnownLogical); err != nil {
				return parserpkg.ObservedLastKnownPositionFullDemoReport{}, err
			}
			return parserpkg.ObservedLastKnownPositionFullDemoReport{
				SchemaID:    parserpkg.ObservedLastKnownPositionFullDemoSchemaID,
				SourceAlias: alias, SourceSHA256: checksum,
			}, nil
		},
		validateLastKnownPositionFullDemo: func(report parserpkg.ObservedLastKnownPositionFullDemoReport) error {
			lastKnownValidated = true
			if report.BuildIdentifier != "test-full-demo-build" ||
				report.RowsArtifact.Path != parserpkg.ObservedLastKnownPositionFullDemoRowsFilename ||
				report.RowsArtifact.Bytes < 1 || !checksumPattern.MatchString(report.RowsArtifact.SHA256) {
				return errors.New("last-known position physical metadata was not completed")
			}
			return nil
		},
		export: func(
			_ *models.DemoContext,
			actualAlias, stagingRoot, _ string,
			_ models.CanonicalExportProvenance,
		) error {
			bundle := filepath.Join(stagingRoot, "match_"+actualAlias)
			if err := os.MkdirAll(bundle, 0o750); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bundle, "marker"), []byte("ok"), 0o600)
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", alias,
		"--checksum", checksum,
		"--maps-root", mapsRoot,
		"--build-id", "test-full-demo-build",
		"--ai-coach-full-demo-validation",
		"--navigation-smoke-queries", "9",
		"--visual-contact-smoke-frames", "16",
	}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if parseCalls != 1 {
		t.Fatalf("parse calls = %d, want exactly one", parseCalls)
	}
	if !navigationValidated || !visualValidated || !weaponValidated || !utilityVelocityValidated || !lastKnownValidated {
		t.Fatalf(
			"completed report validators: navigation=%t visual=%t weapon=%t utility_velocity=%t last_known=%t",
			navigationValidated,
			visualValidated,
			weaponValidated,
			utilityVelocityValidated,
			lastKnownValidated,
		)
	}

	publicationName := "smoke_run_" + alias
	runDir := filepath.Join(outputRoot, publicationName)
	gotArtifacts := relativeRegularFiles(t, runDir)
	wantArtifacts := []string{
		filepath.ToSlash(filepath.Join("match_"+alias, "marker")),
		parserpkg.ObservedNavigationFullDemoReportFilename,
		parserpkg.ObservedNavigationFullDemoRowsFilename,
		"observed_navigation_smoke.json",
		parserpkg.ObservedVisualContactFullDemoReportFilename,
		parserpkg.ObservedVisualContactFullDemoAuditFilename,
		"observed_visual_contact_smoke.json",
		parserpkg.ObservedWeaponStateFullDemoReportFilename,
		parserpkg.ObservedWeaponStateFullDemoRowsFilename,
		parserpkg.ObservedUtilityInitialVelocityFullDemoReportFilename,
		parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename,
		parserpkg.ObservedLastKnownPositionFullDemoReportFilename,
		parserpkg.ObservedLastKnownPositionFullDemoRowsFilename,
	}
	sort.Strings(wantArtifacts)
	if strings.Join(gotArtifacts, "\n") != strings.Join(wantArtifacts, "\n") {
		t.Fatalf("published artifacts = %v, want %v", gotArtifacts, wantArtifacts)
	}

	navigationRowsPath := filepath.Join(runDir, parserpkg.ObservedNavigationFullDemoRowsFilename)
	visualAuditPath := filepath.Join(runDir, parserpkg.ObservedVisualContactFullDemoAuditFilename)
	weaponRowsPath := filepath.Join(runDir, parserpkg.ObservedWeaponStateFullDemoRowsFilename)
	utilityVelocityRowsPath := filepath.Join(runDir, parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename)
	lastKnownRowsPath := filepath.Join(runDir, parserpkg.ObservedLastKnownPositionFullDemoRowsFilename)
	if got := readGzipArtifact(t, navigationRowsPath); !bytes.Equal(got, navigationLogical) {
		t.Fatalf("navigation logical rows = %q, want %q", got, navigationLogical)
	}
	if got := readGzipArtifact(t, visualAuditPath); !bytes.Equal(got, visualLogical) {
		t.Fatalf("visual logical audit = %q, want %q", got, visualLogical)
	}
	if got := readGzipArtifact(t, weaponRowsPath); !bytes.Equal(got, weaponLogical) {
		t.Fatalf("weapon logical rows = %q, want %q", got, weaponLogical)
	}
	if got := readGzipArtifact(t, utilityVelocityRowsPath); !bytes.Equal(got, utilityVelocityLogical) {
		t.Fatalf("utility velocity logical rows = %q, want %q", got, utilityVelocityLogical)
	}
	if got := readGzipArtifact(t, lastKnownRowsPath); !bytes.Equal(got, lastKnownLogical) {
		t.Fatalf("last-known logical rows = %q, want %q", got, lastKnownLogical)
	}

	navigationReportPath := filepath.Join(runDir, parserpkg.ObservedNavigationFullDemoReportFilename)
	navigationPayload, err := os.ReadFile(navigationReportPath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompactJSON(t, navigationPayload)
	var navigationReport parserpkg.ObservedNavigationFullDemoReport
	if err := json.Unmarshal(navigationPayload, &navigationReport); err != nil {
		t.Fatal(err)
	}
	navigationFileSHA, err := sha256File(navigationRowsPath)
	if err != nil {
		t.Fatal(err)
	}
	if navigationReport.RowsArtifact.Bytes != fileSize(t, navigationRowsPath) ||
		navigationReport.RowsArtifact.SHA256 != navigationFileSHA ||
		navigationReport.RowsArtifact.Path != parserpkg.ObservedNavigationFullDemoRowsFilename {
		t.Fatalf("navigation report physical metadata = %#v", navigationReport.RowsArtifact)
	}

	visualReportPath := filepath.Join(runDir, parserpkg.ObservedVisualContactFullDemoReportFilename)
	visualPayload, err := os.ReadFile(visualReportPath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompactJSON(t, visualPayload)
	var visualReport parserpkg.ObservedVisualContactFullDemoReport
	if err := json.Unmarshal(visualPayload, &visualReport); err != nil {
		t.Fatal(err)
	}
	visualFileSHA, err := sha256File(visualAuditPath)
	if err != nil {
		t.Fatal(err)
	}
	if visualReport.Audit.GzipBytes != fileSize(t, visualAuditPath) ||
		visualReport.Audit.GzipSHA256 != visualFileSHA ||
		visualReport.Audit.GzipRelativePath != parserpkg.ObservedVisualContactFullDemoAuditFilename {
		t.Fatalf("visual report physical metadata = %#v", visualReport.Audit)
	}

	weaponReportPath := filepath.Join(runDir, parserpkg.ObservedWeaponStateFullDemoReportFilename)
	weaponPayload, err := os.ReadFile(weaponReportPath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompactJSON(t, weaponPayload)
	var weaponReport parserpkg.ObservedWeaponStateFullDemoReport
	if err := json.Unmarshal(weaponPayload, &weaponReport); err != nil {
		t.Fatal(err)
	}
	weaponFileSHA, err := sha256File(weaponRowsPath)
	if err != nil {
		t.Fatal(err)
	}
	if weaponReport.RowsArtifact.Bytes != fileSize(t, weaponRowsPath) ||
		weaponReport.RowsArtifact.SHA256 != weaponFileSHA ||
		weaponReport.RowsArtifact.Path != parserpkg.ObservedWeaponStateFullDemoRowsFilename {
		t.Fatalf("weapon report physical metadata = %#v", weaponReport.RowsArtifact)
	}

	utilityVelocityReportPath := filepath.Join(runDir, parserpkg.ObservedUtilityInitialVelocityFullDemoReportFilename)
	utilityVelocityPayload, err := os.ReadFile(utilityVelocityReportPath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompactJSON(t, utilityVelocityPayload)
	var utilityVelocityReport parserpkg.ObservedUtilityInitialVelocityFullDemoReport
	if err := json.Unmarshal(utilityVelocityPayload, &utilityVelocityReport); err != nil {
		t.Fatal(err)
	}
	utilityVelocityFileSHA, err := sha256File(utilityVelocityRowsPath)
	if err != nil {
		t.Fatal(err)
	}
	if utilityVelocityReport.RowsArtifact.Bytes != fileSize(t, utilityVelocityRowsPath) ||
		utilityVelocityReport.RowsArtifact.SHA256 != utilityVelocityFileSHA ||
		utilityVelocityReport.RowsArtifact.Path != parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename {
		t.Fatalf("utility velocity report physical metadata = %#v", utilityVelocityReport.RowsArtifact)
	}

	lastKnownReportPath := filepath.Join(runDir, parserpkg.ObservedLastKnownPositionFullDemoReportFilename)
	lastKnownPayload, err := os.ReadFile(lastKnownReportPath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompactJSON(t, lastKnownPayload)
	var lastKnownReport parserpkg.ObservedLastKnownPositionFullDemoReport
	if err := json.Unmarshal(lastKnownPayload, &lastKnownReport); err != nil {
		t.Fatal(err)
	}
	lastKnownFileSHA, err := sha256File(lastKnownRowsPath)
	if err != nil {
		t.Fatal(err)
	}
	if lastKnownReport.RowsArtifact.Bytes != fileSize(t, lastKnownRowsPath) ||
		lastKnownReport.RowsArtifact.SHA256 != lastKnownFileSHA ||
		lastKnownReport.RowsArtifact.Path != parserpkg.ObservedLastKnownPositionFullDemoRowsFilename {
		t.Fatalf("last-known report physical metadata = %#v", lastKnownReport.RowsArtifact)
	}

	var result commandResult
	resultJSON := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(stdout.String()), resultPrefix))
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("decode command result %q: %v", stdout.String(), err)
	}
	wantRoutes := map[string]string{
		"navigation report":       filepath.Join(publicationName, parserpkg.ObservedNavigationFullDemoReportFilename),
		"navigation rows":         filepath.Join(publicationName, parserpkg.ObservedNavigationFullDemoRowsFilename),
		"visual report":           filepath.Join(publicationName, parserpkg.ObservedVisualContactFullDemoReportFilename),
		"visual audit":            filepath.Join(publicationName, parserpkg.ObservedVisualContactFullDemoAuditFilename),
		"weapon report":           filepath.Join(publicationName, parserpkg.ObservedWeaponStateFullDemoReportFilename),
		"weapon rows":             filepath.Join(publicationName, parserpkg.ObservedWeaponStateFullDemoRowsFilename),
		"utility velocity report": filepath.Join(publicationName, parserpkg.ObservedUtilityInitialVelocityFullDemoReportFilename),
		"utility velocity rows":   filepath.Join(publicationName, parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename),
		"last-known report":       filepath.Join(publicationName, parserpkg.ObservedLastKnownPositionFullDemoReportFilename),
		"last-known rows":         filepath.Join(publicationName, parserpkg.ObservedLastKnownPositionFullDemoRowsFilename),
	}
	gotRoutes := map[string]string{
		"navigation report":       result.NavigationFullDemoReport,
		"navigation rows":         result.NavigationFullDemoRows,
		"visual report":           result.VisualContactFullDemoReport,
		"visual audit":            result.VisualContactFullDemoAudit,
		"weapon report":           result.WeaponStateFullDemoReport,
		"weapon rows":             result.WeaponStateFullDemoRows,
		"utility velocity report": result.UtilityInitialVelocityFullDemoReport,
		"utility velocity rows":   result.UtilityInitialVelocityFullDemoRows,
		"last-known report":       result.LastKnownPositionFullDemoReport,
		"last-known rows":         result.LastKnownPositionFullDemoRows,
	}
	for name, want := range wantRoutes {
		got := gotRoutes[name]
		if got != want || filepath.IsAbs(got) || strings.Contains(got, outputRoot) ||
			strings.Contains(got, demoPath) || strings.Contains(got, mapsRoot) {
			t.Fatalf("unsafe or incorrect %s route: got %q want %q", name, got, want)
		}
	}
	if strings.Contains(stdout.String(), outputRoot) || strings.Contains(stdout.String(), demoPath) || strings.Contains(stdout.String(), mapsRoot) {
		t.Fatalf("command result leaked a local path: %s", stdout.String())
	}
}

func TestRunFullDemoValidationFailuresDoNotPublish(t *testing.T) {
	for _, testCase := range []struct {
		name                        string
		failNavigation              bool
		failVisual                  bool
		failWeapon                  bool
		failUtilityVelocity         bool
		failLastKnownBuilder        bool
		failLastKnownValidator      bool
		wantCode                    int
		wantStage                   string
		wantVisualFullCall          bool
		wantWeaponFullCall          bool
		wantUtilityVelocityFullCall bool
		wantLastKnownFullCall       bool
	}{
		{name: "navigation", failNavigation: true, wantCode: 9, wantStage: "navigation_full_demo"},
		{name: "visual contact", failVisual: true, wantCode: 10, wantStage: "visual_contact_full_demo", wantVisualFullCall: true},
		{name: "weapon state", failWeapon: true, wantCode: 11, wantStage: "weapon_state_full_demo", wantVisualFullCall: true, wantWeaponFullCall: true},
		{name: "utility initial velocity", failUtilityVelocity: true, wantCode: 12, wantStage: "utility_initial_velocity_full_demo", wantVisualFullCall: true, wantWeaponFullCall: true, wantUtilityVelocityFullCall: true},
		{name: "last-known builder", failLastKnownBuilder: true, wantCode: 13, wantStage: "last_known_position_full_demo", wantVisualFullCall: true, wantWeaponFullCall: true, wantUtilityVelocityFullCall: true, wantLastKnownFullCall: true},
		{name: "last-known validator", failLastKnownValidator: true, wantCode: 13, wantStage: "last_known_position_full_demo", wantVisualFullCall: true, wantWeaponFullCall: true, wantUtilityVelocityFullCall: true, wantLastKnownFullCall: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			demoPath, checksum := writeDemoFixture(t)
			outputRoot := emptySmokeOutputRoot(t)
			exportCalled := false
			visualFullCalled := false
			weaponFullCalled := false
			utilityVelocityFullCalled := false
			lastKnownFullCalled := false
			deps := dependencies{
				parse: func(string) (*parserpkg.ParseDemoResult, error) {
					return &parserpkg.ParseDemoResult{
						Context: &models.DemoContext{}, ReplayData: &models.ReplayData{},
					}, nil
				},
				navigationSmoke: func(
					*parserpkg.ParseDemoResult, string, string, int,
				) (parserpkg.ObservedNavigationSmokeReport, error) {
					return parserpkg.ObservedNavigationSmokeReport{Status: "passed"}, nil
				},
				visualContactSmoke: func(
					*parserpkg.ParseDemoResult, string, string, string, int,
				) (parserpkg.ObservedVisualContactSmokeReport, error) {
					return parserpkg.ObservedVisualContactSmokeReport{Status: "passed"}, nil
				},
				navigationFullDemo: func(
					*parserpkg.ParseDemoResult, string, string, io.Writer,
				) (parserpkg.ObservedNavigationFullDemoReport, error) {
					if testCase.failNavigation {
						return parserpkg.ObservedNavigationFullDemoReport{}, errors.New("navigation full-demo failed")
					}
					return parserpkg.ObservedNavigationFullDemoReport{Status: "passed"}, nil
				},
				validateNavigationFullDemo: func(parserpkg.ObservedNavigationFullDemoReport) error {
					return nil
				},
				visualContactFullDemo: func(
					*parserpkg.ParseDemoResult, string, string, string, io.Writer,
				) (parserpkg.ObservedVisualContactFullDemoReport, error) {
					visualFullCalled = true
					if testCase.failVisual {
						return parserpkg.ObservedVisualContactFullDemoReport{}, errors.New("visual full-demo failed")
					}
					return parserpkg.ObservedVisualContactFullDemoReport{}, nil
				},
				validateVisualContactFullDemo: func(parserpkg.ObservedVisualContactFullDemoReport) error {
					return nil
				},
				weaponStateFullDemo: func(
					*parserpkg.ParseDemoResult, string, string, io.Writer,
				) (parserpkg.ObservedWeaponStateFullDemoReport, error) {
					weaponFullCalled = true
					if testCase.failWeapon {
						return parserpkg.ObservedWeaponStateFullDemoReport{}, errors.New("weapon full-demo failed")
					}
					return parserpkg.ObservedWeaponStateFullDemoReport{}, nil
				},
				validateWeaponStateFullDemo: func(parserpkg.ObservedWeaponStateFullDemoReport) error {
					return nil
				},
				utilityInitialVelocityFullDemo: func(
					*parserpkg.ParseDemoResult, string, string, io.Writer,
				) (parserpkg.ObservedUtilityInitialVelocityFullDemoReport, error) {
					utilityVelocityFullCalled = true
					if testCase.failUtilityVelocity {
						return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{}, errors.New("utility velocity full-demo failed")
					}
					return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{}, nil
				},
				validateUtilityInitialVelocityFullDemo: func(parserpkg.ObservedUtilityInitialVelocityFullDemoReport) error {
					return nil
				},
				lastKnownPositionFullDemo: func(
					*parserpkg.ParseDemoResult, string, string, io.Writer,
				) (parserpkg.ObservedLastKnownPositionFullDemoReport, error) {
					lastKnownFullCalled = true
					if testCase.failLastKnownBuilder {
						return parserpkg.ObservedLastKnownPositionFullDemoReport{}, errors.New("last-known full-demo failed")
					}
					return parserpkg.ObservedLastKnownPositionFullDemoReport{}, nil
				},
				validateLastKnownPositionFullDemo: func(parserpkg.ObservedLastKnownPositionFullDemoReport) error {
					if testCase.failLastKnownValidator {
						return errors.New("last-known full-demo validator failed")
					}
					return nil
				},
				export: func(
					*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
				) error {
					exportCalled = true
					return nil
				},
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{
				"--demo", demoPath,
				"--output", outputRoot,
				"--alias", "demo-" + checksum[:20],
				"--checksum", checksum,
				"--maps-root", t.TempDir(),
				"--ai-coach-full-demo-validation",
			}, &stdout, &stderr, deps)
			if code != testCase.wantCode || exportCalled || visualFullCalled != testCase.wantVisualFullCall ||
				weaponFullCalled != testCase.wantWeaponFullCall || utilityVelocityFullCalled != testCase.wantUtilityVelocityFullCall ||
				lastKnownFullCalled != testCase.wantLastKnownFullCall {
				t.Fatalf(
					"code=%d exportCalled=%t visualFullCalled=%t weaponFullCalled=%t utilityVelocityFullCalled=%t lastKnownFullCalled=%t stderr=%s",
					code,
					exportCalled,
					visualFullCalled,
					weaponFullCalled,
					utilityVelocityFullCalled,
					lastKnownFullCalled,
					stderr.String(),
				)
			}
			if !strings.Contains(stdout.String(), `"stage":"`+testCase.wantStage+`"`) {
				t.Fatalf("unexpected structured failure: %s", stdout.String())
			}
			entries, err := os.ReadDir(outputRoot)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("failed full-demo validation published files: %v", entries)
			}
		})
	}
}

func TestFullDemoValidationFlagRules(t *testing.T) {
	base := []string{
		"--demo", "fixture.dem",
		"--output", "out",
		"--alias", "demo-0123456789abcdef0123",
		"--checksum", strings.Repeat("0", 64),
	}
	for _, testCase := range []struct {
		name string
		args []string
	}{
		{name: "requires maps root", args: []string{"--ai-coach-full-demo-validation"}},
		{name: "exclusive with navigation smoke", args: []string{"--maps-root", "maps", "--ai-coach-full-demo-validation", "--navigation-smoke"}},
		{name: "exclusive with AI Coach smoke", args: []string{"--maps-root", "maps", "--ai-coach-full-demo-validation", "--ai-coach-smoke"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if _, err := parseOptions(append(append([]string{}, base...), testCase.args...), &stderr); err == nil {
				t.Fatal("invalid flag combination was accepted")
			}
		})
	}
	var stderr bytes.Buffer
	opts, err := parseOptions(append(append([]string{}, base...),
		"--maps-root", "maps", "--ai-coach-full-demo-validation"), &stderr)
	if err != nil {
		t.Fatalf("standalone full-demo mode was rejected: %v", err)
	}
	if !opts.aiCoachFullDemoValidation || opts.aiCoachSmoke || opts.navigationSmoke {
		t.Fatalf("unexpected parsed full-demo options: %#v", opts)
	}
}

func TestExposureAngleValidationFlagRules(t *testing.T) {
	base := []string{
		"--demo", "fixture.dem",
		"--output", "out",
		"--alias", "demo-0123456789abcdef0123",
		"--checksum", strings.Repeat("0", 64),
	}
	for _, testCase := range []struct {
		name string
		args []string
	}{
		{name: "requires maps root", args: []string{"--exposure-packages-root", "packages", "--ai-coach-exposure-angle-validation"}},
		{name: "requires packages root", args: []string{"--maps-root", "maps", "--ai-coach-exposure-angle-validation"}},
		{name: "exclusive with navigation smoke", args: []string{"--maps-root", "maps", "--exposure-packages-root", "packages", "--ai-coach-exposure-angle-validation", "--navigation-smoke"}},
		{name: "exclusive with full validation", args: []string{"--maps-root", "maps", "--exposure-packages-root", "packages", "--ai-coach-exposure-angle-validation", "--ai-coach-full-demo-validation"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if _, err := parseOptions(append(append([]string{}, base...), testCase.args...), &stderr); err == nil {
				t.Fatal("invalid exposure-angle flag combination was accepted")
			}
		})
	}
	var stderr bytes.Buffer
	opts, err := parseOptions(append(append([]string{}, base...),
		"--maps-root", "maps",
		"--exposure-packages-root", "packages",
		"--ai-coach-exposure-angle-validation"), &stderr)
	if err != nil {
		t.Fatalf("standalone exposure-angle mode was rejected: %v", err)
	}
	if !opts.aiCoachExposureValidation || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.navigationSmoke {
		t.Fatalf("unexpected parsed exposure-angle options: %#v", opts)
	}
}

func TestWriteDeterministicGzipArtifact(t *testing.T) {
	payload := []byte("{\"round\":1}\n{\"round\":2}\n")
	firstPath := filepath.Join(t.TempDir(), "first.jsonl.gz")
	secondPath := filepath.Join(t.TempDir(), "second.jsonl.gz")
	write := func(path string) (int64, string) {
		t.Helper()
		bytesWritten, digest, err := writeDeterministicGzipArtifact(path, func(writer io.Writer) error {
			_, err := writer.Write(payload)
			return err
		})
		if err != nil {
			t.Fatalf("write deterministic gzip: %v", err)
		}
		return bytesWritten, digest
	}
	firstBytes, firstSHA := write(firstPath)
	secondBytes, secondSHA := write(secondPath)
	firstPayload, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondPayload, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if firstBytes != int64(len(firstPayload)) || secondBytes != int64(len(secondPayload)) ||
		firstBytes != secondBytes || firstSHA != secondSHA || !bytes.Equal(firstPayload, secondPayload) {
		t.Fatalf("gzip output was not deterministic: bytes=(%d,%d) sha=(%s,%s)", firstBytes, secondBytes, firstSHA, secondSHA)
	}
	if got := readGzipArtifact(t, firstPath); !bytes.Equal(got, payload) {
		t.Fatalf("decompressed payload = %q, want %q", got, payload)
	}
	handle, err := os.Open(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(handle)
	if err != nil {
		_ = handle.Close()
		t.Fatal(err)
	}
	if !reader.ModTime.IsZero() || reader.Name != "" || reader.Comment != "" || reader.OS != 255 {
		t.Fatalf("non-deterministic gzip header: %#v", reader.Header)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}

	failedPath := filepath.Join(t.TempDir(), "failed.jsonl.gz")
	_, _, err = writeDeterministicGzipArtifact(failedPath, func(writer io.Writer) error {
		_, _ = writer.Write([]byte("partial"))
		return errors.New("producer rejected the stream")
	})
	if err == nil {
		t.Fatal("producer failure was ignored")
	}
	if _, statErr := os.Stat(failedPath); !os.IsNotExist(statErr) {
		t.Fatalf("partial gzip was retained: %v", statErr)
	}
}

func TestRunDoesNotExportRejectedVisualContactSmoke(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := emptySmokeOutputRoot(t)
	exportCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{
				Context: &models.DemoContext{}, ReplayData: &models.ReplayData{},
			}, nil
		},
		navigationSmoke: func(
			*parserpkg.ParseDemoResult, string, string, int,
		) (parserpkg.ObservedNavigationSmokeReport, error) {
			return parserpkg.ObservedNavigationSmokeReport{Status: "passed"}, nil
		},
		visualContactSmoke: func(
			*parserpkg.ParseDemoResult, string, string, string, int,
		) (parserpkg.ObservedVisualContactSmokeReport, error) {
			return parserpkg.ObservedVisualContactSmokeReport{}, errors.New("real smoke coverage is incomplete")
		},
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			exportCalled = true
			return nil
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
		"--maps-root", t.TempDir(),
		"--ai-coach-smoke",
	}, &stdout, &stderr, deps)
	if code != 8 || exportCalled {
		t.Fatalf("code=%d exportCalled=%t stderr=%s", code, exportCalled, stderr.String())
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected visual contact smoke published files: %v", entries)
	}
}

func TestRunDoesNotExportRejectedCoverSidecar(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := emptySmokeOutputRoot(t)
	exportCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{Context: &models.DemoContext{}, ReplayData: &models.ReplayData{}}, nil
		},
		coverFullDemo: func(
			*parserpkg.ParseDemoResult, string, string, string, string, io.Writer,
		) (parserpkg.ObservedCoverFullDemoReport, error) {
			return parserpkg.ObservedCoverFullDemoReport{}, errors.New("no affected objective cover records")
		},
		validateCoverFullDemo: parserpkg.ValidateObservedCoverFullDemoReport,
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			exportCalled = true
			return nil
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
		"--maps-root", t.TempDir(),
		"--cover-packages-root", t.TempDir(),
		"--ai-coach-cover-validation",
	}, &stdout, &stderr, deps)
	if code != 15 || exportCalled {
		t.Fatalf("code=%d exportCalled=%t stderr=%s", code, exportCalled, stderr.String())
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected cover sidecar published files: %v", entries)
	}
}

func TestCoverValidationModeIsStandaloneAndRequiresBothRoots(t *testing.T) {
	base := []string{"--demo", "fixture.dem", "--output", "out", "--alias", "demo-aaaaaaaaaaaaaaaaaaaa", "--checksum", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--ai-coach-cover-validation"}
	if _, err := parseOptions(base, io.Discard); err == nil || !strings.Contains(err.Error(), "--maps-root") {
		t.Fatalf("missing maps root was accepted: %v", err)
	}
	withMaps := append(append([]string{}, base...), "--maps-root", "maps")
	if _, err := parseOptions(withMaps, io.Discard); err == nil || !strings.Contains(err.Error(), "--cover-packages-root") {
		t.Fatalf("missing cover packages root was accepted: %v", err)
	}
	conflict := append(append([]string{}, withMaps...), "--cover-packages-root", "cover", "--ai-coach-full-demo-validation")
	if _, err := parseOptions(conflict, io.Discard); err == nil || !strings.Contains(err.Error(), "standalone") {
		t.Fatalf("conflicting cover validation mode was accepted: %v", err)
	}
}

func TestExitRouteValidationModeIsStandaloneAndRequiresBothRoots(t *testing.T) {
	base := []string{"--demo", "fixture.dem", "--output", "out", "--alias", "demo-aaaaaaaaaaaaaaaaaaaa", "--checksum", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--ai-coach-exit-route-validation"}
	if _, err := parseOptions(base, io.Discard); err == nil || !strings.Contains(err.Error(), "--maps-root") {
		t.Fatalf("missing maps root was accepted: %v", err)
	}
	withMaps := append(append([]string{}, base...), "--maps-root", "maps")
	if _, err := parseOptions(withMaps, io.Discard); err == nil || !strings.Contains(err.Error(), "--exit-route-packages-root") {
		t.Fatalf("missing exit-route packages root was accepted: %v", err)
	}
	for _, conflictFlag := range []string{"--navigation-smoke", "--ai-coach-full-demo-validation", "--ai-coach-exposure-angle-validation", "--ai-coach-cover-validation"} {
		conflict := append(append([]string{}, withMaps...), "--exit-route-packages-root", "exit", conflictFlag)
		if _, err := parseOptions(conflict, io.Discard); err == nil || !strings.Contains(err.Error(), "standalone") {
			t.Fatalf("conflict %s was accepted: %v", conflictFlag, err)
		}
	}
}

func TestRunDoesNotParseOrExportWhenExitRouteOutputIsNonTemporary(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	parseCalled := false
	exportCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			parseCalled = true
			return nil, errors.New("must not parse")
		},
		export: func(*models.DemoContext, string, string, string, models.CanonicalExportProvenance) error {
			exportCalled = true
			return nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--demo", demoPath, "--output", filepath.Join(filepath.Dir(demoPath), "not-system-temp"), "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--maps-root", t.TempDir(), "--exit-route-packages-root", t.TempDir(), "--ai-coach-exit-route-validation"}, &stdout, &stderr, deps)
	if code != 6 || parseCalled || exportCalled {
		t.Fatalf("code=%d parse=%t export=%t stderr=%s", code, parseCalled, exportCalled, stderr.String())
	}
}

func TestRunDoesNotExportRejectedExitRouteSidecar(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := emptySmokeOutputRoot(t)
	exportCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{Context: &models.DemoContext{}, ReplayData: &models.ReplayData{}}, nil
		},
		exitRouteFullDemo: func(*parserpkg.ParseDemoResult, string, string, string, string, io.Writer) (parserpkg.ObservedExitRouteFullDemoReport, error) {
			return parserpkg.ObservedExitRouteFullDemoReport{}, errors.New("no affected exit routes")
		},
		validateExitRouteFullDemo: parserpkg.ValidateObservedExitRouteFullDemoReport,
		export: func(*models.DemoContext, string, string, string, models.CanonicalExportProvenance) error {
			exportCalled = true
			return nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--demo", demoPath, "--output", outputRoot, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--maps-root", t.TempDir(), "--exit-route-packages-root", t.TempDir(), "--ai-coach-exit-route-validation"}, &stdout, &stderr, deps)
	if code != 16 || exportCalled {
		t.Fatalf("code=%d export=%t stderr=%s", code, exportCalled, stderr.String())
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected exit-route sidecar published files: %v", entries)
	}
}

type provenanceMapStub struct {
	inputs map[string]mapspkg.InputProvenance
}

func (stub *provenanceMapStub) IsLoaded() bool                      { return true }
func (stub *provenanceMapStub) IsVisible(r3.Vector, r3.Vector) bool { return true }
func (stub *provenanceMapStub) GetCallout(r3.Vector) string         { return "" }
func (stub *provenanceMapStub) RayCast(r3.Vector, r3.Vector, float64) (float64, r3.Vector) {
	return -1, r3.Vector{}
}
func (stub *provenanceMapStub) InputProvenance() map[string]mapspkg.InputProvenance {
	return stub.inputs
}

func TestVisualContactGeometryAssetsHashGLTFAndExternalBuffer(t *testing.T) {
	root := t.TempDir()
	mapDir := filepath.Join(root, "de_test")
	if err := os.MkdirAll(mapDir, 0o750); err != nil {
		t.Fatal(err)
	}
	bufferPath := filepath.Join(mapDir, "physics.bin")
	if err := os.WriteFile(bufferPath, []byte("geometry-buffer"), 0o600); err != nil {
		t.Fatal(err)
	}
	gltfPath := filepath.Join(mapDir, "physics.gltf")
	if err := os.WriteFile(gltfPath, []byte(`{"buffers":[{"uri":"physics.bin"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	gltfSHA, err := sha256File(gltfPath)
	if err != nil {
		t.Fatal(err)
	}
	stub := &provenanceMapStub{inputs: map[string]mapspkg.InputProvenance{
		"physics_map": {
			RelativePath: "de_test/physics.gltf", SHA256: gltfSHA, Used: true,
		},
	}}
	assets, err := visualContactGeometryAssets(stub, root)
	if err != nil {
		t.Fatalf("visualContactGeometryAssets() error = %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("assets = %#v, want GLTF and buffer", assets)
	}
	roles := map[string]bool{}
	for _, asset := range assets {
		roles[asset.Role] = true
		if len(asset.SHA256) != 64 || filepath.IsAbs(asset.Reference) {
			t.Fatalf("unsafe asset provenance: %#v", asset)
		}
	}
	if !roles["physics_gltf"] || !roles["physics_buffer"] {
		t.Fatalf("asset roles = %v", roles)
	}
}

func TestRunDoesNotExportOrPublishRejectedNavigationSmoke(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := emptySmokeOutputRoot(t)
	exportCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{
				Context:    &models.DemoContext{},
				ReplayData: &models.ReplayData{},
			}, nil
		},
		navigationSmoke: func(
			*parserpkg.ParseDemoResult, string, string, int,
		) (parserpkg.ObservedNavigationSmokeReport, error) {
			return parserpkg.ObservedNavigationSmokeReport{}, errors.New("no reachable real route")
		},
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			exportCalled = true
			return nil
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
		"--navigation-smoke",
	}, &stdout, &stderr, deps)
	if code != 7 || exportCalled {
		t.Fatalf("code=%d exportCalled=%t stderr=%s", code, exportCalled, stderr.String())
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected smoke published files: %v", entries)
	}
}

func TestNavigationSmokeOutputMustBeDedicatedEmptyTempDirectory(t *testing.T) {
	filesystemRoot := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	if err := validateNavigationSmokeOutputRoot(filesystemRoot); err == nil {
		t.Fatal("filesystem root was accepted for an isolated navigation smoke")
	}
	if err := validateNavigationSmokeOutputRoot(os.TempDir()); err == nil {
		t.Fatal("system temporary root was accepted as a dedicated smoke directory")
	}
	missing := filepath.Join(t.TempDir(), "caller-did-not-create-this")
	if err := validateNavigationSmokeOutputRoot(missing); err == nil {
		t.Fatal("non-existent smoke output root was accepted")
	}
	existingEmpty := emptySmokeOutputRoot(t)
	if err := validateNavigationSmokeOutputRoot(existingEmpty); err != nil {
		t.Fatalf("dedicated temp descendant was rejected: %v", err)
	}

	demoPath, checksum := writeDemoFixture(t)
	outputRoot := filepath.Join(t.TempDir(), "non-empty-output")
	if err := os.MkdirAll(outputRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputRoot, "unrelated"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	parseCalled := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			parseCalled = true
			return nil, errors.New("must not parse into a non-isolated output")
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
		"--navigation-smoke",
	}, &stdout, &stderr, deps)
	if code != 6 || parseCalled {
		t.Fatalf("code=%d parseCalled=%t stderr=%s", code, parseCalled, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "unrelated")); err != nil {
		t.Fatalf("pre-existing output was changed: %v", err)
	}
}

func TestNavigationSmokeOutputRejectsSymlinkComponent(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "physical-target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(parent, "linked-component")
	if err := os.Symlink(target, linked); err != nil {
		t.Skipf("directory symlink is unavailable on this platform: %v", err)
	}
	outputRoot := filepath.Join(linked, "output")
	if err := os.Mkdir(filepath.Join(target, "output"), 0o750); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == outputRoot {
		t.Fatal("symlink fixture did not change physical resolution")
	}
	if err := validateNavigationSmokeOutputRoot(outputRoot); err == nil {
		t.Fatal("smoke output below a symlink component was accepted")
	}
}

func TestNavigationSmokeOutputRejectsJunctionComponentWhenAvailable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("directory junctions are Windows-specific")
	}
	parent := t.TempDir()
	target := filepath.Join(parent, "junction-target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(parent, "junction-component")
	command := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", junction, target)
	if output, err := command.CombinedOutput(); err != nil {
		t.Skipf("directory junction creation is unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	outputRoot := filepath.Join(junction, "output")
	if err := os.Mkdir(filepath.Join(target, "output"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := validateNavigationSmokeOutputRoot(outputRoot); err == nil {
		t.Fatal("smoke output below a junction component was accepted")
	}
}

func TestRunRefusesToOverwriteBundle(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := t.TempDir()
	finalBundle := filepath.Join(outputRoot, "match_demo-"+checksum[:20])
	if err := os.Mkdir(finalBundle, 0o750); err != nil {
		t.Fatal(err)
	}
	called := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			called = true
			return nil, nil
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 6 {
		t.Fatalf("exit code = %d, want 6", code)
	}
	if called {
		t.Fatal("parser was called before overwrite protection")
	}
}

func writeDemoFixture(t *testing.T) (string, string) {
	t.Helper()
	payload := []byte("PBDEMS2\x00fixture")
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])
	path := filepath.Join(t.TempDir(), "fixture.dem")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, checksum
}

func emptySmokeOutputRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "out")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	return root
}

func relativeRegularFiles(t *testing.T, root string) []string {
	t.Helper()
	files := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func readGzipArtifact(t *testing.T, path string) []byte {
	t.Helper()
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	reader, err := gzip.NewReader(handle)
	if err != nil {
		t.Fatal(err)
	}
	payload, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return payload
}

func assertCompactJSON(t *testing.T, payload []byte) {
	t.Helper()
	if !json.Valid(payload) {
		t.Fatalf("invalid JSON report: %q", payload)
	}
	if bytes.Count(payload, []byte{'\n'}) != 1 || !bytes.HasSuffix(payload, []byte{'\n'}) ||
		bytes.Contains(payload, []byte("\n ")) {
		t.Fatalf("report is not compact single-line JSON: %q", payload)
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}
