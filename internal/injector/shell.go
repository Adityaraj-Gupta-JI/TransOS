package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/transos/transos/internal/wal"
)

const transosHookMarker = "# TransOS Environment Auto-Source Hook"

// InjectShellHooks injects environment sourcing hooks into target shell profiles without duplication
func InjectShellHooks(tx *wal.WALTransaction, targetBaseDir string) (int, error) {
	shellFiles := []string{
		filepath.Join(targetBaseDir, ".bashrc"),
		filepath.Join(targetBaseDir, ".zshrc"),
	}

	hookSnippet := fmt.Sprintf("\n%s\nif [ -f \"$HOME/.config/transos/transos_env.conf\" ]; then\n    source \"$HOME/.config/transos/transos_env.conf\"\nfi\n", transosHookMarker)

	injectedCount := 0

	for _, shellFile := range shellFiles {
		existingContent := ""
		if data, err := os.ReadFile(shellFile); err == nil {
			existingContent = string(data)
		}

		// Strict Idempotency Check: skip if hook marker OR config file path already exists
		if strings.Contains(existingContent, transosHookMarker) || strings.Contains(existingContent, "transos_env.conf") {
			continue
		}

		actionType := wal.ActionModifyFile
		if existingContent == "" {
			actionType = wal.ActionCreateFile
		}

		newContent := strings.TrimRight(existingContent, "\n") + hookSnippet
		// The WAL must write the exact final contents so its content hash and
		// backup accurately represent what can later be rolled back.
		if err := tx.LogAction(actionType, shellFile, newContent); err != nil {
			return injectedCount, fmt.Errorf("log shell hook for %s: %w", shellFile, err)
		}

		injectedCount++
	}

	return injectedCount, nil
}
