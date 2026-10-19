package wal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ActionType string

const (
	ActionCreateFile ActionType = "CREATE_FILE"
	ActionModifyFile ActionType = "MODIFY_FILE"
)

const (
	StatusPending      = "PENDING"
	StatusCommitted    = "COMMITTED"
	StatusRolledBack   = "ROLLED_BACK"
	StatusRollbackFail = "ROLLBACK_FAILED"
)

// WALEntry records one filesystem operation belonging to a transaction.
type WALEntry struct {
	ID           string     `json:"id"`
	Timestamp    string     `json:"timestamp"`
	Action       ActionType `json:"action"`
	TargetPath   string     `json:"target_path"`
	BackupPath   string     `json:"backup_path,omitempty"`
	Description  string     `json:"description"`
	Existed      bool       `json:"existed"`
	ResultSHA256 string     `json:"result_sha256,omitempty"`
	BackupSHA256 string     `json:"backup_sha256,omitempty"`
	RolledBack   bool       `json:"rolled_back,omitempty"`
}

// WALTransaction is a retained checkpoint. LogFile remains the compatibility
// alias target_output/transos.wal; CheckpointPath is the durable per-run record.
type WALTransaction struct {
	TxID           string     `json:"tx_id"`
	CreatedAt      string     `json:"created_at,omitempty"`
	Status         string     `json:"status"`
	Entries        []WALEntry `json:"entries"`
	LogFile        string     `json:"-"`
	CheckpointPath string     `json:"-"`
}

// CheckpointInfo is the public, read-only summary used by the CLI.
type CheckpointInfo struct {
	ID        string
	CreatedAt string
	Status    string
	Entries   int
}

// NewTransaction initializes a transaction and reserves a unique checkpoint
// location while preserving the legacy latest WAL file.
func NewTransaction(logDir string) (*WALTransaction, error) {
	if strings.TrimSpace(logDir) == "" {
		return nil, errors.New("log directory is empty")
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	now := time.Now().UTC()
	txID := fmt.Sprintf("tx_%d", now.UnixNano())
	checkpointDir := filepath.Join(logDir, "checkpoints")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	tx := &WALTransaction{
		TxID:           txID,
		CreatedAt:      now.Format(time.RFC3339Nano),
		Status:         StatusPending,
		Entries:        make([]WALEntry, 0),
		LogFile:        filepath.Join(logDir, "transos.wal"),
		CheckpointPath: filepath.Join(checkpointDir, txID+".wal"),
	}
	return tx, nil
}

// LogAction records and executes one filesystem action. The description is the
// new file content for both CREATE_FILE and MODIFY_FILE actions.
func (tx *WALTransaction) LogAction(action ActionType, targetPath string, content string) error {
	if tx == nil {
		return errors.New("WAL transaction is nil")
	}
	if strings.TrimSpace(targetPath) == "" {
		return errors.New("target path is empty")
	}
	if tx.Status != StatusPending {
		return fmt.Errorf("transaction is not pending: %s", tx.Status)
	}
	if action != ActionCreateFile && action != ActionModifyFile {
		return fmt.Errorf("unsupported WAL action: %q", action)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	info, statErr := os.Stat(targetPath)
	existed := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect target path %q: %w", targetPath, statErr)
	}
	if existed && !info.Mode().IsRegular() {
		return fmt.Errorf("target path %q is not a regular file", targetPath)
	}

	entry := WALEntry{
		ID:          fmt.Sprintf("e_%d", len(tx.Entries)+1),
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Action:      action,
		TargetPath:  targetPath,
		Description: content,
		Existed:     existed,
	}

	if existed {
		backupPath := targetPath + "." + tx.TxID + "." + entry.ID + ".wal.bak"
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read backup source %q: %w", targetPath, err)
		}
		if err := writeFileAtomically(backupPath, data); err != nil {
			return fmt.Errorf("write backup %q: %w", backupPath, err)
		}
		entry.BackupPath = backupPath
		entry.BackupSHA256 = hashBytes(data)
	}

	// Persist intent and backup metadata before changing the target file.
	tx.Entries = append(tx.Entries, entry)
	if err := tx.save(); err != nil {
		tx.Entries = tx.Entries[:len(tx.Entries)-1]
		if entry.BackupPath != "" {
			_ = os.Remove(entry.BackupPath)
		}
		return fmt.Errorf("save WAL before action: %w", err)
	}

	if err := writeFileAtomically(targetPath, []byte(content)); err != nil {
		return fmt.Errorf("apply %s to %q: %w", action, targetPath, err)
	}

	written, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("verify written target %q: %w", targetPath, err)
	}
	tx.Entries[len(tx.Entries)-1].ResultSHA256 = hashBytes(written)
	if err := tx.save(); err != nil {
		return fmt.Errorf("save WAL after action: %w", err)
	}
	return nil
}

