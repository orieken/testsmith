package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFilename = ".testsmith.yaml"

// Load walks upward from startDir looking for a .testsmith.yaml (or pyproject.toml
// fallback for Python projects) and returns the merged Config.
func Load(startDir string) (*Config, error) {
	cfg := Default()

	path, found := findConfigFile(startDir)
	if !found {
		return cfg, nil
	}

	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	cfg.ConfigPath = path
	return cfg, nil
}

// LoadFromFile loads configuration from an explicit file path.
func LoadFromFile(path string) (*Config, error) {
	cfg := Default()
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	cfg.ConfigPath = path
	return cfg, nil
}

func findConfigFile(startDir string) (string, bool) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, configFilename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func loadYAML(path string, dst *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Unmarshal into a raw map first to allow partial overrides without zeroing defaults.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	// Re-marshal and unmarshal into Config to apply type conversion cleanly.
	merged, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(merged, dst)
}
