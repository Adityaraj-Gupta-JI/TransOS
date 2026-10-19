package deployment

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// TargetReport describes a read-only compatibility assessment. A passing
// report does not authorize installation or claim packages are available.
type TargetReport struct {
	OS             string
	Distro         string
	DistroVersion  string
	Architecture   string
	PackageManager string
	Shell          string
	Blockers       []string
	Warnings       []string
}

func (r TargetReport) Ready() bool { return len(r.Blockers) == 0 }

// DetectTarget performs no mutations and never invokes a package manager.
func DetectTarget() TargetReport {
	r := TargetReport{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if runtime.GOOS != "linux" {
		r.Blockers = append(r.Blockers, "target checks must run on the destination Linux machine")
		return r
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		r.Blockers = append(r.Blockers, "cannot read /etc/os-release: "+err.Error())
	} else {
		values := ParseOSRelease(string(data))
		r.Distro, r.DistroVersion = values["ID"], values["VERSION_ID"]
		if r.Distro == "" {
			r.Warnings = append(r.Warnings, "distribution ID missing from /etc/os-release")
		}
	}
	for _, candidate := range []string{"apt-get", "dnf", "pacman", "zypper", "apk"} {
		if _, err := exec.LookPath(candidate); err == nil {
			r.PackageManager = candidate
			break
		}
	}
	if r.PackageManager == "" {
		r.Blockers = append(r.Blockers, "no recognized Linux package manager found")
	}
	for _, candidate := range []string{"bash", "sh"} {
		if _, err := exec.LookPath(candidate); err == nil {
			r.Shell = candidate
			break
		}
	}
	if r.Shell == "" {
		r.Blockers = append(r.Blockers, "no compatible POSIX shell found")
	}
	if r.PackageManager != "apt-get" && r.PackageManager != "dnf" && r.PackageManager != "" {
		r.Warnings = append(r.Warnings, "generated installer may require manual adaptation for "+r.PackageManager)
	}
	return r
}

// ParseOSRelease reads only literal key/value pairs, never evaluating shell code.
func ParseOSRelease(input string) map[string]string {
	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		valid := true
		for _, c := range key {
			if !((c >= 'A' && c <= 'Z') || c == '_') {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\\`, `\`)
		}
		result[key] = value
	}
	return result
}

// CheckTarget verifies the package before assessing the local Linux host.
func CheckTarget(dir string) (TargetReport, error) {
	verification, err := Verify(dir)
	if err != nil {
		return TargetReport{}, fmt.Errorf("verify deployment package: %w", err)
	}
	if !verification.Valid() {
		return TargetReport{}, fmt.Errorf("deployment package failed integrity verification: %s", strings.Join(verification.Problems, "; "))
	}
	return DetectTarget(), nil
}
