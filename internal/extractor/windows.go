package extractor

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/transos/transos/internal/config"
	"github.com/transos/transos/internal/schema"
	"golang.org/x/sys/windows/registry"
)

// ExtractHostEnvironment is retained for callers that use the original API.
// New pipeline code should use ExtractHostEnvironmentWithConfig so invalid
// configuration is reported instead of silently ignored.
func ExtractHostEnvironment() *schema.MigrationProfile {
	profile, err := ExtractHostEnvironmentWithConfig(config.Default())
	if err != nil {
		return nil
	}
	return profile
}

// ExtractHostEnvironmentWithConfig collects the selected, supported host state
// into the canonical profile. Depth controls scan breadth; categories control
// which state families are collected.
func ExtractHostEnvironmentWithConfig(input config.Config) (*schema.MigrationProfile, error) {
	cfg, err := config.Normalize(input)
	if err != nil {
		return nil, fmt.Errorf("invalid extraction configuration: %w", err)
	}

	profile := schema.NewProfile(runtime.GOOS, runtime.GOARCH)
	profile.Metadata.OperationMode = cfg.OperationMode
	profile.Metadata.ExtractionDepth = cfg.ExtractionDepth
	profile.Metadata.ExtractionCategories = append([]string(nil), cfg.Categories...)

	if hostname, err := os.Hostname(); err == nil {
		profile.SourceSystem.Hostname = hostname
	}
	if username := os.Getenv("USERNAME"); username != "" {
		profile.SourceSystem.Username = username
	}

	if hasCategory(cfg, config.CategoryEnvironment) {
		extractProcessEnvironment(profile)
		if runtime.GOOS == "windows" && cfg.ExtractionDepth != config.DepthSurface {
			extractPersistentUserEnv(profile)
		}
		if runtime.GOOS == "windows" && cfg.ExtractionDepth == config.DepthDeep {
			extractMachineEnvironment(profile)
		}
	}

	if runtime.GOOS == "windows" {
		if hasCategory(cfg, config.CategoryRegistry) {
			if cfg.ExtractionDepth == config.DepthSurface {
				extractUserShellFolders(profile, config.DepthSurface)
			} else {
				extractUserShellFolders(profile, cfg.ExtractionDepth)
			}
			if cfg.ExtractionDepth == config.DepthDeep {
				extractDesktopSettings(profile)
			}
		}
		if hasCategory(cfg, config.CategoryTheme) {
			extractSystemColors(profile, cfg.ExtractionDepth)
			if cfg.ExtractionDepth == config.DepthDeep {
				extractPersonalizationSettings(profile)
			}
		}
		if hasCategory(cfg, config.CategorySoftware) {
			detailed := cfg.ExtractionDepth != config.DepthSurface
			extractInstalledSoftware(profile, registry.CURRENT_USER, "HKCU", `Software\Microsoft\Windows\CurrentVersion\Uninstall`, detailed)
			if cfg.ExtractionDepth == config.DepthDeep {
				extractInstalledSoftware(profile, registry.LOCAL_MACHINE, "HKLM", `Software\Microsoft\Windows\CurrentVersion\Uninstall`, true)
				extractInstalledSoftware(profile, registry.LOCAL_MACHINE, "HKLM", `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, true)
			}
		}
	} else if hasCategory(cfg, config.CategoryRegistry) {
		profile.Registry = append(profile.Registry, schema.RegistryEntry{
			Hive:           "N/A",
			Path:           "N/A",
			Name:           "Notice",
			Type:           "STRING",
			Value:          "Non-Windows OS detected. Windows Registry extraction skipped.",
			Source:         "extractor",
			Classification: schema.ClassificationUnsupported,
		})
	}

	return profile, nil
}

// hasCategory reports whether the normalized configuration includes a category.
func hasCategory(cfg config.Config, category string) bool {
	for _, selected := range cfg.Categories {
		if selected == category {
			return true
		}
	}
	return false
}

// extractProcessEnvironment records the environment visible to this process.
func extractProcessEnvironment(profile *schema.MigrationProfile) {
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		name, value := parts[0], parts[1]
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

func inferEnvironmentType(name, value string) string {
	upperName := strings.ToUpper(name)
	switch {
	case upperName == "PATH", upperName == "PATHEXT", upperName == "PSMODULEPATH", strings.HasSuffix(upperName, "_PATH"):
		return schema.EnvironmentTypePathList
	case strings.HasSuffix(upperName, "HOME"), strings.HasSuffix(upperName, "DIR"), strings.HasSuffix(upperName, "DIRECTORY"), strings.HasSuffix(upperName, "ROOT"):
		return schema.EnvironmentTypeDirectory
	case upperName == "TEMP", upperName == "TMP", upperName == "TMPDIR":
		return schema.EnvironmentTypeDirectory
	case upperName == "LANG", upperName == "LC_ALL", strings.HasPrefix(upperName, "LC_"):
		return schema.EnvironmentTypeLocale
	case strings.Contains(strings.ToLower(value), `:\`), strings.Contains(value, `/`):
		return schema.EnvironmentTypePath
	default:
		return schema.EnvironmentTypeString
	}
}

func extractPersistentUserEnv(profile *schema.MigrationProfile) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
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

func extractMachineEnvironment(profile *schema.MigrationProfile) {
	const machineEnvironmentPath = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, machineEnvironmentPath, registry.QUERY_VALUE)
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
			Scope:          schema.ScopeSystem,
			Source:         `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
			Classification: schema.ClassificationUnknown,
		})
	}
}

func extractUserShellFolders(profile *schema.MigrationProfile, depth string) {
	const keyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return
	}
	for _, name := range names {
		if depth == config.DepthSurface && !isSurfaceShellFolder(name) {
			continue
		}
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		profile.Registry = append(profile.Registry, schema.RegistryEntry{
			Hive:           "HKCU",
			Path:           keyPath,
			Name:           name,
			Type:           "STRING",
			Value:          value,
			Source:         "windows_registry",
			Classification: schema.ClassificationConvertible,
		})
	}
}

