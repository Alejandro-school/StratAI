package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cs2-demo-service/db"
	"cs2-demo-service/models"
)

const canonicalValidatorVersion = "stratai.canonical_validator@2"

type publicationError struct {
	Kind  string
	Cause error
}

func (err *publicationError) Error() string {
	return fmt.Sprintf("%s: %v", err.Kind, err.Cause)
}

func (err *publicationError) Unwrap() error { return err.Cause }

func publicationFailureKind(err error, fallback string) string {
	var classified *publicationError
	if errors.As(err, &classified) && classified.Kind != "" {
		return classified.Kind
	}
	return fallback
}

func classifyGoExportFailure(err error) string {
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"quality gate", "reconciliation", "causal", "semantic"} {
		if strings.Contains(message, marker) {
			return "semantic_validation"
		}
	}
	for _, marker := range []string{"contract", "canonical", "schema", "manifest"} {
		if strings.Contains(message, marker) {
			return "contract_validation"
		}
	}
	return "export"
}

type validatorReceipt struct {
	Status           string   `json:"status"`
	ValidatorVersion string   `json:"validator_version"`
	FailureType      *string  `json:"failure_type"`
	Errors           []string `json:"errors"`
	Warnings         []string `json:"warnings"`
}

var stagedBundleValidator = runPythonBundleValidator

func runPythonBundleValidator(
	ctx context.Context, stagedDir, matchID, checksum string,
) (validatorReceipt, error) {
	python, prefixArgs, err := resolvePythonExecutable()
	if err != nil {
		return validatorReceipt{}, &publicationError{Kind: "contract_validation", Cause: err}
	}
	script, err := resolvePublicationValidatorScript()
	if err != nil {
		return validatorReceipt{}, &publicationError{Kind: "contract_validation", Cause: err}
	}
	timeout := durationFromEnv("CANONICAL_VALIDATOR_TIMEOUT_SECONDS", 180*time.Second)
	validationContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := append(append([]string{}, prefixArgs...), script, stagedDir, "--match-id", matchID, "--checksum", checksum)
	command := exec.CommandContext(validationContext, python, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	commandErr := command.Run()
	if errors.Is(validationContext.Err(), context.DeadlineExceeded) {
		return validatorReceipt{}, &publicationError{
			Kind: "validator_timeout", Cause: fmt.Errorf("validator exceeded %s", timeout),
		}
	}
	var receipt validatorReceipt
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &receipt); err != nil {
		return validatorReceipt{}, &publicationError{
			Kind:  "contract_validation",
			Cause: fmt.Errorf("invalid validator response: %w (stderr=%s)", err, strings.TrimSpace(stderr.String())),
		}
	}
	if commandErr != nil || receipt.Status != "passed" || receipt.ValidatorVersion != canonicalValidatorVersion || len(receipt.Errors) > 0 {
		kind := "contract_validation"
		if receipt.FailureType != nil && *receipt.FailureType != "" {
			kind = *receipt.FailureType
		}
		return receipt, &publicationError{Kind: kind, Cause: fmt.Errorf("%s", strings.Join(receipt.Errors, "; "))}
	}
	return receipt, nil
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || seconds < 1 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func resolvePythonExecutable() (string, []string, error) {
	if configured := strings.TrimSpace(os.Getenv("CANONICAL_VALIDATOR_PYTHON")); configured != "" {
		return configured, nil, nil
	}
	for _, candidate := range []string{
		filepath.Join("..", "venv", "Scripts", "python.exe"),
		filepath.Join("backend", "venv", "Scripts", "python.exe"),
		filepath.Join("venv", "Scripts", "python.exe"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil, nil
		}
	}
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil, nil
		}
	}
	if path, err := exec.LookPath("py"); err == nil {
		return path, []string{"-3"}, nil
	}
	return "", nil, fmt.Errorf("no Python interpreter available for canonical validation")
}

func resolvePublicationValidatorScript() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("CANONICAL_VALIDATOR_SCRIPT")); configured != "" {
		if info, err := os.Stat(configured); err == nil && info.Mode().IsRegular() {
			return configured, nil
		}
		return "", fmt.Errorf("configured validator script is unavailable")
	}
	for _, candidate := range []string{
		filepath.Join("scripts", "publication_validator.py"),
		filepath.Join("backend", "go-service", "scripts", "publication_validator.py"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("publication_validator.py is unavailable")
}

type publicationTransaction struct {
	stagedDir string
	finalDir  string
	backupDir string
	committed bool
}

func beginPublicationCommit(stagedDir, finalDir, token string) (*publicationTransaction, error) {
	stagedAbs, err := filepath.Abs(stagedDir)
	if err != nil {
		return nil, &publicationError{Kind: "commit_failure", Cause: err}
	}
	finalAbs, err := filepath.Abs(finalDir)
	if err != nil {
		return nil, &publicationError{Kind: "commit_failure", Cause: err}
	}
	if stagedAbs == finalAbs || !isExactChild(filepath.Dir(stagedAbs), stagedAbs) || !isExactChild(filepath.Dir(finalAbs), finalAbs) {
		return nil, &publicationError{Kind: "commit_failure", Cause: fmt.Errorf("unsafe publication paths")}
	}
	info, err := os.Stat(stagedAbs)
	if err != nil || !info.IsDir() {
		return nil, &publicationError{Kind: "commit_failure", Cause: fmt.Errorf("staging directory unavailable")}
	}
	tokenHash := sha256.Sum256([]byte(token))
	transaction := &publicationTransaction{
		stagedDir: stagedAbs,
		finalDir:  finalAbs,
		backupDir: fmt.Sprintf("%s.rollback.%x", finalAbs, tokenHash[:8]),
	}
	if _, err := os.Stat(transaction.backupDir); err == nil {
		return nil, &publicationError{Kind: "commit_failure", Cause: fmt.Errorf("rollback target already exists")}
	} else if !os.IsNotExist(err) {
		return nil, &publicationError{Kind: "commit_failure", Cause: err}
	}
	if _, err := os.Stat(finalAbs); err == nil {
		if err := renameWithRetry(finalAbs, transaction.backupDir); err != nil {
			return nil, &publicationError{Kind: "commit_failure", Cause: fmt.Errorf("preserve previous export: %w", err)}
		}
	} else if !os.IsNotExist(err) {
		return nil, &publicationError{Kind: "commit_failure", Cause: err}
	} else {
		transaction.backupDir = ""
	}
	if err := renameWithRetry(stagedAbs, finalAbs); err != nil {
		if transaction.backupDir != "" {
			if rollbackErr := renameWithRetry(transaction.backupDir, finalAbs); rollbackErr != nil {
				return nil, &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("commit failed: %v; restore failed: %w", err, rollbackErr)}
			}
		}
		return nil, &publicationError{Kind: "commit_failure", Cause: fmt.Errorf("publish staged export: %w", err)}
	}
	transaction.committed = true
	return transaction, nil
}

