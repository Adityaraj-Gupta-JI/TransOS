package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const hookStart = "# >>> TransOS managed environment >>>"
const hookEnd = "# <<< TransOS managed environment <<<"
const hookBlock = "\n" + hookStart + "\nif [ -f \"$HOME/.config/transos/transos_env.conf\" ]; then\n    . \"$HOME/.config/transos/transos_env.conf\"\nfi\n" + hookEnd + "\n"

type AppliedFile struct {
	Path     string `json:"path"`
	Existed  bool   `json:"existed"`
	Original string `json:"original"`
	Applied  string `json:"applied"`
	Mode     uint32 `json:"mode"`
}
type ApplyRecord struct {
	Version   int           `json:"version"`
	CreatedAt string        `json:"created_at"`
	Home      string        `json:"home"`
	Files     []AppliedFile `json:"files"`
}

func checksum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func checkRegularOrMissing(path string) (bool, os.FileMode, []byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, 0600, nil, nil
	}
	if err != nil {
		return false, 0, nil, err
	}
	if !info.Mode().IsRegular() {
		return false, 0, nil, fmt.Errorf("refusing non-regular target: %s", path)
	}
	b, err := os.ReadFile(path)
	return true, info.Mode().Perm(), b, err
}
func secureParents(home, path string) error {
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	pAbs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(homeAbs, pAbs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == "." {
		return fmt.Errorf("target outside home: %s", path)
	}
	info, err := os.Lstat(homeAbs)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe home directory")
	}
	parent := filepath.Dir(pAbs)
	for parent != homeAbs {
		info, err := os.Lstat(parent)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe target parent: %s", parent)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		parent = filepath.Dir(parent)
	}
	return nil
}
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".transos-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func plannedFiles(bundle, home string) ([]string, [][]byte, error) {
	env, err := os.ReadFile(filepath.Join(bundle, "transos_env.conf"))
	if err != nil {
		return nil, nil, err
	}
	paths := []string{filepath.Join(home, ".config", "transos", "transos_env.conf"), filepath.Join(home, ".bashrc"), filepath.Join(home, ".zshrc")}
	contents := [][]byte{env}
	for _, p := range paths[1:] {
		_, _, original, err := checkRegularOrMissing(p)
		if err != nil {
			return nil, nil, err
		}
		s := string(original)
		if strings.Contains(s, hookStart) || strings.Contains(s, hookEnd) {
			if !strings.Contains(s, hookBlock[1:len(hookBlock)-1]) {
				return nil, nil, fmt.Errorf("existing TransOS managed block differs: %s", p)
			}
			contents = append(contents, original)
			continue
		}
		if strings.Contains(s, "transos_env.conf") {
			return nil, nil, fmt.Errorf("unmanaged TransOS hook already exists: %s", p)
		}
		contents = append(contents, []byte(strings.TrimRight(s, "\n")+hookBlock))
	}
	return paths, contents, nil
}

