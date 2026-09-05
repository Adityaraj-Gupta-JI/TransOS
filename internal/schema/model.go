package schema

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	CurrentSchemaVersion = "2.0.0"

	// Environment value types.
	EnvironmentTypeString     = "STRING"
	EnvironmentTypePath       = "PATH"
	EnvironmentTypePathList   = "PATH_LIST"
	EnvironmentTypeDirectory  = "DIRECTORY"
	EnvironmentTypeFile       = "FILE"
	EnvironmentTypeLocale     = "LOCALE"
	EnvironmentTypeIdentifier = "IDENTIFIER"
	EnvironmentTypeNumber     = "NUMBER"
	EnvironmentTypeBoolean    = "BOOLEAN"
	EnvironmentTypeUnknown    = "UNKNOWN"

	// Migration classifications.
	ClassificationExact              = "EXACT"
	ClassificationConvertible        = "CONVERTIBLE"
	ClassificationNativeEquivalent   = "NATIVE_EQUIVALENT"
	ClassificationCompatibilityLayer = "COMPATIBILITY_LAYER"
	ClassificationManual             = "MANUAL"
	ClassificationUnsupported        = "UNSUPPORTED"
	ClassificationUnknown            = "UNKNOWN"

	// Environment scopes.
	ScopeUser    = "USER"
	ScopeSystem  = "SYSTEM"
	ScopeUnknown = "UNKNOWN"
)

// MigrationProfile is the canonical, platform-neutral representation of
// system state collected by TransOS.
//
// Extraction records what exists on the source system.
// Later pipeline stages are responsible for normalization, analysis,
// planning, application, and verification.
type MigrationProfile struct {
	Metadata     ProfileMetadata  `json:"metadata"`
	SourceSystem SourceSystem     `json:"source_system"`
	Environment  []EnvironmentVar `json:"environment"`
	Software     []Software       `json:"software"`
	Registry     []RegistryEntry  `json:"registry"`
	Theme        ThemeConfig      `json:"theme"`
	Shell        ShellConfig      `json:"shell"`
	Filesystem   []FilesystemItem `json:"filesystem"`
}

// ProfileMetadata identifies the profile and the TransOS schema that produced it.
type ProfileMetadata struct {
	ProfileID        string `json:"profile_id"`
	SchemaVersion    string `json:"schema_version"`
	CreatedAt        string `json:"created_at"`
	Generator        string `json:"generator"`
	GeneratorVersion string `json:"generator_version,omitempty"`
}

// SourceSystem describes the machine and operating system from which the
// migration profile was extracted.
type SourceSystem struct {
	OS           string `json:"os"`
	Version      string `json:"version,omitempty"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname,omitempty"`
	Username     string `json:"username,omitempty"`
}

// EnvironmentVar represents one environment variable while preserving its
// semantic type, scope, provenance, and migration classification.
type EnvironmentVar struct {
	Name           string `json:"name"`
	Value          string `json:"value"`
	Type           string `json:"type"`
	Scope          string `json:"scope"`
	Source         string `json:"source,omitempty"`
	Classification string `json:"classification"`
}

// Software represents an installed application or software component.
type Software struct {
	Name            string `json:"name"`
	Version         string `json:"version,omitempty"`
	Publisher       string `json:"publisher,omitempty"`
	Architecture    string `json:"architecture,omitempty"`
	InstallLocation string `json:"install_location,omitempty"`
	Scope           string `json:"scope,omitempty"`
	Source          string `json:"source,omitempty"`
	Classification  string `json:"classification"`
	MigrationStatus string `json:"migration_status,omitempty"`
}

// RegistryEntry represents a platform registry/configuration entry while
// preserving its source location and value type.
type RegistryEntry struct {
	Hive           string `json:"hive"`
	Path           string `json:"path"`
	Name           string `json:"name"`
	Type           string `json:"type,omitempty"`
	Value          string `json:"value"`
	Source         string `json:"source,omitempty"`
	Classification string `json:"classification"`
}

