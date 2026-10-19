package safety

import (
	"strings"

	"github.com/transos/transos/internal/schema"
)

const SensitiveRedactedClassification = "SENSITIVE_REDACTED"

// IsSensitiveEnvironmentName identifies variable names that commonly carry
// credentials, tokens, private keys, session material, or credential-bearing URLs.
// This intentionally errs on the side of asking the user to configure secrets
// separately on the target system rather than copying them automatically.
func IsSensitiveEnvironmentName(name string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}

	// Treat delimiters consistently while preserving terms such as API_KEY.
	normalized = strings.NewReplacer("-", "_", ".", "_").Replace(normalized)
	parts := strings.Split(normalized, "_")
	for _, part := range parts {
		switch part {
		case "PASSWORD", "PASSWD", "TOKEN", "SECRET", "CREDENTIAL", "CREDENTIALS",
			"PRIVATEKEY", "APIKEY", "ACCESSKEY", "CLIENTSECRET", "REFRESHTOKEN",
			"BEARER", "COOKIE", "AUTHORIZATION":
			return true
		}
	}

	// These names frequently contain credentials even without an obvious secret
	// suffix, e.g. database connection strings and embedded user:password URLs.
	switch normalized {
	case "DATABASE_URL", "DATABASE_URI", "DB_URL", "DB_URI", "REDIS_URL",
		"MONGO_URI", "MONGODB_URI", "CONNECTION_STRING", "CONNECTIONSTRINGS",
		"DSN", "SSH_PRIVATE_KEY", "GPG_PRIVATE_KEY":
		return true
	}

	return strings.Contains(normalized, "PRIVATE_KEY") ||
		strings.Contains(normalized, "API_KEY") ||
		strings.Contains(normalized, "ACCESS_KEY") ||
		strings.Contains(normalized, "CLIENT_SECRET") ||
		strings.Contains(normalized, "AUTH_TOKEN") ||
		strings.Contains(normalized, "SESSION_ID")
}

// RedactSensitiveEnvironment clears values for likely credentials before a
// profile is persisted. Names and metadata remain so users can recreate these
// variables securely on the target system.
func RedactSensitiveEnvironment(profile *schema.MigrationProfile) int {
	if profile == nil {
		return 0
	}
	redacted := 0
	for index := range profile.Environment {
		item := &profile.Environment[index]
		if !IsSensitiveEnvironmentName(item.Name) {
			continue
		}
		if item.Value != "" || item.Classification != SensitiveRedactedClassification {
			redacted++
		}
		item.Value = ""
		item.Classification = SensitiveRedactedClassification
	}
	return redacted
}
