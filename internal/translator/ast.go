package translator

import (
	"strings"
)

// PathNode represents a parsed component AST of a path string
type PathNode struct {
	Drive      string
	Segments   []string
	IsAbsolute bool
	HasEnvVar  bool
}

// ParseWinPath parses a Windows path string into an AST PathNode
func ParseWinPath(winPath string) *PathNode {
	// Normalize backslashes to forward slashes for segment parsing
	normalized := strings.ReplaceAll(winPath, "\\", "/")

	node := &PathNode{
		Segments: []string{},
	}

	// Extract drive letter (e.g., "C:")
	if len(normalized) >= 2 && normalized[1] == ':' {
		node.Drive = strings.ToUpper(normalized[:2])
		normalized = normalized[2:]
		node.IsAbsolute = true
	}

	if strings.HasPrefix(normalized, "/") {
		node.IsAbsolute = true
		normalized = strings.TrimPrefix(normalized, "/")
	}

	// Split path into individual AST segment nodes
	rawSegments := strings.Split(normalized, "/")
	for _, seg := range rawSegments {
		if seg != "" {
			if strings.HasPrefix(seg, "%") && strings.HasSuffix(seg, "%") {
				node.HasEnvVar = true
			}
			node.Segments = append(node.Segments, seg)
		}
	}

	return node
}

// TranslateToPOSIX evaluates the AST and converts it into a POSIX-compliant Linux path
func (node *PathNode) TranslateToPOSIX() string {
	if len(node.Segments) == 0 {
		return "/"
	}

	var posixSegments []string

	for i, seg := range node.Segments {
		upperSeg := strings.ToUpper(seg)
		switch upperSeg {
		case "%USERPROFILE%", "%HOMEPATH%":
			posixSegments = append(posixSegments, "~")
		case "%APPDATA%":
			posixSegments = append(posixSegments, "~/.config")
		case "%LOCALAPPDATA%":
			posixSegments = append(posixSegments, "~/.local/share")
		case "%TEMP%", "%TMP%":
			posixSegments = append(posixSegments, "/tmp")
		default:
			// Handle C:\Users\<Username> pattern
			if i == 0 && upperSeg == "USERS" && len(node.Segments) > 1 {
				continue
			}
			if i == 1 && len(node.Segments) > 1 && strings.ToUpper(node.Segments[0]) == "USERS" {
				posixSegments = append(posixSegments, "~")
				continue
			}
			posixSegments = append(posixSegments, seg)
		}
	}

	result := strings.Join(posixSegments, "/")
	if !strings.HasPrefix(result, "~") && !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	return result
}

// TranslatePathString converts raw path strings or multi-path variables (separated by ;) to POSIX format (separated by :)
func TranslatePathString(rawPath string) string {
	// Handle multi-path lists (e.g. Windows PATH separated by semicolon)
	if strings.Contains(rawPath, ";") {
		paths := strings.Split(rawPath, ";")
		translated := make([]string, 0, len(paths))
		for _, p := range paths {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				node := ParseWinPath(trimmed)
				translated = append(translated, node.TranslateToPOSIX())
			}
		}
		return strings.Join(translated, ":") // Join using Linux PATH separator ':'
	}

	node := ParseWinPath(rawPath)
	return node.TranslateToPOSIX()
}