// Package config owns TransOS' persistent user configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultPath = "transos_config.json"

	ConfigVersion = 1

	ModeAutomatic = "automatic"
	ModeGuided    = "guided"
	ModeManual    = "manual"

	DepthSurface = "surface"
	DepthMedium  = "medium"
	DepthDeep    = "deep"

	CategoryEnvironment = "environment"
	CategorySoftware    = "software"
	CategoryRegistry    = "registry"
	CategoryTheme       = "theme"
)

// Config separates workflow behavior from extraction depth and extraction scope.
// This permits users to change one dimension without silently changing the others.
type Config struct {
	Version         int      `json:"config_version"`
	OperationMode   string   `json:"operation_mode"`
	ExtractionDepth string   `json:"extraction_depth"`
	Categories      []string `json:"categories"`
}

// Default returns the agreed TransOS default: guided operation, medium depth,
// and the environment/software/registry/theme categories.
func Default() Config {
	return Config{
		Version:         ConfigVersion,
		OperationMode:   ModeGuided,
		ExtractionDepth: DepthMedium,
		Categories: []string{
			CategoryEnvironment,
			CategorySoftware,
			CategoryRegistry,
			CategoryTheme,
		},
	}
}

// Normalize validates and canonicalizes a configuration without mutating input.
func Normalize(input Config) (Config, error) {
	cfg := input
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
	}
	if cfg.Version != ConfigVersion {
		return Config{}, fmt.Errorf("unsupported config_version %d; expected %d", cfg.Version, ConfigVersion)
	}

	cfg.OperationMode = strings.ToLower(strings.TrimSpace(cfg.OperationMode))
	if cfg.OperationMode == "" {
		cfg.OperationMode = ModeGuided
	}
	switch cfg.OperationMode {
	case ModeAutomatic, ModeGuided, ModeManual:
	default:
		return Config{}, fmt.Errorf("invalid operation_mode %q; choose automatic, guided, or manual", cfg.OperationMode)
	}

	cfg.ExtractionDepth = strings.ToLower(strings.TrimSpace(cfg.ExtractionDepth))
	if cfg.ExtractionDepth == "" {
		cfg.ExtractionDepth = DepthMedium
	}
	switch cfg.ExtractionDepth {
	case DepthSurface, DepthMedium, DepthDeep:
	default:
		return Config{}, fmt.Errorf("invalid extraction_depth %q; choose surface, medium, or deep", cfg.ExtractionDepth)
	}

	allowed := map[string]struct{}{
		CategoryEnvironment: {},
		CategorySoftware:    {},
		CategoryRegistry:    {},
		CategoryTheme:       {},
	}
	seen := make(map[string]struct{}, len(cfg.Categories))
	categories := make([]string, 0, len(cfg.Categories))
	for _, raw := range cfg.Categories {
		category := strings.ToLower(strings.TrimSpace(raw))
		if category == "" {
			continue
		}
		if _, ok := allowed[category]; !ok {
			return Config{}, fmt.Errorf("unsupported extraction category %q; supported categories: environment, software, registry, theme", category)
		}
		if _, ok := seen[category]; ok {
			return Config{}, fmt.Errorf("duplicate extraction category %q", category)
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	if len(categories) == 0 {
		return Config{}, errors.New("at least one extraction category must be enabled")
	}

	// Use a deterministic category order for predictable config files and demos.
	order := map[string]int{
		CategoryEnvironment: 0,
		CategorySoftware:    1,
		CategoryRegistry:    2,
		CategoryTheme:       3,
	}
	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if order[categories[j]] < order[categories[i]] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}
	cfg.Categories = categories
	return cfg, nil
}

// Load reads and validates a configuration file.
func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read TransOS config %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode TransOS config %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("decode TransOS config %q: multiple JSON values are not allowed", path)
		}
		return Config{}, fmt.Errorf("decode TransOS config %q: %w", path, err)
	}

	normalized, err := Normalize(cfg)
	if err != nil {
		return Config{}, fmt.Errorf("validate TransOS config %q: %w", path, err)
	}
	return normalized, nil
}

// LoadOrCreate returns the existing configuration or creates a default one
// when the file does not yet exist. Invalid existing files are never overwritten.
func LoadOrCreate(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}

	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	cfg = Default()
	if err := Save(path, cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save validates and writes a configuration using a temporary file before
// replacing the destination.
func Save(path string, input Config) error {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}

	cfg, err := Normalize(input)
	if err != nil {
		return fmt.Errorf("refuse to save invalid TransOS config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode TransOS config: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	temp, err := os.CreateTemp(dir, ".transos-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config file: %w", err)
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	if _, err := temp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temporary config file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temporary config file: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("close temporary config file: %w", err)
	}

	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("replace TransOS config %q: %w", path, err)
	}
	return nil
}
