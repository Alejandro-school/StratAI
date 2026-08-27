package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationTransactionRollbackRestoresPreviousExport(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "match_test")
	stagedDir := filepath.Join(root, "staging", "match_test")
	writeTestArtifact(t, filepath.Join(finalDir, "value.txt"), []byte("old"))
	writeTestArtifact(t, filepath.Join(stagedDir, "value.txt"), []byte("new"))

	transaction, err := beginPublicationCommit(stagedDir, finalDir, "rollback-test")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := os.ReadFile(filepath.Join(finalDir, "value.txt")); err != nil || string(value) != "new" {
		t.Fatalf("new export was not atomically visible: value=%q err=%v", value, err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	if value, err := os.ReadFile(filepath.Join(finalDir, "value.txt")); err != nil || string(value) != "old" {
		t.Fatalf("previous export was not restored: value=%q err=%v", value, err)
	}
	if value, err := os.ReadFile(filepath.Join(stagedDir, "value.txt")); err != nil || string(value) != "new" {
		t.Fatalf("failed export was not returned to staging: value=%q err=%v", value, err)
	}
}

func TestPublicationTransactionFinalizeRemovesRollbackBackup(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "match_test")
	stagedDir := filepath.Join(root, "staging", "match_test")
	writeTestArtifact(t, filepath.Join(finalDir, "value.txt"), []byte("old"))
	writeTestArtifact(t, filepath.Join(stagedDir, "value.txt"), []byte("new"))

	transaction, err := beginPublicationCommit(stagedDir, finalDir, "finalize-test")
	if err != nil {
		t.Fatal(err)
	}
	backupDir := transaction.backupDir
	if err := transaction.Finalize(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("rollback backup was not removed: %v", err)
	}
}

func TestRecoverInterruptedPublicationRestoresBackupAndCleansStaging(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "match_test")
	backupDir := finalDir + ".rollback.crash"
	tempParent := filepath.Join(root, ".tmp")
	staleRoot := filepath.Join(tempParent, "test-stale")
	writeTestArtifact(t, filepath.Join(backupDir, "old.txt"), []byte("old"))
	writeTestArtifact(t, filepath.Join(staleRoot, "partial.txt"), []byte("partial"))

	if err := recoverInterruptedPublication(finalDir, tempParent, "test", "unused", defaultParserSchemaVersion); err != nil {
		t.Fatal(err)
	}
	if value, err := os.ReadFile(filepath.Join(finalDir, "old.txt")); err != nil || string(value) != "old" {
		t.Fatalf("backup was not recovered: value=%q err=%v", value, err)
	}
	if _, err := os.Stat(staleRoot); !os.IsNotExist(err) {
		t.Fatalf("stale staging was not cleaned: %v", err)
	}
}
