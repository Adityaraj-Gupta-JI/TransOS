package translator

import (
	"path/filepath"
	"strings"
)

// PathKind describes the semantic role of a Windows path.
type PathKind string

const (
	PathKindUnknown            PathKind = "UNKNOWN"
	PathKindUserHome           PathKind = "USER_HOME"
	PathKindUserConfig         PathKind = "USER_CONFIG"
	PathKindUserData           PathKind = "USER_DATA"
	PathKindUserCache          PathKind = "USER_CACHE"
	PathKindUserDesktop        PathKind = "USER_DESKTOP"
	PathKindUserDocuments      PathKind = "USER_DOCUMENTS"
	PathKindUserDownloads      PathKind = "USER_DOWNLOADS"
	PathKindUserPictures       PathKind = "USER_PICTURES"
	PathKindUserMusic          PathKind = "USER_MUSIC"
	PathKindUserVideos         PathKind = "USER_VIDEOS"
	PathKindTemporary          PathKind = "TEMPORARY"
	PathKindDeveloperTool      PathKind = "DEVELOPER_TOOL"
	PathKindApplicationInstall PathKind = "APPLICATION_INSTALL"
	PathKindWindowsSystem      PathKind = "WINDOWS_SYSTEM"
	PathKindWindowsRuntime     PathKind = "WINDOWS_RUNTIME"
	PathKindNetworkShare       PathKind = "NETWORK_SHARE"
)

// PathStrategy describes what TransOS should eventually do with a path.
type PathStrategy string

const (
	PathStrategyNativeEquivalent PathStrategy = "NATIVE_EQUIVALENT"
	PathStrategyConvertible      PathStrategy = "CONVERTIBLE"
	PathStrategyPreserve         PathStrategy = "PRESERVE"
	PathStrategyManual           PathStrategy = "MANUAL"
	PathStrategyUnsupported      PathStrategy = "UNSUPPORTED"
	PathStrategyIgnore           PathStrategy = "IGNORE"
)

// PathConfidence expresses how confident TransOS is in the semantic mapping.
type PathConfidence string

const (
	PathConfidenceHigh   PathConfidence = "HIGH"
	PathConfidenceMedium PathConfidence = "MEDIUM"
	PathConfidenceLow    PathConfidence = "LOW"
	PathConfidenceNone   PathConfidence = "NONE"
)

// SemanticPath is the canonical semantic representation of a path.
type SemanticPath struct {
	Original string

	Kind       PathKind
	Strategy   PathStrategy
	Confidence PathConfidence

	WindowsPath string
	LinuxPath   string

	Drive      string
	Segments   []string
	EnvVars    []string
	IsAbsolute bool
	IsUNC      bool

	Reason string
}

// AnalyzePath parses and classifies a Windows path without assuming that
// arbitrary Windows filesystem locations can be copied to Linux.
func AnalyzePath(path string) SemanticPath {
	node := ParseWinPath(path)

	result := SemanticPath{
		Original:    node.Original,
		Kind:        PathKindUnknown,
		Strategy:    PathStrategyManual,
		Confidence:  PathConfidenceLow,
		WindowsPath: node.Original,
		Drive:       node.Drive,
		Segments:    append([]string(nil), node.Segments...),
		EnvVars:     append([]string(nil), node.EnvVars...),
		IsAbsolute:  node.IsAbsolute,
		IsUNC:       node.IsUNC,
	}

	if node.Original == "" {
		result.Strategy = PathStrategyIgnore
		result.Confidence = PathConfidenceNone
		result.Reason = "empty path"
		return result
	}

	if node.IsUNC {
		result.Kind = PathKindNetworkShare
		result.Strategy = PathStrategyManual
		result.Confidence = PathConfidenceMedium
		result.Reason = "UNC/network paths require target-side network mapping"
		return result
	}

	normalized := strings.ToLower(
		strings.ReplaceAll(node.Original, "/", "\\"),
	)

	switch {
	case isWindowsTempPath(normalized):
		result.Kind = PathKindTemporary
		result.Strategy = PathStrategyNativeEquivalent
		result.Confidence = PathConfidenceHigh
		result.LinuxPath = "/tmp"
		result.Reason = "Windows temporary storage maps to the Linux temporary directory"

	case isAppDataRoamingPath(normalized):
		result.Kind = PathKindUserConfig
		result.Strategy = PathStrategyNativeEquivalent
		result.Confidence = PathConfidenceHigh
		result.LinuxPath = "$HOME/.config"
		result.Reason = "Windows roaming application data maps to XDG-style user configuration"

	case isAppDataLocalPath(normalized):
		result.Kind = PathKindUserData
		result.Strategy = PathStrategyConvertible
		result.Confidence = PathConfidenceHigh
		result.LinuxPath = "$HOME/.local/share"
		result.Reason = "Windows local application data maps broadly to Linux user application data"

	case isUserHomePath(normalized):
		result.Kind = PathKindUserHome
		result.Strategy = PathStrategyNativeEquivalent
		result.Confidence = PathConfidenceHigh
		result.LinuxPath = "$HOME"
		result.Reason = "Windows user profile maps to the Linux user home directory"

	case isWindowsSystemPath(normalized):
		result.Kind = PathKindWindowsSystem
		result.Strategy = PathStrategyIgnore
		result.Confidence = PathConfidenceHigh
		result.Reason = "Windows system directory has no portable Linux equivalent"

	case isWindowsRuntimePath(normalized):
		result.Kind = PathKindWindowsRuntime
		result.Strategy = PathStrategyIgnore
		result.Confidence = PathConfidenceHigh
		result.Reason = "Windows runtime location should not be migrated as a filesystem path"

	case isApplicationInstallPath(normalized):
		result.Kind = PathKindApplicationInstall
		result.Strategy = PathStrategyManual
		result.Confidence = PathConfidenceHigh
		result.Reason = "Windows application installation paths must be resolved by software identity"

	case isDeveloperToolPath(normalized):
		result.Kind = PathKindDeveloperTool
		result.Strategy = PathStrategyConvertible
		result.Confidence = PathConfidenceMedium
		result.Reason = "Developer tooling may have a Linux-native installation strategy"

	default:
		classifyGenericPath(&result)
	}

	return result
}

