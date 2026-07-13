package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ZhipuAPIKey    string `json:"zhipu_api_key"`
	DeepSeekAPIKey string `json:"deepseek_api_key"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
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
		backupPath := path + ".corrupt"
		_ = os.Rename(path, backupPath)
		fmt.Fprintf(os.Stderr, "config corrupted, backed up to %s: %v\n", backupPath, err)
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
