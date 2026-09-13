package translator

import (
	"sort"
	"strings"

	"github.com/transos/transos/internal/schema"
)

type softwarePlan struct {
	Identity     string
	Strategy     string
	InstallBlock string
	Reason       string
}

func GenerateDependencyScript(software []schema.Software) string {
	plans := buildSoftwarePlans(software)

	var builder strings.Builder

	builder.WriteString(`#!/usr/bin/env bash
set -u

# ============================================================
# TransOS Linux Migration Package
# Generated from Windows migration profile
# ============================================================

set -o pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

MIGRATED=()
ALTERNATIVES=()
MANUAL=()
UNSUPPORTED=()
FAILED=()

log() {
    echo "[TransOS] $*"
}

ok() {
    echo "[TransOS][OK] $*"
}

warn() {
    echo "[TransOS][WARN] $*"
}

fail() {
    echo "[TransOS][FAIL] $*"
}

has_command() {
    command -v "$1" >/dev/null 2>&1
}

echo
echo "============================================================"
echo "                 TransOS Linux Migration"
echo "============================================================"
echo

log "Detecting Linux platform..."

DISTRO_ID="unknown"
if [ -f /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    DISTRO_ID="${ID:-unknown}"
fi

ARCH="$(uname -m)"

echo "[TransOS] Target OS : ${DISTRO_ID}"
echo "[TransOS] Target CPU: ${ARCH}"
echo

# ------------------------------------------------------------
# Environment configuration
# ------------------------------------------------------------

ENV_SOURCE="${SCRIPT_DIR}/transos_env.conf"
ENV_TARGET="${HOME}/.config/transos/transos_env.conf"

if [ -f "$ENV_SOURCE" ]; then
    mkdir -p "$(dirname "$ENV_TARGET")"

    if cp "$ENV_SOURCE" "$ENV_TARGET"; then
        ok "Translated environment configuration installed"
    else
        warn "Unable to install translated environment configuration"
        FAILED+=("environment configuration")
    fi
else
    warn "transos_env.conf not found beside installer; environment migration skipped"
fi

# ------------------------------------------------------------
# Shell hooks
# ------------------------------------------------------------

install_shell_hook() {
    local profile="$1"
    local marker="# >>> TransOS environment >>>"
    local end="# <<< TransOS environment <<<"
    local source_line='if [ -f "$HOME/.config/transos/transos_env.conf" ]; then . "$HOME/.config/transos/transos_env.conf"; fi'

    touch "$profile"

    if ! grep -Fq "$marker" "$profile"; then
        {
            echo
            echo "$marker"
            echo "$source_line"
            echo "$end"
        } >> "$profile"

        ok "Installed TransOS shell hook: $profile"
    else
        ok "TransOS shell hook already present: $profile"
    fi
}

install_shell_hook "$HOME/.bashrc"

if [ -n "${ZSH_VERSION:-}" ] || has_command zsh; then
    install_shell_hook "$HOME/.zshrc"
fi

# ------------------------------------------------------------
# Package manager detection
# ------------------------------------------------------------

APT_AVAILABLE=0
DNF_AVAILABLE=0
FLATPAK_AVAILABLE=0

if has_command apt-get; then
    APT_AVAILABLE=1
fi

if has_command dnf; then
    DNF_AVAILABLE=1
fi

if has_command flatpak; then
    FLATPAK_AVAILABLE=1
fi

echo
echo "[TransOS] Package manager availability:"
echo "  apt-get : ${APT_AVAILABLE}"
echo "  dnf     : ${DNF_AVAILABLE}"
echo "  flatpak : ${FLATPAK_AVAILABLE}"
echo

`)

	// Build the native Linux package lists in Go.
	aptPackages := make([]string, 0)
	dnfPackages := make([]string, 0)

	for _, plan := range plans {
		if plan.InstallBlock != "__APT__" {
			continue
		}

		aptPackage := aptPackageForIdentity(plan.Identity)
		if aptPackage != "" {
			aptPackages = append(aptPackages, aptPackage)
		}

		dnfPackage := dnfPackageForIdentity(plan.Identity)
		if dnfPackage != "" {
			dnfPackages = append(dnfPackages, dnfPackage)
		}
	}

	aptPackages = uniqueStrings(aptPackages)
	dnfPackages = uniqueStrings(dnfPackages)

	sort.Strings(aptPackages)
	sort.Strings(dnfPackages)

	// --------------------------------------------------------
	// APT / DNF native packages
	// --------------------------------------------------------

	if len(aptPackages) > 0 {
		builder.WriteString("if [ \"$APT_AVAILABLE\" -eq 1 ]; then\n")
		builder.WriteString("    log \"Updating APT package metadata...\"\n")
		builder.WriteString("    if sudo apt-get update -y; then\n")
		builder.WriteString("        ok \"APT metadata updated\"\n")
		builder.WriteString("    else\n")
		builder.WriteString("        warn \"APT metadata update failed\"\n")
		builder.WriteString("    fi\n\n")

		builder.WriteString("    log \"Installing native APT dependencies...\"\n")
		builder.WriteString("    if sudo apt-get install -y")

		for _, packageName := range aptPackages {
			builder.WriteString(" ")
			builder.WriteString(packageName)
		}

		builder.WriteString("; then\n")
		builder.WriteString("        ok \"Native APT dependencies installed\"\n")
		builder.WriteString("    else\n")
		builder.WriteString("        fail \"Native APT dependency installation\"\n")
		builder.WriteString("        FAILED+=(\"native APT dependencies\")\n")
		builder.WriteString("    fi\n")

		if len(dnfPackages) > 0 {
			builder.WriteString("elif [ \"$DNF_AVAILABLE\" -eq 1 ]; then\n")
			builder.WriteString("    log \"Installing native DNF dependencies...\"\n")
			builder.WriteString("    if sudo dnf install -y")

			for _, packageName := range dnfPackages {
				builder.WriteString(" ")
				builder.WriteString(packageName)
			}

			builder.WriteString("; then\n")
			builder.WriteString("        ok \"Native DNF dependencies installed\"\n")
			builder.WriteString("    else\n")
			builder.WriteString("        fail \"Native DNF dependency installation\"\n")
			builder.WriteString("        FAILED+=(\"native DNF dependencies\")\n")
			builder.WriteString("    fi\n")
		}

		builder.WriteString("else\n")
		builder.WriteString("    warn \"No supported native package manager found\"\n")
		builder.WriteString("fi\n\n")
	}

	// --------------------------------------------------------
	// Application-specific migration actions
	// --------------------------------------------------------

	for _, plan := range plans {
		switch plan.InstallBlock {
		case "__APT__":
			builder.WriteString("MIGRATED+=(\"")
			builder.WriteString(shellScriptQuote(plan.Identity))
			builder.WriteString("\")\n")

		case "__FLATPAK_DISCORD__":
			builder.WriteString(`if has_command flatpak; then
    log "Installing Discord via Flatpak..."

    if flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo &&
       flatpak install -y flathub com.discordapp.Discord; then
        ok "Discord installed"
        MIGRATED+=("discord")
    else
        warn "Discord installation failed"
        FAILED+=("discord")
    fi
else
    warn "Flatpak is unavailable; Discord requires manual installation"
    MANUAL+=("Discord")
fi

`)

		case "__ZED__":
			builder.WriteString(`if has_command curl; then
    log "Installing Zed using the official Linux installer..."

    if curl -f https://zed.dev/install.sh | sh; then
        ok "Zed installed"
        MIGRATED+=("zed")
    else
        warn "Zed installation failed"
        FAILED+=("zed")
    fi
else
    warn "curl is unavailable; Zed requires manual installation"
    MANUAL+=("Zed")
fi

`)

		case "__OLLAMA__":
			builder.WriteString(`if has_command curl; then
    log "Installing Ollama using the official Linux installer..."

    if curl -fsSL https://ollama.com/install.sh | sh; then
        ok "Ollama installed"
        MIGRATED+=("ollama")
    else
        warn "Ollama installation failed"
        FAILED+=("ollama")
    fi
else
    warn "curl is unavailable; Ollama requires manual installation"
    MANUAL+=("Ollama")
fi

`)

		case "__ALTERNATIVE__":
			builder.WriteString("ALTERNATIVES+=(\"")
			builder.WriteString(shellScriptQuote(plan.Reason))
			builder.WriteString("\")\n")

		case "__MANUAL__":
			builder.WriteString("MANUAL+=(\"")
			builder.WriteString(shellScriptQuote(plan.Reason))
			builder.WriteString("\")\n")

		case "__UNSUPPORTED__":
			builder.WriteString("UNSUPPORTED+=(\"")
			builder.WriteString(shellScriptQuote(plan.Reason))
			builder.WriteString("\")\n")
		}
	}

	builder.WriteString(`
echo
echo "============================================================"
echo "                 TransOS Migration Report"
echo "============================================================"

echo
echo "Migrated / native equivalents:"
if [ "${#MIGRATED[@]}" -eq 0 ]; then
    echo "  - None"
else
    printf '  - %s\n' "${MIGRATED[@]}"
fi

echo
echo "Alternatives:"
if [ "${#ALTERNATIVES[@]}" -eq 0 ]; then
    echo "  - None"
else
    printf '  - %s\n' "${ALTERNATIVES[@]}"
fi

echo
echo "Manual action required:"
if [ "${#MANUAL[@]}" -eq 0 ]; then
    echo "  - None"
else
    printf '  - %s\n' "${MANUAL[@]}"
fi

echo
echo "Unsupported / Windows-specific:"
if [ "${#UNSUPPORTED[@]}" -eq 0 ]; then
    echo "  - None"
else
    printf '  - %s\n' "${UNSUPPORTED[@]}"
fi

echo
echo "Installation failures:"
if [ "${#FAILED[@]}" -eq 0 ]; then
    echo "  - None"
else
    printf '  - %s\n' "${FAILED[@]}"
fi

echo
echo "Translated environment:"
if [ -f "$ENV_TARGET" ]; then
    echo "  $ENV_TARGET"
else
    echo "  Not installed"
fi

echo
echo "Shell profiles:"
echo "  $HOME/.bashrc"
if [ -f "$HOME/.zshrc" ]; then
    echo "  $HOME/.zshrc"
fi

echo
echo "TransOS migration processing complete."
echo
`)

	return builder.String()
}

