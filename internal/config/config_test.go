package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/config"
)

func TestRead_V020(t *testing.T) {
	dir := t.TempDir()
	data := `{
		"version": "0.2.0",
		"threads": 10,
		"retries": 5,
		"tenants": [
			{
				"context": "my-cloud",
				"server": "https://tenant.qlikcloud.com",
				"type": "cloud",
				"lastSync": "2026-04-10T14:30:00Z"
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != "0.2.0" {
		t.Errorf("version = %q, want %q", cfg.Version, "0.2.0")
	}
	if cfg.Threads != 10 {
		t.Errorf("threads = %d, want 10", cfg.Threads)
	}
	if cfg.Retries != 5 {
		t.Errorf("retries = %d, want 5", cfg.Retries)
	}
	if len(cfg.Tenants) != 1 {
		t.Fatalf("tenants count = %d, want 1", len(cfg.Tenants))
	}
	tenant := cfg.Tenants[0]
	if tenant.Context != "my-cloud" {
		t.Errorf("context = %q, want %q", tenant.Context, "my-cloud")
	}
	if tenant.Type != "cloud" {
		t.Errorf("type = %q, want %q", tenant.Type, "cloud")
	}
}

func TestRead_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := config.Read(dir)
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}
