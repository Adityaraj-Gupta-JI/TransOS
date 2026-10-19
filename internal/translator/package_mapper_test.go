package translator

import (
	"strings"
	"testing"

	"github.com/transos/transos/internal/schema"
)

func TestGenerateDependencyScriptWithReviewedPackageOverride(t *testing.T) {
	software := []schema.Software{{Name: "Mystery App $HOME `touch nope`", Classification: schema.ClassificationUnknown}}
	script := GenerateDependencyScriptWithOverrides(software, map[string]string{
		"Mystery App $HOME `touch nope`": "mystery-app",
	})
	if !strings.Contains(script, "mystery-app") {
		t.Fatal("generated script does not contain the reviewed package override")
	}
	if strings.Contains(script, "$(touch nope)") {
		t.Fatal("generated script contains an unescaped command-substitution payload")
	}
	if !strings.Contains(script, `\$HOME`) || !strings.Contains(script, "\\`touch nope\\`") {
		t.Fatal("application name is not escaped for the generated shell context")
	}
}

func TestUnknownSoftwareIsReportedForManualReview(t *testing.T) {
	script := GenerateDependencyScript([]schema.Software{{Name: "Unknown Vendor App", Classification: schema.ClassificationUnknown}})
	if !strings.Contains(script, "Unknown Vendor App -> manual review required") {
		t.Fatal("unknown software disappeared instead of being reported for manual review")
	}
}

func TestInvalidPackageOverrideIsNotEmittedAsCommandArgument(t *testing.T) {
	script := GenerateDependencyScriptWithOverrides(
		[]schema.Software{{Name: "Custom App", Classification: schema.ClassificationUnknown}},
		map[string]string{"Custom App": "$(touch /tmp/pwned)"},
	)
	if strings.Contains(script, "touch /tmp/pwned") {
		t.Fatal("invalid package override was emitted into the installer")
	}
	if !strings.Contains(script, "Custom App -> manual review required") {
		t.Fatal("invalid package override should fall back to manual review")
	}
}
