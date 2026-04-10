package sync_test

import (
	"os"
	"path/filepath"
	"testing"

	sync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

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
