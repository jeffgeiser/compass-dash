// Package config loads and saves dashboard configuration from ~/.compass-dash/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultPort = 7174

// Config holds all user-configurable settings for compass-dash.
type Config struct {
	CompassPath string `json:"compass_path"`
	Port        int    `json:"port"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port: defaultPort,
	}
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".compass-dash", "config.json"), nil
}

// Load reads the config file from disk, returning defaults if the file doesn't exist.
func Load() (Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	return cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0600)
}
