package normalizer

import (
	"testing"

	"github.com/transos/transos/internal/schema"
)

func TestNormalizeProfileDeduplicatesEnvironment(t *testing.T) {
	profile := schema.NewProfile("windows", "amd64")

	profile.Environment = []schema.EnvironmentVar{
		{
			Name:           "Path",
			Value:          "process-path",
			Type:           schema.EnvironmentTypePathList,
			Scope:          schema.ScopeUser,
			Source:         "process_environment",
			Classification: schema.ClassificationUnknown,
		},
		{
			Name:           "PATH",
			Value:          "persistent-path",
			Type:           schema.EnvironmentTypePathList,
			Scope:          schema.ScopeUser,
			Source:         `HKCU\Environment`,
			Classification: schema.ClassificationUnknown,
		},
	}

	normalized, err := NormalizeProfile(profile)
	if err != nil {
		t.Fatalf("NormalizeProfile() error = %v", err)
	}

	if len(normalized.Environment) != 1 {
		t.Fatalf("expected 1 environment variable, got %d", len(normalized.Environment))
	}

	if normalized.Environment[0].Value != "persistent-path" {
		t.Fatalf(
			"expected persistent environment value, got %q",
			normalized.Environment[0].Value,
		)
	}

	if normalized.Environment[0].Classification != schema.ClassificationConvertible {
		t.Fatalf(
			"expected CONVERTIBLE classification, got %q",
			normalized.Environment[0].Classification,
		)
	}
}

func TestNormalizeProfileClassifiesKnownSoftware(t *testing.T) {
	profile := schema.NewProfile("windows", "amd64")

	profile.Software = []schema.Software{
		{
			Name:            "Microsoft Visual Studio Code (User)",
			Version:         "1.136.1",
			Publisher:       "Microsoft Corporation",
			Classification:  schema.ClassificationUnknown,
			MigrationStatus: "DISCOVERED",
		},
	}

	normalized, err := NormalizeProfile(profile)
	if err != nil {
		t.Fatalf("NormalizeProfile() error = %v", err)
	}

	if len(normalized.Software) != 1 {
		t.Fatalf("expected 1 software entry, got %d", len(normalized.Software))
	}

	if normalized.Software[0].Classification != schema.ClassificationNativeEquivalent {
		t.Fatalf(
			"expected NATIVE_EQUIVALENT, got %q",
			normalized.Software[0].Classification,
		)
	}

	if normalized.Software[0].MigrationStatus != "NORMALIZED" {
		t.Fatalf(
			"expected NORMALIZED status, got %q",
			normalized.Software[0].MigrationStatus,
		)
	}
}

func TestNormalizeProfilePreservesUnknownSoftware(t *testing.T) {
	profile := schema.NewProfile("windows", "amd64")

	profile.Software = []schema.Software{
		{
			Name:           "Some Unknown Windows Application",
			Version:        "1.0",
			Classification: schema.ClassificationUnknown,
		},
	}

	normalized, err := NormalizeProfile(profile)
	if err != nil {
		t.Fatalf("NormalizeProfile() error = %v", err)
	}

	if normalized.Software[0].Classification != schema.ClassificationUnknown {
		t.Fatalf(
			"expected UNKNOWN classification, got %q",
			normalized.Software[0].Classification,
		)
	}
}

func TestNormalizeProfilePreservesExtractionSettingsWithoutAliasing(t *testing.T) {
	profile := schema.NewProfile("windows", "amd64")
	profile.Metadata.OperationMode = " GUIDED "
	profile.Metadata.ExtractionDepth = " MEDIUM "
	profile.Metadata.ExtractionCategories = []string{"Environment", "Software", "Registry", "Theme"}

	normalized, err := NormalizeProfile(profile)
	if err != nil {
		t.Fatalf("NormalizeProfile() error = %v", err)
	}

	if normalized.Metadata.OperationMode != "guided" {
		t.Fatalf("operation mode = %q, want guided", normalized.Metadata.OperationMode)
	}
	if normalized.Metadata.ExtractionDepth != "medium" {
		t.Fatalf("extraction depth = %q, want medium", normalized.Metadata.ExtractionDepth)
	}
	if normalized.Metadata.ExtractionCategories[0] != "environment" {
		t.Fatalf("first normalized category = %q, want environment", normalized.Metadata.ExtractionCategories[0])
	}

	normalized.Metadata.ExtractionCategories[0] = "changed"
	if profile.Metadata.ExtractionCategories[0] != "Environment" {
		t.Fatal("normalizing profile metadata unexpectedly mutated the original category slice")
	}
}