// ApplyUserConfig changes only the target user's environment and shell startup files.
// The caller must provide a new, non-existing recovery directory; no packages are executed.
func ApplyUserConfig(bundle, home, recovery string, approved bool) error {
	return applyForHost(bundle, home, recovery, approved, runtime.GOOS)
}
func applyForHost(bundle, home, recovery string, approved bool, host string) error {
	if !approved {
		return fmt.Errorf("explicit approval required")
	}
	if host != "linux" {
		return fmt.Errorf("application is Linux-only")
	}
	if bundle == "" || home == "" || recovery == "" {
		return fmt.Errorf("bundle, home, and recovery paths required")
	}
	v, err := Verify(bundle)
	if err != nil {
		return err
	}
	if !v.Valid() {
		return fmt.Errorf("bundle integrity failure: %v", v.Problems)
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return err
	}
	recovery, err = filepath.Abs(recovery)
	if err != nil {
		return err
	}
	bundle, err = filepath.Abs(bundle)
	if err != nil {
		return err
	}
	if recovery == home || recovery == bundle {
		return fmt.Errorf("unsafe recovery location")
	}
	if rel, e := filepath.Rel(home, recovery); e == nil && (rel == "." || rel == ".." || !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("recovery must be outside target home")
	}
	if _, err := os.Lstat(recovery); err == nil {
		return fmt.Errorf("recovery directory already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	paths, contents, err := plannedFiles(bundle, home)
	if err != nil {
		return err
	}
	record := ApplyRecord{Version: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339), Home: home}
	originals := make([][]byte, len(paths))
	for i, p := range paths {
		if err := secureParents(home, p); err != nil {
			return err
		}
		exists, mode, b, err := checkRegularOrMissing(p)
		if err != nil {
			return err
		}
		originals[i] = b
		if !exists {
			mode = 0600
		}
		record.Files = append(record.Files, AppliedFile{Path: p, Existed: exists, Original: checksum(b), Applied: checksum(contents[i]), Mode: uint32(mode)})
	}
	if err := os.Mkdir(recovery, 0700); err != nil {
		return err
	}
	// Persist recovery before changing any target. Retain it even if application fails.
	for i, f := range record.Files {
		if f.Existed {
			if err := os.WriteFile(filepath.Join(recovery, fmt.Sprintf("original-%d", i)), originals[i], 0600); err != nil {
				return err
			}
		}
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(recovery, "record.json"), data, 0600); err != nil {
		return err
	}
	for i, p := range paths {
		if err := secureParents(home, p); err != nil {
			return err
		}
		exists, _, current, err := checkRegularOrMissing(p)
		if err != nil {
			return err
		}
		if exists != record.Files[i].Existed || checksum(current) != record.Files[i].Original {
			return fmt.Errorf("target changed during application: %s", p)
		}
		if err := writeAtomic(p, contents[i], os.FileMode(record.Files[i].Mode)); err != nil {
			return fmt.Errorf("apply %s: %w (recovery at %s)", p, err, recovery)
		}
	}
	return nil
}
func loadRecord(recovery string) (ApplyRecord, error) {
	var r ApplyRecord
	b, e := os.ReadFile(filepath.Join(recovery, "record.json"))
	if e != nil {
		return r, e
	}
	e = json.Unmarshal(b, &r)
	if e != nil {
		return r, e
	}
	if r.Version != 1 || len(r.Files) != 3 || r.Home == "" {
		return r, fmt.Errorf("invalid recovery record")
	}
	expected := []string{filepath.Join(r.Home, ".config", "transos", "transos_env.conf"), filepath.Join(r.Home, ".bashrc"), filepath.Join(r.Home, ".zshrc")}
	for i, f := range r.Files {
		if f.Path != expected[i] {
			return r, fmt.Errorf("invalid recovery target: %s", f.Path)
		}
		if err := secureParents(r.Home, f.Path); err != nil {
			return r, err
		}
	}
	return r, nil
}
func VerifyApplied(recovery string) error {
	r, err := loadRecord(recovery)
	if err != nil {
		return err
	}
	for _, f := range r.Files {
		_, _, b, e := checkRegularOrMissing(f.Path)
		if e != nil {
			return e
		}
		if checksum(b) != f.Applied {
			return fmt.Errorf("applied configuration differs: %s", f.Path)
		}
	}
	return nil
}

// RestoreUserConfig refuses to overwrite files changed since TransOS applied them.
// RestoreUserConfig can also recover a partially completed application.
// It refuses any file that differs from both the saved original and applied hashes.
func RestoreUserConfig(recovery string, approved bool) error {
	if !approved {
		return fmt.Errorf("explicit approval required")
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("restore is Linux-only")
	}
	r, err := loadRecord(recovery)
	if err != nil {
		return err
	}
	originals := make([][]byte, len(r.Files))
	needsRestore := make([]bool, len(r.Files))
	// Validate all files and backups before writing anything.
	for i, f := range r.Files {
		exists, _, current, e := checkRegularOrMissing(f.Path)
		if e != nil {
			return e
		}
		digest := checksum(current)
		if digest != f.Applied && digest != f.Original {
			return fmt.Errorf("file changed outside TransOS: %s", f.Path)
		}
		if f.Existed {
			b, e := os.ReadFile(filepath.Join(recovery, fmt.Sprintf("original-%d", i)))
			if e != nil {
				return e
			}
			if checksum(b) != f.Original {
				return fmt.Errorf("backup checksum mismatch: %s", f.Path)
			}
			originals[i] = b
			needsRestore[i] = digest != f.Original || !exists
		} else {
			needsRestore[i] = exists
			if !exists && digest != f.Original {
				return fmt.Errorf("unexpected missing state: %s", f.Path)
			}
		}
	}
	for i, f := range r.Files {
		if !needsRestore[i] {
			continue
		}
		if f.Existed {
			if err := writeAtomic(f.Path, originals[i], os.FileMode(f.Mode)); err != nil {
				return err
			}
		} else {
			if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
