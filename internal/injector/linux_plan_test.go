package injector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transos/transos/internal/planner"
	"github.com/transos/transos/internal/schema"
)

func TestInjectProfileUsesOnlyIncludedPlanItems(t *testing.T) {
	profile := &schema.MigrationProfile{
		Metadata: schema.ProfileMetadata{
			ProfileID:     "inject-test-profile",
			SchemaVersion: schema.CurrentSchemaVersion,
			CreatedAt:     "2026-10-19T00:00:00Z",
		},
		SourceSystem: schema.SourceSystem{OS: "windows", Architecture: "amd64"},
		Environment:  []schema.EnvironmentVar{},
		Software: []schema.Software{
			{Name: "Approved Custom App", Classification: schema.ClassificationUnknown},
			{Name: "Pending App", Classification: schema.ClassificationUnknown},
			{Name: "Excluded App", Classification: schema.ClassificationUnknown},
		},
		Registry:   []schema.RegistryEntry{},
		Filesystem: []schema.FilesystemItem{},
	}
	root := t.TempDir()
	planPath := filepath.Join(root, "migration_plan.json")
	outputDir := filepath.Join(root, "target_output")
	plan, err := planner.CreateFromProfile(profile, planPath, false)
	if err != nil {
		t.Fatalf("CreateFromProfile() error = %v", err)
	}
	for i := range plan.Items {
		switch plan.Items[i].Name {
		case "Approved Custom App":
			plan.Items[i].Decision = planner.DecisionInclude
			plan.Items[i].LinuxPackage = "approved-custom-app"
		case "Excluded App":
			plan.Items[i].Decision = planner.DecisionExclude
		}
	}
	if err := planner.Save(planPath, plan); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := InjectProfileWithOptions(profile, planPath, outputDir); err != nil {
		t.Fatalf("InjectProfileWithOptions() error = %v", err)
	}
	scriptPath := filepath.Join(outputDir, "install_dependencies.sh")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read generated installer: %v", err)
	}
	content := string(script)
	if !strings.Contains(content, "approved-custom-app") {
		t.Fatal("approved package override missing from generated installer")
	}
	if strings.Contains(content, "Pending App") {
		t.Fatal("pending software should not be included in generated installer")
	}
	if strings.Contains(content, "Excluded App") {
		t.Fatal("excluded software should not be included in generated installer")
	}
}

func TestInjectProfileRejectsStaleMigrationPlan(t *testing.T) {
	profile := &schema.MigrationProfile{
		Metadata:     schema.ProfileMetadata{ProfileID: "current", SchemaVersion: schema.CurrentSchemaVersion, CreatedAt: "2026-10-19T00:00:00Z"},
		SourceSystem: schema.SourceSystem{OS: "windows", Architecture: "amd64"},
	}
	root := t.TempDir()
	planPath := filepath.Join(root, "migration_plan.json")
	plan := &planner.Plan{
		Version: planner.PlanVersion, ProfileID: "old", CreatedAt: "now", UpdatedAt: "now", Items: []planner.Item{},
	}
	if err := planner.Save(planPath, plan); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	err := InjectProfileWithOptions(profile, planPath, filepath.Join(root, "target_output"))
	if err == nil || !strings.Contains(err.Error(), "belongs to profile") {
		t.Fatalf("expected stale plan error, got %v", err)
	}
}
