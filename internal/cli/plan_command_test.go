package cli

import (
	"reflect"
	"testing"
)

func TestSplitCommandLineKeepsQuotedSoftwareNamesTogether(t *testing.T) {
	got := splitCommandLine(`plan map "Visual Studio Code" code`)
	want := []string{"plan", "map", "Visual Studio Code", "code"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCommandLine() = %#v, want %#v", got, want)
	}
}

func TestSplitCommandLineHandlesSingleQuotesAndEscapedDoubleQuotes(t *testing.T) {
	got := splitCommandLine(`plan include 'Developer Tools'`)
	want := []string{"plan", "include", "Developer Tools"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCommandLine() = %#v, want %#v", got, want)
	}

	got = splitCommandLine(`plan add "My \"Special\" Tool"`)
	want = []string{"plan", "add", `My "Special" Tool`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCommandLine() with escaped quotes = %#v, want %#v", got, want)
	}
}