func isSurfaceShellFolder(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "desktop", "personal":
		return true
	default:
		return false
	}
}

func extractDesktopSettings(profile *schema.MigrationProfile) {
	const keyPath = `Control Panel\Desktop`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	for _, name := range []string{"Wallpaper", "WallpaperStyle", "ScreenSaveActive", "ScreenSaveTimeOut", "TileWallpaper"} {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		profile.Registry = append(profile.Registry, schema.RegistryEntry{
			Hive:           "HKCU",
			Path:           keyPath,
			Name:           name,
			Type:           "STRING",
			Value:          value,
			Source:         "windows_registry",
			Classification: schema.ClassificationManual,
		})
	}
}

func extractSystemColors(profile *schema.MigrationProfile, depth string) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Colors`, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	colorNames := []string{"Window", "WindowText"}
	if depth == config.DepthMedium || depth == config.DepthDeep {
		colorNames = append(colorNames, "Hilight", "ActiveTitle")
	}
	if profile.Theme.Colors == nil {
		profile.Theme.Colors = make(map[string]string)
	}
	for _, name := range colorNames {
		value, _, err := key.GetStringValue(name)
		if err == nil {
			profile.Theme.Colors[name] = value
		}
	}
	if len(profile.Theme.Colors) > 0 {
		profile.Theme.Name = "Windows system colors"
		profile.Theme.Mode = "UNKNOWN"
	}
}

func extractPersonalizationSettings(profile *schema.MigrationProfile) {
	const keyPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	if profile.Theme.Properties == nil {
		profile.Theme.Properties = make(map[string]string)
	}
	for _, name := range []string{"AppsUseLightTheme", "SystemUsesLightTheme", "ColorPrevalence", "EnableTransparency"} {
		value, _, err := key.GetIntegerValue(name)
		if err == nil {
			profile.Theme.Properties[name] = strconv.FormatUint(value, 10)
		}
	}
	if value, ok := profile.Theme.Properties["AppsUseLightTheme"]; ok {
		if value == "0" {
			profile.Theme.Mode = "DARK"
		} else if value == "1" {
			profile.Theme.Mode = "LIGHT"
		}
	}
}

// extractInstalledSoftware reads only the identity in surface mode; medium and
// deep modes also capture version, publisher, and install location.
func extractInstalledSoftware(profile *schema.MigrationProfile, root registry.Key, hive, subPath string, detailed bool) {
	key, err := registry.OpenKey(root, subPath, registry.READ)
	if err != nil {
		return
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(0)
	if err != nil {
		return
	}
	for _, subkeyName := range subkeys {
		appKey, err := registry.OpenKey(key, subkeyName, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		name, _, nameErr := appKey.GetStringValue("DisplayName")
		if nameErr != nil || strings.TrimSpace(name) == "" {
			_ = appKey.Close()
			continue
		}

		software := schema.Software{
			Name:            strings.TrimSpace(name),
			Scope:           "USER",
			Source:          fmt.Sprintf("%s\\%s\\%s", hive, subPath, subkeyName),
			Classification:  schema.ClassificationUnknown,
			MigrationStatus: "DISCOVERED",
		}
		if hive == "HKLM" {
			software.Scope = "SYSTEM"
		}
		if detailed {
			if value, _, err := appKey.GetStringValue("DisplayVersion"); err == nil {
				software.Version = strings.TrimSpace(value)
			}
			if value, _, err := appKey.GetStringValue("Publisher"); err == nil {
				software.Publisher = strings.TrimSpace(value)
			}
			if value, _, err := appKey.GetStringValue("InstallLocation"); err == nil {
				software.InstallLocation = strings.TrimSpace(value)
			}
		}
		_ = appKey.Close()
		profile.Software = append(profile.Software, software)
	}
}
