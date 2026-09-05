package translator

import (
	"regexp"
	"strings"
)

var windowsEnvironmentVariablePattern = regexp.MustCompile(`%([A-Za-z_][A-Za-z0-9_]*)%`)

// PathNode is the parsed representation of a Windows path.
//
// The structure intentionally preserves the original path and environment
// variables instead of immediately destroying Windows-specific semantics.
type PathNode struct {
	Original string

	Drive      string
	Segments   []string
	IsAbsolute bool
	IsUNC      bool

	HasEnvVar bool
	EnvVars   []string
}

// ParseWinPath parses a Windows path without assuming that every Windows
// location has a meaningful Linux filesystem equivalent.
func ParseWinPath(path string) PathNode {
	original := strings.TrimSpace(path)

	node := PathNode{
		Original: original,
		Segments: make([]string, 0),
		EnvVars:  make([]string, 0),
	}

	if original == "" {
		return node
	}

	normalized := strings.ReplaceAll(original, "\\", "/")

	// UNC path:
	// \\server\share\folder
	if strings.HasPrefix(normalized, "//") {
		node.IsUNC = true
		node.IsAbsolute = true
		normalized = strings.TrimLeft(normalized, "/")
	} else if len(normalized) >= 2 &&
		((normalized[0] >= 'A' && normalized[0] <= 'Z') ||
			(normalized[0] >= 'a' && normalized[0] <= 'z')) &&
		normalized[1] == ':' {
		node.Drive = strings.ToUpper(string(normalized[0]))
		normalized = normalized[2:]

		if strings.HasPrefix(normalized, "/") {
			node.IsAbsolute = true
			normalized = strings.TrimLeft(normalized, "/")
		}
	} else if strings.HasPrefix(normalized, "/") {
		node.IsAbsolute = true
		normalized = strings.TrimLeft(normalized, "/")
	}

	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." {
			continue
		}

		if segment == ".." {
			if len(node.Segments) > 0 &&
				node.Segments[len(node.Segments)-1] != ".." {
				node.Segments = node.Segments[:len(node.Segments)-1]
				continue
			}
		}

		node.Segments = append(node.Segments, segment)
	}

	matches := windowsEnvironmentVariablePattern.FindAllStringSubmatch(
		original,
		-1,
	)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		node.HasEnvVar = true
		node.EnvVars = appendUniqueString(node.EnvVars, match[1])
	}

	return node
}

// TranslateToPOSIX provides a conservative textual translation for legacy
// callers. It is intentionally not the authoritative migration decision.
//
// Semantic migration decisions belong to AnalyzePath in path_semantics.go.
func TranslateToPOSIX(node PathNode) string {
	if node.Original == "" {
		return ""
	}

	translated := node.Original

	replacements := map[string]string{
		"%USERPROFILE%":  "$HOME",
		"%HOMEPATH%":     "$HOME",
		"%APPDATA%":      "$HOME/.config",
		"%LOCALAPPDATA%": "$HOME/.local/share",
		"%TEMP%":         "/tmp",
		"%TMP%":          "/tmp",
	}

	for from, to := range replacements {
		translated = strings.ReplaceAll(
			translated,
			from,
			to,
		)
	}

	translated = strings.ReplaceAll(translated, "\\", "/")

	// Preserve UNC paths as UNC-like POSIX paths rather than pretending that
	// they are local filesystem locations.
	if strings.HasPrefix(translated, "//") {
		return translated
	}

	// Windows drive paths are represented under /mnt/<drive> for compatibility
	// with existing callers. This is NOT a statement that the drive will exist
	// on the target Linux system.
	if len(translated) >= 2 &&
		((translated[0] >= 'A' && translated[0] <= 'Z') ||
			(translated[0] >= 'a' && translated[0] <= 'z')) &&
		translated[1] == ':' {
		drive := strings.ToLower(string(translated[0]))
		remainder := strings.TrimLeft(translated[2:], "/")

		if remainder == "" {
			return "/mnt/" + drive
		}

		return "/mnt/" + drive + "/" + remainder
	}

	if strings.HasPrefix(translated, "/") {
		return translated
	}

	// Shell-variable-based paths are already valid target expressions.
	// Do not turn "$HOME/..." into "/$HOME/...".
	if strings.HasPrefix(translated, "$") {
		return translated
	}

	if strings.HasPrefix(translated, "~") {
		return translated
	}

	return "/" + strings.TrimLeft(translated, "/")
}

// TranslatePathString translates a Windows PATH-like string while preserving
// the distinction between individual path entries.
//
// Windows uses ';' as its conventional path-list separator.
func TranslatePathString(value string) string {
	if value == "" {
		return ""
	}

	entries := strings.Split(value, ";")
	translated := make([]string, 0, len(entries))
	seen := make(map[string]struct{})

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		node := ParseWinPath(entry)
		result := TranslateToPOSIX(node)

		if result == "" {
			continue
		}

		if _, exists := seen[result]; exists {
			continue
		}

		seen[result] = struct{}{}
		translated = append(translated, result)
	}

	return strings.Join(translated, ":")
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}

	return append(values, value)
}
