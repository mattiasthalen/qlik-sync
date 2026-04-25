# Qlik CLI Auto-Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `qs setup` automatically downloads and installs the correct qlik-cli binary next to itself, with SHA256 checksum verification.

**Architecture:** New `internal/sync/install.go` with pure functions for URL building, checksum verification, and archive extraction. Orchestrator `EnsureQlikCLI` called from `cmd/setup.go`. Existing `CheckPrerequisites` updated to resolve qlik-cli path relative to `qs` binary. All hardcoded `"qlik"` binary references replaced with resolved path.

**Tech Stack:** Go stdlib only — `net/http`, `crypto/sha256`, `archive/tar`, `archive/zip`, `compress/gzip`, `runtime`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/sync/install.go` | Create | Pure functions: URL builders, checksum verify, archive extract, orchestrator |
| `internal/sync/install_test.go` | Create | Tests for all pure functions in install.go |
| `internal/sync/exec.go` | Modify | `CheckPrerequisites` uses `ResolveQlikPath`, updated error messages |
| `internal/sync/cloud.go` | Modify | `CloudSyncApp` uses passed binary instead of hardcoded `"qlik"` |
| `cmd/setup.go` | Modify | Call `EnsureQlikCLI` before `CheckPrerequisites` |
| `cmd/sync.go` | Modify | Resolve qlik path and pass to all `RunQlikCmd` calls |
| `internal/sync/exec_test.go` | Modify | Update `TestCheckPrerequisites` for new signature |
| `test/integration/sync_test.go` | Modify | Place mock qlik next to built `qs` binary instead of PATH |

---

### Task 1: Pure URL builder functions

**Files:**
- Create: `internal/sync/install.go`
- Create: `internal/sync/install_test.go`

- [ ] **Step 1: Write failing tests for DetectAssetName, BuildDownloadURL, BuildChecksumsURL**

In `internal/sync/install_test.go`:

```go
package sync_test

