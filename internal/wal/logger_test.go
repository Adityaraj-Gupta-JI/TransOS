package wal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createCommittedCheckpoint(t *testing.T, dir, target, content string) string {
	t.Helper()
	tx, err := NewTransaction(dir)
	if err != nil {
		t.Fatalf("NewTransaction() error = %v", err)
	}
	if err := tx.LogAction(ActionModifyFile, target, content); err != nil {
		t.Fatalf("LogAction() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	return tx.TxID
}

func TestRetainsMultipleCheckpointsAndRollbackToOlderState(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "generated.conf")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}

	firstID := createCommittedCheckpoint(t, dir, target, "version-one")
	secondID := createCommittedCheckpoint(t, dir, target, "version-two")
	if firstID == secondID {
		t.Fatalf("expected unique checkpoint IDs, both were %q", firstID)
	}

	checkpoints, err := ListCheckpoints(dir)
	if err != nil {
		t.Fatalf("ListCheckpoints() error = %v", err)
	}
	if len(checkpoints) != 2 {
		t.Fatalf("checkpoint count = %d, want 2", len(checkpoints))
	}
	if checkpoints[0].ID != secondID || checkpoints[1].ID != firstID {
		t.Fatalf("checkpoints were not newest-first: %+v", checkpoints)
	}

	if err := RollbackTo(dir, firstID); err != nil {
		t.Fatalf("RollbackTo() error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("target = %q after rollback, want original", data)
	}

	checkpoints, err = ListCheckpoints(dir)
	if err != nil {
		t.Fatalf("ListCheckpoints() after rollback error = %v", err)
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.Status != StatusRolledBack {
			t.Errorf("checkpoint %s status = %q, want %q", checkpoint.ID, checkpoint.Status, StatusRolledBack)
		}
	}
}

func TestRollbackPreservesExternallyModifiedFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "generated.conf")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	tx, err := NewTransaction(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.LogAction(ActionModifyFile, target, "generated-by-transos"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("manual user edit"), 0600); err != nil {
		t.Fatal(err)
	}

	err = tx.Rollback()
	if err == nil || !strings.Contains(err.Error(), "rollback conflict") {
		t.Fatalf("expected rollback conflict, got %v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "manual user edit" {
		t.Fatalf("rollback overwrote external edit: file now %q", data)
	}
	if tx.Status != StatusRollbackFail {
		t.Fatalf("status = %q, want %q", tx.Status, StatusRollbackFail)
	}
}

func TestLegacyWALWithoutCheckpointDirectoryCanStillLoad(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "transos.wal")
	legacy := `{"tx_id":"tx_legacy","status":"COMMITTED","entries":[]}`
	if err := os.WriteFile(legacyPath, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	tx, err := LoadWAL(legacyPath)
	if err != nil {
		t.Fatalf("LoadWAL() legacy error = %v", err)
	}
	if tx.TxID != "tx_legacy" || tx.CheckpointPath != "" {
		t.Fatalf("unexpected legacy transaction: %+v", tx)
	}
}
