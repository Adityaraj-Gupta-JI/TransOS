package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ManifestName = "transos_manifest.json"
const ManifestVersion = 1

// Only generated, portable deployment artifacts are sealed. WAL/checkpoints and
// backup files are intentionally excluded: they are host-local recovery state.
var portableFiles = []string{".bashrc", ".zshrc", "install_dependencies.sh", "transos_env.conf"}

type FileEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type Manifest struct {
	Version   int         `json:"version"`
	CreatedAt string      `json:"created_at"`
	Files     []FileEntry `json:"files"`
}
type Verification struct {
	FilesChecked int
	Problems     []string
}

func (v Verification) Valid() bool { return len(v.Problems) == 0 }

func fileDigest(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("not a regular file: %s", path)
	}
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
func Seal(dir string) (*Manifest, error) {
	if dir == "" {
		return nil, errors.New("output directory required")
	}
	m := &Manifest{Version: ManifestVersion, CreatedAt: time.Now().UTC().Format(time.RFC3339), Files: make([]FileEntry, 0, len(portableFiles))}
	for _, name := range portableFiles {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("required generated artifact missing: %s", name)
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("artifact must be a regular file (not symlink): %s", name)
		}
		digest, size, err := fileDigest(path)
		if err != nil {
			return nil, err
		}
		m.Files = append(m.Files, FileEntry{Path: name, SHA256: digest, Size: size})
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	path := filepath.Join(dir, ManifestName)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("manifest path is not a regular file: %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	tmp, err := os.CreateTemp(dir, ".transos-manifest-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return nil, err
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	// Windows does not support replacing an existing destination via os.Rename.
	if err = os.Rename(tmpPath, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			return nil, fmt.Errorf("cannot replace existing manifest on this platform; move %s aside and retry: %w", path, err)
		}
		return nil, err
	}
	return m, nil
}
func Load(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if m.Version != ManifestVersion {
		return nil, fmt.Errorf("unsupported manifest version: %d", m.Version)
	}
	if len(m.Files) != len(portableFiles) {
		return nil, fmt.Errorf("manifest must contain exactly %d portable artifacts", len(portableFiles))
	}
	expected := make(map[string]bool, len(portableFiles))
	for _, name := range portableFiles {
		expected[name] = true
	}
	seen := map[string]bool{}
	for _, f := range m.Files {
		if !expected[f.Path] || seen[f.Path] || filepath.Base(f.Path) != f.Path || strings.ContainsAny(f.Path, "/\\") {
			return nil, fmt.Errorf("unexpected or duplicate manifest path: %q", f.Path)
		}
		seen[f.Path] = true
		if len(f.SHA256) != 64 {
			return nil, fmt.Errorf("invalid SHA-256 for %s", f.Path)
		}
		if _, err := hex.DecodeString(f.SHA256); err != nil {
			return nil, fmt.Errorf("invalid SHA-256 for %s: %w", f.Path, err)
		}
		if f.Size < 0 {
			return nil, fmt.Errorf("invalid size for %s", f.Path)
		}
	}
	return &m, nil
}
func Verify(dir string) (Verification, error) {
	m, err := Load(dir)
	if err != nil {
		return Verification{}, err
	}
	v := Verification{}
	for _, entry := range m.Files {
		path := filepath.Join(dir, entry.Path)
		info, err := os.Lstat(path)
		if err != nil {
			v.Problems = append(v.Problems, fmt.Sprintf("%s: %v", entry.Path, err))
			continue
		}
		if !info.Mode().IsRegular() {
			v.Problems = append(v.Problems, entry.Path+": not a regular file")
			continue
		}
		digest, size, err := fileDigest(path)
		if err != nil {
			v.Problems = append(v.Problems, fmt.Sprintf("%s: %v", entry.Path, err))
			continue
		}
		v.FilesChecked++
		if digest != entry.SHA256 || size != entry.Size {
			v.Problems = append(v.Problems, entry.Path+": checksum or size mismatch")
		}
	}
	return v, nil
}
