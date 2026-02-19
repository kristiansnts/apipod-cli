package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultBaseURL = "https://api.apipod.net"
	DefaultModel   = "claude-sonnet-4-20250514"
	ConfigDir      = ".apipod"
	ConfigFile     = "config.json"
)

type Config struct {
	BaseURL  string `json:"base_url,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	Model    string `json:"model,omitempty"`
	Username string `json:"username,omitempty"`
	Plan     string `json:"plan,omitempty"`
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, ConfigFile)
}

func configDirPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir)
}

func Load() (*Config, error) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}

	if env := os.Getenv("APIPOD_BASE_URL"); env != "" {
		cfg.BaseURL = env
	}
	if env := os.Getenv("APIPOD_API_KEY"); env != "" {
		cfg.APIKey = env
	}
	if env := os.Getenv("APIPOD_MODEL"); env != "" {
		cfg.Model = env
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg, nil
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg, nil
	}

	if fileCfg.BaseURL != "" {
		cfg.BaseURL = fileCfg.BaseURL
	}
	if fileCfg.APIKey != "" && cfg.APIKey == "" {
		cfg.APIKey = fileCfg.APIKey
	}
	if fileCfg.Model != "" && os.Getenv("APIPOD_MODEL") == "" {
		cfg.Model = fileCfg.Model
	}
	cfg.Username = fileCfg.Username
	cfg.Plan = fileCfg.Plan

	return cfg, nil
}

func Save(cfg *Config) error {
	dir := configDirPath()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}

func ClearCredentials() error {
	cfg, _ := Load()
	cfg.APIKey = ""
	cfg.Username = ""
	cfg.Plan = ""
	return Save(cfg)
}
