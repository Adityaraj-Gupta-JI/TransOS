package deployment

import (
	"runtime"
	"strings"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	got := ParseOSRelease("# comment\nID=ubuntu\nVERSION_ID=\"24.04\"\nPRETTY_NAME=\"Ubuntu Linux\"\nBAD-KEY=ignored\n")
	if got["ID"] != "ubuntu" || got["VERSION_ID"] != "24.04" || got["PRETTY_NAME"] != "Ubuntu Linux" {
		t.Fatalf("unexpected values: %#v", got)
	}
	if _, exists := got["BAD-KEY"]; exists {
		t.Fatal("invalid key accepted")
	}
}

func TestCheckTargetRejectsTampering(t *testing.T) {
	dir := t.TempDir()
	// No manifest means no host checks can pass.
	_, err := CheckTarget(dir)
	if err == nil || !strings.Contains(err.Error(), "verify deployment package") {
		t.Fatalf("expected package integrity error, got %v", err)
	}
}

func TestDetectTargetNeverClaimsWindowsReady(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific assertion")
	}
	report := DetectTarget()
	if report.Ready() {
		t.Fatal("Windows cannot be a Linux deployment target")
	}
}
