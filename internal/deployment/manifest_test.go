package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeArtifacts(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range portableFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("sample:"+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
func TestSealVerifyAndTamper(t *testing.T) {
	dir := makeArtifacts(t)
	if _, err := Seal(dir); err != nil {
		t.Fatal(err)
	}
	v, err := Verify(dir)
	if err != nil || !v.Valid() || v.FilesChecked != len(portableFiles) {
		t.Fatalf("verification: %+v, %v", v, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transos_env.conf"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	v, err = Verify(dir)
	if err != nil || v.Valid() || !strings.Contains(strings.Join(v.Problems, " "), "transos_env.conf") {
		t.Fatalf("tamper undetected: %+v, %v", v, err)
	}
}
func TestSealRequiresAllArtifacts(t *testing.T) {
	dir := t.TempDir()
	if _, err := Seal(dir); err == nil {
		t.Fatal("missing files accepted")
	}
}
func TestVerifyRejectsPathTraversal(t *testing.T) {
	dir := makeArtifacts(t)
	if _, err := Seal(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ManifestName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"path": ".bashrc"`, `"path": "../outside"`, 1))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(dir); err == nil {
		t.Fatal("path traversal accepted")
	}
}