// Commit marks the transaction complete and persists both its checkpoint and
// the compatibility WAL alias.
func (tx *WALTransaction) Commit() error {
	if tx == nil {
		return errors.New("WAL transaction is nil")
	}
	if tx.Status != StatusPending {
		return fmt.Errorf("cannot commit transaction with status %q", tx.Status)
	}
	tx.Status = StatusCommitted
	if err := tx.save(); err != nil {
		return fmt.Errorf("save committed WAL: %w", err)
	}
	return nil
}

// Rollback restores this transaction's filesystem changes in reverse order.
// It refuses to overwrite files that no longer match the content recorded when
// the transaction completed, unless the file already matches its backup state.
func (tx *WALTransaction) Rollback() error {
	if tx == nil {
		return errors.New("WAL transaction is nil")
	}
	if tx.Status == StatusRolledBack {
		return fmt.Errorf("checkpoint %s has already been rolled back", tx.TxID)
	}
	if tx.Status != StatusCommitted && tx.Status != StatusPending && tx.Status != StatusRollbackFail {
		return fmt.Errorf("checkpoint %s cannot be rolled back from status %q", tx.TxID, tx.Status)
	}

	fmt.Printf("[*] Rolling back checkpoint %s...\n", tx.TxID)
	var rollbackErrors []error

	for i := len(tx.Entries) - 1; i >= 0; i-- {
		entry := &tx.Entries[i]
		if entry.RolledBack {
			continue
		}

		current, readErr := os.ReadFile(entry.TargetPath)
		currentExists := readErr == nil
		if readErr != nil && !os.IsNotExist(readErr) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("inspect target %q: %w", entry.TargetPath, readErr))
			continue
		}

		if !currentExists && !entry.Existed {
			// A created file that is already absent is already in its rollback state.
			entry.RolledBack = true
			if err := tx.save(); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("record rollback of %q: %w", entry.TargetPath, err))
			}
			continue
		}

		// New transactions record ResultSHA256 after writing the final content.
		// CreatedAt identifies the newer WAL format, where Description is also the
		// exact content written. Older single-WAL runs may have logged a label
		// instead of final content (the historical shell-hook path did this), so
		// do not misinterpret that legacy label as a content hash.
		expectedHash := entry.ResultSHA256
		if expectedHash == "" && tx.CreatedAt != "" {
			expectedHash = hashBytes([]byte(entry.Description))
		}
		if expectedHash != "" {
			if !currentExists {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %q: generated file is missing; preserving current state", entry.TargetPath))
				continue
			}
			currentHash := hashBytes(current)
			if currentHash != expectedHash {
				if entry.Existed && entry.BackupSHA256 != "" && currentHash == entry.BackupSHA256 {
					// Handles interruption after restore but before persisting RolledBack.
					entry.RolledBack = true
					if err := tx.save(); err != nil {
						rollbackErrors = append(rollbackErrors, fmt.Errorf("record restored state for %q: %w", entry.TargetPath, err))
					} else if entry.BackupPath != "" {
						if err := os.Remove(entry.BackupPath); err != nil && !os.IsNotExist(err) {
							rollbackErrors = append(rollbackErrors, fmt.Errorf("remove backup %q: %w", entry.BackupPath, err))
						}
					}
					continue
				}
				rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %q: file contents changed after this checkpoint; preserving the changed file", entry.TargetPath))
				continue
			}
		}

		if entry.BackupPath != "" {
			backup, err := os.ReadFile(entry.BackupPath)
			if err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("read backup %q: %w", entry.BackupPath, err))
				continue
			}
			if err := writeFileAtomically(entry.TargetPath, backup); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore %q: %w", entry.TargetPath, err))
				continue
			}
		} else {
			if err := os.Remove(entry.TargetPath); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove created file %q: %w", entry.TargetPath, err))
				continue
			}
		}

		entry.RolledBack = true
		if err := tx.save(); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("record rollback of %q: %w", entry.TargetPath, err))
			continue
		}
		// Backups are removed only after the per-entry rollback state is durable.
		if entry.BackupPath != "" {
			if err := os.Remove(entry.BackupPath); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove backup %q: %w", entry.BackupPath, err))
			}
		}
	}

	if len(rollbackErrors) > 0 {
		tx.Status = StatusRollbackFail
	} else {
		tx.Status = StatusRolledBack
	}
	if err := tx.save(); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("save final rollback status: %w", err))
	}
	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback of checkpoint %s completed with errors: %v", tx.TxID, errors.Join(rollbackErrors...))
	}
	return nil
}

