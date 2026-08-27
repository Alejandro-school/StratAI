package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var buildID = "development"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--release-metadata" {
		_ = json.NewEncoder(os.Stdout).Encode(releaseMetadata(buildID))
		return
	}
	if len(os.Args) != 1 {
		log.Fatal("unsupported command-line arguments")
	}

	config, err := configFromEnvironment()
	if err != nil {
		log.Fatalf("invalid offline service configuration: %v", err)
	}
	service := newOfflineService(config, buildID, nil)
	server := &http.Server{
		Addr:              config.address,
		Handler:           service.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("event=offline_demo_service_started addr=%s build_id=%s", config.address, buildID)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case signal := <-stop:
		log.Printf("event=offline_demo_service_stopping signal=%s", signal)
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("offline HTTP service failed: %v", err)
		}
		return
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("offline HTTP shutdown failed: %v", err)
	}
}

func configFromEnvironment() (offlineConfig, error) {
	address := strings.TrimSpace(os.Getenv("GO_SERVICE_ADDR"))
	if address == "" {
		address = "127.0.0.1:18080"
	}
	if err := validateLoopbackAddress(address); err != nil {
		return offlineConfig{}, err
	}
	demosRoot, err := existingDirectoryFromEnv("FACEIT_DEMOS_DIR")
	if err != nil {
		return offlineConfig{}, err
	}
	exportRootValue := strings.TrimSpace(os.Getenv("FACEIT_EXPORT_ROOT"))
	if exportRootValue == "" {
		return offlineConfig{}, errors.New("FACEIT_EXPORT_ROOT is required")
	}
	exportRoot, err := filepath.Abs(exportRootValue)
	if err != nil {
		return offlineConfig{}, fmt.Errorf("resolve FACEIT_EXPORT_ROOT: %w", err)
	}
	if err := os.MkdirAll(exportRoot, 0o750); err != nil {
		return offlineConfig{}, fmt.Errorf("prepare FACEIT_EXPORT_ROOT: %w", err)
	}
	info, err := os.Lstat(exportRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return offlineConfig{}, errors.New("FACEIT_EXPORT_ROOT must be a regular directory")
	}

	concurrency := 1
	if raw := strings.TrimSpace(os.Getenv("GO_MAX_CONCURRENT_ANALYSES")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 32 {
			return offlineConfig{}, errors.New("GO_MAX_CONCURRENT_ANALYSES must be between 1 and 32")
		}
		concurrency = parsed
	}
	return offlineConfig{
		address:       address,
		demosRoot:     demosRoot,
		exportRoot:    exportRoot,
		maxConcurrent: concurrency,
	}, nil
}

func existingDirectoryFromEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s must be an existing directory", name)
	}
	return resolved, nil
}

func validateLoopbackAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return errors.New("GO_SERVICE_ADDR must be a loopback host and port")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("GO_SERVICE_ADDR must bind only to loopback")
	}
	return nil
}
