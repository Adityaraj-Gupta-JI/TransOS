package translator

import (
	"fmt"
	"strings"
)

// PackageMap holds Linux package identifiers for cross-platform app mapping
type PackageMap struct {
	AptPackage string
	FlatpakApp string
}

// SoftwareCatalog maps Windows app name keywords to Linux package managers
var SoftwareCatalog = map[string]PackageMap{
	"visual studio code": {AptPackage: "code", FlatpakApp: "com.visualstudio.code"},
	"vscode":             {AptPackage: "code", FlatpakApp: "com.visualstudio.code"},
	"git":                {AptPackage: "git"},
	"python":             {AptPackage: "python3 python3-pip"},
	"node.js":            {AptPackage: "nodejs npm"},
	"node":               {AptPackage: "nodejs npm"},
	"docker":             {AptPackage: "docker.io docker-compose"},
	"google chrome":      {FlatpakApp: "com.google.Chrome"},
	"chrome":             {FlatpakApp: "com.google.Chrome"},
	"firefox":            {AptPackage: "firefox", FlatpakApp: "org.mozilla.firefox"},
	"vlc":                {AptPackage: "vlc", FlatpakApp: "org.videolan.VLC"},
	"7-zip":              {AptPackage: "p7zip-full"},
	"go programming":     {AptPackage: "golang-go"},
	"golang":             {AptPackage: "golang-go"},
	"neovim":             {AptPackage: "neovim"},
	"curl":               {AptPackage: "curl"},
	"powershell":         {AptPackage: "powershell"},
}

// MapWindowsAppToLinux matches a Windows display name against the catalog
func MapWindowsAppToLinux(winAppName string) (string, bool) {
	lower := strings.ToLower(winAppName)
	for key, pkg := range SoftwareCatalog {
		if strings.Contains(lower, key) {
			if pkg.AptPackage != "" {
				return fmt.Sprintf("sudo apt-get install -y %s", pkg.AptPackage), true
			} else if pkg.FlatpakApp != "" {
				return fmt.Sprintf("flatpak install -y flathub %s", pkg.FlatpakApp), true
			}
		}
	}
	return "", false
}