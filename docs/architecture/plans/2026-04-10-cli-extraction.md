# CLI Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `qs` — a standalone Go CLI that syncs Qlik Cloud and on-prem apps to a local `qlik/` directory.

**Architecture:** Cobra CLI with internal packages for config, sync orchestration, QVF/QVW parsing, and terminal UI. Cloud sync shells out to `qlik-cli` (v1). On-prem sync downloads QVF bytes and streams them into an internal parser. Parallel execution via `errgroup` with configurable retries.

**Tech Stack:** Go 1.25+, Cobra (CLI), zerolog (logging), lipgloss (terminal styling), errgroup (concurrency), GoReleaser (distribution), GitHub Actions (CI/CD), golangci-lint (linting), Lefthook (git hooks).

**Spec:** `docs/architecture/specs/2026-04-10-cli-extraction-design.md`

**Working directory for all commands:** `/workspaces/qlik-sync/.worktrees/cli-extraction`

---

### Task 1: Project Scaffold

**Files:**
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/version.go`
- Create: `go.mod`
- Create: `Makefile`
- Create: `.golangci.yml`
- Create: `.goreleaser.yml`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Modify: `.devcontainer/devcontainer.json`
- Modify: `lefthook.yml`
- Modify: `.gitignore`

- [ ] **Step 1: Enable Go in devcontainer**

In `.devcontainer/devcontainer.json`, uncomment the Go feature line:

```json
"ghcr.io/devcontainers/features/go:1": {},
```

- [ ] **Step 2: Initialize Go module**

Run: `go mod init github.com/mattiasthalen/qlik-sync`
Expected: creates `go.mod`

- [ ] **Step 3: Install Cobra dependency**

Run: `go get github.com/spf13/cobra@latest`
Expected: adds cobra to `go.mod` and `go.sum`

- [ ] **Step 4: Write main.go**

```go
// main.go
package main

