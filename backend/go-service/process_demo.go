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
	demoFilename := "match_6DhZ28LiQKRMfj4tDG8CKUGnK.dem"
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

	// 2. Export
	matchID := "test_verification"
	err = parser.ExportTimelineData(ctx, matchID, outputDir)
	if err != nil {
		log.Fatalf("Error exporting timeline: %v", err)
	}

	err = parser.ExportAnalysisData(ctx, matchID, outputDir)
	if err != nil {
		log.Fatalf("Error exporting analysis: %v", err)
	}

	fmt.Println("Done!")
}
