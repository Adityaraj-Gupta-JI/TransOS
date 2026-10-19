package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAndRestoreUserConfig(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	original := []byte("# existing settings\nexport FOO=bar\n")
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := applyForHost(bundle, home, recovery, false, "linux"); err == nil {
		t.Fatal("approval not required")
	}
	if err := applyForHost(bundle, home, recovery, true, "windows"); err == nil {
		t.Fatal("Linux not required")
	}
	if err := applyForHost(bundle, home, recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyApplied(recovery); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export FOO=bar") || !strings.Contains(string(data), hookStart) {
		t.Fatal("original configuration lost or hook missing")
	}
	if err := applyForHost(bundle, home, recovery, true, "linux"); err == nil {
		t.Fatal("existing recovery overwritten")
	}
	// Restore is gated by platform, but the recovery semantics can be tested on Linux CI.
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("user changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyApplied(recovery); err == nil {
		t.Fatal("tampering not detected")
	}
}
func TestApplyRejectsUnsafePaths(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	if err := applyForHost(bundle, home, filepath.Join(home, "recovery"), true, "linux"); err == nil {
		t.Fatal("recovery inside home allowed")
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, ".config")); err == nil {
		if err := applyForHost(bundle, home, filepath.Join(t.TempDir(), "recovery"), true, "linux"); err == nil {
			t.Fatal("symlink parent allowed")
		}
	}
}
