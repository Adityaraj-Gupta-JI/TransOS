//go:build !windows

package extractor

import (
	"errors"
	"testing"

	"github.com/transos/transos/internal/config"
)

func TestNonWindowsExtractionIsExplicitlyUnsupported(t *testing.T) {
	if got := ExtractHostEnvironment(); got != nil {
		t.Fatal("legacy extraction returned a profile on a non-Windows host")
	}
	profile, err := ExtractHostEnvironmentWithConfig(config.Default())
	if profile != nil || !errors.Is(err, ErrWindowsExtractionUnsupported) {
		t.Fatalf("profile=%v, err=%v; want nil profile and unsupported error", profile, err)
	}
}
