//go:build !windows

package extractor

import (
	"errors"

	"github.com/transos/transos/internal/config"
	"github.com/transos/transos/internal/schema"
)

// ErrWindowsExtractionUnsupported indicates that source capture requires Windows.
// Linux hosts can still inspect, verify, stage, apply, and restore existing bundles.
var ErrWindowsExtractionUnsupported = errors.New("Windows source extraction is supported only on Windows; transfer a generated bundle from the Windows source")

// ExtractHostEnvironment retains the legacy API without pretending that a
// non-Windows host can capture Windows registry and software state.
func ExtractHostEnvironment() *schema.MigrationProfile {
	return nil
}

// ExtractHostEnvironmentWithConfig deliberately rejects source capture on
// non-Windows hosts. Deployment operations do not depend on this function.
func ExtractHostEnvironmentWithConfig(_ config.Config) (*schema.MigrationProfile, error) {
	return nil, ErrWindowsExtractionUnsupported
}
