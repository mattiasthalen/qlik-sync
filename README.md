# qs

Sync Qlik Sense apps to local files for version control and offline inspection.

[![CI](https://github.com/mattiasthalen/qlik-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/mattiasthalen/qlik-sync/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/mattiasthalen/qlik-sync)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mattiasthalen/qlik-sync)](https://github.com/mattiasthalen/qlik-sync/releases)

## What It Does

`qs` pulls Qlik Cloud apps — load scripts, master measures, dimensions, variables, and sheet objects — into a local `qlik/` directory. Track changes in git, diff across versions, search across apps.

```
$ qs sync
Syncing my-tenant (47 apps, 5 threads)...
35 synced, 12 skipped, 0 errors
```

Re-running skips already-synced apps. Use `--force` to re-sync everything.

## Install

**Go install:**

```bash
go install github.com/mattiasthalen/qlik-sync@latest
```

**Binary download:**

Grab the latest release from [GitHub Releases](https://github.com/mattiasthalen/qlik-sync/releases). Available for Linux, macOS, and Windows (amd64/arm64).

**Build from source:**

```bash
git clone https://github.com/mattiasthalen/qlik-sync.git
cd qlik-sync
make build
```

## Prerequisites

- [qlik-cli](https://qlik.dev/toolkits/qlik-cli/) — `qs` uses it for Qlik Cloud API access

## Quick Start

```bash
# 1. Configure your tenant
qs setup

# 2. Sync all apps
qs sync

# 3. Browse your apps
ls qlik/
```

## Usage

### `qs setup`

Interactive tenant configuration. Creates `qlik/config.json` and tests connectivity.

```bash
qs setup
```

### `qs sync`

Pull apps from Qlik Cloud into `qlik/`.

```bash
# Sync everything
qs sync

# Sync a specific space
qs sync --space "Finance Prod"

# Sync apps matching a pattern
qs sync --app "Sales.*"

# Force re-sync a single app
qs sync --force --id 204be326-6892-494d-a186-376e6d1f6c85

# Use 10 threads and 5 retries
qs sync --threads 10 --retries 5

# Sync a specific tenant (multi-tenant setups)
qs sync --tenant my-cloud
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--space` | Filter by space name |
| `--stream` | Filter by stream name (on-prem, future) |
| `--app` | Regex filter on app name |
| `--id` | Exact app ID |
| `--tenant` | Filter by tenant context |
| `--force` | Skip cache, re-sync all |
| `--threads` | Concurrent syncs (overrides config) |
| `--retries` | Retry count per app (overrides config) |

### `qs version`

```bash
$ qs version
qs 1.0.0
```

## Configuration

`qs` stores config and synced data in `qlik/` at your project root.

### qlik/config.json

```json
{
  "version": "0.2.0",
  "threads": 5,
  "retries": 3,
  "tenants": [
    {
      "context": "my-cloud",
      "server": "https://tenant.qlikcloud.com",
      "type": "cloud"
    }
  ]
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `threads` | 5 | Number of concurrent app syncs |
| `retries` | 3 | Retry count per app (exponential backoff) |

Command-line flags override config values.

## Directory Structure

```
qlik/
├── config.json
├── index.json
└── my-tenant (HZJStxN8fU...)/
    ├── managed/
    │   └── Finance Prod (abc123...)/
    │       └── analytics/
    │           └── Sales Dashboard (def456...)/
    │               ├── script.qvs
    │               ├── measures.json
    │               ├── dimensions.json
    │               ├── variables.json
    │               ├── connections.yml
    │               └── objects/
    ├── shared/
    │   └── ...
    └── personal/
        └── Jane Doe (user123...)/
            └── ...
```

Each app directory contains the full extracted app: load script, master items, data connections, and sheet objects.

## How It Works

`qs` shells out to [qlik-cli](https://qlik.dev/toolkits/qlik-cli/) for API access:

1. **Prep** — fetches spaces and app list, caches results for 5 minutes
2. **Sync** — runs `qlik app unbuild` in parallel (configurable threads)
3. **Finalize** — builds `index.json` with all app metadata

Resume detection skips apps that already have local artifacts. `--force` bypasses both cache and resume.

On-prem support is planned — the internal QVF/QVW parser is already built for streaming extraction without temp files.

## Claude Code Plugin

`qs` works great standalone. For a natural language interface, check out [qlik-plugin](https://github.com/mattiasthalen/qlik-plugin) — a Claude Code plugin that wraps `qs` commands and adds AI-powered search across synced apps.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

[MIT](LICENSE)
