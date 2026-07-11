package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ZhipuAPIKey string `json:"zhipu_api_key"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "token-tray", "config.json")
}

func LoadConfig() Config {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}
	}
	return c
}

func SaveConfig(c Config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// Write with 0600 (owner-only, sensitive credential).
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	// WriteFile honours umask, force 0600 explicitly.
	_ = os.Chmod(path, 0600)
	return nil
}
