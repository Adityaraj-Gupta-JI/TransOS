package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range portableFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("safe test contents"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Seal(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStageRequiresApprovalAndLinux(t *testing.T) {
	src := testBundle(t)
	dst := filepath.Join(t.TempDir(), "stage")
	if err := stageForHost(src, dst, false, "linux"); err == nil {
		t.Fatal("approval required")
	}
	if err := stageForHost(src, dst, true, "windows"); err == nil {
		t.Fatal("Linux required")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("unexpected destination: %v", err)
	}
}

func TestStageCopiesAndVerifiesWithoutExecution(t *testing.T) {
	src := testBundle(t)
	dst := filepath.Join(t.TempDir(), "stage")
	if err := stageForHost(src, dst, true, "linux"); err != nil {
		t.Fatal(err)
	}
	v, err := Verify(dst)
	if err != nil || !v.Valid() {
		t.Fatalf("invalid stage: %+v %v", v, err)
	}
	if err := stageForHost(src, dst, true, "linux"); err == nil {
		t.Fatal("must refuse overwrite")
	}
	if info, err := os.Stat(filepath.Join(dst, "install_dependencies.sh")); err != nil || info.Mode().Perm()&0111 != 0 {
		t.Fatalf("script should not be executable: %v %v", info, err)
	}
}

func TestStageRejectsTamperingAndNestedDestination(t *testing.T) {
	src := testBundle(t)
	if err := stageForHost(src, filepath.Join(src, "nested"), true, "linux"); err == nil {
		t.Fatal("nested destination accepted")
	}
	if err := os.WriteFile(filepath.Join(src, ".bashrc"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "stage")
	err := stageForHost(src, dst, true, "linux")
	if err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity failure: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("created directory on invalid source")
	}
}

func TestStageRejectsSymlinkedParentWithoutCreatingDestination(t *testing.T) {
	src := testBundle(t)
	realParent := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realParent, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dst := filepath.Join(alias, "stage")
	if err := stageForHost(src, dst, true, "linux"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink parent rejection, got %v", err)
	}
	if _, err := os.Lstat(filepath.Join(realParent, "stage")); !os.IsNotExist(err) {
		t.Fatalf("destination created despite rejection: %v", err)
	}
}

func TestStageRejectsSymlinkedBundleDirectory(t *testing.T) {
	src := testBundle(t)
	alias := filepath.Join(t.TempDir(), "bundle-alias")
	if err := os.Symlink(src, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "stage")
	if err := stageForHost(alias, dst, true, "linux"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink bundle rejection, got %v", err)
	}
}
