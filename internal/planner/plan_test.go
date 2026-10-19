package planner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transos/transos/internal/schema"
)

func testProfile() *schema.MigrationProfile {
	return &schema.MigrationProfile{
		Metadata: schema.ProfileMetadata{
			ProfileID:     "profile-test-001",
			SchemaVersion: schema.CurrentSchemaVersion,
			CreatedAt:     "2026-10-19T00:00:00Z",
		},
		SourceSystem: schema.SourceSystem{OS: "windows", Architecture: "amd64"},
		Software: []schema.Software{
			{Name: "Visual Studio Code", Version: "1.0", Classification: schema.ClassificationNativeEquivalent},
			{Name: "Unmapped Tool", Classification: schema.ClassificationUnknown},
		},
	}
}

func TestCreateFromProfileRequiresExplicitDecisions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration_plan.json")
	plan, err := CreateFromProfile(testProfile(), path, false)
	if err != nil {
		t.Fatalf("CreateFromProfile() error = %v", err)
	}
	if len(plan.Items) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan.Items))
	}
	for _, item := range plan.Items {
		if item.Decision != DecisionPending {
			t.Errorf("item %q decision = %q; want pending", item.Name, item.Decision)
		}
	}
	if _, err := CreateFromProfile(testProfile(), path, false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing-plan protection, got %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ProfileID != "profile-test-001" {
		t.Fatalf("loaded profile ID = %q", loaded.ProfileID)
	}
}

func TestPlanDecisionsAndPackageMapping(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion, ProfileID: "p1", CreatedAt: "now", UpdatedAt: "now",
		Items: []Item{{Name: "Tool One", Decision: DecisionPending}},
	}
	if _, err := MapPackage(plan, "Tool One", "tool-one"); err != nil {
		t.Fatalf("MapPackage() error = %v", err)
	}
	if plan.Items[0].LinuxPackage != "tool-one" {
		t.Fatalf("LinuxPackage = %q", plan.Items[0].LinuxPackage)
	}
	if _, err := SetDecision(plan, "tool one", DecisionInclude); err != nil {
		t.Fatalf("SetDecision() error = %v", err)
	}
	if plan.Items[0].Decision != DecisionInclude {
		t.Fatalf("Decision = %q", plan.Items[0].Decision)
	}
	if _, err := MapPackage(plan, "Tool One", "$(touch pwned)"); err == nil {
		t.Fatal("expected unsafe package override to be rejected")
	}
}

func TestManualAddCanBeRemovedButDiscoveredItemsCannot(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion, ProfileID: "p1", CreatedAt: "now", UpdatedAt: "now",
		Items: []Item{{Name: "Discovered", Decision: DecisionPending}},
	}
	if err := AddManual(plan, "Manual Utility"); err != nil {
		t.Fatalf("AddManual() error = %v", err)
	}
	if err := AddManual(plan, "manual utility"); err == nil {
		t.Fatal("expected duplicate manual app to be rejected")
	}
	if err := RemoveManual(plan, "Discovered"); err == nil {
		t.Fatal("expected discovered software removal to be rejected")
	}
	if err := RemoveManual(plan, "Manual Utility"); err != nil {
		t.Fatalf("RemoveManual() error = %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected only the discovered item to remain, got %d", len(plan.Items))
	}
}

func TestResolveForInjectionOnlyIncludesApprovedItems(t *testing.T) {
	profile := testProfile()
	plan := &Plan{
		Version: PlanVersion, ProfileID: profile.Metadata.ProfileID, CreatedAt: "now", UpdatedAt: "now",
		Items: []Item{
			{Name: "Visual Studio Code", Version: "1.0", Classification: schema.ClassificationNativeEquivalent, Decision: DecisionInclude, LinuxPackage: "code-custom"},
			{Name: "Unmapped Tool", Classification: schema.ClassificationUnknown, Decision: DecisionPending},
			{Name: "Do Not Install", Classification: schema.ClassificationUnknown, Decision: DecisionExclude},
			{Name: "Manual Program", Classification: schema.ClassificationUnknown, Decision: DecisionInclude, Manual: true},
		},
	}
	software, overrides, summary, err := ResolveForInjection(profile, plan)
	if err != nil {
		t.Fatalf("ResolveForInjection() error = %v", err)
	}
	if len(software) != 2 {
		t.Fatalf("approved software count = %d; want 2", len(software))
	}
	if overrides["Visual Studio Code"] != "code-custom" {
		t.Fatalf("package override = %q", overrides["Visual Studio Code"])
	}
	if summary.Included != 2 || summary.Pending != 1 || summary.Excluded != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if _, _, _, err := ResolveForInjection(profile, &Plan{Version: PlanVersion, ProfileID: "other", CreatedAt: "now", UpdatedAt: "now"}); err == nil {
		t.Fatal("expected mismatched profile ID to be rejected")
	}
}

func TestLoadRejectsMalformedPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration_plan.json")
	if err := os.WriteFile(path, []byte(`{"plan_version":`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected malformed plan to be rejected")
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "plan.json")
	plan := &Plan{Version: PlanVersion, ProfileID: "p1", CreatedAt: "now", UpdatedAt: "now", Items: []Item{}}
	if err := Save(path, plan); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved plan missing: %v", err)
	}
}

func TestUniqueSubstringMatchesLongSoftwareName(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion, ProfileID: "p1", CreatedAt: "now", UpdatedAt: "now",
		Items: []Item{
			{Name: "Microsoft Visual Studio Code (User Install)", Decision: DecisionPending},
			{Name: "Canva", Decision: DecisionPending},
		},
	}
	count, err := SetDecision(plan, "Visual Studio Code", DecisionInclude)
	if err != nil {
		t.Fatalf("SetDecision() unique substring error = %v", err)
	}
	if count != 1 || plan.Items[0].Decision != DecisionInclude || plan.Items[1].Decision != DecisionPending {
		t.Fatalf("unexpected decisions after unique substring match: count=%d items=%+v", count, plan.Items)
	}
}

func TestAmbiguousSubstringMatchIsRejected(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion, ProfileID: "p1", CreatedAt: "now", UpdatedAt: "now",
		Items: []Item{
			{Name: "Python 3.10.11", Decision: DecisionPending},
			{Name: "Python 3.12.10", Decision: DecisionPending},
		},
	}
	if _, err := SetDecision(plan, "Python", DecisionInclude); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous substring error, got %v", err)
	}
	for _, item := range plan.Items {
		if item.Decision != DecisionPending {
			t.Fatalf("ambiguous match changed %q to %q", item.Name, item.Decision)
		}
	}
}
