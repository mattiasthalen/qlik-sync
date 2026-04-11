# qlik-cli Version Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Verify installed qlik-cli version is compatible (`>= 3.0.0, < 3.1.0`) before any API calls, with a `--skip-version-check` escape hatch.

**Architecture:** Extend `CheckPrerequisites()` in `internal/sync/exec.go` with version parsing after the existing PATH check. Pure `ParseVersion` function extracts semver from `qlik version` output; `CheckVersion` validates range. Root command gets `--skip-version-check` persistent flag passed through to callers.

**Tech Stack:** Go stdlib only (`strings`, `strconv`, `fmt`, `os/exec`). No new dependencies.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/sync/version.go` | Create | `ParseVersion` (pure) and `CheckVersion` (calls qlik binary) |
| `internal/sync/version_test.go` | Create | Unit tests for `ParseVersion` and `CheckVersion` |
| `internal/sync/exec.go` | Modify | Update `CheckPrerequisites` signature to accept `skipVersionCheck bool` |
| `internal/sync/exec_test.go` | Modify | Update `TestCheckPrerequisites` call |
| `cmd/root.go` | Modify | Add `--skip-version-check` persistent flag |
| `cmd/setup.go` | Modify | Pass `skipVersionCheck` to `CheckPrerequisites` |
| `cmd/sync.go` | Modify | Pass `skipVersionCheck` to `CheckPrerequisites` |
| `test/integration/mock-qlik.sh` | Modify | Handle `version` subcommand |

---

### Task 1: ParseVersion — pure semver extraction

**Files:**
- Create: `internal/sync/version.go`
- Create: `internal/sync/version_test.go`

`ParseVersion` takes raw `qlik version` output and returns `(major, minor, patch int, err error)`. Pure function — no side effects.

- [ ] **Step 1: Write failing tests for ParseVersion**

Add to `internal/sync/version_test.go`:

```go
package sync_test

