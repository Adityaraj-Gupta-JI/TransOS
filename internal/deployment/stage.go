package deployment

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// rejectSymlinkAncestors checks every existing path component. Staging must not
// traverse symlinked source or destination directories, even if the final path
// itself does not exist. This is a preflight check, not a defense against a
// concurrent attacker replacing directories during the copy.
func rejectSymlinkAncestors(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for current := abs; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in staging path: %s", current)
			}
			if current != abs && !info.IsDir() {
				return fmt.Errorf("non-directory staging ancestor: %s", current)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return nil
}

// Stage copies verified artifacts to a new private directory for manual review.
// It never executes scripts, edits shell startup files, or invokes a package manager.
// Approval must be supplied explicitly by the caller.
func Stage(bundleDir, destination string, approved bool) error {
	return stageForHost(bundleDir, destination, approved, runtime.GOOS)
}

func stageForHost(bundleDir, destination string, approved bool, hostOS string) error {
	if !approved {
		return fmt.Errorf("explicit --approve required; nothing staged")
	}
	if hostOS != "linux" {
		return fmt.Errorf("staging is Linux-only; nothing staged")
	}
	if bundleDir == "" || destination == "" {
		return fmt.Errorf("bundle and destination paths are required")
	}
	bundleAbs, err := filepath.Abs(bundleDir)
	if err != nil {
		return err
	}
	destAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if destAbs == bundleAbs || destAbs == string(filepath.Separator) || destAbs == "." {
		return fmt.Errorf("unsafe staging destination")
	}
	// A destination inside the bundle could make the bundle itself mutable.
	rel, err := filepath.Rel(bundleAbs, destAbs)
	if err != nil {
		return err
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("destination must not be inside the source bundle")
	}
	if err := rejectSymlinkAncestors(bundleAbs); err != nil {
		return err
	}
	if err := rejectSymlinkAncestors(destAbs); err != nil {
		return err
	}
	verification, err := Verify(bundleAbs)
	if err != nil {
		return fmt.Errorf("verify source: %w", err)
	}
	if !verification.Valid() {
		return fmt.Errorf("source bundle integrity failed: %v", verification.Problems)
	}
	if _, err := os.Lstat(destAbs); err == nil {
		return fmt.Errorf("destination already exists: %s", destAbs)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Mkdir(destAbs, 0700); err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(destAbs)
		}
	}()
	for _, name := range append(append([]string{}, portableFiles...), ManifestName) {
		source := filepath.Join(bundleAbs, name)
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source is not a regular file: %s", name)
		}
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(filepath.Join(destAbs, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
		if closeInErr != nil {
			return closeInErr
		}
	}
	checked, err := Verify(destAbs)
	if err != nil {
		return err
	}
	if !checked.Valid() {
		return fmt.Errorf("staged bundle failed verification: %v", checked.Problems)
	}
	completed = true
	return nil
}
