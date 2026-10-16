package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

// These are the actual source bytes bundled into this executable, not a caller
// supplied version label. Both platform guards are recorded; Go's selected
// GOOS and complete module build information are recorded alongside them.
//
//go:embed main.go contract.go native.go recorder.go output_guard.go path_reparse_other.go path_reparse_windows.go
var producerSources embed.FS

type artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

func hashBytes(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), n, err
}
func exclusiveJSON(path string, value any) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	encodeErr := json.NewEncoder(f).Encode(value)
	closeErr := f.Close()
	return errors.Join(encodeErr, closeErr)
}

type runOptions struct{ input, output, targets, sourceSHA, targetsSHA string }

func run(o runOptions) (result error) {
	// All output checks precede reading any source, target or parser input.
	if err := validateOutput(o.output); err != nil {
		return err
	}
	if !hashPattern.MatchString(o.sourceSHA) || !hashPattern.MatchString(o.targetsSHA) {
		return errors.New("explicit lowercase source and target SHA-256 required")
	}
	start := time.Now()
	raw, err := os.ReadFile(o.targets)
	if err != nil {
		return err
	}
	if hashBytes(raw) != o.targetsSHA {
		return errors.New("targets file hash differs")
	}
	cfg, err := decodeConfig(raw, o.sourceSHA)
	if err != nil {
		return err
	}
	before, inputBytes, err := hashFile(o.input)
	if err != nil {
		return err
	}
	if before != o.sourceSHA {
		return errors.New("source file hash differs")
	}
	if inputBytes == 0 {
		return errors.New("empty demo source")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executableHash, executableBytes, err := hashFile(executable)
	if err != nil {
		return err
	}
	buildInfo, buildAvailable := debug.ReadBuildInfo()
	if !buildAvailable {
		return errors.New("Go module build information unavailable")
	}
	sources := []artifact{}
	entries, err := producerSources.ReadDir(".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := producerSources.ReadFile(e.Name())
		if err != nil {
			return err
		}
		sources = append(sources, artifact{e.Name(), hashBytes(b), int64(len(b))})
	}
	// Check again immediately before exclusive file creation after hashing.
	if err := validateOutput(o.output); err != nil {
		return err
	}
	files := []*os.File{}
	defer func() {
		for _, f := range files {
			_ = f.Close()
		}
	}()
	for _, name := range []string{"snapshots.jsonl", "deliveries.jsonl", "state_history.jsonl"} {
		f, err := os.OpenFile(filepath.Join(o.output, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		files = append(files, f)
	}
	in, err := os.Open(o.input)
	if err != nil {
		return err
	}
	defer in.Close()
	parserConfig := dem.DefaultParserConfig
	parserConfig.MsgQueueBufferSize = 0 // Do not pre-process a later frame.
	p := dem.NewParserWithConfig(in, parserConfig)
	defer p.Close()
	r := &recorder{p: p, cfg: cfg, outputSnapshots: json.NewEncoder(files[0]), outputDeliveries: json.NewEncoder(files[1]), outputChanges: json.NewEncoder(files[2])}
	r.register()
	last := cfg.Targets[len(cfg.Targets)-1].Cut
	for r.frame < last.Frame && r.err == nil {
		more, err := p.ParseNextFrame()
		if err != nil {
			return errors.New("native demo parser failed; partial output retained")
		}
		if !more {
			return errors.New("EOF before last target cut; partial output retained")
		}
	}
	if r.err != nil {
		return r.err
	}
	if r.snapshots != len(cfg.Targets) || r.now() != last {
		return errors.New("not every requested exact cut was captured")
	}
	if r.header == nil || r.header["map_name"] != cfg.MapName {
		return errors.New("demo map header differs from target")
	}
	for _, f := range files {
		if err := f.Close(); err != nil {
			return err
		}
	}
	after, afterBytes, err := hashFile(o.input)
	if err != nil {
		return err
	}
	if after != before || afterBytes != inputBytes {
		return errors.New("demo source changed during capture")
	}
	afterTargets, targetBytes, err := hashFile(o.targets)
	if err != nil {
		return err
	}
	if afterTargets != o.targetsSHA || targetBytes != int64(len(raw)) {
		return errors.New("targets changed during capture")
	}
	artifacts := map[string]any{}
	for _, name := range []string{"snapshots.jsonl", "deliveries.jsonl", "state_history.jsonl"} {
		h, n, err := hashFile(filepath.Join(o.output, name))
		if err != nil {
			return err
		}
		artifacts[name] = map[string]any{"sha256": h, "bytes": n}
	}
	manifest := map[string]any{
		"schema_id": "stratai.native_place_manifest@1", "producer_version": producerVersion,
		"created_at": time.Now().UTC().Format(time.RFC3339Nano), "duration_seconds": time.Since(start).Seconds(),
		"source_alias": cfg.SourceAlias, "source_sha256": before, "source_sha256_after": after, "source_bytes": inputBytes,
		"targets_sha256": o.targetsSHA, "targets_sha256_after": afterTargets, "targets_bytes": len(raw),
		"target_bindings": cfg.Targets, "header": r.header, "property_name": placeField,
		"code":      map[string]any{"version": producerVersion, "source_files": sources, "executable_sha256": executableHash, "executable_bytes": executableBytes, "go_build_info": buildInfo},
		"capture":   map[string]any{"message_queue_buffer_size": 0, "availability_boundary": "FrameDone", "sdk": "demoinfocs/v5@v5.2.0", "last_processed_cut": r.now(), "snapshots": r.snapshots, "property_deliveries": r.deliveries, "state_changes": r.changes, "frames_sampled": r.framesSampled, "actors_sampled": r.actorsSampled},
		"artifacts": artifacts, "map_assets_loaded": false, "future_frames_processed": false, "training_allowed": false, "semantic_zone_accredited": false, "tactical_prediction_created": false,
		"limitations": []string{
			"Native last-place property is preserved verbatim; it is not geometric zone accreditation or a tactical destination.",
			"Property delivery is the completed-frame availability boundary, not the time of a physical zone transition.",
			"Last native delivery is limited to the same round instance and tick epoch; null may mean an inherited native value.",
			"Current raw property does not establish individual player perception, communication, route viability or intent.",
			"Candidate context hashes bind caller-supplied targets; an independent consumer must verify those context bytes and actor completeness.",
			"Native source data are an additional factual layer; availability in an earlier human instrument is not asserted.",
		},
	}
	return exclusiveJSON(filepath.Join(o.output, "manifest.json"), manifest)
}

func main() {
	var o runOptions
	flag.StringVar(&o.input, "input", "", "read-only approved source demo")
	flag.StringVar(&o.output, "output", "", "existing, empty dedicated TEMP directory")
	flag.StringVar(&o.targets, "targets", "", "strict factual target document")
	flag.StringVar(&o.sourceSHA, "source-sha256", "", "expected source SHA-256")
	flag.StringVar(&o.targetsSHA, "targets-sha256", "", "expected targets SHA-256")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(1)
	}
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
