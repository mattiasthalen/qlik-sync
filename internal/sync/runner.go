package sync

import (
	"context"
	"fmt"
	gosync "sync"
	"time"
)

type SyncFunc func(ctx context.Context, app App, configDir string, qlikBinary string) error

func RunParallel(ctx context.Context, apps []App, configDir string, threads, retries int, qlikBinary string, fn SyncFunc) []Result {
	results := make([]Result, len(apps))
	sem := make(chan struct{}, threads)
	var wg gosync.WaitGroup

	for i, app := range apps {
		if app.Skip {
			results[i] = Result{ResourceID: app.ResourceID, Status: "skipped"}
			continue
		}
		wg.Add(1)
		go func(idx int, a App) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := retry(ctx, retries, func() error { return fn(ctx, a, configDir, qlikBinary) })
			if err != nil {
				results[idx] = Result{ResourceID: a.ResourceID, Status: "error", Error: err.Error()}
			} else {
				results[idx] = Result{ResourceID: a.ResourceID, Status: "synced"}
			}
		}(i, app)
	}
	wg.Wait()
	return results
}

func retry(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("cancelled: %w", err)
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt < maxAttempts {
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("cancelled: %w", ctx.Err())
			}
		}
	}
	return lastErr
}

func Summarize(results []Result) (synced, skipped, errors int) {
	for _, r := range results {
		switch r.Status {
		case "synced":
			synced++
		case "skipped":
			skipped++
		case "error":
			errors++
		}
	}
	return
}
