package translator

import (
	"fmt"
	"strings"
)

type PackageMap struct {
	AptPackage string
	FlatpakApp string
}

// SoftwareCatalog contains mapped package names across popular Windows developer and desktop software
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
	"spotify":            {FlatpakApp: "com.spotify.Client"},
	"steam":              {AptPackage: "steam", FlatpakApp: "com.valvesoftware.Steam"},
	"gimp":               {AptPackage: "gimp", FlatpakApp: "org.gimp.GIMP"},
	"inkscape":           {AptPackage: "inkscape", FlatpakApp: "org.inkscape.Inkscape"},
	"obs studio":         {AptPackage: "obs-studio", FlatpakApp: "com.obsproject.Studio"},
	"postman":            {FlatpakApp: "com.getpostman.Postman"},
	"notepad++":          {AptPackage: "notepadqq"},
	"filezilla":          {AptPackage: "filezilla"},
	"wireshark":          {AptPackage: "wireshark"},
	"build-essential":    {AptPackage: "build-essential"},
	"cmake":              {AptPackage: "cmake"},
	"htop":               {AptPackage: "htop"},
	"tmux":               {AptPackage: "tmux"},
}

// MapWindowsAppToLinux attempts keyword matching against known Linux software targets
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
