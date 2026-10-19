package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfiguration(t *testing.T) {
	cfg, err := Normalize(Default())
	if err != nil {
		t.Fatalf("Normalize(Default()) error = %v", err)
	}
	if cfg.OperationMode != ModeGuided {
		t.Fatalf("operation mode = %q, want %q", cfg.OperationMode, ModeGuided)
	}
	if cfg.ExtractionDepth != DepthMedium {
		t.Fatalf("extraction depth = %q, want %q", cfg.ExtractionDepth, DepthMedium)
	}
	if strings.Join(cfg.Categories, ",") != "environment,software,registry,theme" {
		t.Fatalf("unexpected default categories: %#v", cfg.Categories)
	}
}

func TestNormalizeRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{"unknown mode", func(c *Config) { c.OperationMode = "turbo" }},
		{"unknown depth", func(c *Config) { c.ExtractionDepth = "extreme" }},
		{"unknown category", func(c *Config) { c.Categories = []string{"environment", "drivers"} }},
		{"duplicate category", func(c *Config) { c.Categories = []string{"software", "software"} }},
		{"empty categories", func(c *Config) { c.Categories = nil }},
		{"unknown version", func(c *Config) { c.Version = 99 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.edit(&cfg)
			if _, err := Normalize(cfg); err == nil {
				t.Fatal("Normalize() succeeded for invalid config")
			}
		})
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "transos_config.json")
	cfg := Default()
	cfg.OperationMode = ModeManual
	cfg.ExtractionDepth = DepthDeep
	cfg.Categories = []string{"theme", "environment", "software"}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.OperationMode != ModeManual || loaded.ExtractionDepth != DepthDeep {
		t.Fatalf("loaded config does not preserve mode/depth: %#v", loaded)
	}
	if strings.Join(loaded.Categories, ",") != "environment,software,theme" {
		t.Fatalf("categories were not canonicalized: %#v", loaded.Categories)
	}
}

func TestLoadOrCreateDoesNotOverwriteInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transos_config.json")
	bad := []byte(`{"config_version":1,"operation_mode":"unsafe","extraction_depth":"medium","categories":["environment"]}`)
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(path); err == nil {
		t.Fatal("LoadOrCreate() succeeded for invalid existing file")
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(bad) {
		t.Fatal("invalid configuration was unexpectedly overwritten")
	}
}
