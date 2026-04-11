# qlik-cli Version Check Design Spec

## Problem

`qs` shells out to `qlik-cli` for API access but doesn't verify the installed version is compatible. Users with incompatible versions get confusing errors instead of a clear message.

## Solution

Extend `CheckPrerequisites()` in `internal/sync/exec.go` to verify qlik-cli version is within compatible range (`>= 3.0.0, < 3.1.0`) before any API calls.

## Version Compatibility

- **Minimum:** 3.0.0 (v3.0.0 changed API surface from v2.x)
- **Maximum:** < 3.1.0 (pin major.minor, allow patches)
- **Developed against:** qlik-cli 3.0.0

## CheckPrerequisites() Flow

1. Check `qlik` binary exists in PATH (existing behavior)
2. If `--skip-version-check` flag is set, return early (skip steps 3-5)
3. Run `qlik version`, capture stdout
4. Parse output format: `version: X.Y.Z<TAB>commit: ...<TAB>date: ...` — extract `X.Y.Z`
5. Split into major, minor, patch integers; verify `major == 3 && minor == 0`
6. On mismatch, return error: `qlik-cli version X.Y.Z is not compatible; requires >= 3.0.0, < 3.1.0`

## Skip Flag

- `--skip-version-check` boolean flag on root command in `cmd/root.go`
- Applies globally to all subcommands
- Passed into `CheckPrerequisites()` as parameter
- When true, PATH check still runs but version check is skipped

## Version Parsing

Pure stdlib implementation — no external semver library.

1. Split `qlik version` stdout on `\t`, take first field
2. Strip `"version: "` prefix
3. Split on `.` into 3 parts
4. `strconv.Atoi` each part
5. Return error on malformed output

## Testing

### Unit Tests (`internal/sync/exec_test.go`)

| Case | Input | Expected |
|------|-------|----------|
| Valid version | `version: 3.0.0\tcommit: abc\tdate: ...` | pass |
| Patch variant | `version: 3.0.5\tcommit: abc\tdate: ...` | pass |
| Major too low | `version: 2.9.0\t...` | error |
| Major too high | `version: 4.0.0\t...` | error |
| Minor too high | `version: 3.1.0\t...` | error |
| Malformed output | `not a version` | error |
| Skip flag | any | pass (no check) |

### Integration Test

Extend `test/integration/mock-qlik.sh` to handle `qlik version` command, returning compatible version string so existing sync flow passes.

## Files Changed

- `internal/sync/exec.go` — version parsing + check in `CheckPrerequisites()`
- `internal/sync/exec_test.go` — unit tests for version check
- `cmd/root.go` — add `--skip-version-check` flag
- `test/integration/mock-qlik.sh` — handle `version` subcommand
- `test/integration/sync_test.go` — verify mock handles version check

## No New Dependencies

Semver parsing is ~20 lines of stdlib code (`strings.Cut`, `strconv.Atoi`). No need for `golang.org/x/mod/semver` or other packages.
