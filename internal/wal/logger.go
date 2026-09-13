package wal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ActionType string

const (
	ActionCreateFile ActionType = "CREATE_FILE"
	ActionModifyFile ActionType = "MODIFY_FILE"
)

// WALEntry records one filesystem operation belonging to a transaction.
type WALEntry struct {
	ID          string     `json:"id"`
	Timestamp   string     `json:"timestamp"`
	Action      ActionType `json:"action"`
	TargetPath  string     `json:"target_path"`
	BackupPath  string     `json:"backup_path,omitempty"`
	Description string     `json:"description"`
	Existed     bool       `json:"existed"`
}

// WALTransaction encapsulates an injection transaction.
type WALTransaction struct {
	TxID    string     `json:"tx_id"`
	Status  string     `json:"status"`
	Entries []WALEntry `json:"entries"`
	LogFile string     `json:"-"`
}

// NewTransaction initializes a new transaction.
func NewTransaction(logDir string) (*WALTransaction, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	logFile := filepath.Join(logDir, "transos.wal")

	tx := &WALTransaction{
		TxID:    txID,
		Status:  "PENDING",
		Entries: make([]WALEntry, 0),
		LogFile: logFile,
	}

	return tx, nil
}

// LogAction records and executes one filesystem action.
//
// The description argument contains the file content for file operations.
func (tx *WALTransaction) LogAction(
	action ActionType,
	targetPath string,
	desc string,
) error {
	if tx == nil {
		return fmt.Errorf("WAL transaction is nil")
	}

	if targetPath == "" {
		return fmt.Errorf("target path is empty")
	}

	if tx.Status != "PENDING" {
		return fmt.Errorf("transaction is not pending: %s", tx.Status)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	_, statErr := os.Stat(targetPath)
	existed := statErr == nil

	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect target path %q: %w", targetPath, statErr)
	}

	entry := WALEntry{
		ID:          fmt.Sprintf("e_%d", len(tx.Entries)+1),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Action:      action,
		TargetPath:  targetPath,
		Description: desc,
		Existed:     existed,
	}

	// When an existing file is about to be replaced, create a backup.
	if existed {
		backupPath := targetPath + ".wal.bak"

		data, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read backup source %q: %w", targetPath, err)
		}

		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("write backup %q: %w", backupPath, err)
		}

		entry.BackupPath = backupPath
	}

	// Record the intent before applying it.
	tx.Entries = append(tx.Entries, entry)

	if err := tx.save(); err != nil {
		tx.Entries = tx.Entries[:len(tx.Entries)-1]

		if entry.BackupPath != "" {
			_ = os.Remove(entry.BackupPath)
		}

		return fmt.Errorf("save WAL before action: %w", err)
	}

	// Execute the actual filesystem operation.
	switch action {
	case ActionCreateFile, ActionModifyFile:
		if err := writeFileAtomically(targetPath, []byte(desc)); err != nil {
			return fmt.Errorf(
				"apply %s to %q: %w",
				action,
				targetPath,
				err,
			)
		}

	default:
		return fmt.Errorf("unsupported WAL action: %q", action)
	}

	// Persist the final entry state.
	if err := tx.save(); err != nil {
		return fmt.Errorf("save WAL after action: %w", err)
	}

	return nil
}

// Commit marks the transaction complete.
func (tx *WALTransaction) Commit() error {
	if tx == nil {
		return fmt.Errorf("WAL transaction is nil")
	}

	if tx.Status != "PENDING" {
		return fmt.Errorf("cannot commit transaction with status %q", tx.Status)
	}

	tx.Status = "COMMITTED"

	if err := tx.save(); err != nil {
		return fmt.Errorf("save committed WAL: %w", err)
	}

	return nil
}

// Rollback restores the filesystem to the state recorded by the WAL.
func (tx *WALTransaction) Rollback() error {
	if tx == nil {
		return fmt.Errorf("WAL transaction is nil")
	}

	fmt.Println("[*] Executing atomic rollback from Write-Ahead Log (WAL)...")

	var rollbackErrors []error

	for i := len(tx.Entries) - 1; i >= 0; i-- {
		entry := tx.Entries[i]

		if entry.BackupPath != "" {
			fmt.Printf("[*] Restoring backup for %s\n", entry.TargetPath)

			data, err := os.ReadFile(entry.BackupPath)
			if err != nil {
				rollbackErrors = append(
					rollbackErrors,
					fmt.Errorf(
						"read backup %q: %w",
						entry.BackupPath,
						err,
					),
				)
				continue
			}

			if err := writeFileAtomically(
				entry.TargetPath,
				data,
			); err != nil {
				rollbackErrors = append(
					rollbackErrors,
					fmt.Errorf(
						"restore %q: %w",
						entry.TargetPath,
						err,
					),
				)
				continue
			}

			if err := os.Remove(entry.BackupPath); err != nil &&
				!os.IsNotExist(err) {
				rollbackErrors = append(
					rollbackErrors,
					fmt.Errorf(
						"remove backup %q: %w",
						entry.BackupPath,
						err,
					),
				)
			}

			continue
		}

		// The file did not exist before the transaction.
		fmt.Printf("[*] Reverting: removing created file %s\n", entry.TargetPath)

		if err := os.Remove(entry.TargetPath); err != nil &&
			!os.IsNotExist(err) {
			rollbackErrors = append(
				rollbackErrors,
				fmt.Errorf(
					"remove %q: %w",
					entry.TargetPath,
					err,
				),
			)
		}
	}

	tx.Status = "ROLLED_BACK"

	if err := tx.save(); err != nil {
		rollbackErrors = append(
			rollbackErrors,
			fmt.Errorf("save rolled-back WAL: %w", err),
		)
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback completed with errors: %v", rollbackErrors)
	}

	return nil
}

func (tx *WALTransaction) save() error {
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomically(tx.LogFile, data)
}

// LoadWAL loads an existing WAL transaction.
func LoadWAL(logFile string) (*WALTransaction, error) {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	var tx WALTransaction

	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}

	tx.LogFile = logFile

	return &tx, nil
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

	// Windows cannot rename over an existing file directly.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing target: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace target: %w", err)
	}

	return nil
}
