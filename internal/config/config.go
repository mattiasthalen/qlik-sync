package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Version string   `json:"version"`
	Threads int      `json:"threads"`
	Retries int      `json:"retries"`
	Tenants []Tenant `json:"tenants"`
}

type Tenant struct {
	Context  string `json:"context"`
	Server   string `json:"server"`
	Type     string `json:"type"`
	LastSync string `json:"lastSync,omitempty"`
}

func Read(dir string) (*Config, error) {
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
