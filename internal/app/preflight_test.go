package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transos/transos/internal/planner"
	"github.com/transos/transos/internal/schema"
)

func writePreflightProfile(t *testing.T, path string, profile *schema.MigrationProfile) {
	t.Helper()
	data, err := profile.Serialize()
	if err != nil {
		t.Fatalf("serialize profile: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write profile: %v", err)
	}
}

func TestAnalyzePreflightIsReadOnlyAndReportsPendingPlan(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.json")
	planPath := filepath.Join(dir, "plan.json")
	outputDir := filepath.Join(dir, "target_output")

	profile := schema.NewProfile("windows", "amd64")
	profile.Environment = []schema.EnvironmentVar{
		{Name: "PATH", Value: `C:\Windows`, Type: schema.EnvironmentTypePath, Scope: schema.ScopeUser, Classification: schema.ClassificationConvertible},
		{Name: "GITHUB_TOKEN", Value: "test-secret-value", Type: schema.EnvironmentTypeString, Scope: schema.ScopeUser, Classification: schema.ClassificationNativeEquivalent},
	}
	profile.Software = []schema.Software{{Name: "aria2", Version: "1.0", Classification: schema.ClassificationNativeEquivalent}}
	writePreflightProfile(t, profilePath, profile)

	loaded, err := LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("load fixture profile: %v", err)
	}
	if _, err := planner.CreateFromProfile(loaded, planPath, false); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	report, err := AnalyzePreflight(profilePath, planPath, outputDir)
	if err != nil {
		t.Fatalf("AnalyzePreflight() error = %v", err)
	}
	if !report.Ready() {
		t.Fatalf("preflight unexpectedly blocked: %v", report.Blockers)
	}
	if report.Pending != 1 || report.Included != 0 {
		t.Fatalf("unexpected plan summary: included=%d pending=%d", report.Included, report.Pending)
	}
	if len(report.SensitiveEnvironment) != 1 || report.SensitiveEnvironment[0] != "GITHUB_TOKEN" {
		t.Fatalf("sensitive environment names = %#v", report.SensitiveEnvironment)
	}
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Fatalf("preflight should not create output directory; stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "target_output", "install_dependencies.sh")); !os.IsNotExist(err) {
		t.Fatalf("preflight should not create installer; stat error = %v", err)
	}
}

func TestAnalyzePreflightBlocksStalePlan(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.json")
	planPath := filepath.Join(dir, "plan.json")
	outputDir := filepath.Join(dir, "target_output")

	first := schema.NewProfile("windows", "amd64")
	first.Software = []schema.Software{{Name: "aria2", Classification: schema.ClassificationNativeEquivalent}}
	writePreflightProfile(t, profilePath, first)
	if _, err := planner.CreateFromProfile(first, planPath, false); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	second := schema.NewProfile("windows", "amd64")
	second.Software = []schema.Software{{Name: "aria2", Classification: schema.ClassificationNativeEquivalent}}
	writePreflightProfile(t, profilePath, second)

	report, err := AnalyzePreflight(profilePath, planPath, outputDir)
	if err != nil {
		t.Fatalf("AnalyzePreflight() error = %v", err)
	}
	if report.Ready() || len(report.Blockers) == 0 {
		t.Fatalf("stale plan should block preflight, got blockers=%v", report.Blockers)
	}
	if !strings.Contains(strings.Join(report.Blockers, " "), "belongs to profile") {
		t.Fatalf("stale-plan blocker not clear enough: %v", report.Blockers)
	}
}
