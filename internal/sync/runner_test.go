package sync_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestRunParallel(t *testing.T) {
	apps := []qsync.App{
		{ResourceID: "1", Name: "App1", TargetPath: "path/1"},
		{ResourceID: "2", Name: "App2", TargetPath: "path/2"},
		{ResourceID: "3", Name: "App3", TargetPath: "path/3", Skip: true},
	}

	var called atomic.Int32
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		called.Add(1)
		return nil
	}

	results := qsync.RunParallel(context.Background(), apps, "testdir", 2, 1, "qlik", syncFn)

	if int(called.Load()) != 2 {
		t.Errorf("syncFn called %d times, want 2", called.Load())
	}
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	synced, skipped := 0, 0
	for _, r := range results {
		switch r.Status {
		case "synced":
			synced++
		case "skipped":
			skipped++
		}
	}
	if synced != 2 {
		t.Errorf("synced = %d, want 2", synced)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestRunParallel_Retry(t *testing.T) {
	var attempts atomic.Int32
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	}

	apps := []qsync.App{{ResourceID: "1", Name: "Flaky", TargetPath: "path/1"}}
	results := qsync.RunParallel(context.Background(), apps, "testdir", 1, 3, "qlik", syncFn)

	if results[0].Status != "synced" {
		t.Errorf("status = %q, want synced", results[0].Status)
	}
	if int(attempts.Load()) != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
}

func TestRunParallel_ExhaustedRetries(t *testing.T) {
	syncFn := func(ctx context.Context, app qsync.App, configDir string, qlikBinary string) error {
		return errors.New("permanent error")
	}

	apps := []qsync.App{{ResourceID: "1", Name: "Broken", TargetPath: "path/1"}}
	results := qsync.RunParallel(context.Background(), apps, "testdir", 1, 2, "qlik", syncFn)

	if results[0].Status != "error" {
		t.Errorf("status = %q, want error", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected error message")
	}
}
