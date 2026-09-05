package extractor

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/transos/transos/internal/schema"
	"golang.org/x/sys/windows/registry"
)

// ExtractHostEnvironment collects transferable host state into the canonical
// TransOS migration profile.
//
// The extractor records source information. It does not perform Linux
// translation or migration planning.
func ExtractHostEnvironment() *schema.MigrationProfile {
	profile := schema.NewProfile(runtime.GOOS, runtime.GOARCH)

	if hostname, err := os.Hostname(); err == nil {
		profile.SourceSystem.Hostname = hostname
	}

	if username := os.Getenv("USERNAME"); username != "" {
		profile.SourceSystem.Username = username
	}

	extractProcessEnvironment(profile)

	if runtime.GOOS == "windows" {
		extractPersistentUserEnv(profile)
		extractUserShellFolders(profile)
		extractSystemColors(profile)
		extractInstalledSoftware(profile)
	} else {
		profile.Registry = append(profile.Registry, schema.RegistryEntry{
			Hive:           "N/A",
			Path:           "",
			Name:           "Notice",
			Type:           "STRING",
			Value:          "Non-Windows OS detected. Registry extraction skipped.",
			Source:         "extractor",
			Classification: schema.ClassificationUnsupported,
		})
	}

	return profile
}

// extractProcessEnvironment records the current process environment.
//
// Environment variables are intentionally classified conservatively here.
// The normalization stage will perform deeper semantic classification later.
func extractProcessEnvironment(profile *schema.MigrationProfile) {
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}

		name := parts[0]
		value := parts[1]

		profile.Environment = append(profile.Environment, schema.EnvironmentVar{
			Name:           name,
			Value:          value,
			Type:           inferEnvironmentType(name, value),
			Scope:          schema.ScopeUser,
			Source:         "process_environment",
			Classification: schema.ClassificationUnknown,
		})
	}
}

// inferEnvironmentType performs conservative first-pass typing.
//
// This is intentionally not the final normalization algorithm.
func inferEnvironmentType(name, value string) string {
	upperName := strings.ToUpper(name)

	switch {
	case upperName == "PATH",
		upperName == "PATHEXT",
		upperName == "PSMODULEPATH",
		strings.HasSuffix(upperName, "_PATH"):
		return schema.EnvironmentTypePathList

	case strings.HasSuffix(upperName, "HOME"),
		strings.HasSuffix(upperName, "DIR"),
		strings.HasSuffix(upperName, "DIRECTORY"),
		strings.HasSuffix(upperName, "ROOT"):
		return schema.EnvironmentTypeDirectory

	case upperName == "TEMP",
		upperName == "TMP",
		upperName == "TMPDIR":
		return schema.EnvironmentTypeDirectory

	case upperName == "LANG",
		upperName == "LC_ALL",
		strings.HasPrefix(upperName, "LC_"):
		return schema.EnvironmentTypeLocale

	case strings.Contains(strings.ToLower(value), `:\`),
		strings.Contains(value, `/`):
		return schema.EnvironmentTypePath

	default:
		return schema.EnvironmentTypeString
	}
}

// extractPersistentUserEnv extracts persistent user-level environment
// variables from the Windows registry.
func extractPersistentUserEnv(profile *schema.MigrationProfile) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Environment`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return
	}

	for _, name := range names {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}

		profile.Environment = append(profile.Environment, schema.EnvironmentVar{
			Name:           name,
			Value:          value,
			Type:           inferEnvironmentType(name, value),
			Scope:          schema.ScopeUser,
			Source:         `HKCU\Environment`,
			Classification: schema.ClassificationUnknown,
		})
	}
}

// extractUserShellFolders extracts Windows shell-folder mappings.
func extractUserShellFolders(profile *schema.MigrationProfile) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return
	}

	for _, name := range names {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}

		profile.Registry = append(profile.Registry, schema.RegistryEntry{
			Hive:           "HKCU",
			Path:           `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`,
			Name:           name,
			Type:           "STRING",
			Value:          value,
			Source:         "windows_registry",
			Classification: schema.ClassificationConvertible,
		})
	}
}

// extractSystemColors extracts a first-pass representation of Windows theme
// colors.
func extractSystemColors(profile *schema.MigrationProfile) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Control Panel\Colors`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer key.Close()

	colorNames := []string{
		"Window",
		"WindowText",
		"Hilight",
		"ActiveTitle",
	}

	if profile.Theme.Colors == nil {
		profile.Theme.Colors = make(map[string]string)
	}

	for _, name := range colorNames {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}

		profile.Theme.Colors[name] = value
	}

	if len(profile.Theme.Colors) > 0 {
		profile.Theme.Name = "Windows system colors"
		profile.Theme.Mode = "UNKNOWN"
	}
}

// extractInstalledSoftware extracts installed application identities from the
// current user's Windows uninstall registry.
func extractInstalledSoftware(profile *schema.MigrationProfile) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Uninstall`,
		registry.READ,
	)
	if err != nil {
		return
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(0)
	if err != nil {
		return
	}

	for _, subkeyName := range subkeys {
		appKey, err := registry.OpenKey(
			key,
			subkeyName,
			registry.QUERY_VALUE,
		)
		if err != nil {
			continue
		}

		name, _, nameErr := appKey.GetStringValue("DisplayName")
		version, _, versionErr := appKey.GetStringValue("DisplayVersion")
		publisher, _, publisherErr := appKey.GetStringValue("Publisher")
		installLocation, _, locationErr := appKey.GetStringValue("InstallLocation")

		appKey.Close()

		if nameErr != nil || strings.TrimSpace(name) == "" {
			continue
		}

		software := schema.Software{
			Name:            name,
			Source:          fmt.Sprintf("HKCU\\...\\Uninstall\\%s", subkeyName),
			Classification:  schema.ClassificationUnknown,
			MigrationStatus: "DISCOVERED",
		}

		if versionErr == nil {
			software.Version = version
		}

		if publisherErr == nil {
			software.Publisher = publisher
		}

		if locationErr == nil {
			software.InstallLocation = installLocation
		}

		profile.Software = append(profile.Software, software)
	}
}
