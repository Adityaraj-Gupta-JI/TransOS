package normalizer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/transos/transos/internal/schema"
)

func NormalizeProfile(profile *schema.MigrationProfile) (*schema.MigrationProfile, error) {
	if profile == nil {
		return nil, fmt.Errorf("migration profile is nil")
	}

	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("validate input migration profile: %w", err)
	}

	normalized := cloneProfile(profile)

	normalizeMetadata(normalized)
	normalizeEnvironment(normalized)
	normalizeSoftware(normalized)
	normalizeRegistry(normalized)
	normalizeFilesystem(normalized)
	normalizeTheme(normalized)
	normalizeShell(normalized)

	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validate normalized migration profile: %w", err)
	}

	return normalized, nil
}

func cloneProfile(profile *schema.MigrationProfile) *schema.MigrationProfile {
	result := *profile

	result.Environment = append([]schema.EnvironmentVar(nil), profile.Environment...)
	result.Software = append([]schema.Software(nil), profile.Software...)
	result.Registry = append([]schema.RegistryEntry(nil), profile.Registry...)
	result.Filesystem = append([]schema.FilesystemItem(nil), profile.Filesystem...)

	result.Theme.Colors = cloneMap(profile.Theme.Colors)
	result.Theme.Properties = cloneMap(profile.Theme.Properties)

	result.Shell.ProfileFiles = append([]string(nil), profile.Shell.ProfileFiles...)
	result.Shell.Variables = cloneMap(profile.Shell.Variables)
	result.Shell.Properties = cloneMap(profile.Shell.Properties)

	return &result
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}

	return result
}

func normalizeMetadata(profile *schema.MigrationProfile) {
	profile.Metadata.Generator = strings.TrimSpace(profile.Metadata.Generator)
	profile.Metadata.GeneratorVersion = strings.TrimSpace(profile.Metadata.GeneratorVersion)

	profile.SourceSystem.OS = strings.ToLower(strings.TrimSpace(profile.SourceSystem.OS))
	profile.SourceSystem.Version = strings.TrimSpace(profile.SourceSystem.Version)
	profile.SourceSystem.Architecture = strings.ToLower(strings.TrimSpace(profile.SourceSystem.Architecture))
	profile.SourceSystem.Hostname = strings.TrimSpace(profile.SourceSystem.Hostname)
	profile.SourceSystem.Username = strings.TrimSpace(profile.SourceSystem.Username)
}

func normalizeEnvironment(profile *schema.MigrationProfile) {
	deduplicated := make(map[string]schema.EnvironmentVar)

	for _, env := range profile.Environment {
		env.Name = strings.TrimSpace(env.Name)
		env.Value = strings.TrimSpace(env.Value)
		env.Type = strings.ToUpper(strings.TrimSpace(env.Type))
		env.Scope = strings.ToUpper(strings.TrimSpace(env.Scope))
		env.Source = strings.TrimSpace(env.Source)
		env.Classification = strings.ToUpper(strings.TrimSpace(env.Classification))

		if env.Name == "" {
			continue
		}

		key := strings.ToUpper(env.Name)

		// Prefer persistent configuration over the transient
		// process environment when both describe the same variable.
		existing, exists := deduplicated[key]
		if !exists || environmentPriority(env) > environmentPriority(existing) {
			deduplicated[key] = env
		}
	}

	profile.Environment = profile.Environment[:0]

	for _, env := range deduplicated {
		classifyEnvironment(&env)
		profile.Environment = append(profile.Environment, env)
	}

	sort.Slice(profile.Environment, func(i, j int) bool {
		return strings.ToUpper(profile.Environment[i].Name) <
			strings.ToUpper(profile.Environment[j].Name)
	})
}

func environmentPriority(env schema.EnvironmentVar) int {
	source := strings.ToLower(env.Source)

	switch {
	case strings.Contains(source, "hkcu\\environment"):
		return 100
	case strings.Contains(source, "process_environment"):
		return 50
	default:
		return 10
	}
}

func classifyEnvironment(env *schema.EnvironmentVar) {
	switch env.Type {
	case schema.EnvironmentTypePath,
		schema.EnvironmentTypePathList,
		schema.EnvironmentTypeDirectory,
		schema.EnvironmentTypeFile:
		env.Classification = schema.ClassificationConvertible
	default:
		env.Classification = schema.ClassificationNativeEquivalent
	}
}

