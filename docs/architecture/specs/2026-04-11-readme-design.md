# README + CONTRIBUTING Design Spec

## Problem

The repo has no README or contributing guide. Users landing on GitHub have no idea what `qs` does, how to install it, or how to use it.

## Solution

Two files: `README.md` (user-focused with contributing pointer) and `CONTRIBUTING.md` (dev setup and workflow).

## README.md

### 1. Title + Tagline

```
# qs

Sync Qlik Sense apps to local files for version control and offline inspection.
```

### 2. Badges

CI status, Go version (from go.mod), license (MIT), latest release.

### 3. What It Does

2-3 sentences explaining the value prop: pull Qlik Cloud apps (load scripts, measures, dimensions, variables) into a local directory. Version control your Qlik work. Search and diff across apps.

Short example showing `qs sync` output:

```
$ qs sync
Syncing my-tenant (47 apps, 5 threads)...
35 synced, 12 skipped, 0 errors
```

### 4. Install

Three methods:
- `go install github.com/mattiasthalen/qlik-sync@latest`
- Binary download from GitHub Releases
- Build from source: `git clone` + `make build`

### 5. Prerequisites

- `qlik-cli` — link to install page, note version compat (ref #2)

### 6. Quick Start

Three commands:
1. `qs setup` — configure tenant
2. `qs sync` — pull all apps
3. Browse `qlik/` directory

### 7. Usage

Command reference with examples:

- `qs setup` — interactive tenant config
- `qs sync` — all flags with examples (`--space`, `--app`, `--id`, `--force`, `--threads`, `--retries`, `--tenant`)
- `qs version` — version info

### 8. Configuration

Show `qlik/config.json` format. Explain `threads` (default 5), `retries` (default 3). Flag overrides config.

### 9. Directory Structure

Show actual output tree:
```
qlik/
├── config.json
├── index.json
└── tenant-name (tenantId)/
    ├── managed/
    │   └── Space Name (spaceId)/
    │       └── analytics/
    │           └── App Name (appId)/
    │               ├── script.qvs
    │               ├── measures.json
    │               ├── dimensions.json
    │               └── variables.json
    └── personal/
        └── Owner Name (ownerId)/
            └── ...
```

### 10. How It Works

Brief: shells out to `qlik-cli` for API access (v1). Parallel sync via goroutines. Prep results cached for 5 minutes. Resume detection skips already-synced apps.

Mention: on-prem support planned (internal QVF parser already built).

### 11. Claude Code Plugin

Link to mattiasthalen/qlik-plugin. Explain: `qs` works standalone, plugin adds natural language convenience and AI-powered inspect.

### 12. Contributing

One line: "See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines."

### 13. License

MIT.

## CONTRIBUTING.md

### 1. Prerequisites

- Go 1.23+
- qlik-cli
- golangci-lint v2
- Lefthook

### 2. Development Setup

```bash
git clone https://github.com/mattiasthalen/qlik-sync.git
cd qlik-sync
lefthook install
make build
make test
```

### 3. Testing

- `make test` — unit tests with race detector
- `make lint` — golangci-lint
- `make vet` — go vet
- `go test ./test/integration/` — integration test with mock qlik-cli

### 4. Workflow

- Work in git worktrees on feature branches
- Open draft PR when starting work
- TDD: test first, verify fail, implement, verify pass, commit
- Conventional commits with scope
- Push after every commit
- Merge commits (preserve history)

### 5. CI

Tests must pass before merge. CI runs vet, lint, and test on every PR.
