package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const AppVersion = "1.0.0"

type Config struct {
	DataFilePath string `json:"data_file_path"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "stashcut", "config.json"), nil
}

// DefaultDataPath returns the default location for shortcuts.json.
func DefaultDataPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Stashcut", "shortcuts.json"), nil
}

// Load reads the config file, creating it with defaults if it doesn't exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return createDefault(path)
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func createDefault(path string) (*Config, error) {
	defaultData, err := DefaultDataPath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{DataFilePath: defaultData}
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
