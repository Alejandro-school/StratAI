// Read-only native source survey. No hearing model, replay export or promotion.
package main

import (
	"crypto/sha256"
	"cs2-demo-service/pkg/sound"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func run(input, output, expectedSHA string) error {
	// This check MUST precede opening the demo, even if it does not exist.
	if err := validateOutput(output); err != nil {
		return err
	}
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	bytesRead, err := io.Copy(hash, f)
	if err != nil {
		return err
	}
	actualSHA := hex.EncodeToString(hash.Sum(nil))
	if len(expectedSHA) != 64 || actualSHA != expectedSHA {
		return errors.New("demo checksum differs from explicit approved SHA-256")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := validateOutput(output); err != nil {
		return err
	}
	artifact, err := os.OpenFile(filepath.Join(output, "native-sound-audit.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer artifact.Close()
	p := dem.NewParser(f)
	defer p.Close()
	audit := sound.RegisterSourceAudit(p)
	warnings := map[int]int{}
	unknownMessages := map[int]int{}
	frames, roundEnds := 0, 0
	p.RegisterEventHandler(func(e events.ParserWarn) {
		warnings[int(e.Type)]++
		if e.Type == events.WarnTypeUnknownProtobufMessage {
			var id int
			if _, err := fmt.Sscanf(e.Message, "unknown message type: %d", &id); err == nil {
				unknownMessages[id]++
			}
		}
	})
	p.RegisterEventHandler(func(e events.FrameDone) { frames++ })
	p.RegisterEventHandler(func(e events.RoundEnd) { roundEnds++ })
	started := time.Now()
	if err := p.ParseToEnd(); err != nil {
		return err
	}
	// Alias every observed participant before retention. There is no retained
	// reverse map and no native participant name in this survey's contract.
	ids := []uint64{}
	for _, signal := range audit.Signals {
		if signal.EmitterID != 0 {
			ids = append(ids, signal.EmitterID)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	aliases := map[uint64]uint64{}
	for i, id := range ids {
		aliases[id] = uint64(i + 1)
	}
	for i := range audit.Signals {
		audit.Signals[i].EmitterID = aliases[audit.Signals[i].EmitterID]
	}
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("producer build info unavailable")
	}
	result := map[string]any{
		"source_alias": "demo-" + actualSHA[:20], "source_sha256": actualSHA, "source_bytes": bytesRead,
		"audit": audit, "seconds": time.Since(started).Seconds(), "frame_done_count": frames,
		"round_end_count": roundEnds, "last_ingame_tick": p.GameState().IngameTick(),
		"tick_rate": p.TickRate(), "opaque_signal_participants": len(ids), "warnings_by_type": warnings,
		"unknown_native_message_ids": unknownMessages,
		"producer_build":             build.String(), "reception_records": 0,
	}
	return json.NewEncoder(artifact).Encode(result)
}

func main() {
	demo := flag.String("demo", "", "read-only approved demo")
	output := flag.String("output", "", "new empty isolated directory under system TEMP")
	checksum := flag.String("checksum", "", "explicit approved source SHA-256")
	flag.Parse()
	if err := run(*demo, *output, *checksum); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
