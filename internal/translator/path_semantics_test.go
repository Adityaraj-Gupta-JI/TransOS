package translator

import "testing"

func TestParseWinPath(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDrive    string
		wantAbsolute bool
		wantUNC      bool
		wantEnv      []string
	}{
		{
			name:         "absolute user path",
			input:        `C:\Users\arcti`,
			wantDrive:    "C",
			wantAbsolute: true,
		},
		{
			name:         "embedded environment variable",
			input:        `%USERPROFILE%\.dotnet\tools`,
			wantAbsolute: false,
			wantEnv:      []string{"USERPROFILE"},
		},
		{
			name:    "multiple environment variables",
			input:   `%PNPM_HOME%\%USERNAME%\bin`,
			wantEnv: []string{"PNPM_HOME", "USERNAME"},
		},
		{
			name:         "UNC path",
			input:        `\\server\share\folder`,
			wantAbsolute: true,
			wantUNC:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ParseWinPath(test.input)

			if got.Drive != test.wantDrive {
				t.Fatalf("drive = %q, want %q", got.Drive, test.wantDrive)
			}

			if got.IsAbsolute != test.wantAbsolute {
				t.Fatalf(
					"isAbsolute = %v, want %v",
					got.IsAbsolute,
					test.wantAbsolute,
				)
			}

			if got.IsUNC != test.wantUNC {
				t.Fatalf(
					"isUNC = %v, want %v",
					got.IsUNC,
					test.wantUNC,
				)
			}

			if len(got.EnvVars) != len(test.wantEnv) {
				t.Fatalf(
					"environment variables = %#v, want %#v",
					got.EnvVars,
					test.wantEnv,
				)
			}

			for index := range test.wantEnv {
				if got.EnvVars[index] != test.wantEnv[index] {
					t.Fatalf(
						"EnvVars[%d] = %q, want %q",
						index,
						got.EnvVars[index],
						test.wantEnv[index],
					)
				}
			}
		})
	}
}

func TestAnalyzePath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantKind   PathKind
		wantStrat  PathStrategy
		wantTarget string
	}{
		{
			name:       "user profile",
			input:      `C:\Users\arcti`,
			wantKind:   PathKindUserHome,
			wantStrat:  PathStrategyNativeEquivalent,
			wantTarget: "$HOME",
		},
		{
			name:       "roaming application data",
			input:      `C:\Users\arcti\AppData\Roaming`,
			wantKind:   PathKindUserConfig,
			wantStrat:  PathStrategyNativeEquivalent,
			wantTarget: "$HOME/.config",
		},
		{
			name:       "local application data",
			input:      `C:\Users\arcti\AppData\Local`,
			wantKind:   PathKindUserData,
			wantStrat:  PathStrategyConvertible,
			wantTarget: "$HOME/.local/share",
		},
		{
			name:       "temporary directory",
			input:      `C:\Users\arcti\AppData\Local\Temp`,
			wantKind:   PathKindTemporary,
			wantStrat:  PathStrategyNativeEquivalent,
			wantTarget: "/tmp",
		},
		{
			name:      "program files",
			input:     `C:\Program Files\Microsoft VS Code`,
			wantKind:  PathKindApplicationInstall,
			wantStrat: PathStrategyManual,
		},
		{
			name:      "windows system",
			input:     `C:\Windows\System32`,
			wantKind:  PathKindWindowsSystem,
			wantStrat: PathStrategyIgnore,
		},
		{
			name:      "developer path",
			input:     `D:\Flutter\flutter\bin`,
			wantKind:  PathKindDeveloperTool,
			wantStrat: PathStrategyConvertible,
		},
		{
			name:      "unc path",
			input:     `\\server\share\folder`,
			wantKind:  PathKindNetworkShare,
			wantStrat: PathStrategyManual,
		},
		{
			name:      "unknown drive path",
			input:     `D:\Custom\Application\Data`,
			wantKind:  PathKindUnknown,
			wantStrat: PathStrategyManual,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := AnalyzePath(test.input)

			if got.Kind != test.wantKind {
				t.Fatalf(
					"kind = %q, want %q; reason=%s",
					got.Kind,
					test.wantKind,
					got.Reason,
				)
			}

			if got.Strategy != test.wantStrat {
				t.Fatalf(
					"strategy = %q, want %q; reason=%s",
					got.Strategy,
					test.wantStrat,
					got.Reason,
				)
			}

			if test.wantTarget != "" && got.LinuxPath != test.wantTarget {
				t.Fatalf(
					"linux path = %q, want %q",
					got.LinuxPath,
					test.wantTarget,
				)
			}
		})
	}
}

func TestAnalyzePathDoesNotInventLinuxLocation(t *testing.T) {
	result := AnalyzePath(`D:\Ansys\Installation\Data`)

	if result.LinuxPath != "" {
		t.Fatalf(
			"unexpected Linux path %q for unknown application path",
			result.LinuxPath,
		)
	}

	if result.Strategy != PathStrategyManual {
		t.Fatalf(
			"strategy = %q, want %q",
			result.Strategy,
			PathStrategyManual,
		)
	}
}

func TestTranslatePathString(t *testing.T) {
	input := `%USERPROFILE%\go\bin;C:\Windows\System32;C:\Tools`

	got := TranslatePathString(input)

	want := "$HOME/go/bin:/mnt/c/Windows/System32:/mnt/c/Tools"

	if got != want {
		t.Fatalf(
			"translated path = %q, want %q",
			got,
			want,
		)
	}
}

func TestNormalizePathForComparison(t *testing.T) {
	got := NormalizePathForComparison(
		`C:/Users/arcti/AppData/Roaming/`,
	)

	want := `c:\users\arcti\appdata\roaming`

	if got != want {
		t.Fatalf(
			"normalized path = %q, want %q",
			got,
			want,
		)
	}
}