func normalizeSoftware(profile *schema.MigrationProfile) {
	deduplicated := make(map[string]schema.Software)

	for _, software := range profile.Software {
		software.Name = strings.TrimSpace(software.Name)
		software.Version = strings.TrimSpace(software.Version)
		software.Publisher = strings.TrimSpace(software.Publisher)
		software.Architecture = strings.TrimSpace(software.Architecture)
		software.InstallLocation = strings.TrimSpace(software.InstallLocation)
		software.Scope = strings.ToUpper(strings.TrimSpace(software.Scope))
		software.Source = strings.TrimSpace(software.Source)

		if software.Name == "" {
			continue
		}

		key := softwareIdentity(software)

		existing, exists := deduplicated[key]
		if !exists || softwarePriority(software) > softwarePriority(existing) {
			deduplicated[key] = software
		}
	}

	profile.Software = profile.Software[:0]

	for _, software := range deduplicated {
		classifySoftware(&software)
		profile.Software = append(profile.Software, software)
	}

	sort.Slice(profile.Software, func(i, j int) bool {
		left := strings.ToLower(profile.Software[i].Name)
		right := strings.ToLower(profile.Software[j].Name)

		if left == right {
			return profile.Software[i].Version < profile.Software[j].Version
		}

		return left < right
	})
}

func softwareIdentity(software schema.Software) string {
	return strings.ToLower(strings.TrimSpace(software.Name)) +
		"|" +
		strings.ToLower(strings.TrimSpace(software.Version))
}

func softwarePriority(software schema.Software) int {
	score := 0

	if software.Version != "" {
		score += 20
	}

	if software.Publisher != "" {
		score += 20
	}

	if software.InstallLocation != "" {
		score += 10
	}

	return score
}

func classifySoftware(software *schema.Software) {
	name := strings.ToLower(software.Name)
	publisher := strings.ToLower(software.Publisher)

	switch {
	case strings.Contains(name, "microsoft visual studio code"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "python"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "git"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "docker"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "msys2"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "arduino"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "obsidian"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "zed"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(name, "discord"),
		strings.Contains(name, "zoom"),
		strings.Contains(name, "figma"),
		strings.Contains(name, "notion"),
		strings.Contains(name, "canva"):
		software.Classification = schema.ClassificationNativeEquivalent

	case strings.Contains(publisher, "microsoft"):
		software.Classification = schema.ClassificationConvertible

	default:
		software.Classification = schema.ClassificationUnknown
	}

	software.MigrationStatus = "NORMALIZED"
}

func normalizeRegistry(profile *schema.MigrationProfile) {
	for index := range profile.Registry {
		entry := &profile.Registry[index]

		entry.Hive = strings.ToUpper(strings.TrimSpace(entry.Hive))
		entry.Path = strings.TrimSpace(entry.Path)
		entry.Name = strings.TrimSpace(entry.Name)
		entry.Type = strings.ToUpper(strings.TrimSpace(entry.Type))
		entry.Value = strings.TrimSpace(entry.Value)
		entry.Source = strings.TrimSpace(entry.Source)

		if entry.Classification == "" ||
			entry.Classification == schema.ClassificationUnknown {
			if entry.Value != "" {
				entry.Classification = schema.ClassificationConvertible
			}
		}
	}

	sort.Slice(profile.Registry, func(i, j int) bool {
		left := strings.ToLower(
			profile.Registry[i].Hive + "\\" +
				profile.Registry[i].Path + "\\" +
				profile.Registry[i].Name,
		)

		right := strings.ToLower(
			profile.Registry[j].Hive + "\\" +
				profile.Registry[j].Path + "\\" +
				profile.Registry[j].Name,
		)

		return left < right
	})
}

func normalizeFilesystem(profile *schema.MigrationProfile) {
	for index := range profile.Filesystem {
		item := &profile.Filesystem[index]

		item.Path = strings.TrimSpace(item.Path)
		item.Type = strings.ToUpper(strings.TrimSpace(item.Type))
		item.Scope = strings.ToUpper(strings.TrimSpace(item.Scope))
		item.Source = strings.TrimSpace(item.Source)

		if item.Classification == "" {
			item.Classification = schema.ClassificationUnknown
		}
	}
}

func normalizeTheme(profile *schema.MigrationProfile) {
	profile.Theme.Name = strings.TrimSpace(profile.Theme.Name)
	profile.Theme.Mode = strings.ToUpper(strings.TrimSpace(profile.Theme.Mode))

	if profile.Theme.Colors == nil {
		profile.Theme.Colors = make(map[string]string)
	}

	if profile.Theme.Properties == nil {
		profile.Theme.Properties = make(map[string]string)
	}
}

func normalizeShell(profile *schema.MigrationProfile) {
	profile.Shell.DefaultShell = strings.TrimSpace(profile.Shell.DefaultShell)

	if profile.Shell.ProfileFiles == nil {
		profile.Shell.ProfileFiles = make([]string, 0)
	}

	if profile.Shell.Variables == nil {
		profile.Shell.Variables = make(map[string]string)
	}

	if profile.Shell.Properties == nil {
		profile.Shell.Properties = make(map[string]string)
	}
}