func buildSoftwarePlans(software []schema.Software) []softwarePlan {
	seen := make(map[string]bool)
	plans := make([]softwarePlan, 0, len(software))

	for _, item := range software {
		identity := identifySoftware(item.Name)

		if identity == "" {
			continue
		}

		if seen[identity] {
			continue
		}

		seen[identity] = true
		plans = append(plans, planForIdentity(identity, item.Name))
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Identity < plans[j].Identity
	})

	return plans
}

func identifySoftware(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))

	switch {
	case strings.Contains(n, "visual studio code"),
		strings.HasPrefix(n, "vs code"):
		return "vscode"

	case strings.HasPrefix(n, "python "):
		return "python"

	case strings.HasPrefix(n, "discord"):
		return "discord"

	case strings.HasPrefix(n, "zed"):
		return "zed"

	case strings.HasPrefix(n, "ollama"):
		return "ollama"

	case strings.HasPrefix(n, "aria2"):
		return "aria2"

	case strings.HasPrefix(n, "obsidian"):
		return "obsidian"

	case strings.HasPrefix(n, "arduino ide"):
		return "arduino"

	case strings.HasPrefix(n, "bitwarden"):
		return "bitwarden"

	case strings.HasPrefix(n, "figma"):
		return "figma"

	case strings.HasPrefix(n, "canva"):
		return "canva"

	case strings.HasPrefix(n, "notion"):
		return "notion"

	case strings.HasPrefix(n, "linear"):
		return "linear"

	case strings.HasPrefix(n, "pitch"):
		return "pitch"

	case strings.HasPrefix(n, "perplexity"):
		return "perplexity"

	case strings.HasPrefix(n, "proton mail"):
		return "proton-mail"

	case strings.HasPrefix(n, "msys2"):
		return "msys2"

	case strings.HasPrefix(n, "sourcetree"):
		return "sourcetree"

	case strings.HasPrefix(n, "superf4"):
		return "superf4"

	case strings.HasPrefix(n, "easyeda"):
		return "easyeda"

	case strings.HasPrefix(n, "lm studio"):
		return "lmstudio"

	case strings.HasPrefix(n, "edex-ui"):
		return "edex"

	case strings.HasPrefix(n, "marktext"):
		return "marktext"

	case strings.HasPrefix(n, "qt"):
		return "qt"

	case strings.HasPrefix(n, "binary ninja"):
		return "binary-ninja"

	case strings.HasPrefix(n, "modelio"):
		return "modelio"

	case strings.HasPrefix(n, "openstego"):
		return "openstego"

	case strings.HasPrefix(n, "zoom workplace"):
		return "zoom"

	case strings.HasPrefix(n, "nvidia isaac sim webrtc"):
		return "isaac-sim"

	case strings.HasPrefix(n, "antigravity"):
		return "antigravity"

	default:
		return ""
	}
}

