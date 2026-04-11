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

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Version: "0.2.0",
		Threads: 5,
		Retries: 3,
		Tenants: []config.Tenant{
			{Context: "test", Server: "https://test.qlikcloud.com", Type: "cloud"},
		},
	}

	if err := config.Write(dir, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := config.Read(dir)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Tenants[0].Context != "test" {
		t.Errorf("context = %q, want %q", got.Tenants[0].Context, "test")
	}
}

func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	data := `{"version": "0.2.0", "tenants": []}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolved := config.WithDefaults(cfg)
	if resolved.Threads != 5 {
		t.Errorf("default threads = %d, want 5", resolved.Threads)
	}
	if resolved.Retries != 3 {
		t.Errorf("default retries = %d, want 3", resolved.Retries)
	}
}

func TestRead_V010Migration(t *testing.T) {
	dir := t.TempDir()
	data := `{
		"context": "legacy-tenant",
		"server": "https://legacy.qlikcloud.com",
		"lastSync": "2026-01-01T00:00:00Z"
	}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != "0.2.0" {
		t.Errorf("migrated version = %q, want %q", cfg.Version, "0.2.0")
	}
	if len(cfg.Tenants) != 1 {
		t.Fatalf("migrated tenants count = %d, want 1", len(cfg.Tenants))
	}
	if cfg.Tenants[0].Context != "legacy-tenant" {
		t.Errorf("migrated context = %q, want %q", cfg.Tenants[0].Context, "legacy-tenant")
	}
	if cfg.Tenants[0].Type != "cloud" {
		t.Errorf("migrated type = %q, want %q", cfg.Tenants[0].Type, "cloud")
	}
}

func TestDetectTenantType(t *testing.T) {
	tests := []struct {
		server string
		want   string
	}{
		{"https://tenant.qlikcloud.com", "cloud"},
		{"https://qlik.corp.local/jwt", "on-prem"},
		{"https://us.qlikcloud.com", "cloud"},
	}
	for _, tt := range tests {
		got := config.DetectTenantType(tt.server)
		if got != tt.want {
			t.Errorf("DetectTenantType(%q) = %q, want %q", tt.server, got, tt.want)
		}
	}
}

func TestResolve(t *testing.T) {
	cfg := &config.Config{
		Version: "0.2.0",
		Threads: 10,
		Retries: 5,
		Tenants: []config.Tenant{
			{Context: "a", Server: "https://a.qlikcloud.com", Type: "cloud"},
			{Context: "b", Server: "https://b.corp.local", Type: "on-prem"},
		},
	}

	t.Run("flag overrides config", func(t *testing.T) {
		flagThreads := 20
		flagRetries := 1
		resolved := config.Resolve(cfg, &flagThreads, &flagRetries)
		if resolved.Threads != 20 {
			t.Errorf("threads = %d, want 20", resolved.Threads)
		}
		if resolved.Retries != 1 {
			t.Errorf("retries = %d, want 1", resolved.Retries)
		}
	})

	t.Run("nil flags use config values", func(t *testing.T) {
		resolved := config.Resolve(cfg, nil, nil)
		if resolved.Threads != 10 {
			t.Errorf("threads = %d, want 10", resolved.Threads)
		}
		if resolved.Retries != 5 {
			t.Errorf("retries = %d, want 5", resolved.Retries)
		}
	})

	t.Run("filter tenants by context", func(t *testing.T) {
		tenants := config.FilterTenants(cfg.Tenants, "a")
		if len(tenants) != 1 {
			t.Fatalf("filtered count = %d, want 1", len(tenants))
		}
		if tenants[0].Context != "a" {
			t.Errorf("context = %q, want %q", tenants[0].Context, "a")
		}
	})

	t.Run("empty filter returns all tenants", func(t *testing.T) {
		tenants := config.FilterTenants(cfg.Tenants, "")
		if len(tenants) != 2 {
			t.Fatalf("filtered count = %d, want 2", len(tenants))
		}
	})
}
