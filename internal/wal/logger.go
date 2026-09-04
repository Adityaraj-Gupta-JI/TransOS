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

// WALEntry records an individual atomic filesystem state change
type WALEntry struct {
	ID          string     `json:"id"`
	Timestamp   string     `json:"timestamp"`
	Action      ActionType `json:"action"`
	TargetPath  string     `json:"target_path"`
	BackupPath  string     `json:"backup_path,omitempty"`
	Description string     `json:"description"`
}

// WALTransaction encapsulates the entire injection log session
type WALTransaction struct {
	TxID    string     `json:"tx_id"`
	Status  string     `json:"status"` // "PENDING", "COMMITTED", "ROLLED_BACK"
	Entries []WALEntry `json:"entries"`
	LogFile string     `json:"-"`
}

// NewTransaction initializes a new transactional session
func NewTransaction(logDir string) (*WALTransaction, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	logFile := filepath.Join(logDir, "transos.wal")

	tx := &WALTransaction{
		TxID:    txID,
		Status:  "PENDING",
		Entries: []WALEntry{},
		LogFile: logFile,
	}

	return tx, nil
}

// LogAction records an intentional file creation or modification before execution
func (tx *WALTransaction) LogAction(action ActionType, targetPath string, desc string) error {
	entry := WALEntry{
		ID:          fmt.Sprintf("e_%d", len(tx.Entries)+1),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Action:      action,
		TargetPath:  targetPath,
		Description: desc,
	}

	// Create a backup snapshot if modifying an existing file
	if _, err := os.Stat(targetPath); err == nil && action == ActionModifyFile {
		backupPath := targetPath + ".wal.bak"
		data, err := os.ReadFile(targetPath)
		if err == nil {
			_ = os.WriteFile(backupPath, data, 0644)
			entry.BackupPath = backupPath
		}
	}

	tx.Entries = append(tx.Entries, entry)
	return tx.save()
}

// Commit marks the transaction as successfully applied
func (tx *WALTransaction) Commit() error {
	tx.Status = "COMMITTED"
	return tx.save()
}

// Rollback restores target system state using recorded WAL entries in reverse order
func (tx *WALTransaction) Rollback() error {
	fmt.Println("[*] Executing atomic rollback from Write-Ahead Log (WAL)...")
	for i := len(tx.Entries) - 1; i >= 0; i-- {
		entry := tx.Entries[i]
		if entry.Action == ActionCreateFile {
			fmt.Printf("[*] Reverting: removing created file %s\n", entry.TargetPath)
			_ = os.Remove(entry.TargetPath)
		} else if entry.BackupPath != "" {
			fmt.Printf("[*] Restoring backup for %s\n", entry.TargetPath)
			data, err := os.ReadFile(entry.BackupPath)
			if err == nil {
				_ = os.WriteFile(entry.TargetPath, data, 0644)
				_ = os.Remove(entry.BackupPath)
			}
		}
	}
	tx.Status = "ROLLED_BACK"
	return tx.save()
}

func (tx *WALTransaction) save() error {
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tx.LogFile, data, 0644)
}

// LoadWAL loads an existing WAL file from disk
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
