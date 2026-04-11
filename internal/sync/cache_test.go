package sync_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestCacheWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	key := "test-cache-key"
	data := []byte(`{"tenant":"test","apps":[]}`)
	if err := qsync.CacheWrite(dir, key, data); err != nil { t.Fatalf("write: %v", err) }
	got, err := qsync.CacheRead(dir, key, 5*time.Minute)
	if err != nil { t.Fatalf("read: %v", err) }
	if string(got) != string(data) { t.Errorf("data = %q, want %q", string(got), string(data)) }
}

func TestCacheRead_Expired(t *testing.T) {
	dir := t.TempDir()
	key := "expired-key"
	data := []byte(`{"expired":true}`)
	path := filepath.Join(dir, "qs-cache-"+key+".json")
	if err := os.WriteFile(path, data, 0644); err != nil { t.Fatal(err) }
	past := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, past, past); err != nil { t.Fatal(err) }
	got, err := qsync.CacheRead(dir, key, 5*time.Minute)
	if err != nil { t.Fatalf("unexpected: %v", err) }
	if got != nil { t.Error("expected nil for expired cache") }
}

func TestCacheRead_Missing(t *testing.T) {
	dir := t.TempDir()
	got, err := qsync.CacheRead(dir, "nonexistent", 5*time.Minute)
	if err != nil { t.Fatalf("unexpected: %v", err) }
	if got != nil { t.Error("expected nil for missing cache") }
}

func TestBuildCacheKey(t *testing.T) {
	key1 := qsync.BuildCacheKey("ctx1", "Finance", "", "", "/workdir")
	key2 := qsync.BuildCacheKey("ctx1", "HR", "", "", "/workdir")
	key3 := qsync.BuildCacheKey("ctx1", "Finance", "", "", "/workdir")
	if key1 == key2 { t.Error("different filters should produce different keys") }
	if key1 != key3 { t.Error("same inputs should produce same key") }
}
