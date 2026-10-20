package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// All framing uses ASCII, so menus remain aligned in PowerShell, SSH and logs.
func uiWidth() int {
	width := 78
	if v, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && v > 0 {
		width = v
	}
	if width < 48 {
		return 48
	}
	if width > 100 {
		return 100
	}
	return width
}

func uiColorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" || os.Getenv("CLICOLOR") == "0" {
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

// uiPaint applies restrained styling to interactive console chrome only.
func uiPaint(code, value string) string {
	if !uiColorEnabled() {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func uiRule(char string) { fmt.Println("  " + strings.Repeat(char, uiWidth()-4)) }

func uiHeader(title, subtitle string) {
	w := uiWidth() - 4
	fmt.Println()
	fmt.Println("  " + uiPaint("36", strings.Repeat("=", w)))
	fmt.Println("  " + uiPaint("1;36", "TRANSOS") + "  " + uiPaint("2", "/") + "  " + uiPaint("1", uiClip(title, w-12)))
	for _, line := range uiWrap(subtitle, w) {
		fmt.Println("  " + uiPaint("2", line))
	}
	fmt.Println("  " + uiPaint("36", strings.Repeat("=", w)))
}

func uiSection(title string) {
	fmt.Println()
	caption := "[ " + strings.ToUpper(title) + " ]"
	fmt.Println("  " + uiPaint("1;36", uiClip(caption, uiWidth()-4)))
	fmt.Println("  " + uiPaint("2", strings.Repeat("-", uiWidth()-4)))
}

func uiKeyValue(key, value string) {
	labelWidth := 19
	if uiWidth() < 60 {
		labelWidth = 18
	}
	valueWidth := uiWidth() - labelWidth - 5
	lines := uiWrap(value, valueWidth)
	if len(lines) == 0 {
		lines = []string{""}
	}
	fmt.Printf("  %-*s %s\n", labelWidth, uiClip(key, labelWidth), lines[0])
	for _, line := range lines[1:] {
		fmt.Printf("  %-*s %s\n", labelWidth, "", line)
	}
}

func uiMenuItem(n int, label, description string) {
	w := uiWidth() - 4
	prefix := fmt.Sprintf("[%d]", n)
	// Descriptions get their own line instead of losing essential guidance.
	fmt.Printf("  %s  %s\n", uiPaint("1;36", prefix), uiPaint("1", uiClip(label, w-6)))
	for _, line := range uiWrap(description, w-6) {
		fmt.Println("       " + uiPaint("2", line))
	}
}

func uiFooter(hint string) {
	fmt.Println()
	uiRule("-")
	for _, line := range uiWrap(hint, uiWidth()-6) {
		fmt.Println("  " + uiPaint("36", line))
	}
}

func uiNote(text string) {
	for _, line := range uiWrap(text, uiWidth()-8) {
		fmt.Println("  " + uiPaint("33", "! ") + line)
	}
}

func uiWrap(s string, width int) []string {
	if width < 1 {
		return nil
	}
	words := strings.Fields(s)
	var lines []string
	line := ""
	for _, word := range words {
		if line != "" && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) > width {
			lines = append(lines, line)
			line = ""
		}
		for utf8.RuneCountInString(word) > width {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			runes := []rune(word)
			lines = append(lines, string(runes[:width]))
			word = string(runes[width:])
		}
		if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
