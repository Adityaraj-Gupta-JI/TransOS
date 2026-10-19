package safety

import (
	"testing"

	"github.com/transos/transos/internal/schema"
)

func TestIsSensitiveEnvironmentName(t *testing.T) {
	sensitive := []string{
		"GITHUB_TOKEN", "OPENAI_API_KEY", "AWS_SECRET_ACCESS_KEY", "DB_PASSWORD",
		"AZURE_CLIENT_SECRET", "SSH_PRIVATE_KEY", "DATABASE_URL", "NPM-TOKEN",
	}
	for _, name := range sensitive {
		if !IsSensitiveEnvironmentName(name) {
			t.Errorf("IsSensitiveEnvironmentName(%q) = false, want true", name)
		}
	}

	ordinary := []string{"PATH", "HOME", "TEMP", "JAVA_HOME", "PROCESSOR_ARCHITECTURE", "LANG"}
	for _, name := range ordinary {
		if IsSensitiveEnvironmentName(name) {
			t.Errorf("IsSensitiveEnvironmentName(%q) = true, want false", name)
		}
	}
}

func TestRedactSensitiveEnvironment(t *testing.T) {
	profile := &schema.MigrationProfile{Environment: []schema.EnvironmentVar{
		{Name: "GITHUB_TOKEN", Value: "secret-value", Classification: schema.ClassificationNativeEquivalent},
		{Name: "PATH", Value: "/usr/bin", Classification: schema.ClassificationConvertible},
	}}
	if got := RedactSensitiveEnvironment(profile); got != 1 {
		t.Fatalf("RedactSensitiveEnvironment() = %d, want 1", got)
	}
	if profile.Environment[0].Value != "" || profile.Environment[0].Classification != SensitiveRedactedClassification {
		t.Fatalf("sensitive environment entry was not redacted: %#v", profile.Environment[0])
	}
	if profile.Environment[1].Value != "/usr/bin" {
		t.Fatalf("ordinary environment value changed: %#v", profile.Environment[1])
	}
}