func isUserHomePath(path string) bool {
	if path == `c:\users` {
		return false
	}

	// Environment variables that represent the user's home directory
	// are only classified as USER_HOME when they are the complete path.
	if path == `%userprofile%` ||
		path == `%homepath%` {
		return true
	}

	// A concrete Windows user profile is the root immediately below
	// C:\Users. Descendants such as AppData must be classified by their
	// more specific semantic rules.
	if !strings.HasPrefix(path, `c:\users\`) {
		return false
	}

	remainder := strings.TrimPrefix(path, `c:\users\`)

	return remainder != "" &&
		!strings.Contains(remainder, `\`)
}

func isAppDataRoamingPath(path string) bool {
	return strings.Contains(path, `\appdata\roaming`) ||
		strings.Contains(path, `%appdata%`)
}

func isAppDataLocalPath(path string) bool {
	return strings.Contains(path, `\appdata\local`) ||
		strings.Contains(path, `%localappdata%`)
}

func isWindowsTempPath(path string) bool {
	return strings.Contains(path, `\appdata\local\temp`) ||
		strings.Contains(path, `%temp%`) ||
		strings.Contains(path, `%tmp%`)
}

func isWindowsSystemPath(path string) bool {
	systemMarkers := []string{
		`\windows\system32`,
		`\windows\syswow64`,
		`\windows\winsxs`,
		`\windows\servicing`,
		`\windows\system`,
	}

	for _, marker := range systemMarkers {
		if strings.Contains(path, marker) {
			return true
		}
	}

	return false
}

func isWindowsRuntimePath(path string) bool {
	runtimeMarkers := []string{
		`%comspec%`,
		`%systemroot%`,
		`%windir%`,
		`\system32\cmd.exe`,
		`\system32\powershell`,
	}

	for _, marker := range runtimeMarkers {
		if strings.Contains(path, marker) {
			return true
		}
	}

	return false
}

func isApplicationInstallPath(path string) bool {
	return strings.Contains(path, `\program files\`) ||
		strings.Contains(path, `\program files (x86)\`) ||
		strings.Contains(path, `\appdata\local\programs\`)
}

func isDeveloperToolPath(path string) bool {
	developerMarkers := []string{
		`\go\bin`,
		`\go`,
		`\flutter`,
		`\python`,
		`\nodejs`,
		`\npm`,
		`\pnpm`,
		`\.dotnet\tools`,
		`\cargo\bin`,
		`\rust`,
		`\jdk`,
		`\java`,
		`\android\sdk`,
		`\android\ndk`,
	}

	for _, marker := range developerMarkers {
		if strings.Contains(path, marker) {
			return true
		}
	}

	return false
}

func classifyGenericPath(result *SemanticPath) {
	if result == nil {
		return
	}

	// Relative paths should be preserved rather than invented as absolute
	// Linux paths.
	if !result.IsAbsolute && result.Drive == "" {
		result.Strategy = PathStrategyPreserve
		result.Confidence = PathConfidenceMedium
		result.Reason = "relative path has no source filesystem root"
		return
	}

	if result.Drive != "" {
		result.Strategy = PathStrategyManual
		result.Confidence = PathConfidenceLow
		result.Reason = "drive-qualified Windows path requires context before migration"
		return
	}

	result.Strategy = PathStrategyManual
	result.Confidence = PathConfidenceLow
	result.Reason = "path semantics could not be determined safely"
}

// SuggestedLinuxPath returns the semantic Linux target when one is known.
//
// It deliberately does not convert arbitrary Windows drive paths into fake
// Linux locations.
func SuggestedLinuxPath(path string) string {
	result := AnalyzePath(path)

	if result.LinuxPath != "" {
		return result.LinuxPath
	}

	return ""
}

// IsPathListSeparator reports whether the supplied value appears to contain
// Windows PATH-list semantics.
func IsPathListSeparator(value string) bool {
	return strings.Contains(value, ";")
}

// NormalizePathForComparison provides a stable representation for comparing
// Windows paths without changing their migration semantics.
func NormalizePathForComparison(path string) string {
	value := strings.TrimSpace(path)
	value = strings.ReplaceAll(value, "/", "\\")
	value = strings.TrimRight(value, `\`)

	return strings.ToLower(filepath.Clean(value))
}