import (
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantMaj        int
		wantMin        int
		wantPat        int
		wantErr        bool
	}{
		{"valid 3.0.0", "version: 3.0.0\tcommit: abc\tdate: 2026-01-01", 3, 0, 0, false},
		{"valid 3.0.5", "version: 3.0.5\tcommit: abc\tdate: 2026-01-01", 3, 0, 5, false},
		{"valid 2.9.1", "version: 2.9.1\tcommit: abc\tdate: 2026-01-01", 2, 9, 1, false},
		{"no tabs", "version: 3.0.0", 3, 0, 0, false},
		{"malformed no prefix", "3.0.0\tcommit: abc", 0, 0, 0, true},
		{"malformed empty", "", 0, 0, 0, true},
		{"malformed two parts", "version: 3.0\tcommit: abc", 0, 0, 0, true},
		{"malformed non-numeric", "version: a.b.c\tcommit: abc", 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maj, min, pat, err := qsync.ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if maj != tt.wantMaj || min != tt.wantMin || pat != tt.wantPat {
				t.Errorf("got %d.%d.%d, want %d.%d.%d", maj, min, pat, tt.wantMaj, tt.wantMin, tt.wantPat)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ -run TestParseVersion -v`
Expected: FAIL — `ParseVersion` not defined

- [ ] **Step 3: Implement ParseVersion**

Create `internal/sync/version.go`:

```go
package sync

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseVersion extracts major, minor, patch from qlik version output.
// Expected format: "version: X.Y.Z\tcommit: ...\tdate: ..."
func ParseVersion(raw string) (major, minor, patch int, err error) {
	field := raw
	if i := strings.IndexByte(raw, '\t'); i >= 0 {
		field = raw[:i]
	}

	ver, found := strings.CutPrefix(field, "version: ")
	if !found {
		return 0, 0, 0, fmt.Errorf("unexpected version output: %q", raw)
	}

	parts := strings.Split(ver, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected major.minor.patch, got %q", ver)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return major, minor, patch, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ -run TestParseVersion -v`
Expected: PASS (all 8 subtests)

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/version-check
git add internal/sync/version.go internal/sync/version_test.go
git commit -m "feat(sync): add ParseVersion for qlik-cli output"
git push
```

---

### Task 2: CheckVersion — validate version range

**Files:**
- Modify: `internal/sync/version.go`
- Modify: `internal/sync/version_test.go`

`CheckVersion` takes `qlik version` output string, parses it, and returns error if outside `>= 3.0.0, < 3.1.0`.

- [ ] **Step 1: Write failing tests for CheckVersion**

Append to `internal/sync/version_test.go`:

```go
func TestCheckVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"compatible 3.0.0", "version: 3.0.0\tcommit: abc\tdate: 2026-01-01", false, ""},
		{"compatible 3.0.5", "version: 3.0.5\tcommit: abc\tdate: 2026-01-01", false, ""},
		{"major too low", "version: 2.9.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 2.9.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"major too high", "version: 4.0.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 4.0.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"minor too high", "version: 3.1.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 3.1.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"malformed", "garbage", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qsync.CheckVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
				t.Errorf("err = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ -run TestCheckVersion -v`
Expected: FAIL — `CheckVersion` not defined

- [ ] **Step 3: Implement CheckVersion**

Append to `internal/sync/version.go`:

```go
// CheckVersion validates that qlik version output is within compatible range.
// Compatible range: major == 3, minor == 0, any patch.
func CheckVersion(raw string) error {
	major, minor, patch, err := ParseVersion(raw)
	if err != nil {
		return fmt.Errorf("cannot determine qlik-cli version: %w", err)
	}

	if major != 3 || minor != 0 {
		return fmt.Errorf("qlik-cli version %d.%d.%d is not compatible; requires >= 3.0.0, < 3.1.0", major, minor, patch)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ -run TestCheckVersion -v`
Expected: PASS (all 6 subtests)

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/version-check
git add internal/sync/version.go internal/sync/version_test.go
git commit -m "feat(sync): add CheckVersion for compatibility range"
git push
```

---

### Task 3: Wire version check into CheckPrerequisites with root flag

**Files:**
- Modify: `cmd/root.go`
- Modify: `internal/sync/exec.go`
- Modify: `internal/sync/exec_test.go`
- Modify: `cmd/setup.go`
- Modify: `cmd/sync.go`

Add `--skip-version-check` flag to root command, update `CheckPrerequisites` signature to accept `skipVersionCheck bool`, and update all callers. All changes compile together.

- [ ] **Step 1: Write failing test for updated CheckPrerequisites**

Replace `TestCheckPrerequisites` in `internal/sync/exec_test.go`:

```go
func TestCheckPrerequisites(t *testing.T) {
	t.Run("skip version check", func(t *testing.T) {
		err := qsync.CheckPrerequisites(true)
		_ = err // depends on environment having qlik in PATH
	})

	t.Run("does not skip version check", func(t *testing.T) {
		// When not skipping, CheckPrerequisites calls qlik version.
		// In test env with real qlik installed, should pass.
		// Just verify function accepts false without panic.
		_ = qsync.CheckPrerequisites(false)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ -run TestCheckPrerequisites -v`
Expected: FAIL — `CheckPrerequisites` takes 0 args, called with 1

- [ ] **Step 3: Update CheckPrerequisites, root flag, and all callers**

Update `cmd/root.go` — add `skipVersionCheck` to var block and flag to `init()`:

```go
var (
	logLevel         string
	configDir        string
	skipVersionCheck bool
)
```

```go
func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "disabled", "log level (debug|info|warn|error|disabled)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "qlik", "config and sync directory")
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "skip qlik-cli version compatibility check")
}
```

Update `internal/sync/exec.go` — replace `CheckPrerequisites` function:

```go
func CheckPrerequisites(skipVersionCheck bool) error {
	binary, err := exec.LookPath("qlik")
	if err != nil {
		return fmt.Errorf("qlik-cli not found in PATH\n  Install: https://qlik.dev/toolkits/qlik-cli/")
	}

	if skipVersionCheck {
		return nil
	}

	out, err := RunQlikCmd(context.Background(), binary, "version")
	if err != nil {
		return fmt.Errorf("cannot determine qlik-cli version: %w", err)
	}

	return CheckVersion(strings.TrimSpace(string(out)))
}
```

Add `"context"` and `"strings"` to imports in `exec.go`.

Update `cmd/setup.go` line 27:

```go
if err := qsync.CheckPrerequisites(skipVersionCheck); err != nil {
```

Update `cmd/sync.go` line 51:

```go
if err := qsync.CheckPrerequisites(skipVersionCheck); err != nil {
```

- [ ] **Step 4: Run unit tests to verify they pass**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./internal/sync/ ./cmd/ -v`
Expected: PASS (integration test may fail until mock is updated in Task 4)

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/version-check
git add cmd/root.go internal/sync/exec.go internal/sync/exec_test.go cmd/setup.go cmd/sync.go
git commit -m "feat(sync): wire version check into CheckPrerequisites with root flag"
git push
```

---

### Task 4: Update integration test mock

**Files:**
- Modify: `test/integration/mock-qlik.sh`

The integration test builds `qs` and runs `qs sync` with a mock `qlik` binary. The mock must now handle `version` to pass `CheckPrerequisites`.

- [ ] **Step 1: Run integration test to confirm it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./test/integration/ -v`
Expected: FAIL — mock returns "unknown command: version" causing version check error

- [ ] **Step 2: Add version case to mock**

Update `test/integration/mock-qlik.sh` — add `version` case before the wildcard `*` case:

```bash
  "version")
    printf 'version: 3.0.0\tcommit: mock\tdate: 2026-01-01T00:00:00Z'
    ;;
```

The full case block becomes:

```bash
case "$*" in
  "version")
    printf 'version: 3.0.0\tcommit: mock\tdate: 2026-01-01T00:00:00Z'
    ;;
  "space ls --json")
    cat "$SCRIPT_DIR/testdata/spaces.json"
    ;;
  "app ls --json --limit 1000")
    cat "$SCRIPT_DIR/testdata/apps.json"
    ;;
  "app unbuild --app"*)
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

- [ ] **Step 3: Run integration test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./test/integration/ -v`
Expected: PASS

- [ ] **Step 4: Run full test suite**

Run: `cd /workspaces/qlik-sync/.worktrees/version-check && go test ./... -v`
Expected: PASS (all packages)

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/version-check
git add test/integration/mock-qlik.sh
git commit -m "test(sync): add version response to integration mock"
git push
```
