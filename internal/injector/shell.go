package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/transos/transos/internal/wal"
)

const transosHookMarker = "# TransOS Environment Auto-Source Hook"

// InjectShellHooks injects environment sourcing hooks into target shell profiles (.bashrc, .zshrc)
func InjectShellHooks(tx *wal.WALTransaction, targetBaseDir string) (int, error) {
	// Target standard shell configuration profiles inside target_output
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

		// Prevent duplicate injection if hook already exists
		if strings.Contains(existingContent, transosHookMarker) {
			continue
		}

		// Log WAL modification action prior to filesystem write
		actionType := wal.ActionModifyFile
		if existingContent == "" {
			actionType = wal.ActionCreateFile
		}

		if err := tx.LogAction(actionType, shellFile, "Injecting TransOS shell profile hook"); err != nil {
			return injectedCount, err
		}

		newContent := existingContent + hookSnippet
		if err := os.WriteFile(shellFile, []byte(newContent), 0644); err != nil {
			return injectedCount, fmt.Errorf("failed to write shell hook to %s: %w", shellFile, err)
		}

		injectedCount++
	}

	return injectedCount, nil
}
