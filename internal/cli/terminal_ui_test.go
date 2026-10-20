package cli

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestUIWrapNeverExceedsWidth(t *testing.T) {
	for _, width := range []int{1, 8, 24, 44} {
		for _, line := range uiWrap("[1-6] open workspace | help [topic] | status | exit | exceptionallylongcommandname", width) {
			if utf8.RuneCountInString(line) > width {
				t.Fatalf("width %d: %q", width, line)
			}
		}
	}
}
func TestUIClipWidth(t *testing.T) {
	for _, width := range []int{1, 2, 3, 4, 12} {
		if got := uiClip("Example Unicode → navigation", width); utf8.RuneCountInString(got) > width {
			t.Fatalf("clip %d: %q", width, got)
		}
	}
}
func TestInteractiveParsing(t *testing.T) {
	got := splitCommandLine(`plan map "Visual Studio Code" code`)
	if strings.Join(got, "|") != "plan|map|Visual Studio Code|code" {
		t.Fatalf("unexpected tokens: %q", got)
	}
	if got := splitCommandLine(`plan map "unclosed`); len(got) != 0 {
		t.Fatalf("unclosed quote should not execute: %q", got)
	}
}
