package deployment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutionPlanReadOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range portableFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("sample"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Seal(dir); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanExecution(dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.VerifiedFiles != len(portableFiles) || len(plan.Actions) != 4 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	for _, a := range plan.Actions {
		if !a.RequiresApproval {
			t.Fatalf("unapproved action: %+v", a)
		}
	}
	after, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("manifest modified")
	}
}
func TestExecutionPlanRejectsTampering(t *testing.T) {
	dir := t.TempDir()
	for _, name := range portableFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("sample"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Seal(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".bashrc"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanExecution(dir); err == nil {
		t.Fatal("expected integrity failure")
	}
}
