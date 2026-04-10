# CLI Extraction Design Spec

## Problem

The sync and setup scripts in mattiasthalen/qlik-plugin are a CLI in disguise. The skill layer adds overhead — token cost, backgrounding issues, timeout problems, redundant retries — for what is essentially "run these scripts with these flags." Four of the issues fixed in qlik-plugin#13 were caused by indirection between Claude and the bash scripts.

## Solution

Extract sync and setup into `qs` (qlik-sync) — a standalone Go CLI that replaces the bash scripts with native Go, adds parallel sync via `errgroup`, and streams QVF parsing for on-prem without temp files.

## CLI Command Tree

```
qs
├── setup                     # interactive tenant config
├── sync                      # pull apps from Qlik Cloud/on-prem
│   ├── --space "Name"        # filter by space (cloud)
│   ├── --stream "Name"       # filter by stream (on-prem)
│   ├── --app "Pattern"       # regex filter on app name
│   ├── --id <GUID>           # exact app ID
│   ├── --tenant "context"    # filter tenant (multi-tenant)
│   ├── --force               # skip cache, re-sync all
│   ├── --threads N           # concurrent syncs (overrides config)
│   └── --retries N           # retry count per app (overrides config)
└── version                   # print version info

Global flags: --log-level (debug|info|warn|error|disabled), --config (default: qlik/)
```

## Project Structure

```
qlik-sync/
├── cmd/
│   ├── root.go               # Cobra root, global flags
│   ├── setup.go              # qs setup
│   ├── sync.go               # qs sync (flag parsing, orchestration)
│   └── version.go            # qs version
├── internal/
│   ├── config/               # qlik/ config read/write, v0.1→v0.2 migration
│   ├── sync/
│   │   ├── cloud.go          # cloud prep + app sync (qlik-cli shelling)
│   │   ├── onprem.go         # on-prem prep + app sync (QRS API + parser)
│   │   ├── finalize.go       # index.json + config update
│   │   └── runner.go         # parallel orchestration (errgroup)
│   ├── parser/               # QVF/QVW extraction (inspired by qlik-parser)
│   │   ├── qvf.go            # zlib block scanning, JSON extraction
│   │   └── qvw.go            # header strip, script extraction
│   └── ui/                   # terminal output, progress, spinners
├── main.go
├── go.mod                    # module: github.com/mattiasthalen/qlik-sync
├── .goreleaser.yml
├── .golangci.yml
├── Makefile
└── qlik/                     # project-local config + synced data (tracked in git)
```

## Config & State

All state lives in `qlik/` at project root. Fully tracked in git — no secrets stored here.

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
      "type": "cloud",
      "lastSync": "2026-04-10T14:30:00Z"
    }
  ]
}
```

Flag precedence: `--threads`/`--retries` flag > config value > default (5 threads, 3 retries).

### qlik/index.json

```json
{
  "lastSync": "2026-04-10T14:30:00Z",
  "context": "my-context",
  "server": "https://tenant.qlikcloud.com",
  "tenant": "my-tenant",
  "tenantId": "abc-123",
  "appCount": 47,
  "apps": {
    "app-001": {
      "name": "Sales Dashboard",
      "space": "Finance Prod",
      "spaceId": "space-001",
      "spaceType": "managed",
      "appType": "analytics",
      "owner": "user-001",
      "ownerName": "jane.doe",
      "description": "Monthly sales KPIs",
      "tags": ["finance"],
      "published": true,
      "lastReloadTime": "2026-04-08T02:00:00Z",
      "path": "my-tenant (abc-123)/managed/Finance Prod (space-001)/analytics/Sales Dashboard (app-001)/"
    }
  }
}
```

### Synced File Tree

```
qlik/
├── config.json
├── index.json
└── <tenant> (<tenantId>)/
    └── <spaceType>/
        └── <spaceName> (<spaceId>)/
            └── <appType>/
                └── <appName> (<resourceId>)/
                    ├── script.qvs
                    ├── measures.json
                    ├── dimensions.json
                    ├── variables.json
                    ├── connections.yml
                    └── objects/
```

On-prem omits the `<appType>/` level and uses `<hostname>/` instead of `<tenant> (<tenantId>)/`.

## Setup Flow

```
qs setup
    │
    ├─ check prerequisites (qlik-cli, jq in PATH)
    │   └─ missing → print install instructions with URLs, exit 1
    ├─ list existing qlik contexts: qlik context ls
    ├─ prompt: select existing context or create new
    │   └─ new: prompt server URL → detect type (cloud/on-prem) → prompt API key
    │          → qlik context create
    ├─ test connectivity: qlik app ls (cloud) or qlik qrs app (on-prem)
    │   └─ fail → print auth troubleshooting, exit 1
    ├─ write/update qlik/config.json (append tenant to array)
    └─ print success + suggest: qs sync
