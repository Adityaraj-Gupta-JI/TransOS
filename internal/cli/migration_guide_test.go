package cli

import (
	"strings"
	"testing"
)

func TestMigrationNextStepEvidence(t *testing.T) {
	cases := []struct {
		state    sessionState
		contains string
	}{
		{sessionState{}, "extract"},
		{sessionState{profileReady: true}, "validate"},
		{sessionState{profileReady: true, profileValid: true}, "plan show"},
		{sessionState{profileReady: true, profileValid: true, outputReady: true}, "package verify"},
	}
	for _, tc := range cases {
		if got := migrationNextStep(tc.state); !strings.Contains(got, tc.contains) {
			t.Errorf("%+v: %q lacks %q", tc.state, got, tc.contains)
		}
	}
}
