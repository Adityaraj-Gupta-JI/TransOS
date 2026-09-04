package schema

import (
	"encoding/json"
	"time"
)

// ConfigProfile represents the schema v7 compliant migration payload structure
type ConfigProfile struct {
	Version     string            `json:"version"`
	Timestamp   string            `json:"timestamp"`
	OSSource    string            `json:"os_source"`
	Environment map[string]string `json:"environment"`
	Registry    map[string]string `json:"registry_keys"`
	ThemeConfig map[string]string `json:"theme_config"`
}

// Serialize converts the config profile into a pretty-printed JSON byte array
func (cp *ConfigProfile) Serialize() ([]byte, error) {
	return json.MarshalIndent(cp, "", "  ")
}

// Deserialize parses a JSON byte array back into a ConfigProfile struct
func Deserialize(data []byte) (*ConfigProfile, error) {
	var cp ConfigProfile
	err := json.Unmarshal(data, &cp)
	if err != nil {
		return nil, err
	}
	return &cp, nil
}

func NewProfile(source string) *ConfigProfile {
	return &ConfigProfile{
		Version:     "1.0.0",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		OSSource:    source,
		Environment: make(map[string]string),
		Registry:    make(map[string]string),
		ThemeConfig: make(map[string]string),
	}
}