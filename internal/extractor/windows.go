package extractor

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/transos/transos/internal/schema"
	"golang.org/x/sys/windows/registry"
)

// ExtractHostEnvironment captures real environment variables, Registry keys, and installed software
func ExtractHostEnvironment() *schema.ConfigProfile {
	profile := schema.NewProfile(fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH))

	// 1. Harvest Process Environment Variables
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			profile.Environment[pair[0]] = pair[1]
		}
	}

	// 2. Query Windows Registry (only when running on Windows host)
	if runtime.GOOS == "windows" {
		extractPersistentUserEnv(profile)
		extractUserShellFolders(profile)
		extractSystemColors(profile)
		extractInstalledSoftware(profile)
	} else {
		profile.Registry["Notice"] = "Non-Windows OS detected. Registry extraction skipped."
	}

	return profile
}

// extractPersistentUserEnv reads persistent user variables from HKCU\Environment
func extractPersistentUserEnv(profile *schema.ConfigProfile) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return
	}

	for _, name := range names {
		val, _, err := k.GetStringValue(name)
		if err == nil {
			profile.Registry["HKCU\\Environment\\"+name] = val
		}
	}
}

// extractUserShellFolders reads special directory locations (AppData, Desktop, Documents)
func extractUserShellFolders(profile *schema.ConfigProfile) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()

	folders := []string{"AppData", "Local AppData", "Personal", "Desktop"}
	for _, folder := range folders {
		val, _, err := k.GetStringValue(folder)
		if err == nil {
			profile.Registry["HKCU\\UserShellFolders\\"+folder] = val
		}
	}
}

// extractSystemColors reads system UI color choices
func extractSystemColors(profile *schema.ConfigProfile) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Colors`, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()

	colorKeys := []string{"Window", "WindowText", "Hilight", "ActiveTitle"}
	for _, ck := range colorKeys {
		val, _, err := k.GetStringValue(ck)
		if err == nil {
			profile.ThemeConfig["Color_"+ck] = val
		}
	}
}

// extractInstalledSoftware scans uninstall registry keys to generate a package list for migration
func extractInstalledSoftware(profile *schema.ConfigProfile) {
	uninstallPath := `Software\Microsoft\Windows\CurrentVersion\Uninstall`
	k, err := registry.OpenKey(registry.CURRENT_USER, uninstallPath, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(0)
	if err != nil {
		return
	}

	appCount := 0
	for _, subkey := range subkeys {
		sk, err := registry.OpenKey(registry.CURRENT_USER, uninstallPath+`\`+subkey, registry.READ)
		if err != nil {
			continue
		}

		appName, _, err := sk.GetStringValue("DisplayName")
		sk.Close()

		if err == nil && appName != "" {
			profile.Registry[fmt.Sprintf("InstalledApp_%d", appCount)] = appName
			appCount++
		}
	}
}
