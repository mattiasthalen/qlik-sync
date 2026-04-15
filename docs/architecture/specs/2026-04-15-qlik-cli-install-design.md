# Qlik CLI Auto-Install Design

## Problem

Users must manually find, download, and install the correct version of qlik-cli before using `qs`. This creates friction — wrong versions cause confusing errors, and the download/install process is undocumented tribal knowledge.

## Solution

`qs setup` automatically installs the correct qlik-cli binary next to itself. Other commands (`qs sync`) provide clear error messages pointing users to `qs setup` if qlik-cli is missing or incompatible.

## Design Decisions

- **Install location:** Next to the `qs` binary. If `qs` is at `/usr/local/bin/qs`, qlik-cli installs to `/usr/local/bin/qlik`. They are a paired toolchain — same scope, same location.
- **Version pinning:** Hardcoded constant `QlikCLIVersion = "3.0.0"` in source. Deterministic — same `qs` binary always installs same qlik-cli. Version bumps are deliberate commits.
- **Existing install handling:** Check version if qlik-cli exists at target path. Skip if compatible (3.0.x range). Replace if missing or incompatible.
- **Explicit install only:** Only `qs setup` downloads qlik-cli. Other commands fail with actionable error message. No surprise 6MB downloads.
- **Checksum verification:** Download SHA256 checksums file from GitHub release. Verify downloaded archive matches before extracting. Fail on mismatch.

## Binary Resolution

All commands resolve qlik-cli path relative to their own executable:

```
os.Executable() → /usr/local/bin/qs
filepath.Dir()  → /usr/local/bin/
qlik target     → /usr/local/bin/qlik
```

This replaces `exec.LookPath("qlik")` in `CheckPrerequisites`. `RunQlikCmd` callers pass the resolved path instead of hardcoded `"qlik"`.

## Install Flow

Current `qs setup` flow:
```
CheckPrerequisites → context ls → prompt → context create → test connectivity → write config
```

New flow:
```
ResolveQlikPath → EnsureQlikCLI → CheckPrerequisites → context ls → ...
```

`EnsureQlikCLI(ctx, targetPath)` orchestrates:

1. Check if `targetPath` exists.
2. If exists: run `qlik version`, parse, check compatibility (3.0.x range).
3. If compatible: print status message, return.
4. If missing or incompatible: download from GitHub releases.
   - Detect OS/arch via `runtime.GOOS`, `runtime.GOARCH`.
   - Build asset name: `qlik-{Darwin|Linux|Windows}-x86_64.{tar.gz|zip}`.
   - Download archive from `github.com/qlik-oss/qlik-cli/releases/download/v3.0.0/`.
   - Download checksums file.
   - Verify SHA256.
   - Extract binary from archive.
   - Write to `targetPath`, set executable permission (unix).
   - Verify: run `qlik version` on new binary.

## Download Details

URL template:
```
https://github.com/qlik-oss/qlik-cli/releases/download/v{version}/qlik-{OS}-{arch}.tar.gz
```

Checksums URL:
```
https://github.com/qlik-oss/qlik-cli/releases/download/v{version}/qlik_{version}_checksums.txt
```

Available assets (v3.0.0):
- `qlik-Darwin-x86_64.tar.gz` (6.6MB)
- `qlik-Linux-x86_64.tar.gz` (6.5MB)
- `qlik-Windows-x86_64.zip` (6.6MB)
- `qlik_3.0.0_checksums.txt`

## Error Messages

Missing qlik-cli (from `qs sync` or other commands):
```
qlik-cli not found at /usr/local/bin/qlik

Run "qs setup" to install it automatically.
```

Wrong version:
```
qlik-cli 2.30.0 found at /usr/local/bin/qlik — need 3.0.x

Run "qs setup" to install the correct version.
```

Permission denied during install:
```
cannot install qlik-cli to /usr/local/bin/ — check permissions or run with sudo
```

## Functions and File Layout

New file `internal/sync/install.go`:

| Function | Pure | Purpose |
|----------|------|---------|
| `ResolveQlikPath() (string, error)` | No | `os.Executable()` → dir → `filepath.Join(dir, "qlik")` |
| `DetectAssetName() string` | Yes | `runtime.GOOS` + `runtime.GOARCH` → asset filename |
| `BuildDownloadURL(version, asset) string` | Yes | GitHub release URL |
| `BuildChecksumsURL(version) string` | Yes | GitHub checksums URL |
| `DownloadFile(ctx, url) ([]byte, error)` | No | `net/http` GET, read body |
| `VerifyChecksum(data, checksumsBody, assetName) error` | Yes | Parse checksums, find match, compare SHA256 |
| `ExtractBinary(archive, goos) ([]byte, error)` | Yes | tar.gz (linux/darwin) or zip (windows) extraction |
| `EnsureQlikCLI(ctx, targetPath) error` | No | Orchestrator: check → download → verify → extract → write |

## Changes to Existing Files

- **`internal/sync/exec.go`** — `CheckPrerequisites` uses `ResolveQlikPath()` instead of `exec.LookPath("qlik")`. Error messages updated to reference `qs setup`.
- **`cmd/setup.go`** — Calls `EnsureQlikCLI` before context configuration.
- **`cmd/sync.go`** — Passes resolved qlik-cli path to `RunQlikCmd` instead of `"qlik"`.

## Version Constants

Reuses existing `ParseVersion` and `CheckVersion` from `internal/sync/version.go`:

```go
const QlikCLIVersion = "3.0.0"
```

Compatibility range already enforced: `major == 3 && minor == 0` (any patch).

## Edge Cases

- **Windows:** `.zip` extraction instead of `.tar.gz`. No `chmod` needed.
- **Permission denied:** Clear error with suggestion to check permissions or use sudo.
- **Network failure:** Error with retry suggestion.
- **Symlinked `qs`:** `os.Executable()` resolves symlinks via `filepath.EvalSymlinks` — installs next to real binary, not the symlink.
- **Existing qlik-cli is different tool:** Version parse will fail or return incompatible version — triggers replacement with user-visible messaging about what's happening.
