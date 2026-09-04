package translator

import (
	"fmt"
	"sort"
	"strings"
)

type PackageMap struct {
	AptPackage  string
	FlatpakApp  string
	SnapPackage string
}

// SoftwareCatalog contains mapped package names across popular software
var SoftwareCatalog = map[string]PackageMap{
	"visual studio code": {AptPackage: "code", SnapPackage: "code --classic"},
	"vscode":             {AptPackage: "code", SnapPackage: "code --classic"},
	"git":                {AptPackage: "git"},
	"python":             {AptPackage: "python3 python3-pip"},
	"node.js":            {AptPackage: "nodejs npm"},
	"node":               {AptPackage: "nodejs npm"},
	"docker":             {AptPackage: "docker.io docker-compose"},
	"google chrome":      {FlatpakApp: "com.google.Chrome"},
	"chrome":             {FlatpakApp: "com.google.Chrome"},
	"firefox":            {AptPackage: "firefox", FlatpakApp: "org.mozilla.firefox"},
	"brave":              {FlatpakApp: "com.brave.Browser"},
	"vlc":                {AptPackage: "vlc", FlatpakApp: "org.videolan.VLC"},
	"7-zip":              {AptPackage: "p7zip-full"},
	"go programming":     {AptPackage: "golang-go"},
	"golang":             {AptPackage: "golang-go"},
	"neovim":             {AptPackage: "neovim"},
	"vim":                {AptPackage: "vim"},
	"curl":               {AptPackage: "curl"},
	"wget":               {AptPackage: "wget"},
	"powershell":         {AptPackage: "powershell"},
	"discord":            {FlatpakApp: "com.discordapp.Discord"},
	"spotify":            {FlatpakApp: "com.spotify.Client", SnapPackage: "spotify"},
	"steam":              {AptPackage: "steam", FlatpakApp: "com.valvesoftware.Steam"},
	"gimp":               {AptPackage: "gimp", FlatpakApp: "org.gimp.GIMP"},
	"inkscape":           {AptPackage: "inkscape", FlatpakApp: "org.inkscape.Inkscape"},
	"obs studio":         {AptPackage: "obs-studio", FlatpakApp: "com.obsproject.Studio"},
	"postman":            {FlatpakApp: "com.getpostman.Postman", SnapPackage: "postman"},
	"notepad++":          {AptPackage: "notepadqq"},
	"filezilla":          {AptPackage: "filezilla"},
	"wireshark":          {AptPackage: "wireshark"},
	"build-essential":    {AptPackage: "build-essential"},
	"cmake":              {AptPackage: "cmake"},
	"htop":               {AptPackage: "htop"},
	"tmux":               {AptPackage: "tmux"},
}

type AggregatedPackages struct {
	AptPackages map[string]bool
	FlatpakApps map[string]bool
	Unmapped    []string
}

// Aggregates and deduplicates installed Windows apps into unique Linux package targets
func AggregateInstalledSoftware(registry map[string]string) AggregatedPackages {
	agg := AggregatedPackages{
		AptPackages: make(map[string]bool),
		FlatpakApps: make(map[string]bool),
		Unmapped:    []string{},
	}

	for k, winAppName := range registry {
		if !strings.HasPrefix(k, "InstalledApp_") {
			continue
		}

		lower := strings.ToLower(winAppName)
		matched := false

		for key, pkg := range SoftwareCatalog {
			if strings.Contains(lower, key) {
				matched = true
				if pkg.AptPackage != "" {
					// Split multiple packages (e.g. "python3 python3-pip")
					for _, p := range strings.Fields(pkg.AptPackage) {
						agg.AptPackages[p] = true
					}
				} else if pkg.FlatpakApp != "" {
					agg.FlatpakApps[pkg.FlatpakApp] = true
				}
				break
			}
		}

		if !matched {
			agg.Unmapped = append(agg.Unmapped, winAppName)
		}
	}

	return agg
}

// GenerateDependencyScript creates a pre-flight bootstrapped script
func GenerateDependencyScript(agg AggregatedPackages) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# TransOS Automated Dependency & Tool Installer Script\n")
	sb.WriteString("# Deduplicated & Bootstrapped Package Resolution\n\n")

	// Pre-flight setup checks
	sb.WriteString("echo \"[TransOS] Checking pre-requisites and package managers...\"\n")
	if len(agg.FlatpakApps) > 0 {
		sb.WriteString("if command -v flatpak &> /dev/null; then\n")
		sb.WriteString("    echo \"[TransOS] Ensuring Flathub repository is configured...\"\n")
		sb.WriteString("    flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo\n")
		sb.WriteString("fi\n\n")
	}

	// APT Installations (Unified single command)
	if len(agg.AptPackages) > 0 {
		var aptList []string
		for pkg := range agg.AptPackages {
			aptList = append(aptList, pkg)
		}
		sort.Strings(aptList)

		sb.WriteString("echo \"[TransOS] Updating local APT repositories...\"\n")
		sb.WriteString("sudo apt-get update -y\n\n")

		sb.WriteString("echo \"[TransOS] Installing aggregated APT dependencies...\"\n")
		sb.WriteString(fmt.Sprintf("sudo apt-get install -y %s\n\n", strings.Join(aptList, " ")))
	}

	// Flatpak Installations
	if len(agg.FlatpakApps) > 0 {
		var flatList []string
		for app := range agg.FlatpakApps {
			flatList = append(flatList, app)
		}
		sort.Strings(flatList)

		sb.WriteString("echo \"[TransOS] Installing Flatpak applications...\"\n")
		for _, app := range flatList {
			sb.WriteString(fmt.Sprintf("flatpak install -y flathub %s\n", app))
		}
		sb.WriteString("\n")
	}

	// Unmapped applications reference block
	if len(agg.Unmapped) > 0 {
		sort.Strings(agg.Unmapped)
		sb.WriteString("# =========================================================\n")
		sb.WriteString("# Unmapped Windows Applications (Review for manual install):\n")
		for _, app := range agg.Unmapped {
			sb.WriteString(fmt.Sprintf("# - %s\n", app))
		}
		sb.WriteString("# =========================================================\n\n")
	}

	sb.WriteString("echo \"[TransOS] All mapped software dependencies processed successfully!\"\n")
	return sb.String()
}
