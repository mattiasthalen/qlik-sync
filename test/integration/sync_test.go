package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestSyncEndToEnd(t *testing.T) {
	// Build the binary — resolve module root relative to the test file location.
	testDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(testDir, "..", "..")
	binPath := filepath.Join(t.TempDir(), "qs")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Set up mock qlik next to the qs binary (ResolveQlikPath looks there)
	mockDir, _ := filepath.Abs(".")
	mockScript := filepath.Join(mockDir, "mock-qlik.sh")
	if err := os.Chmod(mockScript, 0755); err != nil {
		t.Fatal(err)
	}

	// Create symlink so mock is found as "qlik" next to qs binary
	binDir := filepath.Dir(binPath)
	mockLink := filepath.Join(binDir, "qlik")
	if err := os.Symlink(mockScript, mockLink); err != nil {
		t.Fatal(err)
	}
	pathDir := binDir

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

func TestSyncRejectsIncompatibleVersion(t *testing.T) {
	// Build the binary
	testDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(testDir, "..", "..")
	binPath := filepath.Join(t.TempDir(), "qs")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Create mock qlik next to qs binary that returns incompatible version
	binDir := filepath.Dir(binPath)
	pathDir := binDir
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
	cmd.Env = append(os.Environ(), "PATH="+pathDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected sync to fail with incompatible version, but it succeeded")
	}
	if !strings.Contains(string(out), "not compatible") {
		t.Errorf("expected 'not compatible' in output, got: %s", string(out))
	}
}
