//go:build ignore

package main

import (
	"cs2-demo-service/parser"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	demoFilename := "match_EM3x9b7fGC2uZEMFCtKfZjenL.dem"
	demoPath := filepath.Join("..", "data", "demos", demoFilename)
	outputDir := filepath.Join("..", "data", "exports")

	f, err := os.Create("debug_output.log")
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	fmt.Printf("Processing demo: %s\n", demoPath)
	start := time.Now()

	// 1. Parse
	ctx, err := parser.ParseDemo(demoPath)
	if err != nil {
		log.Fatalf("Error parsing demo: %v", err)
	}

	fmt.Printf("Parsing took: %v\n", time.Since(start))
	fmt.Printf("Rounds: %d\n", len(ctx.RoundTimelines))

	// 2. Export (date comes from Node via API - empty for local testing)
	matchID := "test_verification"

	// For local testing, date would normally come from Steam GC via Node service
	// Pass empty string since we don't have GC access here
	err = parser.ExportAIModels(ctx, matchID, outputDir)
	if err != nil {
		log.Fatalf("Error exporting AI models: %v", err)
	}

	fmt.Println("Done!")
}
