package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Terminal UI deliberately uses plain ASCII for reliable alignment on Windows
// PowerShell, SSH sessions and terminals without Unicode box-drawing support.
func uiWidth() int {
	width := 78
	if v, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && v > 0 {
		width = v
	}
	if width < 48 {
		width = 48
	}
	if width > 100 {
		width = 100
	}
	return width
}

func uiColorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		return true
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func uiAccent(s string) string {
	if !uiColorEnabled() {
		return s
	}
	return "\x1b[36m" + s + "\x1b[0m"
}

func uiClip(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	if limit <= 3 {
		return strings.Repeat(".", limit)
	}
	return string([]rune(s)[:limit-3]) + "..."
}

func uiHeader(title, subtitle string) {
	width := uiWidth()
	fmt.Println()
	fmt.Println(uiAccent("  " + strings.Repeat("=", width-4)))
	fmt.Printf("  %s\n", uiAccent("TRANSOS  /  "+uiClip(title, width-17)))
	fmt.Printf("  %s\n", uiClip(subtitle, width-4))
	fmt.Println(uiAccent("  " + strings.Repeat("=", width-4)))
}

func uiSection(title string) {
	fmt.Printf("\n  %s\n", uiAccent(title))
	fmt.Println("  " + strings.Repeat("-", uiWidth()-4))
}

func uiKeyValue(key, value string) {
	fmt.Printf("  %-19s %s\n", key, uiClip(value, uiWidth()-25))
}

func uiMenuItem(n int, label, description string) {
	width := uiWidth()
	labelWidth := 25
	if width < 64 {
		labelWidth = 18
	}
	label = uiClip(label, labelWidth)
	descWidth := width - labelWidth - 11
	fmt.Printf("  %2d  %-*s %s\n", n, labelWidth, label, uiClip(description, descWidth))
}

func uiFooter(hint string) {
	fmt.Println()
	fmt.Println("  " + strings.Repeat("-", uiWidth()-4))
	fmt.Printf("  %s\n", uiClip(hint, uiWidth()-4))
}