func planForIdentity(identity string, originalName string) softwarePlan {
	switch identity {
	case "vscode":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__APT__",
			Reason:       "Visual Studio Code -> native Linux package",
		}

	case "python":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__APT__",
			Reason:       originalName + " -> native Python 3 runtime",
		}

	case "aria2":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__APT__",
			Reason:       "aria2 -> native Linux package",
		}

	case "discord":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__FLATPAK_DISCORD__",
			Reason:       "Discord -> Linux Flatpak",
		}

	case "zed":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__ZED__",
			Reason:       "Zed -> official Linux installer",
		}

	case "ollama":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "NATIVE_REINSTALL",
			InstallBlock: "__OLLAMA__",
			Reason:       "Ollama -> official Linux installer",
		}

	case "obsidian":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "ALTERNATIVE",
			InstallBlock: "__ALTERNATIVE__",
			Reason:       "Obsidian -> target-specific Linux installation",
		}

	case "arduino":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       originalName + " -> manual Linux installation",
		}

	case "binary-ninja":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       "Binary Ninja -> manual licensed application installation",
		}

	case "bitwarden",
		"figma",
		"canva",
		"notion",
		"linear",
		"pitch",
		"perplexity",
		"proton-mail":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "ALTERNATIVE",
			InstallBlock: "__ALTERNATIVE__",
			Reason:       originalName + " -> web/PWA or Linux-compatible alternative",
		}

	case "msys2":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "ALTERNATIVE",
			InstallBlock: "__ALTERNATIVE__",
			Reason:       "MSYS2 -> native Linux toolchain",
		}

	case "sourcetree":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "ALTERNATIVE",
			InstallBlock: "__ALTERNATIVE__",
			Reason:       "SourceTree -> native Git GUI/CLI alternative",
		}

	case "qt":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       "Qt -> development framework; install when required",
		}

	case "lmstudio":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       "LM Studio -> target-specific Linux installation",
		}

	case "easyeda",
		"modelio",
		"openstego",
		"zoom",
		"isaac-sim",
		"antigravity",
		"edex",
		"marktext":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       originalName + " -> manual Linux/compatibility review",
		}

	case "superf4":
		return softwarePlan{
			Identity:     identity,
			Strategy:     "UNSUPPORTED",
			InstallBlock: "__UNSUPPORTED__",
			Reason:       "SuperF4 -> Windows-specific utility; no direct Linux equivalent",
		}

	default:
		return softwarePlan{
			Identity:     identity,
			Strategy:     "MANUAL_INSTALL",
			InstallBlock: "__MANUAL__",
			Reason:       originalName + " -> manual review required",
		}
	}
}

func aptPackageForIdentity(identity string) string {
	switch identity {
	case "vscode":
		return "code"
	case "python":
		return "python3"
	case "aria2":
		return "aria2"
	default:
		return ""
	}
}

func dnfPackageForIdentity(identity string) string {
	switch identity {
	case "vscode":
		return "code"
	case "python":
		return "python3"
	case "aria2":
		return "aria2"
	default:
		return ""
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)
	}

	return result
}

func shellScriptQuote(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
