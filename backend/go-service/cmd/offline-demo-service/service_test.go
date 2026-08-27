package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthReportsImmutableReleaseIdentity(t *testing.T) {
	service := newOfflineService(offlineConfig{maxConcurrent: 1}, "stratai-test-build", nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	service.routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d", recorder.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" || payload["service"] != "stratai-offline-demo-service" {
		t.Fatalf("unexpected health payload: %#v", payload)
	}
	if payload["build_id"] != "stratai-test-build" || payload["parser_schema_version"] != parserVersion ||
		payload["export_format_version"] != exportVersion || payload["validator_version"] != validatorVersion {
		t.Fatalf("health did not preserve release identity: %#v", payload)
	}
	if payload["quality_schema_version"] != float64(qualityVersion) {
		t.Fatalf("quality schema version = %#v", payload["quality_schema_version"])
	}
}

func TestProcessDemoPublishesPendingRootManifestAndCanonicalBundle(t *testing.T) {
	demosRoot := t.TempDir()
	exportRoot := t.TempDir()
	demoPath, checksum := writeDemo(t, demosRoot, "match.dem")
	processor := func(
		_ context.Context, payload processDemoRequest, outputRoot, buildIdentifier string,
	) (processingResult, error) {
		if buildIdentifier != "stratai-test-build" || payload.MatchID != "match-123" {
			t.Fatalf("unexpected processor identity: %q %#v", buildIdentifier, payload)
		}
		writeCanonicalFixture(t, filepath.Join(outputRoot, "match_"+payload.MatchID))
		return processingResult{Kills: 7, Rounds: 2}, nil
	}
	service := newOfflineService(offlineConfig{
		demosRoot: demosRoot, exportRoot: exportRoot, maxConcurrent: 1,
	}, "stratai-test-build", processor)
	payload := processDemoRequest{
		DemoPath: demoPath, MatchID: "match-123", Checksum: checksum,
		SchemaVersion: parserVersion, Source: "faceit",
	}
	recorder := postDemo(t, service, payload)
	if recorder.Code != http.StatusOK {
		t.Fatalf("process status = %d, body = %q", recorder.Code, recorder.Body.String())
	}

	bundle := filepath.Join(exportRoot, "match_match-123")
	manifestData, err := os.ReadFile(filepath.Join(bundle, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest exportManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ValidationStatus != "pending" || manifest.MatchID != "match-123" || manifest.Checksum != checksum {
		t.Fatalf("unexpected root manifest: %#v", manifest)
	}
	if manifest.ParserSchemaVersion != parserVersion || manifest.ExportFormatVersion != exportVersion ||
		manifest.ValidatorVersion != validatorVersion || manifest.CommittedAt.IsZero() {
		t.Fatalf("root manifest versions are incomplete: %#v", manifest)
	}
	paths := make([]string, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		paths = append(paths, artifact.Path)
		artifactPath := filepath.Join(bundle, filepath.FromSlash(artifact.Path))
		actual, hashErr := hashFile(artifactPath)
		if hashErr != nil || actual != artifact.SHA256 {
			t.Fatalf("artifact %s hash mismatch", artifact.Path)
		}
	}
	if fmt.Sprint(paths) != "[canonical/core/match.json canonical/manifest.json]" {
		t.Fatalf("unexpected artifact inventory: %v", paths)
	}
	entries, err := os.ReadDir(exportRoot)
	if err != nil || len(entries) != 1 || entries[0].Name() != "match_match-123" {
		t.Fatalf("publication was not atomic: %v, %v", entries, err)
	}
}

func TestProcessDemoRejectsPathOutsideConfiguredRoot(t *testing.T) {
	demosRoot := t.TempDir()
	exportRoot := t.TempDir()
	outsidePath, checksum := writeDemo(t, t.TempDir(), "outside.dem")
	called := false
	service := newOfflineService(offlineConfig{
		demosRoot: demosRoot, exportRoot: exportRoot, maxConcurrent: 1,
	}, "stratai-test-build", func(context.Context, processDemoRequest, string, string) (processingResult, error) {
		called = true
		return processingResult{}, nil
	})

	recorder := postDemo(t, service, processDemoRequest{
		DemoPath: outsidePath, MatchID: "match-123", Checksum: checksum,
	})
	if recorder.Code != http.StatusBadRequest || called {
		t.Fatalf("outside path status = %d, processor called = %v", recorder.Code, called)
	}
}

func TestFailedExportLeavesNoPublishedOrTemporaryBundle(t *testing.T) {
	demosRoot := t.TempDir()
	exportRoot := t.TempDir()
	demoPath, checksum := writeDemo(t, demosRoot, "match.dem")
	service := newOfflineService(offlineConfig{
		demosRoot: demosRoot, exportRoot: exportRoot, maxConcurrent: 1,
	}, "stratai-test-build", func(
		_ context.Context, payload processDemoRequest, outputRoot, _ string,
	) (processingResult, error) {
		if err := os.MkdirAll(filepath.Join(outputRoot, "match_"+payload.MatchID), 0o750); err != nil {
			t.Fatal(err)
		}
		return processingResult{}, errors.New("quality gate rejected demo")
	})

	recorder := postDemo(t, service, processDemoRequest{
		DemoPath: demoPath, MatchID: "match-123", Checksum: checksum,
	})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("failed export status = %d", recorder.Code)
	}
	entries, err := os.ReadDir(exportRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed export left data behind: %v, %v", entries, err)
	}
}

func TestLoopbackAddressIsRequired(t *testing.T) {
	for _, address := range []string{"0.0.0.0:18080", "192.0.2.1:18080", ":18080"} {
		if err := validateLoopbackAddress(address); err == nil {
			t.Fatalf("address %q should be rejected", address)
		}
	}
	for _, address := range []string{"127.0.0.1:18080", "[::1]:18080", "localhost:18080"} {
		if err := validateLoopbackAddress(address); err != nil {
			t.Fatalf("address %q rejected: %v", address, err)
		}
	}
}

func postDemo(t *testing.T, service *offlineService, payload processDemoRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/process-demo", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	service.routes().ServeHTTP(recorder, request)
	return recorder
}

func writeDemo(t *testing.T, root, name string) (string, string) {
	t.Helper()
	path := filepath.Join(root, name)
	data := bytes.Repeat([]byte("demo"), 25*1024+1)
	if err := os.WriteFile(path, data, 0o640); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return path, fmt.Sprintf("%x", digest[:])
}

func writeCanonicalFixture(t *testing.T, bundle string) {
	t.Helper()
	matchPath := filepath.Join(bundle, "canonical", "core", "match.json")
	if err := os.MkdirAll(filepath.Dir(matchPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(matchPath, []byte(`{"map_name":"de_mirage"}`), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]interface{}{
		"schema_id": "stratai.canonical_manifest@3",
		"artifacts": []map[string]interface{}{{
			"artifact_type": "match", "path": "core/match.json", "format": "json",
			"compression": "none", "schema_id": "stratai.match@1", "record_count": 1,
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "canonical", "manifest.json"), data, 0o640); err != nil {
		t.Fatal(err)
	}
}