import (
	"runtime"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestDetectAssetName(t *testing.T) {
	name := qsync.DetectAssetName()

	switch runtime.GOOS {
	case "linux":
		if name != "qlik-Linux-x86_64.tar.gz" {
			t.Errorf("got %q, want qlik-Linux-x86_64.tar.gz", name)
		}
	case "darwin":
		if name != "qlik-Darwin-x86_64.tar.gz" {
			t.Errorf("got %q, want qlik-Darwin-x86_64.tar.gz", name)
		}
	case "windows":
		if name != "qlik-Windows-x86_64.zip" {
			t.Errorf("got %q, want qlik-Windows-x86_64.zip", name)
		}
	}
}

func TestBuildDownloadURL(t *testing.T) {
	got := qsync.BuildDownloadURL("3.0.0", "qlik-Linux-x86_64.tar.gz")
	want := "https://github.com/qlik-oss/qlik-cli/releases/download/v3.0.0/qlik-Linux-x86_64.tar.gz"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildChecksumsURL(t *testing.T) {
	got := qsync.BuildChecksumsURL("3.0.0")
	want := "https://github.com/qlik-oss/qlik-cli/releases/download/v3.0.0/qlik_3.0.0_checksums.txt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestDetectAssetName|TestBuildDownloadURL|TestBuildChecksumsURL"`
Expected: FAIL — `DetectAssetName`, `BuildDownloadURL`, `BuildChecksumsURL` not defined

- [ ] **Step 3: Implement the functions**

In `internal/sync/install.go`:

```go
package sync

import (
	"fmt"
	"runtime"
	"strings"
)

const QlikCLIVersion = "3.0.0"

func DetectAssetName() string {
	osName := strings.ToTitle(runtime.GOOS[:1]) + runtime.GOOS[1:]
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("qlik-%s-x86_64.%s", osName, ext)
}

func BuildDownloadURL(version, asset string) string {
	return fmt.Sprintf("https://github.com/qlik-oss/qlik-cli/releases/download/v%s/%s", version, asset)
}

func BuildChecksumsURL(version string) string {
	return fmt.Sprintf("https://github.com/qlik-oss/qlik-cli/releases/download/v%s/qlik_%s_checksums.txt", version, version)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestDetectAssetName|TestBuildDownloadURL|TestBuildChecksumsURL"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/install.go internal/sync/install_test.go
git commit -m "feat(sync): add URL builder functions for qlik-cli download"
git push
```

---

### Task 2: Checksum verification

**Files:**
- Modify: `internal/sync/install.go`
- Modify: `internal/sync/install_test.go`

- [ ] **Step 1: Write failing test for VerifyChecksum**

Append to `internal/sync/install_test.go`:

```go
func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world")
	// SHA256 of "hello world"
	checksumsBody := []byte(
		"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  hello.tar.gz\n" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.tar.gz\n",
	)

	t.Run("valid checksum", func(t *testing.T) {
		err := qsync.VerifyChecksum(data, checksumsBody, "hello.tar.gz")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid checksum", func(t *testing.T) {
		bad := []byte("wrong data")
		err := qsync.VerifyChecksum(bad, checksumsBody, "hello.tar.gz")
		if err == nil {
			t.Error("expected error for mismatched checksum")
		}
	})

	t.Run("asset not found in checksums", func(t *testing.T) {
		err := qsync.VerifyChecksum(data, checksumsBody, "missing.tar.gz")
		if err == nil {
			t.Error("expected error for missing asset")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestVerifyChecksum"`
Expected: FAIL — `VerifyChecksum` not defined

- [ ] **Step 3: Implement VerifyChecksum**

Add to `internal/sync/install.go` (add `"crypto/sha256"`, `"encoding/hex"`, `"bufio"`, `"bytes"` to imports):

```go
func VerifyChecksum(data, checksumsBody []byte, assetName string) error {
	expected := ""
	scanner := bufio.NewScanner(bytes.NewReader(checksumsBody))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "<hash>  <filename>"
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 && parts[1] == assetName {
			expected = parts[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestVerifyChecksum"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/install.go internal/sync/install_test.go
git commit -m "feat(sync): add SHA256 checksum verification for downloads"
git push
```

---

### Task 3: Archive extraction

**Files:**
- Modify: `internal/sync/install.go`
- Modify: `internal/sync/install_test.go`

- [ ] **Step 1: Write failing tests for ExtractBinary**

Append to `internal/sync/install_test.go`:

```go
import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
)

func TestExtractBinary_TarGz(t *testing.T) {
	content := []byte("fake-qlik-binary")
	archive := buildTarGz(t, "qlik", content)

	got, err := qsync.ExtractBinary(archive, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	content := []byte("fake-qlik-binary")
	archive := buildZip(t, "qlik.exe", content)

	got, err := qsync.ExtractBinary(archive, "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinary_NoBinary(t *testing.T) {
	archive := buildTarGz(t, "README.md", []byte("not a binary"))

	_, err := qsync.ExtractBinary(archive, "linux")
	if err == nil {
		t.Error("expected error for missing qlik binary in archive")
	}
}

func buildTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0755}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func buildZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	zw.Close()
	return buf.Bytes()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestExtractBinary"`
Expected: FAIL — `ExtractBinary` not defined

- [ ] **Step 3: Implement ExtractBinary**

Add to `internal/sync/install.go` (add `"archive/tar"`, `"archive/zip"`, `"compress/gzip"`, `"io"` to imports):

```go
func ExtractBinary(archive []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		return extractFromZip(archive)
	}
	return extractFromTarGz(archive)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompressing archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if hdr.Name == "qlik" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("qlik binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}
	for _, f := range r.File {
		if f.Name == "qlik.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening qlik.exe in zip: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("qlik.exe not found in archive")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestExtractBinary"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/install.go internal/sync/install_test.go
git commit -m "feat(sync): add tar.gz and zip extraction for qlik-cli binary"
git push
```

---

### Task 4: ResolveQlikPath

**Files:**
- Modify: `internal/sync/install.go`
- Modify: `internal/sync/install_test.go`

- [ ] **Step 1: Write failing test for ResolveQlikPath**

Append to `internal/sync/install_test.go`:

```go
import (
	"os"
	"path/filepath"
	"strings"
)

func TestResolveQlikPath(t *testing.T) {
	got, err := qsync.ResolveQlikPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should end with /qlik (or \qlik on windows)
	base := filepath.Base(got)
	if runtime.GOOS == "windows" {
		if base != "qlik.exe" {
			t.Errorf("base = %q, want qlik.exe", base)
		}
	} else {
		if base != "qlik" {
			t.Errorf("base = %q, want qlik", base)
		}
	}
	// Should be an absolute path
	if !filepath.IsAbs(got) {
		t.Errorf("path %q is not absolute", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestResolveQlikPath"`
Expected: FAIL — `ResolveQlikPath` not defined

- [ ] **Step 3: Implement ResolveQlikPath**

Add to `internal/sync/install.go` (add `"os"`, `"path/filepath"` to imports):

```go
func ResolveQlikPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine qs location: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("cannot resolve qs symlink: %w", err)
	}
	dir := filepath.Dir(exe)
	name := "qlik"
	if runtime.GOOS == "windows" {
		name = "qlik.exe"
	}
	return filepath.Join(dir, name), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestResolveQlikPath"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/install.go internal/sync/install_test.go
git commit -m "feat(sync): add ResolveQlikPath for sibling binary resolution"
git push
```

---

### Task 5: DownloadFile and EnsureQlikCLI orchestrator

**Files:**
- Modify: `internal/sync/install.go`
- Modify: `internal/sync/install_test.go`

- [ ] **Step 1: Write failing test for DownloadFile**

Append to `internal/sync/install_test.go`:

```go
import (
	"context"
	"net/http"
	"net/http/httptest"
)

func TestDownloadFile(t *testing.T) {
	content := []byte("file-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	got, err := qsync.DownloadFile(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestDownloadFile_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := qsync.DownloadFile(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestDownloadFile"`
Expected: FAIL — `DownloadFile` not defined

- [ ] **Step 3: Implement DownloadFile**

Add to `internal/sync/install.go` (add `"context"`, `"net/http"` to imports):

```go
func DownloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestDownloadFile"`
Expected: PASS

- [ ] **Step 5: Implement EnsureQlikCLI orchestrator**

Add to `internal/sync/install.go`:

```go
func EnsureQlikCLI(ctx context.Context, targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		out, err := RunQlikCmd(ctx, targetPath, "version")
		if err == nil {
			if verErr := CheckVersion(strings.TrimSpace(string(out))); verErr == nil {
				fmt.Printf("qlik-cli found at %s\n", targetPath)
				return nil
			}
		}
		fmt.Printf("Replacing incompatible qlik-cli at %s\n", targetPath)
	}

	fmt.Printf("Installing qlik-cli %s to %s\n", QlikCLIVersion, targetPath)

	asset := DetectAssetName()
	archiveURL := BuildDownloadURL(QlikCLIVersion, asset)
	checksumsURL := BuildChecksumsURL(QlikCLIVersion)

	archive, err := DownloadFile(ctx, archiveURL)
	if err != nil {
		return fmt.Errorf("downloading qlik-cli: %w", err)
	}

	checksums, err := DownloadFile(ctx, checksumsURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	if err := VerifyChecksum(archive, checksums, asset); err != nil {
		return fmt.Errorf("verifying qlik-cli: %w", err)
	}

	binary, err := ExtractBinary(archive, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("extracting qlik-cli: %w", err)
	}

	if err := os.WriteFile(targetPath, binary, 0755); err != nil {
		dir := filepath.Dir(targetPath)
		return fmt.Errorf("cannot install qlik-cli to %s — check permissions or run with sudo", dir)
	}

	out, err := RunQlikCmd(ctx, targetPath, "version")
	if err != nil {
		return fmt.Errorf("verifying installed qlik-cli: %w", err)
	}
	if err := CheckVersion(strings.TrimSpace(string(out))); err != nil {
		return fmt.Errorf("installed qlik-cli version mismatch: %w", err)
	}

	fmt.Printf("qlik-cli %s installed successfully\n", QlikCLIVersion)
	return nil
}
```

- [ ] **Step 6: Run all install tests**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestDownloadFile|TestDetectAsset|TestBuild|TestVerifyChecksum|TestExtractBinary|TestResolveQlikPath"`
Expected: PASS — all install tests pass, `EnsureQlikCLI` compiles

- [ ] **Step 7: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/install.go internal/sync/install_test.go
git commit -m "feat(sync): add DownloadFile and EnsureQlikCLI orchestrator"
git push
```

---

### Task 6: Update CheckPrerequisites to use resolved path

**Files:**
- Modify: `internal/sync/exec.go`
- Modify: `internal/sync/exec_test.go`

- [ ] **Step 1: Write failing test for updated CheckPrerequisites**

`CheckPrerequisites` currently takes `skipVersionCheck bool`. We need to change its signature to accept the resolved binary path. Update `internal/sync/exec_test.go`:

Replace the existing `TestCheckPrerequisites` with:

```go
func TestCheckPrerequisites(t *testing.T) {
	t.Run("skip version check with existing binary", func(t *testing.T) {
		dir := t.TempDir()
		mockPath := filepath.Join(dir, "qlik")
		script := "#!/bin/sh\nprintf 'version: 3.0.0\\tcommit: abc\\tdate: 2026-01-01'\n"
		if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		err := qsync.CheckPrerequisites(mockPath, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing binary", func(t *testing.T) {
		err := qsync.CheckPrerequisites("/nonexistent/path/qlik", false)
		if err == nil {
			t.Error("expected error for missing binary")
		}
		if !strings.Contains(err.Error(), "qs setup") {
			t.Errorf("error should mention 'qs setup', got: %v", err)
		}
	})

	t.Run("incompatible version", func(t *testing.T) {
		dir := t.TempDir()
		mockPath := filepath.Join(dir, "qlik")
		script := "#!/bin/sh\nprintf 'version: 2.0.0\\tcommit: abc\\tdate: 2026-01-01'\n"
		if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		err := qsync.CheckPrerequisites(mockPath, false)
		if err == nil {
			t.Error("expected error for incompatible version")
		}
		if !strings.Contains(err.Error(), "qs setup") {
			t.Errorf("error should mention 'qs setup', got: %v", err)
		}
	})

	t.Run("compatible version", func(t *testing.T) {
		dir := t.TempDir()
		mockPath := filepath.Join(dir, "qlik")
		script := "#!/bin/sh\nprintf 'version: 3.0.0\\tcommit: abc\\tdate: 2026-01-01'\n"
		if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		err := qsync.CheckPrerequisites(mockPath, false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
```

Add `"strings"` to the imports if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestCheckPrerequisites"`
Expected: FAIL — `CheckPrerequisites` signature mismatch (expects 1 arg, now gets 2)

- [ ] **Step 3: Update CheckPrerequisites**

Replace the entire `CheckPrerequisites` function in `internal/sync/exec.go`:

```go
func CheckPrerequisites(qlikPath string, skipVersionCheck bool) error {
	if _, err := os.Stat(qlikPath); err != nil {
		return fmt.Errorf("qlik-cli not found at %s\n\n  Run \"qs setup\" to install it automatically.", qlikPath)
	}

	if skipVersionCheck {
		return nil
	}

	out, err := RunQlikCmd(context.Background(), qlikPath, "version")
	if err != nil {
		return fmt.Errorf("cannot determine qlik-cli version: %w", err)
	}

	if err := CheckVersion(strings.TrimSpace(string(out))); err != nil {
		return fmt.Errorf("%w\n\n  Run \"qs setup\" to install the correct version.", err)
	}

	return nil
}
```

Add `"os"` to the imports in `exec.go`. Remove `"os/exec"` (no longer needed — `exec.LookPath` removed).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestCheckPrerequisites"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/exec.go internal/sync/exec_test.go
git commit -m "refactor(sync): update CheckPrerequisites to accept resolved binary path"
git push
```

---

### Task 7: Update CloudSyncApp, SyncFunc, and RunParallel to accept binary path

**Files:**
- Modify: `internal/sync/exec.go` (CloudSyncApp)
- Modify: `internal/sync/runner.go` (SyncFunc type, RunParallel signature)
- Modify: `internal/sync/runner_test.go` (update test closures and calls)

- [ ] **Step 1: Update SyncFunc type and RunParallel in runner.go**

In `internal/sync/runner.go`, change line 10:

```go
type SyncFunc func(ctx context.Context, app App, configDir string, qlikBinary string) error
```

Change `RunParallel` signature (line 12):

```go
func RunParallel(ctx context.Context, apps []App, configDir string, threads, retries int, qlikBinary string, fn SyncFunc) []Result {
```

Change the `fn` call inside the goroutine (line 28):

```go
err := retry(ctx, retries, func() error { return fn(ctx, a, configDir, qlikBinary) })
```

- [ ] **Step 2: Update CloudSyncApp in exec.go**

In `internal/sync/exec.go`, change `CloudSyncApp`:

```go
func CloudSyncApp(ctx context.Context, app App, configDir string, qlikBinary string) error {
	targetDir := fmt.Sprintf("%s/%s", configDir, app.TargetPath)
	args := BuildUnbuildArgs(app.ResourceID, targetDir)
	_, err := RunQlikCmd(ctx, qlikBinary, args...)
	return err
}
```

- [ ] **Step 3: Update runner_test.go**

All three test functions need updated closures and `RunParallel` calls.

In `TestRunParallel` (line 20-25):

```go
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		called.Add(1)
		return nil
	}

	results := qsync.RunParallel(context.Background(), apps, "testdir", 2, 1, "qlik", syncFn)
```

In `TestRunParallel_Retry` (line 53-62):

```go
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	}

	apps := []qsync.App{{ResourceID: "1", Name: "Flaky", TargetPath: "path/1"}}
	results := qsync.RunParallel(context.Background(), apps, "testdir", 1, 3, "qlik", syncFn)
```

In `TestRunParallel_ExhaustedRetries` (line 73-78):

```go
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		return errors.New("permanent error")
	}

	apps := []qsync.App{{ResourceID: "1", Name: "Broken", TargetPath: "path/1"}}
	results := qsync.RunParallel(context.Background(), apps, "testdir", 1, 2, "qlik", syncFn)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./internal/sync/ -v -run "TestRunParallel"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add internal/sync/exec.go internal/sync/runner.go internal/sync/runner_test.go
git commit -m "refactor(sync): add qlikBinary param to CloudSyncApp, SyncFunc, and RunParallel"
git push
```

---

### Task 8: Update cmd callers to use resolved qlik path

**Files:**
- Modify: `cmd/setup.go`
- Modify: `cmd/sync.go`

- [ ] **Step 1: Update cmd/setup.go**

Replace the `runSetup` function in `cmd/setup.go`:

```go
func runSetup(cmd *cobra.Command, args []string) error {
	qlikPath, err := qsync.ResolveQlikPath()
	if err != nil {
		return err
	}

	if err := qsync.EnsureQlikCLI(cmd.Context(), qlikPath); err != nil {
		return err
	}

	if err := qsync.CheckPrerequisites(qlikPath, skipVersionCheck); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	// List existing contexts
	out, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "context", "ls")
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
	checkCmd := exec.Command(qlikPath, "context", "get", contextName)
	if err := checkCmd.Run(); err != nil {
		fmt.Print("Enter API key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		createArgs := []string{"context", "create", contextName, "--server", server, "--api-key", apiKey}
		if tenantType == "on-prem" {
			createArgs = append(createArgs, "--server-type", "Windows", "--insecure")
		}
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, createArgs...); err != nil {
			return fmt.Errorf("creating context: %w", err)
		}
		fmt.Println("Context created.")
	}

	// Set active context
	if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "context", "use", contextName); err != nil {
		return fmt.Errorf("setting active context: %w", err)
	}

	// Test connectivity
	fmt.Println("Testing connectivity...")
	if tenantType == "cloud" {
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "app", "ls", "--limit", "1"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	} else {
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "qrs", "app", "count"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	}
	fmt.Println("Connected successfully.")

	// Read or create config
	cfg, err := config.Read(configDir)
	if err != nil {
		cfg = &config.Config{Version: "0.2.0"}
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
		cfg.Tenants = append(cfg.Tenants, config.Tenant{Context: contextName, Server: server, Type: tenantType})
	}

	if err := config.Write(configDir, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nSetup complete. Run: qs sync\n")
	return nil
}
```

- [ ] **Step 2: Update cmd/sync.go**

Add `qlikPath` resolution at the top of `runSync` and pass it through:

Replace the beginning of `runSync`:

```go
func runSync(cmd *cobra.Command, args []string) error {
	qlikPath, err := qsync.ResolveQlikPath()
	if err != nil {
		return err
	}

	if err := qsync.CheckPrerequisites(qlikPath, skipVersionCheck); err != nil {
		return err
	}
```

Update the `ResolveOwnerNames` call (line 93):

```go
apps = qsync.ResolveOwnerNames(ctx, apps, qlikPath)
```

Update the `RunParallel` call (line 108):

```go
results := qsync.RunParallel(ctx, apps, configDir, resolved.Threads, resolved.Retries, qlikPath, qsync.CloudSyncApp)
```

Update `prepTenant` signature to accept `qlikPath`:

```go
func prepTenant(ctx context.Context, tenant config.Tenant, filters qsync.Filters, qlikPath string) ([]qsync.App, map[string]qsync.SpaceInfo, error) {
```

And update the `RunQlikCmd` calls inside `prepTenant` (currently at lines 148 and 167):

```go
spacesOut, err := qsync.RunQlikCmd(ctx, qlikPath, qsync.BuildSpaceListArgs()...)
```

```go
appsOut, err := qsync.RunQlikCmd(ctx, qlikPath, qsync.BuildAppListArgs(spaceID)...)
```

Update the call to `prepTenant` in the loop (line 84):

```go
apps, spaces, err := prepTenant(ctx, tenant, filters, qlikPath)
```

- [ ] **Step 3: Run all tests**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./... 2>&1`
Expected: PASS — all packages compile and tests pass

- [ ] **Step 4: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add cmd/setup.go cmd/sync.go
git commit -m "refactor(cmd): replace hardcoded qlik binary with resolved path"
git push
```

---

### Task 9: Update integration tests

**Files:**
- Modify: `test/integration/sync_test.go`

- [ ] **Step 1: Update TestSyncEndToEnd**

The integration test currently puts a mock qlik in PATH. Now `qs` resolves qlik next to itself, so the mock needs to be placed next to the built `qs` binary.

In `test/integration/sync_test.go`, replace the mock setup section of `TestSyncEndToEnd`:

Change from symlinking into a PATH directory to placing mock next to `qs`:

```go
func TestSyncEndToEnd(t *testing.T) {
	// Build the binary — resolve module root relative to the test file location.
	testDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(testDir, "..", "..")
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "qs")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Place mock qlik next to qs binary
	mockDir, _ := filepath.Abs(".")
	mockScript := filepath.Join(mockDir, "mock-qlik.sh")
	if err := os.Chmod(mockScript, 0755); err != nil {
		t.Fatal(err)
	}
	mockLink := filepath.Join(binDir, "qlik")
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

- [ ] **Step 2: Update TestSyncRejectsIncompatibleVersion**

Same change — place mock qlik next to built qs binary:

```go
func TestSyncRejectsIncompatibleVersion(t *testing.T) {
	// Build the binary
	testDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(testDir, "..", "..")
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "qs")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Create mock qlik next to qs that returns incompatible version
	mockPath := filepath.Join(binDir, "qlik")
	mockScript := "#!/bin/sh\nprintf 'version: 2.0.0\\tcommit: mock\\tdate: 2026-01-01T00:00:00Z'\n"
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
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

	// Run qs sync — should fail due to version mismatch
	cmd := exec.Command(binPath, "sync")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected sync to fail with incompatible version, but it succeeded")
	}
	if !strings.Contains(string(out), "not compatible") {
		t.Errorf("expected 'not compatible' in output, got: %s", string(out))
	}
}
```

- [ ] **Step 3: Run integration tests**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./test/integration/ -v -timeout 30s`
Expected: PASS

- [ ] **Step 4: Run full test suite**

Run: `cd /workspaces/qlik-sync/.worktrees/qlik-cli-install && go test ./...`
Expected: PASS — all packages pass

- [ ] **Step 5: Commit**

```bash
cd /workspaces/qlik-sync/.worktrees/qlik-cli-install
git add test/integration/sync_test.go
git commit -m "test(integration): place mock qlik next to qs binary instead of PATH"
git push
```
