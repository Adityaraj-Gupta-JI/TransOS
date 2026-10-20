package cli

import "testing"

func TestPackageBadArgumentsReturnUsageCode(t *testing.T) {
	for _, args := range [][]string{
		{}, {"verify-applied"}, {"restore"}, {"restore", "not-approved", "backup"},
		{"apply"}, {"stage"}, {"seal", "", "extra"}, {"unknown"},
	} {
		if got := runPackage(args); got != 2 {
			t.Errorf("runPackage(%q) = %d; want 2", args, got)
		}
	}
}