// ListCheckpoints returns retained per-injection checkpoints newest first.
func ListCheckpoints(logDir string) ([]CheckpointInfo, error) {
	if strings.TrimSpace(logDir) == "" {
		return nil, errors.New("log directory is empty")
	}
	checkpointDir := filepath.Join(logDir, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CheckpointInfo{}, nil
		}
		return nil, fmt.Errorf("read checkpoint directory: %w", err)
	}

	checkpoints := make([]CheckpointInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wal" {
			continue
		}
		path := filepath.Join(checkpointDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read checkpoint %q: %w", entry.Name(), err)
		}
		var tx WALTransaction
		if err := json.Unmarshal(data, &tx); err != nil {
			return nil, fmt.Errorf("parse checkpoint %q: %w", entry.Name(), err)
		}
		if strings.TrimSpace(tx.TxID) == "" {
			return nil, fmt.Errorf("checkpoint %q has no transaction ID", entry.Name())
		}
		if entry.Name() != tx.TxID+".wal" {
			return nil, fmt.Errorf("checkpoint file %q does not match its transaction ID %q", entry.Name(), tx.TxID)
		}
		createdAt := tx.CreatedAt
		if createdAt == "" && len(tx.Entries) > 0 {
			createdAt = tx.Entries[0].Timestamp
		}
		checkpoints = append(checkpoints, CheckpointInfo{
			ID:        tx.TxID,
			CreatedAt: createdAt,
			Status:    tx.Status,
			Entries:   len(tx.Entries),
		})
	}
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].ID > checkpoints[j].ID
	})
	return checkpoints, nil
}

// RollbackLatest reverses the newest checkpoint that has not already been
// rolled back. It also retries a previously failed checkpoint, subject to the
// same file-integrity checks.
func RollbackLatest(logDir string) error {
	checkpoints, err := ListCheckpoints(logDir)
	if err != nil {
		return err
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.Status == StatusRolledBack {
			continue
		}
		if checkpoint.Status != StatusCommitted && checkpoint.Status != StatusPending && checkpoint.Status != StatusRollbackFail {
			return fmt.Errorf("checkpoint %s has unsupported status %q", checkpoint.ID, checkpoint.Status)
		}
		return RollbackTo(logDir, checkpoint.ID)
	}
	return errors.New("no active TransOS checkpoints are available to roll back")
}

// RollbackTo rolls back the target checkpoint and every newer active checkpoint
// in reverse creation order, preserving the state that existed before target.
func RollbackTo(logDir, targetID string) error {
	if strings.TrimSpace(targetID) == "" {
		return errors.New("checkpoint ID is empty")
	}
	checkpoints, err := ListCheckpoints(logDir)
	if err != nil {
		return err
	}
	targetIndex := -1
	for i, checkpoint := range checkpoints {
		if checkpoint.ID == targetID {
			targetIndex = i
			if checkpoint.Status == StatusRolledBack {
				return fmt.Errorf("checkpoint %s has already been rolled back", targetID)
			}
			break
		}
	}
	if targetIndex < 0 {
		return fmt.Errorf("checkpoint %q was not found; run 'transos rollback list' to see retained checkpoints", targetID)
	}

	checkpointDir := filepath.Join(logDir, "checkpoints")
	for i := 0; i <= targetIndex; i++ {
		checkpoint := checkpoints[i]
		if checkpoint.Status == StatusRolledBack {
			continue
		}
		path := filepath.Join(checkpointDir, checkpoint.ID+".wal")
		tx, err := LoadWAL(path)
		if err != nil {
			return fmt.Errorf("load checkpoint %s: %w", checkpoint.ID, err)
		}
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("stopped at checkpoint %s; resolve this rollback before older checkpoints are changed: %w", checkpoint.ID, err)
		}
	}
	return nil
}

func (tx *WALTransaction) save() error {
	if tx == nil {
		return errors.New("WAL transaction is nil")
	}
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	// The durable checkpoint is written first. The legacy latest-WAL alias is
	// then refreshed for existing tooling and old TransOS versions.
	if tx.CheckpointPath != "" {
		if err := writeFileAtomically(tx.CheckpointPath, data); err != nil {
			return fmt.Errorf("write checkpoint WAL: %w", err)
		}
	}
	if tx.LogFile != "" && filepath.Clean(tx.LogFile) != filepath.Clean(tx.CheckpointPath) {
		if err := writeFileAtomically(tx.LogFile, data); err != nil {
			return fmt.Errorf("write latest WAL alias: %w", err)
		}
	}
	if tx.LogFile == "" && tx.CheckpointPath == "" {
		return errors.New("WAL log path is empty")
	}
	return nil
}

// LoadWAL supports both retained checkpoint files and the legacy latest WAL.
func LoadWAL(logFile string) (*WALTransaction, error) {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, err
	}
	var tx WALTransaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}

	cleanPath := filepath.Clean(logFile)
	if filepath.Base(filepath.Dir(cleanPath)) == "checkpoints" {
		tx.CheckpointPath = cleanPath
		tx.LogFile = filepath.Join(filepath.Dir(filepath.Dir(cleanPath)), "transos.wal")
	} else {
		tx.LogFile = cleanPath
		if tx.TxID != "" && filepath.Base(cleanPath) == "transos.wal" {
			candidate := filepath.Join(filepath.Dir(cleanPath), "checkpoints", tx.TxID+".wal")
			if _, statErr := os.Stat(candidate); statErr == nil {
				tx.CheckpointPath = candidate
			}
		}
	}
	return &tx, nil
}

func hashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// writeFileAtomically writes content through a temporary file in the same
// directory and then replaces the target.
func writeFileAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".transos-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	// Windows cannot always rename over an existing file, so replace it
	// explicitly. Backup copies protect the values managed by the WAL.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing target: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace target: %w", err)
	}
	return nil
}
