package extractor

import (
	"os"
	"strings"

	"github.com/transos/transos/internal/schema"
)

// ExtractHostEnvironment captures real system environment variables and parameters from the host OS
func ExtractHostEnvironment() *schema.ConfigProfile {
	profile := schema.NewProfile("host-system-detected")

	// Harvest real environment variables from host OS
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			profile.Environment[pair[0]] = pair[1]
		}
	}

	// Capture baseline configuration markers
	profile.Registry["SystemRoot"] = os.Getenv("SystemRoot")
	profile.Registry["UserProfile"] = os.Getenv("USERPROFILE")
	profile.ThemeConfig["active-engine"] = "transos-native-extractor-v1"

	return profile
}