// ThemeConfig stores platform-neutral visual configuration.
type ThemeConfig struct {
	Name       string            `json:"name,omitempty"`
	Mode       string            `json:"mode,omitempty"`
	Colors     map[string]string `json:"colors,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ShellConfig describes shell-related source configuration.
type ShellConfig struct {
	DefaultShell string            `json:"default_shell,omitempty"`
	ProfileFiles []string          `json:"profile_files,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

// FilesystemItem represents a filesystem object that may be relevant to
// migration.
type FilesystemItem struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	Scope          string `json:"scope,omitempty"`
	Size           int64  `json:"size,omitempty"`
	Classification string `json:"classification"`
	Source         string `json:"source,omitempty"`
}

// Serialize converts the canonical migration profile into indented JSON.
func (p *MigrationProfile) Serialize() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	return json.MarshalIndent(p, "", "  ")
}

// Deserialize converts canonical migration-profile JSON into a validated
// MigrationProfile.
func Deserialize(data []byte) (*MigrationProfile, error) {
	var profile MigrationProfile

	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("decode migration profile: %w", err)
	}

	if err := profile.Validate(); err != nil {
		return nil, err
	}

	return &profile, nil
}

// NewProfile creates an empty canonical migration profile.
func NewProfile(sourceOS, architecture string) *MigrationProfile {
	now := time.Now().UTC()

	return &MigrationProfile{
		Metadata: ProfileMetadata{
			ProfileID:     fmt.Sprintf("transos-%d", now.UnixNano()),
			SchemaVersion: CurrentSchemaVersion,
			CreatedAt:     now.Format(time.RFC3339),
			Generator:     "TransOS",
		},
		SourceSystem: SourceSystem{
			OS:           sourceOS,
			Architecture: architecture,
		},
		Environment: make([]EnvironmentVar, 0),
		Software:    make([]Software, 0),
		Registry:    make([]RegistryEntry, 0),
		Theme: ThemeConfig{
			Colors:     make(map[string]string),
			Properties: make(map[string]string),
		},
		Shell: ShellConfig{
			ProfileFiles: make([]string, 0),
			Variables:    make(map[string]string),
			Properties:   make(map[string]string),
		},
		Filesystem: make([]FilesystemItem, 0),
	}
}

// Validate performs structural validation of the canonical profile.
func (p *MigrationProfile) Validate() error {
	if p == nil {
		return fmt.Errorf("migration profile is nil")
	}

	if p.Metadata.SchemaVersion == "" {
		return fmt.Errorf("metadata.schema_version is required")
	}

	if p.Metadata.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf(
			"unsupported migration profile schema version %q, expected %q",
			p.Metadata.SchemaVersion,
			CurrentSchemaVersion,
		)
	}

	if p.Metadata.ProfileID == "" {
		return fmt.Errorf("metadata.profile_id is required")
	}

	if p.Metadata.CreatedAt == "" {
		return fmt.Errorf("metadata.created_at is required")
	}

	if p.SourceSystem.OS == "" {
		return fmt.Errorf("source_system.os is required")
	}

	if p.SourceSystem.Architecture == "" {
		return fmt.Errorf("source_system.architecture is required")
	}

	for index, env := range p.Environment {
		if env.Name == "" {
			return fmt.Errorf("environment[%d].name is required", index)
		}

		if env.Type == "" {
			return fmt.Errorf("environment[%d].type is required", index)
		}

		if env.Classification == "" {
			return fmt.Errorf("environment[%d].classification is required", index)
		}
	}

	for index, software := range p.Software {
		if software.Name == "" {
			return fmt.Errorf("software[%d].name is required", index)
		}

		if software.Classification == "" {
			return fmt.Errorf("software[%d].classification is required", index)
		}
	}

	for index, registry := range p.Registry {
		if registry.Path == "" {
			return fmt.Errorf("registry[%d].path is required", index)
		}

		if registry.Classification == "" {
			return fmt.Errorf("registry[%d].classification is required", index)
		}
	}

	for index, item := range p.Filesystem {
		if item.Path == "" {
			return fmt.Errorf("filesystem[%d].path is required", index)
		}

		if item.Type == "" {
			return fmt.Errorf("filesystem[%d].type is required", index)
		}

		if item.Classification == "" {
			return fmt.Errorf("filesystem[%d].classification is required", index)
		}
	}

	return nil
}