```

Interactive prompts via Go TUI library. Falls back to plain stdin if not TTY.

## Sync Flow

### Cloud

```
qs sync [flags]
    │
    ├─ read qlik/config.json → resolve tenant(s)
    ├─ shell out: qlik app ls, qlik space ls → build app list
    ├─ apply filters (--space, --app, --id)
    ├─ check resume: skip apps with existing artifacts (unless --force)
    ├─ parallel sync via errgroup (--threads N):
    │   └─ per app (with --retries N, exponential backoff):
    │       └─ shell out: qlik app unbuild → write to qlik/<tenant>/<space>/<app>/
    ├─ finalize: build/merge index.json, update config.json lastSync
    └─ print summary: "35 synced, 12 skipped, 0 errors"
```

### On-Prem

```
qs sync [flags]
    │
    ├─ read qlik/config.json → resolve tenant(s)
    ├─ shell out: qlik qrs app full, qlik qrs stream ls → build app list
    ├─ apply filters (--stream, --app, --id)
    ├─ check resume
    ├─ parallel sync via errgroup (--threads N):
    │   └─ per app (with --retries N, exponential backoff):
    │       ├─ shell out: qlik qrs app export → download QVF bytes
    │       └─ stream into internal/parser → write artifacts (no temp file)
    ├─ finalize
    └─ print summary
```

### Cache

Prep results (app list from API) cached in OS temp dir with 5-min TTL. Cache key derived from context + filters + working dir hash. `--force` bypasses cache.

## Error Handling

### Exit Codes

- `0` — success
- `1` — general error (config missing, auth fail, dependency missing)
- `2` — partial success (some apps synced, some failed)

### Error Messages

Actionable, suggest next step:

```
Error: qlik-cli not found in PATH
  Install: https://qlik.dev/toolkits/qlik-cli/
```

```
Error: authentication failed for tenant "my-cloud"
  Run: qlik context login
```

```
Error: 3 of 47 apps failed to sync (after 3 retries each)
  Re-run with: qs sync --force --id <failed-id>
```

### Retries

Per-app retry with exponential backoff. Default 3 attempts, configurable via `retries` in config or `--retries` flag.

## UX

### TTY Output (Interactive)

```
Syncing my-cloud (47 apps, 5 threads)...
  ✓ Sales Dashboard          ⠋ Finance Report          ⊘ HR Metrics (skipped)
  35/47 synced | 12 skipped | 0 errors
```

### Non-TTY Output (Piped)

Plain text, one line per app — machine parseable.

### Logging

`--log-level debug` for troubleshooting, disabled by default.

## Testing Strategy

### Unit Tests

`*_test.go` alongside source:
- `internal/config/` — config read/write, migration, defaults, flag precedence
- `internal/parser/` — QVF/QVW extraction from byte streams
- `internal/sync/` — filter logic, resume detection, index building, retry logic
- `internal/ui/` — TTY vs non-TTY output formatting

### Integration Tests

- Mock `qlik-cli` binary returning fixture JSON
- End-to-end: `qs sync` against mock → verify file tree + index.json

### CI

- Linux + `go test -race`
- Coverage via `coverprofile`
- No real API tests in CI (requires Qlik Cloud auth)

## CI/CD

### GitHub Actions

- **ci.yml** — on PR + push to main: `golangci-lint`, `go test -race`, `go vet`
- **release.yml** — on tag push (`v*`): GoReleaser builds + publishes

### GoReleaser

- Binary: `qs`
- Platforms: linux/mac/win × amd64/arm64
- Formats: `.tar.gz` (linux/mac), `.zip` (win)
- SHA256 checksums
- Version injection: `-X github.com/mattiasthalen/qlik-sync/cmd.Version`

### Lefthook (Pre-commit)

```yaml
pre-commit:
  commands:
    go-vet:
      glob: "*.go"
      run: go vet ./...
    go-lint:
      glob: "*.go"
      run: golangci-lint run
    go-test:
      glob: "*.go"
      run: go test ./...
```

### Branch Protection

Ruleset on `main`:
- Require PR (no direct push)
- No force push
- No deletion
- CI status checks required (added once workflows exist)

## Dependencies

### Hard (v1)

- `qlik-cli` — API access via shell exec
- `jq` — JSON processing (evaluate if still needed with Go JSON handling)

### Soft (v1)

- Go TUI library (e.g., `huh` or `survey`) — interactive setup prompts

### Removed (v2)

- `qlik-cli` — replaced by direct REST API calls
- `jq` — unnecessary with native Go JSON

## Migration Path

1. Users of qlik-plugin continue using skills as-is
2. `qs` ships as standalone alternative
3. qlik-plugin sync skill updated to shell out to `qs` (thin NL wrapper)
4. qlik-plugin inspect skill unchanged (AI-powered search retains value)

## Out of Scope (v1)

- `qs inspect` command (local search — potential v2 feature)
- Direct REST API (replaces qlik-cli shelling — v2)
- Push/reload support (write-back to Qlik — future)
- Multi-repo config (global `~/.config/qs/` — not needed, project-local only)