func (transaction *publicationTransaction) Rollback() error {
	if transaction == nil || !transaction.committed {
		return nil
	}
	if _, err := os.Stat(transaction.finalDir); err != nil {
		return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("published directory missing: %w", err)}
	}
	if err := renameWithRetry(transaction.finalDir, transaction.stagedDir); err != nil {
		return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("quarantine failed publication: %w", err)}
	}
	if transaction.backupDir != "" {
		if err := renameWithRetry(transaction.backupDir, transaction.finalDir); err != nil {
			_ = renameWithRetry(transaction.stagedDir, transaction.finalDir)
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("restore previous export: %w", err)}
		}
	}
	transaction.committed = false
	return nil
}

func (transaction *publicationTransaction) Finalize() error {
	if transaction == nil || transaction.backupDir == "" {
		return nil
	}
	if !strings.HasPrefix(filepath.Base(transaction.backupDir), filepath.Base(transaction.finalDir)+".rollback.") || !isExactChild(filepath.Dir(transaction.finalDir), transaction.backupDir) {
		return fmt.Errorf("unsafe rollback cleanup target")
	}
	if err := os.RemoveAll(transaction.backupDir); err != nil {
		return fmt.Errorf("remove publication rollback backup: %w", err)
	}
	transaction.backupDir = ""
	return nil
}

func isExactChild(parent, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(parentAbs, childAbs)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func renameWithRetry(source, destination string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := os.Rename(source, destination); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
		}
	}
	return lastErr
}

func recoverInterruptedPublication(finalDir, tempParent, matchID, checksum, schemaVersion string) error {
	backups, err := filepath.Glob(finalDir + ".rollback.*")
	if err != nil {
		return &publicationError{Kind: "rollback_failure", Cause: err}
	}
	if _, err := os.Stat(finalDir); os.IsNotExist(err) && len(backups) > 0 {
		sort.Slice(backups, func(i, j int) bool {
			left, leftErr := os.Stat(backups[i])
			right, rightErr := os.Stat(backups[j])
			return leftErr == nil && (rightErr != nil || left.ModTime().After(right.ModTime()))
		})
		if err := renameWithRetry(backups[0], finalDir); err != nil {
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("recover previous export: %w", err)}
		}
		backups = backups[1:]
	}
	if len(backups) > 0 && !hasCommittedManifest(finalDir, checksum, schemaVersion) {
		return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("ambiguous interrupted publication requires manual recovery")}
	}
	for _, backup := range backups {
		if !isExactChild(filepath.Dir(finalDir), backup) || !strings.HasPrefix(filepath.Base(backup), filepath.Base(finalDir)+".rollback.") {
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("unsafe recovery backup path")}
		}
		if err := os.RemoveAll(backup); err != nil {
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("clean recovered backup: %w", err)}
		}
	}
	stagingRoots, err := filepath.Glob(filepath.Join(tempParent, matchID+"-*"))
	if err != nil {
		return &publicationError{Kind: "rollback_failure", Cause: err}
	}
	for _, stagingRoot := range stagingRoots {
		if !isExactChild(tempParent, stagingRoot) || !strings.HasPrefix(filepath.Base(stagingRoot), matchID+"-") {
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("unsafe staging recovery path")}
		}
		if err := os.RemoveAll(stagingRoot); err != nil {
			return &publicationError{Kind: "rollback_failure", Cause: fmt.Errorf("clean stale staging: %w", err)}
		}
	}
	return nil
}

func snapshotMatchDataWithRetry(ctx context.Context, matchID string) (db.MatchDataSnapshot, error) {
	var snapshot db.MatchDataSnapshot
	err := retryBounded(ctx, func(attemptContext context.Context) error {
		var err error
		snapshot, err = db.SnapshotMatchData(attemptContext, matchID)
		return err
	})
	return snapshot, err
}

func saveMatchDataWithRetry(ctx context.Context, matchID string, matchData *models.MatchData) error {
	return retryBounded(ctx, func(attemptContext context.Context) error {
		return db.SaveMatchDataContext(attemptContext, matchID, matchData)
	})
}

func restoreMatchDataWithRetry(ctx context.Context, matchID string, snapshot db.MatchDataSnapshot) error {
	return retryBounded(ctx, func(attemptContext context.Context) error {
		return db.RestoreMatchData(attemptContext, matchID, snapshot)
	})
}

func retryBounded(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		attemptContext, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = operation(attemptContext)
		cancel()
		if lastErr == nil {
			return nil
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
			}
		}
	}
	return lastErr
}
