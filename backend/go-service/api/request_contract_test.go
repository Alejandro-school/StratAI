package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessDemoRejectsUnsupportedVersionBeforeAnyWork(t *testing.T) {
	// Invalid path and a full slot semaphore prove rejection precedes I/O and capacity acquisition.
	oldSlots := analysisSlots
	analysisSlots = make(chan struct{}, 1)
	analysisSlots <- struct{}{}
	t.Cleanup(func() { analysisSlots = oldSlots })
	root := t.TempDir()
	t.Chdir(root)
	for _, version := range []string{"v11", "v3", "v999", " v16", "V16"} {
		payload, err := json.Marshal(ProcessDemoRequest{
			DemoPath: "forbidden.dem", MatchID: "contract-match", Checksum: strings.Repeat("a", 64), SchemaVersion: version,
		})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		request := httptest.NewRequest(http.MethodPost, "/process-demo", bytes.NewReader(payload)).WithContext(ctx)
		HandleProcessDemo(response, request)
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "parser_schema_version") {
			t.Fatalf("version %q: status=%d body=%s", version, response.Code, response.Body.String())
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejection mutated filesystem: %v %v", entries, err)
	}
	if len(analysisSlots) != 1 {
		t.Fatal("rejection changed processing slots")
	}
}

func TestProcessDemoAcceptsNodeRequestsAndOmittedVersion(t *testing.T) {
	helper, err := filepath.Abs(filepath.Join("..", "..", "node-service", "test", "helpers", "emitAnalysisRequest.js"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("node", helper).Output()
	if err != nil {
		t.Fatalf("Node contract fixture failed: %v", err)
	}
	var requests []ProcessDemoRequest
	if err := json.Unmarshal(output, &requests); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 {
		t.Fatalf("expected new, default and recovered Node requests: %d", len(requests))
	}
	omitted := requests[0]
	omitted.SchemaVersion = ""
	requests = append(requests, omitted)
	root := t.TempDir()
	service := filepath.Join(root, "service")
	if err := os.Mkdir(service, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(service)
	for i, request := range requests {
		// Only the fixture path changes: the Node-emitted version is sent unchanged.
		request.DemoPath = filepath.Join(root, "data", "demos", "absent.dem")
		payload, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		HandleProcessDemo(response, httptest.NewRequest(http.MethodPost, "/process-demo", bytes.NewReader(payload)))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Invalid demo path") {
			t.Fatalf("request %d version %q did not pass version gate: %d %s", i, request.SchemaVersion, response.Code, response.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatalf("request mutated data directory: %v", err)
	}
}
