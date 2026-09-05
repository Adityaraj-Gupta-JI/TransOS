package app

import (
	"fmt"
	"os"

	"github.com/transos/transos/internal/extractor"
	"github.com/transos/transos/internal/injector"
	"github.com/transos/transos/internal/schema"
	"github.com/transos/transos/internal/wal"
)

const (
	DefaultProfilePath = "migration_profile.json"
	DefaultOutputDir   = "target_output"
	DefaultWALPath     = "target_output/transos.wal"
)

// ExtractProfile extracts the current host environment and persists it as
// a canonical TransOS migration profile.
func ExtractProfile(profilePath string) error {
	if profilePath == "" {
		profilePath = DefaultProfilePath
	}

	profile := extractor.ExtractHostEnvironment()
	if profile == nil {
		return fmt.Errorf("extract host environment returned a nil profile")
	}

	data, err := profile.Serialize()
	if err != nil {
		return fmt.Errorf("serialize migration profile: %w", err)
	}

	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		return fmt.Errorf("write migration profile %q: %w", profilePath, err)
	}

	return nil
}

// LoadProfile reads and validates an existing canonical TransOS migration
// profile.
func LoadProfile(profilePath string) (*schema.MigrationProfile, error) {
	if profilePath == "" {
		profilePath = DefaultProfilePath
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("read migration profile %q: %w", profilePath, err)
	}

	profile, err := schema.Deserialize(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize migration profile %q: %w", profilePath, err)
	}

	return profile, nil
}

// InjectProfile loads a migration profile and passes it to the injection
// engine.
func InjectProfile(profilePath string) error {
	profile, err := LoadProfile(profilePath)
	if err != nil {
		return err
	}

	if err := injector.InjectProfile(profile); err != nil {
		return fmt.Errorf("inject migration profile: %w", err)
	}

	return nil
}

// ValidateProfile verifies that a migration profile exists, contains valid
// JSON, and conforms to the current TransOS canonical schema.
func ValidateProfile(profilePath string) error {
	_, err := LoadProfile(profilePath)
	if err != nil {
		return err
	}

	return nil
}

// PreviewProfile reads and validates a migration profile without modifying it.
func PreviewProfile(profilePath string) ([]byte, error) {
	if profilePath == "" {
		profilePath = DefaultProfilePath
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("read migration profile %q: %w", profilePath, err)
	}

	if _, err := schema.Deserialize(data); err != nil {
		return nil, fmt.Errorf("validate migration profile %q: %w", profilePath, err)
	}

	return data, nil
}

// Rollback restores the previous state recorded by the TransOS WAL.
func Rollback(walPath string) error {
	if walPath == "" {
		walPath = DefaultWALPath
	}

	tx, err := wal.LoadWAL(walPath)
	if err != nil {
		return fmt.Errorf("load WAL %q: %w", walPath, err)
	}

	if err := tx.Rollback(); err != nil {
		return fmt.Errorf("rollback transaction: %w", err)
	}

	return nil
}

// RunExtraction is the compatibility entry point used by the current CLI.
func RunExtraction() error {
	return ExtractProfile(DefaultProfilePath)
}

// RunTranslation is temporarily retained for compatibility with the current
// CLI. Translation will become a standalone pipeline stage after canonical
// normalization and analysis are implemented.
func RunTranslation() error {
	return nil
}

// RunInjection is the compatibility entry point used by the current CLI.
func RunInjection() error {
	return InjectProfile(DefaultProfilePath)
}

// RunRollback is the compatibility entry point used by the current CLI.
func RunRollback() error {
	return Rollback(DefaultWALPath)
}
