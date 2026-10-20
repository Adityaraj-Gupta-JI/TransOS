package deployment

import (
	"encoding/json"
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

// Recovery tests simulate the Linux execution path on Windows without touching
// a real user's home. The public restore command remains Linux-only.
func TestRestoreRoundTripAndRefusesModifiedTarget(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	original := []byte("# original bash settings\n")
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := applyForHost(bundle, home, recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	if err := restoreForHost(recovery, false, "linux"); err == nil {
		t.Fatal("restore accepted without approval")
	}
	if err := restoreForHost(recovery, true, "windows"); err == nil {
		t.Fatal("restore accepted non-Linux host")
	}
	bash := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(bash, []byte("# externally modified\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreForHost(recovery, true, "linux"); err == nil {
		t.Fatal("restore overwrote externally modified target")
	}
	if got, err := os.ReadFile(bash); err != nil || string(got) != "# externally modified\n" {
		t.Fatalf("unexpected modified file state: %q, %v", got, err)
	}
	// Return the file to its applied state, then restore the complete set.
	// Re-applying from a new recovery location is forbidden here because the
	// target contains a TransOS block. Instead reconstruct the applied state
	// from the saved original and the known managed hook.
	if err := os.WriteFile(bash, []byte(strings.TrimRight(string(original), "\n")+hookBlock), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreForHost(recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(bash); err != nil || string(got) != string(original) {
		t.Fatalf("original bash settings not restored: %q, %v", got, err)
	}
	for _, p := range []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".config", "transos", "transos_env.conf")} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatalf("new file was not removed on restore: %s: %v", p, err)
		}
	}
	// Restore is safe to retry after completion.
	if err := restoreForHost(recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsSymlinkedBackupWithoutChangingHome(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	bash := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(bash, []byte("# before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := applyForHost(bundle, home, recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(bash)
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(recovery, "original-1")
	if err := os.Remove(backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(bash, backup); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := restoreForHost(recovery, true, "linux"); err == nil {
		t.Fatal("restore followed symlinked recovery backup")
	}
	if after, err := os.ReadFile(bash); err != nil || string(after) != string(before) {
		t.Fatalf("restore changed target despite invalid backup: %q %v", after, err)
	}
}

func TestRestoreRejectsCorruptRecordBeforeChanges(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	if err := applyForHost(bundle, home, recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(recovery, "record.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var record ApplyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record.Files[0].Original = "invalid-checksum"
	data, err = json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreForHost(recovery, true, "linux"); err == nil {
		t.Fatal("restore accepted corrupt recovery metadata")
	}
	if err := VerifyApplied(recovery); err == nil {
		t.Fatal("verification accepted corrupt recovery metadata")
	}
}

func TestApplyRejectsSymlinkedRecoveryParent(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	parent := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := applyForHost(bundle, home, filepath.Join(alias, "recovery"), true, "linux"); err == nil {
		t.Fatal("apply accepted symlinked recovery parent")
	}
	if _, err := os.Lstat(filepath.Join(parent, "recovery")); !os.IsNotExist(err) {
		t.Fatalf("recovery created despite unsafe parent: %v", err)
	}
}

func TestRestoreRejectsBackupChecksumMismatch(t *testing.T) {
	bundle := testBundle(t)
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	bash := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(bash, []byte("# before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := applyForHost(bundle, home, recovery, true, "linux"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(bash)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recovery, "original-1"), []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreForHost(recovery, true, "linux"); err == nil {
		t.Fatal("restore accepted corrupted backup")
	}
	if after, err := os.ReadFile(bash); err != nil || string(after) != string(before) {
		t.Fatalf("restore changed target despite corrupt backup: %q %v", after, err)
	}
}

func TestApplyRejectsSymlinkedBundle(t *testing.T) {
	bundle := testBundle(t)
	alias := filepath.Join(t.TempDir(), "bundle-link")
	if err := os.Symlink(bundle, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	home := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery")
	if err := applyForHost(alias, home, recovery, true, "linux"); err == nil {
		t.Fatal("apply accepted symlinked bundle")
	}
	if _, err := os.Lstat(recovery); !os.IsNotExist(err) {
		t.Fatalf("created recovery on invalid source: %v", err)
	}
}
