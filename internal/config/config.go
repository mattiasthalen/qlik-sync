package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultThreads = 5
	DefaultRetries = 3
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

	// Check if this is v0.1.0 (no "version" field, has top-level "context")
	var probe struct {
		Version string `json:"version"`
		Context string `json:"context"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if probe.Version == "" && probe.Context != "" {
		return migrateV010(data)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func migrateV010(data []byte) (*Config, error) {
	var legacy struct {
		Context  string `json:"context"`
		Server   string `json:"server"`
		LastSync string `json:"lastSync,omitempty"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("parsing legacy config: %w", err)
	}

	return &Config{
		Version: "0.2.0",
		Tenants: []Tenant{
			{
				Context:  legacy.Context,
				Server:   legacy.Server,
				Type:     DetectTenantType(legacy.Server),
				LastSync: legacy.LastSync,
			},
		},
	}, nil
}

func Write(dir string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func WithDefaults(cfg *Config) *Config {
	out := *cfg
	if out.Threads == 0 {
		out.Threads = DefaultThreads
	}
	if out.Retries == 0 {
		out.Retries = DefaultRetries
	}
	return &out
}

func DetectTenantType(server string) string {
	if strings.Contains(server, "qlikcloud.com") {
		return "cloud"
	}
	return "on-prem"
}