import (
	"os"

	"github.com/mattiasthalen/qlik-sync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Write cmd/root.go**

```go
// cmd/root.go
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	logLevel  string
	configDir string
)

var rootCmd = &cobra.Command{
	Use:   "qs",
	Short: "Qlik Sync — sync Qlik apps to local files",
	Long:  "qs syncs Qlik Sense cloud and on-prem apps to a local qlik/ directory for version control and offline inspection.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "disabled", "log level (debug|info|warn|error|disabled)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "qlik", "config and sync directory")
}
```

- [ ] **Step 6: Write cmd/version.go**

```go
// cmd/version.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print qs version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("qs %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

- [ ] **Step 7: Verify CLI builds and runs**

Run: `go build -o qs . && ./qs version`
Expected: `qs dev`

Run: `./qs --help`
Expected: help text with "Qlik Sync" description, `setup`, `sync`, `version` subcommands listed (setup/sync added in later tasks — for now just version)

- [ ] **Step 8: Write Makefile**

```makefile
# Makefile
.PHONY: build test lint vet coverage clean

VERSION ?= dev
LDFLAGS := -X github.com/mattiasthalen/qlik-sync/cmd.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o qs .

test:
	go test -race ./...

lint:
	golangci-lint run

vet:
	go vet ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f qs coverage.out
```

- [ ] **Step 9: Write .golangci.yml**

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - gosimple
```

- [ ] **Step 10: Write .goreleaser.yml**

```yaml
# .goreleaser.yml
version: 2

builds:
  - binary: qs
    ldflags:
      - -X github.com/mattiasthalen/qlik-sync/cmd.Version={{ .Version }}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - formats:
      - tar.gz
    format_overrides:
      - goos: windows
        formats:
          - zip

checksum:
  name_template: checksums.txt

changelog:
  filters:
    exclude:
      - "^chore"
      - "^docs"
      - "^test"
```

- [ ] **Step 11: Write .github/workflows/ci.yml**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
  push:
    branches: [main]

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Vet
        run: go vet ./...
      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      - name: Test
        run: go test -race -coverprofile=coverage.out ./...
```

- [ ] **Step 12: Write .github/workflows/release.yml**

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 13: Update lefthook.yml with Go hooks**

Add Go-specific pre-commit hooks after the existing `gitkeep-cleanup`:

```yaml
pre-commit:
  commands:
    gitkeep-cleanup:
      glob: ".gitkeep"
      run: |
        for f in {staged_files}; do
          dir=$(dirname "$f")
          if [ -f "$dir/.gitkeep" ] && [ "$(ls -1 "$dir" | wc -l)" -gt 1 ]; then
            rm "$dir/.gitkeep"
            git add "$dir/.gitkeep"
          fi
        done
    go-vet:
      glob: "*.go"
      run: go vet ./...
    go-lint:
      glob: "*.go"
      run: golangci-lint run
    go-test:
      glob: "*.go"
      run: go test ./...

commit-msg:
  commands:
    conventional-commit:
      run: |
        msg=$(head -1 {1})
        if ! echo "$msg" | grep -qE '^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\(.+\))?!?: .+'; then
          echo "Commit message must follow Conventional Commits: <type>[scope]: <description>"
          exit 1
        fi
```

- [ ] **Step 14: Update .gitignore**

Add build artifacts:

```
.worktrees/
qs
coverage.out
dist/
```

- [ ] **Step 15: Verify build, vet, and test pass**

Run: `make build && make vet && make test`
Expected: all pass (no tests yet, but no errors)

- [ ] **Step 16: Commit**

```bash
git add main.go cmd/ go.mod go.sum Makefile .golangci.yml .goreleaser.yml .github/ .devcontainer/devcontainer.json lefthook.yml .gitignore
git commit -m "feat(cli): scaffold Go project with Cobra, CI/CD, and tooling"
git push
```

---

### Task 2: Config Package — Types and Read

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for config types and reading**

```go
// internal/config/config_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write implementation**

```go
// internal/config/config.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add config types and Read function"
git push
```

---

### Task 3: Config Package — Write, Defaults, and Migration

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for Write, Defaults, and v0.1.0 migration**

Append to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `Write`, `WithDefaults`, `DetectTenantType` not defined

- [ ] **Step 3: Write implementation**

Add to `internal/config/config.go`:

```go
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

// migrateV010 converts legacy single-tenant config to v0.2.0 multi-tenant format.
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
```

Update `Read` to detect v0.1.0:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add Write, WithDefaults, DetectTenantType, and v0.1.0 migration"
git push
```

---

### Task 4: Config Package — Resolve with Flag Precedence

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for Resolve**

Append to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v -run TestResolve`
Expected: FAIL — `Resolve`, `FilterTenants` not defined

- [ ] **Step 3: Write implementation**

Add to `internal/config/config.go`:

```go
func Resolve(cfg *Config, flagThreads, flagRetries *int) *Config {
	out := WithDefaults(cfg)
	if flagThreads != nil {
		out.Threads = *flagThreads
	}
	if flagRetries != nil {
		out.Retries = *flagRetries
	}
	return out
}

func FilterTenants(tenants []Tenant, context string) []Tenant {
	if context == "" {
		return tenants
	}
	var filtered []Tenant
	for _, t := range tenants {
		if t.Context == context {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add Resolve with flag precedence and FilterTenants"
git push
```

---

### Task 5: Sync Types and Filter Functions

**Files:**
- Create: `internal/sync/types.go`
- Create: `internal/sync/filter.go`
- Create: `internal/sync/filter_test.go`

- [ ] **Step 1: Write failing test for filter functions**

```go
// internal/sync/filter_test.go
package sync_test

import (
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestFilterBySpace(t *testing.T) {
	apps := []sync.App{
		{ResourceID: "1", Name: "Sales", SpaceName: "Finance"},
		{ResourceID: "2", Name: "HR Report", SpaceName: "HR"},
		{ResourceID: "3", Name: "Budget", SpaceName: "Finance"},
	}

	filtered := sync.FilterBySpace(apps, "Finance")
	if len(filtered) != 2 {
		t.Fatalf("count = %d, want 2", len(filtered))
	}
}

func TestFilterByApp(t *testing.T) {
	apps := []sync.App{
		{ResourceID: "1", Name: "Sales Dashboard"},
		{ResourceID: "2", Name: "HR Report"},
		{ResourceID: "3", Name: "Sales Summary"},
	}

	filtered := sync.FilterByApp(apps, "Sales")
	if len(filtered) != 2 {
		t.Fatalf("count = %d, want 2", len(filtered))
	}
}

func TestFilterByID(t *testing.T) {
	apps := []sync.App{
		{ResourceID: "abc-123", Name: "Sales"},
		{ResourceID: "def-456", Name: "HR"},
	}

	filtered := sync.FilterByID(apps, "abc-123")
	if len(filtered) != 1 {
		t.Fatalf("count = %d, want 1", len(filtered))
	}
	if filtered[0].Name != "Sales" {
		t.Errorf("name = %q, want %q", filtered[0].Name, "Sales")
	}
}

func TestApplyFilters(t *testing.T) {
	apps := []sync.App{
		{ResourceID: "1", Name: "Sales Dashboard", SpaceName: "Finance"},
		{ResourceID: "2", Name: "HR Report", SpaceName: "HR"},
		{ResourceID: "3", Name: "Sales Summary", SpaceName: "Finance"},
	}

	filtered := sync.ApplyFilters(apps, sync.Filters{
		Space: "Finance",
		App:   "Summary",
	})
	if len(filtered) != 1 {
		t.Fatalf("count = %d, want 1", len(filtered))
	}
	if filtered[0].ResourceID != "3" {
		t.Errorf("id = %q, want %q", filtered[0].ResourceID, "3")
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Sales/Revenue", "Sales_Revenue"},
		{`My App: "Test"`, "My App_ _Test_"},
		{"Normal Name", "Normal Name"},
	}
	for _, tt := range tests {
		got := sync.Sanitize(tt.input)
		if got != tt.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildTargetPath(t *testing.T) {
	app := sync.App{
		ResourceID: "app-001",
		Name:       "Sales Dashboard",
		SpaceName:  "Finance Prod",
		SpaceID:    "space-001",
		SpaceType:  "managed",
		AppType:    "analytics",
		Tenant:     "my-tenant",
		TenantID:   "abc-123",
	}

	got := sync.BuildTargetPath(app)
	want := "my-tenant (abc-123)/managed/Finance Prod (space-001)/analytics/Sales Dashboard (app-001)"
	if got != want {
		t.Errorf("BuildTargetPath = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write types**

```go
// internal/sync/types.go
package sync

type App struct {
	ResourceID     string   `json:"resourceId"`
	Name           string   `json:"name"`
	SpaceID        string   `json:"spaceId"`
	SpaceName      string   `json:"spaceName"`
	SpaceType      string   `json:"spaceType"`
	AppType        string   `json:"appType"`
	OwnerID        string   `json:"ownerId"`
	OwnerName      string   `json:"ownerName"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Published      bool     `json:"published"`
	LastReloadTime string   `json:"lastReloadTime"`
	Tenant         string   `json:"tenant"`
	TenantID       string   `json:"tenantId"`
	TargetPath     string   `json:"targetPath"`
	Skip           bool     `json:"skip"`
	SkipReason     string   `json:"skipReason"`
}

type Filters struct {
	Space  string
	Stream string
	App    string
	ID     string
}

type Result struct {
	ResourceID string `json:"resourceId"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type PrepOutput struct {
	Tenant   string `json:"tenant"`
	TenantID string `json:"tenantId"`
	Context  string `json:"context"`
	Server   string `json:"server"`
	Apps     []App  `json:"apps"`
}

type IndexEntry struct {
	Name           string   `json:"name"`
	Space          string   `json:"space"`
	SpaceID        string   `json:"spaceId"`
	SpaceType      string   `json:"spaceType"`
	AppType        string   `json:"appType"`
	Owner          string   `json:"owner"`
	OwnerName      string   `json:"ownerName"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Published      bool     `json:"published"`
	LastReloadTime string   `json:"lastReloadTime"`
	Path           string   `json:"path"`
}

type Index struct {
	LastSync string                `json:"lastSync"`
	Context  string                `json:"context"`
	Server   string                `json:"server"`
	Tenant   string                `json:"tenant"`
	TenantID string                `json:"tenantId"`
	AppCount int                   `json:"appCount"`
	Apps     map[string]IndexEntry `json:"apps"`
}
```

- [ ] **Step 4: Write filter implementation**

```go
// internal/sync/filter.go
package sync

import (
	"path/filepath"
	"regexp"
	"strings"
)

func FilterBySpace(apps []App, space string) []App {
	var out []App
	for _, a := range apps {
		if a.SpaceName == space {
			out = append(out, a)
		}
	}
	return out
}

func FilterByApp(apps []App, pattern string) []App {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	var out []App
	for _, a := range apps {
		if re.MatchString(a.Name) {
			out = append(out, a)
		}
	}
	return out
}

func FilterByID(apps []App, id string) []App {
	var out []App
	for _, a := range apps {
		if a.ResourceID == id {
			out = append(out, a)
		}
	}
	return out
}

func ApplyFilters(apps []App, f Filters) []App {
	result := apps
	if f.ID != "" {
		return FilterByID(result, f.ID)
	}
	if f.Space != "" {
		result = FilterBySpace(result, f.Space)
	}
	if f.Stream != "" {
		result = FilterBySpace(result, f.Stream)
	}
	if f.App != "" {
		result = FilterByApp(result, f.App)
	}
	return result
}

func Sanitize(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		`"`, "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

func BuildTargetPath(app App) string {
	tenant := Sanitize(app.Tenant) + " (" + app.TenantID + ")"
	space := Sanitize(app.SpaceName) + " (" + app.SpaceID + ")"
	appDir := Sanitize(app.Name) + " (" + app.ResourceID + ")"

	if app.AppType != "" {
		return filepath.Join(tenant, app.SpaceType, space, app.AppType, appDir)
	}
	return filepath.Join(tenant, app.SpaceType, space, appDir)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/sync/
git commit -m "feat(sync): add App types, filter functions, and path builder"
git push
```

---

### Task 6: Resume Detection

**Files:**
- Create: `internal/sync/resume.go`
- Create: `internal/sync/resume_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/sync/resume_test.go
package sync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestMarkSkipped(t *testing.T) {
	dir := t.TempDir()

	apps := []sync.App{
		{ResourceID: "1", Name: "Synced App", TargetPath: "tenant/managed/space/app1"},
		{ResourceID: "2", Name: "New App", TargetPath: "tenant/managed/space/app2"},
	}

	// Create config.yml for app1 (already synced)
	app1Dir := filepath.Join(dir, "tenant/managed/space/app1")
	if err := os.MkdirAll(app1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app1Dir, "config.yml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	marked := sync.MarkSkipped(apps, dir)

	if !marked[0].Skip {
		t.Error("app1 should be marked as skip")
	}
	if marked[0].SkipReason != "already synced" {
		t.Errorf("skipReason = %q, want %q", marked[0].SkipReason, "already synced")
	}
	if marked[1].Skip {
		t.Error("app2 should not be skipped")
	}
}

func TestMarkSkipped_Force(t *testing.T) {
	dir := t.TempDir()

	apps := []sync.App{
		{ResourceID: "1", Name: "Synced App", TargetPath: "tenant/managed/space/app1"},
	}

	app1Dir := filepath.Join(dir, "tenant/managed/space/app1")
	if err := os.MkdirAll(app1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app1Dir, "config.yml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	marked := sync.MarkSkippedForce(apps)

	if marked[0].Skip {
		t.Error("force mode should not skip any app")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run TestMarkSkip`
Expected: FAIL — `MarkSkipped`, `MarkSkippedForce` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/sync/resume.go
package sync

import (
	"os"
	"path/filepath"
)

func MarkSkipped(apps []App, configDir string) []App {
	out := make([]App, len(apps))
	copy(out, apps)

	for i := range out {
		targetDir := filepath.Join(configDir, out[i].TargetPath)
		if fileExists(filepath.Join(targetDir, "config.yml")) || fileExists(filepath.Join(targetDir, "script.qvs")) {
			out[i].Skip = true
			out[i].SkipReason = "already synced"
		}
	}
	return out
}

func MarkSkippedForce(apps []App) []App {
	out := make([]App, len(apps))
	copy(out, apps)
	for i := range out {
		out[i].Skip = false
		out[i].SkipReason = ""
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sync/resume.go internal/sync/resume_test.go
git commit -m "feat(sync): add resume detection with skip marking"
git push
```

---

### Task 7: Cloud Prep — Fetch App List via qlik-cli

**Files:**
- Create: `internal/sync/cloud.go`
- Create: `internal/sync/cloud_test.go`
- Create: `testdata/cloud/spaces.json`
- Create: `testdata/cloud/apps.json`

- [ ] **Step 1: Create test fixtures**

```json
// testdata/cloud/spaces.json
[
  {"id": "space-001", "name": "Finance Prod", "type": "managed"},
  {"id": "space-002", "name": "HR", "type": "shared"}
]
```

```json
// testdata/cloud/apps.json
[
  {
    "resourceId": "app-001",
    "name": "Sales Dashboard",
    "resourceType": "app",
    "spaceId": "space-001",
    "ownerId": "user-001",
    "description": "Monthly sales KPIs",
    "resourceAttributes": {
      "usage": "ANALYTICS",
      "lastReloadTime": "2026-04-08T02:00:00Z"
    },
    "resourceCreatedAt": "2026-01-01T00:00:00Z"
  },
  {
    "resourceId": "app-002",
    "name": "HR Report",
    "resourceType": "app",
    "spaceId": "space-002",
    "ownerId": "user-002",
    "description": "Employee metrics",
    "resourceAttributes": {
      "usage": "ANALYTICS",
      "lastReloadTime": "2026-04-07T12:00:00Z"
    },
    "resourceCreatedAt": "2026-02-01T00:00:00Z"
  }
]
```

- [ ] **Step 2: Write failing test**

```go
// internal/sync/cloud_test.go
package sync_test

import (
	"os"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestParseCloudSpaces(t *testing.T) {
	data, err := os.ReadFile("../../testdata/cloud/spaces.json")
	if err != nil {
		t.Fatal(err)
	}

	spaces, err := sync.ParseCloudSpaces(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spaces) != 2 {
		t.Fatalf("count = %d, want 2", len(spaces))
	}
	if spaces["space-001"].Name != "Finance Prod" {
		t.Errorf("name = %q, want %q", spaces["space-001"].Name, "Finance Prod")
	}
}

func TestParseCloudApps(t *testing.T) {
	spacesData, err := os.ReadFile("../../testdata/cloud/spaces.json")
	if err != nil {
		t.Fatal(err)
	}
	spaces, err := sync.ParseCloudSpaces(spacesData)
	if err != nil {
		t.Fatal(err)
	}

	appsData, err := os.ReadFile("../../testdata/cloud/apps.json")
	if err != nil {
		t.Fatal(err)
	}

	apps, err := sync.ParseCloudApps(appsData, spaces, "my-tenant", "tenant-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("count = %d, want 2", len(apps))
	}

	app := apps[0]
	if app.ResourceID != "app-001" {
		t.Errorf("resourceId = %q, want %q", app.ResourceID, "app-001")
	}
	if app.SpaceName != "Finance Prod" {
		t.Errorf("spaceName = %q, want %q", app.SpaceName, "Finance Prod")
	}
	if app.SpaceType != "managed" {
		t.Errorf("spaceType = %q, want %q", app.SpaceType, "managed")
	}
	if app.AppType != "analytics" {
		t.Errorf("appType = %q, want %q", app.AppType, "analytics")
	}
}

func TestNormalizeAppType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ANALYTICS", "analytics"},
		{"DATAFLOW_PREP", "dataflow-prep"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sync.NormalizeAppType(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeAppType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run "TestParseCloud|TestNormalize"`
Expected: FAIL — `ParseCloudSpaces`, `ParseCloudApps`, `NormalizeAppType` not defined

- [ ] **Step 4: Write implementation**

```go
// internal/sync/cloud.go
package sync

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SpaceInfo struct {
	ID   string
	Name string
	Type string
}

type cloudAppRaw struct {
	ResourceID         string `json:"resourceId"`
	Name               string `json:"name"`
	SpaceID            string `json:"spaceId"`
	OwnerID            string `json:"ownerId"`
	Description        string `json:"description"`
	ResourceAttributes struct {
		Usage          string `json:"usage"`
		LastReloadTime string `json:"lastReloadTime"`
	} `json:"resourceAttributes"`
}

func ParseCloudSpaces(data []byte) (map[string]SpaceInfo, error) {
	var raw []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing spaces: %w", err)
	}

	spaces := make(map[string]SpaceInfo, len(raw))
	for _, s := range raw {
		spaces[s.ID] = SpaceInfo{ID: s.ID, Name: s.Name, Type: s.Type}
	}
	return spaces, nil
}

func ParseCloudApps(data []byte, spaces map[string]SpaceInfo, tenant, tenantID string) ([]App, error) {
	var raw []cloudAppRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing apps: %w", err)
	}

	apps := make([]App, 0, len(raw))
	for _, r := range raw {
		space := spaces[r.SpaceID]
		app := App{
			ResourceID:     r.ResourceID,
			Name:           r.Name,
			SpaceID:        r.SpaceID,
			SpaceName:      space.Name,
			SpaceType:      space.Type,
			AppType:        NormalizeAppType(r.ResourceAttributes.Usage),
			OwnerID:        r.OwnerID,
			Description:    r.Description,
			LastReloadTime: r.ResourceAttributes.LastReloadTime,
			Tenant:         tenant,
			TenantID:       tenantID,
		}
		app.TargetPath = BuildTargetPath(app)
		apps = append(apps, app)
	}
	return apps, nil
}

func NormalizeAppType(usage string) string {
	if usage == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(usage, "_", "-"))
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/sync/cloud.go internal/sync/cloud_test.go testdata/
git commit -m "feat(sync): add cloud space/app parsing and app type normalization"
git push
```

---

### Task 8: Finalize — Build Index and Update Config

**Files:**
- Create: `internal/sync/finalize.go`
- Create: `internal/sync/finalize_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/sync/finalize_test.go
package sync_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestBuildIndex(t *testing.T) {
	prep := sync.PrepOutput{
		Tenant:   "my-tenant",
		TenantID: "abc-123",
		Context:  "my-context",
		Server:   "https://tenant.qlikcloud.com",
		Apps: []sync.App{
			{
				ResourceID: "app-001",
				Name:       "Sales Dashboard",
				SpaceName:  "Finance",
				SpaceID:    "space-001",
				SpaceType:  "managed",
				AppType:    "analytics",
				OwnerID:    "user-001",
				OwnerName:  "jane.doe",
				TargetPath: "my-tenant (abc-123)/managed/Finance (space-001)/analytics/Sales Dashboard (app-001)",
			},
			{
				ResourceID: "app-002",
				Name:       "HR Report",
				SpaceName:  "HR",
				SpaceID:    "space-002",
				SpaceType:  "shared",
				AppType:    "analytics",
				OwnerID:    "user-002",
				OwnerName:  "john.smith",
				TargetPath: "my-tenant (abc-123)/shared/HR (space-002)/analytics/HR Report (app-002)",
			},
		},
	}

	results := []sync.Result{
		{ResourceID: "app-001", Status: "synced"},
		{ResourceID: "app-002", Status: "skipped"},
	}

	index := sync.BuildIndex(prep, results)

	if index.Tenant != "my-tenant" {
		t.Errorf("tenant = %q, want %q", index.Tenant, "my-tenant")
	}
	if index.AppCount != 2 {
		t.Errorf("appCount = %d, want 2", index.AppCount)
	}
	if _, ok := index.Apps["app-001"]; !ok {
		t.Fatal("app-001 missing from index")
	}
	if index.Apps["app-001"].Space != "Finance" {
		t.Errorf("space = %q, want %q", index.Apps["app-001"].Space, "Finance")
	}
}

func TestWriteIndex(t *testing.T) {
	dir := t.TempDir()
	index := sync.Index{
		Tenant:   "test",
		TenantID: "123",
		AppCount: 1,
		Apps: map[string]sync.IndexEntry{
			"app-001": {Name: "Test App", Path: "test/path"},
		},
	}

	if err := sync.WriteIndex(dir, index); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got sync.Index
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.AppCount != 1 {
		t.Errorf("appCount = %d, want 1", got.AppCount)
	}
}

func TestMergeIndex(t *testing.T) {
	existing := sync.Index{
		Tenant:   "test",
		AppCount: 1,
		Apps: map[string]sync.IndexEntry{
			"old-app": {Name: "Old App"},
		},
	}

	new := sync.Index{
		Tenant:   "test",
		AppCount: 1,
		Apps: map[string]sync.IndexEntry{
			"new-app": {Name: "New App"},
		},
	}

	merged := sync.MergeIndex(existing, new)
	if merged.AppCount != 2 {
		t.Errorf("appCount = %d, want 2", merged.AppCount)
	}
	if _, ok := merged.Apps["old-app"]; !ok {
		t.Error("old-app missing after merge")
	}
	if _, ok := merged.Apps["new-app"]; !ok {
		t.Error("new-app missing after merge")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run "TestBuildIndex|TestWriteIndex|TestMergeIndex"`
Expected: FAIL — `BuildIndex`, `WriteIndex`, `MergeIndex` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/sync/finalize.go
package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func BuildIndex(prep PrepOutput, results []Result) Index {
	apps := make(map[string]IndexEntry, len(prep.Apps))
	for _, a := range prep.Apps {
		apps[a.ResourceID] = IndexEntry{
			Name:           a.Name,
			Space:          a.SpaceName,
			SpaceID:        a.SpaceID,
			SpaceType:      a.SpaceType,
			AppType:        a.AppType,
			Owner:          a.OwnerID,
			OwnerName:      a.OwnerName,
			Description:    a.Description,
			Tags:           a.Tags,
			Published:      a.Published,
			LastReloadTime: a.LastReloadTime,
			Path:           a.TargetPath,
		}
	}

	return Index{
		LastSync: time.Now().UTC().Format(time.RFC3339),
		Context:  prep.Context,
		Server:   prep.Server,
		Tenant:   prep.Tenant,
		TenantID: prep.TenantID,
		AppCount: len(apps),
		Apps:     apps,
	}
}

func WriteIndex(dir string, index Index) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	path := filepath.Join(dir, "index.json")
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}

	return nil
}

func ReadIndex(dir string) (Index, error) {
	path := filepath.Join(dir, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Index{Apps: make(map[string]IndexEntry)}, nil
		}
		return Index{}, fmt.Errorf("reading index: %w", err)
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return Index{}, fmt.Errorf("parsing index: %w", err)
	}

	return index, nil
}

func MergeIndex(existing, new Index) Index {
	merged := new
	if merged.Apps == nil {
		merged.Apps = make(map[string]IndexEntry)
	}

	for id, entry := range existing.Apps {
		if _, exists := merged.Apps[id]; !exists {
			merged.Apps[id] = entry
		}
	}

	merged.AppCount = len(merged.Apps)
	return merged
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sync/finalize.go internal/sync/finalize_test.go
git commit -m "feat(sync): add index building, writing, and merging"
git push
```

---

### Task 9: Runner — Parallel Execution with Retries

**Files:**
- Create: `internal/sync/runner.go`
- Create: `internal/sync/runner_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/sync/runner_test.go
package sync_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestRunParallel(t *testing.T) {
	apps := []qsync.App{
		{ResourceID: "1", Name: "App1", TargetPath: "path/1"},
		{ResourceID: "2", Name: "App2", TargetPath: "path/2"},
		{ResourceID: "3", Name: "App3", TargetPath: "path/3", Skip: true},
	}

	var called atomic.Int32
	syncFn := func(ctx context.Context, app qsync.App, configDir string) error {
		called.Add(1)
		return nil
	}

	results := qsync.RunParallel(context.Background(), apps, "qlik", 2, 1, syncFn)

	if int(called.Load()) != 2 {
		t.Errorf("syncFn called %d times, want 2 (skip 1)", called.Load())
	}
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	synced := 0
	skipped := 0
	for _, r := range results {
		switch r.Status {
		case "synced":
			synced++
		case "skipped":
			skipped++
		}
	}
	if synced != 2 {
		t.Errorf("synced = %d, want 2", synced)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestRunParallel_Retry(t *testing.T) {
	var attempts atomic.Int32
	syncFn := func(ctx context.Context, app qsync.App, configDir string) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	}

	apps := []qsync.App{
		{ResourceID: "1", Name: "Flaky", TargetPath: "path/1"},
	}

	results := qsync.RunParallel(context.Background(), apps, "qlik", 1, 3, syncFn)

	if results[0].Status != "synced" {
		t.Errorf("status = %q, want %q", results[0].Status, "synced")
	}
	if int(attempts.Load()) != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
}

func TestRunParallel_ExhaustedRetries(t *testing.T) {
	syncFn := func(ctx context.Context, app qsync.App, configDir string) error {
		return errors.New("permanent error")
	}

	apps := []qsync.App{
		{ResourceID: "1", Name: "Broken", TargetPath: "path/1"},
	}

	results := qsync.RunParallel(context.Background(), apps, "qlik", 1, 2, syncFn)

	if results[0].Status != "error" {
		t.Errorf("status = %q, want %q", results[0].Status, "error")
	}
	if results[0].Error == "" {
		t.Error("expected error message, got empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run TestRunParallel`
Expected: FAIL — `RunParallel` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/sync/runner.go
package sync

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type SyncFunc func(ctx context.Context, app App, configDir string) error

func RunParallel(ctx context.Context, apps []App, configDir string, threads, retries int, fn SyncFunc) []Result {
	results := make([]Result, len(apps))
	sem := make(chan struct{}, threads)
	var wg sync.WaitGroup

	for i, app := range apps {
		if app.Skip {
			results[i] = Result{
				ResourceID: app.ResourceID,
				Status:     "skipped",
			}
			continue
		}

		wg.Add(1)
		go func(idx int, a App) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := retry(ctx, retries, func() error {
				return fn(ctx, a, configDir)
			})

			if err != nil {
				results[idx] = Result{
					ResourceID: a.ResourceID,
					Status:     "error",
					Error:      err.Error(),
				}
			} else {
				results[idx] = Result{
					ResourceID: a.ResourceID,
					Status:     "synced",
				}
			}
		}(i, app)
	}

	wg.Wait()
	return results
}

func retry(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("cancelled: %w", err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt < maxAttempts {
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("cancelled during backoff: %w", ctx.Err())
			}
		}
	}
	return lastErr
}

func Summarize(results []Result) (synced, skipped, errors int) {
	for _, r := range results {
		switch r.Status {
		case "synced":
			synced++
		case "skipped":
			skipped++
		case "error":
			errors++
		}
	}
	return
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sync/runner.go internal/sync/runner_test.go
git commit -m "feat(sync): add parallel runner with retries and exponential backoff"
git push
```

---

### Task 10: Cloud App Sync — qlik-cli Shelling

**Files:**
- Create: `internal/sync/exec.go`
- Create: `internal/sync/exec_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/sync/exec_test.go
package sync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestCloudSyncApp(t *testing.T) {
	// Skip if qlik-cli not available (CI won't have it)
	if _, err := exec.LookPath("qlik"); err != nil {
		t.Skip("qlik-cli not in PATH, skipping integration test")
	}
	t.Skip("requires live qlik-cli auth — run manually")
}

func TestBuildUnbuildArgs(t *testing.T) {
	args := qsync.BuildUnbuildArgs("app-001", "/tmp/test/path")
	expected := []string{"app", "unbuild", "--app", "app-001", "--dir", "/tmp/test/path"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d", len(args), len(expected))
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildSpaceListArgs(t *testing.T) {
	args := qsync.BuildSpaceListArgs()
	expected := []string{"space", "ls", "--json"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d", len(args), len(expected))
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildAppListArgs(t *testing.T) {
	t.Run("no space filter", func(t *testing.T) {
		args := qsync.BuildAppListArgs("")
		expected := []string{"app", "ls", "--json", "--limit", "1000"}
		if len(args) != len(expected) {
			t.Fatalf("args len = %d, want %d", len(args), len(expected))
		}
	})

	t.Run("with space filter", func(t *testing.T) {
		args := qsync.BuildAppListArgs("space-001")
		expected := []string{"app", "ls", "--json", "--limit", "1000", "--spaceId", "space-001"}
		if len(args) != len(expected) {
			t.Fatalf("args len = %d, want %d", len(args), len(expected))
		}
	})
}

func TestCheckPrerequisites(t *testing.T) {
	// This test checks the function signature exists
	// Actual behavior depends on PATH
	err := qsync.CheckPrerequisites()
	// Don't assert pass/fail — depends on environment
	_ = err
}

func TestRunQlikCmd(t *testing.T) {
	// Create a mock script that echoes JSON
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "qlik")
	script := "#!/bin/sh\necho '[{\"id\":\"test\"}]'\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	out, err := qsync.RunQlikCmd(context.Background(), mockPath, "space", "ls", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `[{"id":"test"}]`+"\n" {
		t.Errorf("output = %q, want %q", string(out), `[{"id":"test"}]`+"\n")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run "TestBuild|TestCheck|TestRunQlik"`
Expected: FAIL — `BuildUnbuildArgs`, `BuildSpaceListArgs`, etc. not defined

- [ ] **Step 3: Write implementation**

```go
// internal/sync/exec.go
package sync

import (
	"context"
	"fmt"
	"os/exec"
)

func CheckPrerequisites() error {
	if _, err := exec.LookPath("qlik"); err != nil {
		return fmt.Errorf("qlik-cli not found in PATH\n  Install: https://qlik.dev/toolkits/qlik-cli/")
	}
	return nil
}

func RunQlikCmd(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("qlik %v failed: %s", args, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("qlik %v failed: %w", args, err)
	}
	return out, nil
}

func BuildSpaceListArgs() []string {
	return []string{"space", "ls", "--json"}
}

func BuildAppListArgs(spaceID string) []string {
	args := []string{"app", "ls", "--json", "--limit", "1000"}
	if spaceID != "" {
		args = append(args, "--spaceId", spaceID)
	}
	return args
}

func BuildUnbuildArgs(resourceID, targetDir string) []string {
	return []string{"app", "unbuild", "--app", resourceID, "--dir", targetDir}
}

func CloudSyncApp(ctx context.Context, app App, configDir string) error {
	targetDir := fmt.Sprintf("%s/%s", configDir, app.TargetPath)
	args := BuildUnbuildArgs(app.ResourceID, targetDir)
	_, err := RunQlikCmd(ctx, "qlik", args...)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v -race`
Expected: PASS (integration tests skipped)

- [ ] **Step 5: Commit**

```bash
git add internal/sync/exec.go internal/sync/exec_test.go
git commit -m "feat(sync): add qlik-cli command builders and execution"
git push
```

---

### Task 11: Sync Command — Wire Everything Together

**Files:**
- Create: `cmd/sync.go`
- Create: `cmd/sync_test.go`

- [ ] **Step 1: Write failing test**

```go
// cmd/sync_test.go
package cmd

import (
	"testing"
)

func TestSyncCmdFlags(t *testing.T) {
	cmd := syncCmd

	flags := []struct {
		name     string
		defValue string
	}{
		{"space", ""},
		{"stream", ""},
		{"app", ""},
		{"id", ""},
		{"tenant", ""},
		{"threads", "0"},
		{"retries", "0"},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag %q not found", f.name)
			continue
		}
		if flag.DefValue != f.defValue {
			t.Errorf("flag %q default = %q, want %q", f.name, flag.DefValue, f.defValue)
		}
	}

	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("flag 'force' not found")
	}
}

func TestSyncCmdRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "sync" {
			found = true
			break
		}
	}
	if !found {
		t.Error("sync command not registered on root")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -v -run TestSyncCmd`
Expected: FAIL — `syncCmd` not defined

- [ ] **Step 3: Write implementation**

```go
// cmd/sync.go
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	"github.com/mattiasthalen/qlik-sync/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncSpace   string
	syncStream  string
	syncApp     string
	syncID      string
	syncTenant  string
	syncForce   bool
	syncThreads int
	syncRetries int
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Qlik apps to local files",
	Long:  "Pull Qlik Sense cloud and on-prem apps into the local qlik/ directory.",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncSpace, "space", "", "filter by space name (cloud)")
	syncCmd.Flags().StringVar(&syncStream, "stream", "", "filter by stream name (on-prem)")
	syncCmd.Flags().StringVar(&syncApp, "app", "", "regex filter on app name")
	syncCmd.Flags().StringVar(&syncID, "id", "", "exact app ID")
	syncCmd.Flags().StringVar(&syncTenant, "tenant", "", "filter by tenant context")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip cache, re-sync all")
	syncCmd.Flags().IntVar(&syncThreads, "threads", 0, "concurrent syncs (overrides config)")
	syncCmd.Flags().IntVar(&syncRetries, "retries", 0, "retry count per app (overrides config)")

	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := sync.CheckPrerequisites(); err != nil {
		return err
	}

	cfg, err := config.Read(configDir)
	if err != nil {
		return fmt.Errorf("reading config: %w\n  Run: qs setup", err)
	}

	var flagThreads, flagRetries *int
	if cmd.Flags().Changed("threads") {
		flagThreads = &syncThreads
	}
	if cmd.Flags().Changed("retries") {
		flagRetries = &syncRetries
	}
	resolved := config.Resolve(cfg, flagThreads, flagRetries)

	tenants := config.FilterTenants(resolved.Tenants, syncTenant)
	if len(tenants) == 0 {
		return fmt.Errorf("no tenants found for context %q\n  Run: qs setup", syncTenant)
	}

	ctx := context.Background()
	filters := sync.Filters{
		Space:  syncSpace,
		Stream: syncStream,
		App:    syncApp,
		ID:     syncID,
	}

	exitCode := 0
	for _, tenant := range tenants {
		if tenant.Type != "cloud" {
			fmt.Fprintf(os.Stderr, "Skipping on-prem tenant %q (not yet supported)\n", tenant.Context)
			continue
		}

		spacesOut, err := sync.RunQlikCmd(ctx, "qlik", sync.BuildSpaceListArgs()...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching spaces for %q: %v\n", tenant.Context, err)
			exitCode = 2
			continue
		}
		spaces, err := sync.ParseCloudSpaces(spacesOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing spaces for %q: %v\n", tenant.Context, err)
			exitCode = 2
			continue
		}

		spaceID := ""
		if syncSpace != "" {
			for _, s := range spaces {
				if s.Name == syncSpace {
					spaceID = s.ID
					break
				}
			}
		}

		appsOut, err := sync.RunQlikCmd(ctx, "qlik", sync.BuildAppListArgs(spaceID)...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching apps for %q: %v\n", tenant.Context, err)
			exitCode = 2
			continue
		}

		tenantName := tenant.Context
		tenantID := ""
		apps, err := sync.ParseCloudApps(appsOut, spaces, tenantName, tenantID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing apps for %q: %v\n", tenant.Context, err)
			exitCode = 2
			continue
		}

		apps = sync.ApplyFilters(apps, filters)

		if syncForce {
			apps = sync.MarkSkippedForce(apps)
		} else {
			apps = sync.MarkSkipped(apps, configDir)
		}

		fmt.Printf("Syncing %s (%d apps, %d threads)...\n", tenant.Context, len(apps), resolved.Threads)

		results := sync.RunParallel(ctx, apps, configDir, resolved.Threads, resolved.Retries, sync.CloudSyncApp)

		prep := sync.PrepOutput{
			Tenant:   tenantName,
			TenantID: tenantID,
			Context:  tenant.Context,
			Server:   tenant.Server,
			Apps:     apps,
		}

		index := sync.BuildIndex(prep, results)
		existing, _ := sync.ReadIndex(configDir)
		merged := sync.MergeIndex(existing, index)
		if err := sync.WriteIndex(configDir, merged); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing index: %v\n", err)
		}

		synced, skipped, errors := sync.Summarize(results)
		fmt.Printf("%d synced, %d skipped, %d errors\n", synced, skipped, errors)

		if errors > 0 {
			exitCode = 2
		}
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -v -race`
Expected: PASS

- [ ] **Step 5: Verify CLI builds with sync command**

Run: `go build -o qs . && ./qs sync --help`
Expected: help text showing all sync flags

- [ ] **Step 6: Commit**

```bash
git add cmd/sync.go cmd/sync_test.go
git commit -m "feat(cli): add sync command with cloud orchestration"
git push
```

---

### Task 12: Setup Command

**Files:**
- Create: `cmd/setup.go`
- Create: `cmd/setup_test.go`

- [ ] **Step 1: Write failing test**

```go
// cmd/setup_test.go
package cmd

import (
	"testing"
)

func TestSetupCmdRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "setup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("setup command not registered on root")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -v -run TestSetupCmd`
Expected: FAIL — setup command not registered

- [ ] **Step 3: Write implementation**

```go
// cmd/setup.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Qlik tenant connection",
	Long:  "Interactive setup for connecting to a Qlik Cloud or on-prem tenant.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	if err := qsync.CheckPrerequisites(); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	// List existing contexts
	out, err := qsync.RunQlikCmd(cmd.Context(), "qlik", "context", "ls")
	if err == nil {
		fmt.Printf("Existing qlik contexts:\n%s\n", string(out))
	}

	fmt.Print("Enter qlik context name (existing or new): ")
	contextName, _ := reader.ReadString('\n')
	contextName = strings.TrimSpace(contextName)

	fmt.Print("Enter server URL (e.g., https://tenant.qlikcloud.com): ")
	server, _ := reader.ReadString('\n')
	server = strings.TrimSpace(server)

	tenantType := config.DetectTenantType(server)
	fmt.Printf("Detected tenant type: %s\n", tenantType)

	// Check if context exists, create if not
	checkCmd := exec.Command("qlik", "context", "get", contextName)
	if err := checkCmd.Run(); err != nil {
		fmt.Print("Enter API key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		createArgs := []string{"context", "create", contextName, "--server", server, "--api-key", apiKey}
		if tenantType == "on-prem" {
			createArgs = append(createArgs, "--server-type", "Windows", "--insecure")
		}
		if _, err := qsync.RunQlikCmd(cmd.Context(), "qlik", createArgs...); err != nil {
			return fmt.Errorf("creating context: %w", err)
		}
		fmt.Println("Context created.")
	}

	// Set active context
	if _, err := qsync.RunQlikCmd(cmd.Context(), "qlik", "context", "use", contextName); err != nil {
		return fmt.Errorf("setting active context: %w", err)
	}

	// Test connectivity
	fmt.Println("Testing connectivity...")
	if tenantType == "cloud" {
		if _, err := qsync.RunQlikCmd(cmd.Context(), "qlik", "app", "ls", "--limit", "1"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	} else {
		if _, err := qsync.RunQlikCmd(cmd.Context(), "qlik", "qrs", "app", "count"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	}
	fmt.Println("Connected successfully.")

	// Read or create config
	cfg, err := config.Read(configDir)
	if err != nil {
		cfg = &config.Config{
			Version: "0.2.0",
		}
	}

	// Add tenant (or update existing)
	found := false
	for i, t := range cfg.Tenants {
		if t.Context == contextName {
			cfg.Tenants[i].Server = server
			cfg.Tenants[i].Type = tenantType
			found = true
			break
		}
	}
	if !found {
		cfg.Tenants = append(cfg.Tenants, config.Tenant{
			Context: contextName,
			Server:  server,
			Type:    tenantType,
		})
	}

	if err := config.Write(configDir, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nSetup complete. Run: qs sync\n")
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -v -race`
Expected: PASS

- [ ] **Step 5: Verify CLI builds with setup command**

Run: `go build -o qs . && ./qs setup --help`
Expected: help text for setup

- [ ] **Step 6: Commit**

```bash
git add cmd/setup.go cmd/setup_test.go
git commit -m "feat(cli): add interactive setup command"
git push
```

---

### Task 13: QVF Parser — Extract Scripts, Measures, Dimensions, Variables

**Files:**
- Create: `internal/parser/qvf.go`
- Create: `internal/parser/qvf_test.go`
- Create: `testdata/parser/minimal.qvf` (generated by test helper)

- [ ] **Step 1: Write failing test**

```go
// internal/parser/qvf_test.go
package parser_test

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func TestExtractScript(t *testing.T) {
	scriptJSON := map[string]string{"qScript": "LOAD * FROM data.qvd;"}
	data, _ := json.Marshal(scriptJSON)
	compressed := zlibCompress(data)

	// Build a minimal byte stream with zlib block
	var input bytes.Buffer
	input.Write([]byte{0x00, 0x00}) // padding
	input.Write(compressed)
	input.Write([]byte{0x00, 0x00}) // padding

	result, err := parser.ExtractQVF(bytes.NewReader(input.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q, want %q", result.Script, "LOAD * FROM data.qvd;")
	}
}

func TestExtractMeasures(t *testing.T) {
	measures := []map[string]interface{}{
		{
			"qMeasure": map[string]interface{}{
				"qLabel": "Total Sales",
				"qDef":   "Sum(Sales)",
			},
		},
	}
	block := map[string]interface{}{"qMeasureList": measures}
	data, _ := json.Marshal(block)
	compressed := zlibCompress(data)

	var input bytes.Buffer
	input.Write(compressed)

	result, err := parser.ExtractQVF(bytes.NewReader(input.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Measures) != 1 {
		t.Fatalf("measures count = %d, want 1", len(result.Measures))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parser/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write implementation**

```go
// internal/parser/qvf.go
package parser

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
)

type QVFResult struct {
	Script     string            `json:"script,omitempty"`
	Measures   []json.RawMessage `json:"measures,omitempty"`
	Dimensions []json.RawMessage `json:"dimensions,omitempty"`
	Variables  []json.RawMessage `json:"variables,omitempty"`
}

// zlibMarkers are the second bytes of zlib headers (first byte is always 0x78)
var zlibMarkers = []byte{0x01, 0x5E, 0x9C, 0xDA}

func ExtractQVF(r io.Reader) (*QVFResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading QVF data: %w", err)
	}

	result := &QVFResult{}
	blocks := findZlibBlocks(data)

	for _, block := range blocks {
		decompressed, err := decompressZlib(block)
		if err != nil {
			continue
		}

		parseQVFBlock(decompressed, result)
	}

	return result, nil
}

func findZlibBlocks(data []byte) [][]byte {
	var blocks [][]byte
	for i := 0; i < len(data)-1; i++ {
		if data[i] != 0x78 {
			continue
		}
		for _, marker := range zlibMarkers {
			if data[i+1] == marker {
				block := data[i:]
				blocks = append(blocks, block)
				break
			}
		}
	}
	return blocks
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func parseQVFBlock(data []byte, result *QVFResult) {
	// Try script extraction
	var scriptBlock struct {
		QScript string `json:"qScript"`
	}
	if json.Unmarshal(data, &scriptBlock) == nil && scriptBlock.QScript != "" {
		result.Script = scriptBlock.QScript
	}

	// Try measures
	var measuresBlock struct {
		QMeasureList []json.RawMessage `json:"qMeasureList"`
	}
	if json.Unmarshal(data, &measuresBlock) == nil && len(measuresBlock.QMeasureList) > 0 {
		result.Measures = measuresBlock.QMeasureList
	}

	// Try dimensions
	var dimensionsBlock struct {
		QDimensionList []json.RawMessage `json:"qDimensionList"`
	}
	if json.Unmarshal(data, &dimensionsBlock) == nil && len(dimensionsBlock.QDimensionList) > 0 {
		result.Dimensions = dimensionsBlock.QDimensionList
	}

	// Try variables
	var variablesBlock struct {
		QVariableList []json.RawMessage `json:"qVariableList"`
	}
	if json.Unmarshal(data, &variablesBlock) == nil && len(variablesBlock.QVariableList) > 0 {
		result.Variables = variablesBlock.QVariableList
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): add QVF extraction for scripts, measures, dimensions, variables"
git push
```

---

### Task 14: QVW Parser — Extract Load Scripts

**Files:**
- Create: `internal/parser/qvw.go`
- Create: `internal/parser/qvw_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/parser/qvw_test.go
package parser_test

import (
	"bytes"
	"compress/zlib"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func TestExtractQVW(t *testing.T) {
	script := "///\r\nLOAD * FROM data.qvd;\r\n"
	scriptBytes := []byte(script)

	// Compress the content
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(scriptBytes)
	w.Close()

	// Build QVW: 23-byte header + compressed data
	var qvw bytes.Buffer
	header := make([]byte, 23)
	qvw.Write(header)
	qvw.Write(compressed.Bytes())

	result, err := parser.ExtractQVW(bytes.NewReader(qvw.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "LOAD * FROM data.qvd;\r\n" {
		t.Errorf("script = %q, want %q", result.Script, "LOAD * FROM data.qvd;\r\n")
	}
}

func TestExtractQVW_NoScript(t *testing.T) {
	content := []byte("just some random data without script marker")

	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(content)
	w.Close()

	var qvw bytes.Buffer
	header := make([]byte, 23)
	qvw.Write(header)
	qvw.Write(compressed.Bytes())

	result, err := parser.ExtractQVW(bytes.NewReader(qvw.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "" {
		t.Errorf("expected empty script, got %q", result.Script)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parser/ -v -run TestExtractQVW`
Expected: FAIL — `ExtractQVW` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/parser/qvw.go
package parser

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strings"
)

const qvwHeaderSize = 23
const scriptMarker = "///"

type QVWResult struct {
	Script string `json:"script,omitempty"`
}

func ExtractQVW(r io.Reader) (*QVWResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading QVW data: %w", err)
	}

	if len(data) <= qvwHeaderSize {
		return nil, fmt.Errorf("QVW file too small: %d bytes", len(data))
	}

	// Strip 23-byte header
	compressed := data[qvwHeaderSize:]

	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("decompressing QVW: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("reading decompressed QVW: %w", err)
	}

	content := string(decompressed)
	result := &QVWResult{}

	// Find script between /// marker and end (null bytes)
	idx := strings.Index(content, scriptMarker)
	if idx == -1 {
		return result, nil
	}

	script := content[idx+len(scriptMarker):]
	// Trim leading newlines after marker
	script = strings.TrimLeft(script, "\r\n")

	// Find end of script (null bytes)
	if nullIdx := strings.Index(script, "\x00\x00"); nullIdx != -1 {
		script = script[:nullIdx]
	}

	result.Script = script
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/qvw.go internal/parser/qvw_test.go
git commit -m "feat(parser): add QVW extraction for load scripts"
git push
```

---

### Task 15: Parser — Write Artifacts to Disk

**Files:**
- Create: `internal/parser/writer.go`
- Create: `internal/parser/writer_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/parser/writer_test.go
package parser_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func TestWriteArtifacts_QVF(t *testing.T) {
	dir := t.TempDir()
	result := &parser.QVFResult{
		Script:   "LOAD * FROM data.qvd;",
		Measures: []json.RawMessage{[]byte(`{"qMeasure":{"qLabel":"Sales"}}`)},
	}

	if err := parser.WriteQVFArtifacts(dir, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check script.qvs
	script, err := os.ReadFile(filepath.Join(dir, "script.qvs"))
	if err != nil {
		t.Fatalf("script.qvs not found: %v", err)
	}
	if string(script) != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q, want %q", string(script), "LOAD * FROM data.qvd;")
	}

	// Check measures.json
	measures, err := os.ReadFile(filepath.Join(dir, "measures.json"))
	if err != nil {
		t.Fatalf("measures.json not found: %v", err)
	}
	if len(measures) == 0 {
		t.Error("measures.json is empty")
	}
}

func TestWriteArtifacts_QVW(t *testing.T) {
	dir := t.TempDir()
	result := &parser.QVWResult{
		Script: "LOAD * FROM data.qvd;",
	}

	if err := parser.WriteQVWArtifacts(dir, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	script, err := os.ReadFile(filepath.Join(dir, "script.qvs"))
	if err != nil {
		t.Fatalf("script.qvs not found: %v", err)
	}
	if string(script) != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q, want %q", string(script), "LOAD * FROM data.qvd;")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parser/ -v -run TestWriteArtifacts`
Expected: FAIL — `WriteQVFArtifacts`, `WriteQVWArtifacts` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/parser/writer.go
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteQVFArtifacts(dir string, result *QVFResult) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if result.Script != "" {
		if err := os.WriteFile(filepath.Join(dir, "script.qvs"), []byte(result.Script), 0644); err != nil {
			return fmt.Errorf("writing script: %w", err)
		}
	}

	if len(result.Measures) > 0 {
		if err := writeJSON(filepath.Join(dir, "measures.json"), result.Measures); err != nil {
			return fmt.Errorf("writing measures: %w", err)
		}
	}

	if len(result.Dimensions) > 0 {
		if err := writeJSON(filepath.Join(dir, "dimensions.json"), result.Dimensions); err != nil {
			return fmt.Errorf("writing dimensions: %w", err)
		}
	}

	if len(result.Variables) > 0 {
		if err := writeJSON(filepath.Join(dir, "variables.json"), result.Variables); err != nil {
			return fmt.Errorf("writing variables: %w", err)
		}
	}

	return nil
}

func WriteQVWArtifacts(dir string, result *QVWResult) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if result.Script != "" {
		if err := os.WriteFile(filepath.Join(dir, "script.qvs"), []byte(result.Script), 0644); err != nil {
			return fmt.Errorf("writing script: %w", err)
		}
	}

	return nil
}

func writeJSON(path string, data interface{}) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/writer.go internal/parser/writer_test.go
git commit -m "feat(parser): add artifact writer for QVF and QVW results"
git push
```

---

### Task 16: Cache — Prep Result Caching with TTL

**Files:**
- Create: `internal/sync/cache.go`
- Create: `internal/sync/cache_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/sync/cache_test.go
package sync_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestCacheWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	key := "test-cache-key"
	data := []byte(`{"tenant":"test","apps":[]}`)

	if err := qsync.CacheWrite(dir, key, data); err != nil {
		t.Fatalf("write error: %v", err)
	}

	got, err := qsync.CacheRead(dir, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("data = %q, want %q", string(got), string(data))
	}
}

func TestCacheRead_Expired(t *testing.T) {
	dir := t.TempDir()
	key := "expired-key"
	data := []byte(`{"expired":true}`)

	path := filepath.Join(dir, "qs-cache-"+key+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Set mtime to 10 minutes ago
	past := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	got, err := qsync.CacheRead(dir, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired cache, got data")
	}
}

func TestCacheRead_Missing(t *testing.T) {
	dir := t.TempDir()
	got, err := qsync.CacheRead(dir, "nonexistent", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing cache, got data")
	}
}

func TestCacheKey(t *testing.T) {
	key1 := qsync.BuildCacheKey("ctx1", "Finance", "", "", "/workdir")
	key2 := qsync.BuildCacheKey("ctx1", "HR", "", "", "/workdir")
	key3 := qsync.BuildCacheKey("ctx1", "Finance", "", "", "/workdir")

	if key1 == key2 {
		t.Error("different filters should produce different keys")
	}
	if key1 != key3 {
		t.Error("same inputs should produce same key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/ -v -run TestCache`
Expected: FAIL — `CacheWrite`, `CacheRead`, `BuildCacheKey` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/sync/cache.go
package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func BuildCacheKey(context, space, stream, app, workdir string) string {
	input := fmt.Sprintf("%s|%s|%s|%s|%s", context, space, stream, app, workdir)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash[:8])
}

func CacheWrite(dir, key string, data []byte) error {
	path := cachePath(dir, key)
	return os.WriteFile(path, data, 0644)
}

func CacheRead(dir, key string, ttl time.Duration) ([]byte, error) {
	path := cachePath(dir, key)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if time.Since(info.ModTime()) > ttl {
		return nil, nil
	}

	return os.ReadFile(path)
}

func cachePath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("qs-cache-%s.json", key))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/ -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sync/cache.go internal/sync/cache_test.go
git commit -m "feat(sync): add prep result caching with TTL"
git push
```

---

### Task 17: Integration Test — End-to-End Sync with Mock qlik-cli

**Files:**
- Create: `test/integration/sync_test.go`
- Create: `test/integration/mock-qlik.sh`
- Create: `test/integration/testdata/spaces.json`
- Create: `test/integration/testdata/apps.json`

- [ ] **Step 1: Create mock qlik-cli script**

```bash
#!/bin/sh
# test/integration/mock-qlik.sh
# Mock qlik-cli that returns fixture data

case "$*" in
  "space ls --json")
    cat "$(dirname "$0")/testdata/spaces.json"
    ;;
  "app ls --json --limit 1000")
    cat "$(dirname "$0")/testdata/apps.json"
    ;;
  "app unbuild --app"*)
    # Extract target dir from args
    DIR=$(echo "$*" | sed 's/.*--dir //')
    mkdir -p "$DIR"
    echo "resourceId: test" > "$DIR/config.yml"
    echo "LOAD * FROM test.qvd;" > "$DIR/script.qvs"
    echo "[]" > "$DIR/measures.json"
    echo "[]" > "$DIR/dimensions.json"
    echo "[]" > "$DIR/variables.json"
    ;;
  "context ls")
    echo "default *"
    ;;
  *)
    echo "mock: unknown command: $*" >&2
    exit 1
    ;;
esac
```

- [ ] **Step 2: Create test fixtures**

```json
// test/integration/testdata/spaces.json
[
  {"id": "space-001", "name": "Finance", "type": "managed"}
]
```

```json
// test/integration/testdata/apps.json
[
  {
    "resourceId": "app-001",
    "name": "Sales Dashboard",
    "resourceType": "app",
    "spaceId": "space-001",
    "ownerId": "user-001",
    "description": "Test app",
    "resourceAttributes": {
      "usage": "ANALYTICS",
      "lastReloadTime": "2026-04-08T02:00:00Z"
    },
    "resourceCreatedAt": "2026-01-01T00:00:00Z"
  }
]
```

- [ ] **Step 3: Write integration test**

```go
// test/integration/sync_test.go
package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestSyncEndToEnd(t *testing.T) {
	// Build the binary
	binPath := filepath.Join(t.TempDir(), "qs")
	build := exec.Command("go", "build", "-o", binPath, "../../.")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Set up mock qlik in PATH
	mockDir, _ := filepath.Abs(".")
	mockScript := filepath.Join(mockDir, "mock-qlik.sh")
	if err := os.Chmod(mockScript, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a temp dir to act as mock "qlik" binary
	pathDir := t.TempDir()
	mockLink := filepath.Join(pathDir, "qlik")
	if err := os.Symlink(mockScript, mockLink); err != nil {
		t.Fatal(err)
	}

	// Set up working directory with config
	workDir := t.TempDir()
	qlikDir := filepath.Join(workDir, "qlik")
	cfg := &config.Config{
		Version: "0.2.0",
		Threads: 2,
		Retries: 1,
		Tenants: []config.Tenant{
			{Context: "test", Server: "https://test.qlikcloud.com", Type: "cloud"},
		},
	}
	if err := config.Write(qlikDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Run qs sync
	cmd := exec.Command(binPath, "sync")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "PATH="+pathDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync failed: %s\n%s", err, out)
	}

	// Verify index.json was created
	indexData, err := os.ReadFile(filepath.Join(qlikDir, "index.json"))
	if err != nil {
		t.Fatalf("index.json not found: %v", err)
	}

	var index qsync.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		t.Fatalf("parsing index: %v", err)
	}

	if index.AppCount != 1 {
		t.Errorf("appCount = %d, want 1", index.AppCount)
	}

	if _, ok := index.Apps["app-001"]; !ok {
		t.Error("app-001 missing from index")
	}

	// Verify synced files exist
	appEntry := index.Apps["app-001"]
	scriptPath := filepath.Join(qlikDir, appEntry.Path, "script.qvs")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("script.qvs not found at %s: %v", scriptPath, err)
	}
}
```

- [ ] **Step 4: Run integration test**

Run: `go test ./test/integration/ -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add test/
git commit -m "test(sync): add end-to-end integration test with mock qlik-cli"
git push
```

---

### Task 18: Add CI Status Checks to Branch Protection

**Files:** None (GitHub API only)

- [ ] **Step 1: Update branch protection ruleset to require CI checks**

Run:
```bash
RULESET_ID=$(gh api repos/mattiasthalen/qlik-sync/rulesets --jq '.[0].id')
gh api repos/mattiasthalen/qlik-sync/rulesets/$RULESET_ID --method PUT --input - <<'EOF'
{
  "name": "Protect main",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/main"],
      "exclude": []
    }
  },
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": false,
        "required_status_checks": [
          {"context": "ci"}
        ]
      }
    },
    {
      "type": "deletion"
    },
    {
      "type": "non_fast_forward"
    }
  ]
}
EOF
```

Expected: 200 OK, ruleset updated

- [ ] **Step 2: Commit** (nothing to commit — API-only change)

Verify: `gh api repos/mattiasthalen/qlik-sync/rulesets --jq '.[0].rules[].type'`
Expected: output includes `required_status_checks`
