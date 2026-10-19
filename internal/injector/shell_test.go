package injector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transos/transos/internal/wal"
)

func TestShellHooksAreFullyRecordedAndRollbackable(t *testing.T) {
	outputDir := t.TempDir()
	originalBashrc := "export KEEP_ME=1\n"
	if err := os.WriteFile(filepath.Join(outputDir, ".bashrc"), []byte(originalBashrc), 0600); err != nil {
		t.Fatalf("seed existing .bashrc: %v", err)
	}
	tx, err := wal.NewTransaction(outputDir)
	if err != nil {
		t.Fatalf("NewTransaction() error = %v", err)
	}
	count, err := InjectShellHooks(tx, outputDir)
	if err != nil {
		t.Fatalf("InjectShellHooks() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("injected hook count = %d, want 2", count)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	for _, name := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(outputDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(data), transosHookMarker) {
			t.Errorf("%s does not contain TransOS hook", name)
		}
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	bashrc, err := os.ReadFile(filepath.Join(outputDir, ".bashrc"))
	if err != nil {
		t.Fatalf("read restored .bashrc: %v", err)
	}
	if string(bashrc) != originalBashrc {
		t.Fatalf(".bashrc restored to %q, want %q", bashrc, originalBashrc)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf(".zshrc should be removed after rollback, stat error = %v", err)
	}
}
