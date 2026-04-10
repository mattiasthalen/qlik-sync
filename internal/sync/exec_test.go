package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestBuildUnbuildArgs(t *testing.T) {
	args := qsync.BuildUnbuildArgs("app-001", "/tmp/test/path")
	expected := []string{"app", "unbuild", "--app", "app-001", "--dir", "/tmp/test/path"}
	if len(args) != len(expected) { t.Fatalf("len = %d, want %d", len(args), len(expected)) }
	for i, arg := range args {
		if arg != expected[i] { t.Errorf("args[%d] = %q, want %q", i, arg, expected[i]) }
	}
}

func TestBuildSpaceListArgs(t *testing.T) {
	args := qsync.BuildSpaceListArgs()
	expected := []string{"space", "ls", "--json"}
	if len(args) != len(expected) { t.Fatalf("len = %d, want %d", len(args), len(expected)) }
	for i, arg := range args {
		if arg != expected[i] { t.Errorf("args[%d] = %q, want %q", i, arg, expected[i]) }
	}
}

func TestBuildAppListArgs(t *testing.T) {
	t.Run("no space filter", func(t *testing.T) {
		args := qsync.BuildAppListArgs("")
		if len(args) != 5 { t.Fatalf("len = %d, want 5", len(args)) }
	})
	t.Run("with space filter", func(t *testing.T) {
		args := qsync.BuildAppListArgs("space-001")
		if len(args) != 7 { t.Fatalf("len = %d, want 7", len(args)) }
		if args[5] != "--spaceId" || args[6] != "space-001" {
			t.Errorf("last args = %v, want [--spaceId space-001]", args[5:])
		}
	})
}

func TestRunQlikCmd(t *testing.T) {
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "qlik")
	script := "#!/bin/sh\necho '[{\"id\":\"test\"}]'\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil { t.Fatal(err) }

	out, err := qsync.RunQlikCmd(context.Background(), mockPath, "space", "ls", "--json")
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if string(out) != "[{\"id\":\"test\"}]\n" {
		t.Errorf("output = %q, want %q", string(out), "[{\"id\":\"test\"}]\n")
	}
}

func TestCheckPrerequisites(t *testing.T) {
	// Just verify function exists and returns error type
	err := qsync.CheckPrerequisites()
	_ = err // depends on environment
}
