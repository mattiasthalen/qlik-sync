package sync_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestBuildIndex(t *testing.T) {
	prep := sync.PrepOutput{
		Tenant: "my-tenant", TenantID: "abc-123",
		Context: "my-context", Server: "https://tenant.qlikcloud.com",
		Apps: []sync.App{
			{ResourceID: "app-001", Name: "Sales Dashboard", SpaceName: "Finance", SpaceID: "space-001", SpaceType: "managed", AppType: "analytics", OwnerID: "user-001", OwnerName: "jane.doe", TargetPath: "my-tenant (abc-123)/managed/Finance (space-001)/analytics/Sales Dashboard (app-001)"},
			{ResourceID: "app-002", Name: "HR Report", SpaceName: "HR", SpaceID: "space-002", SpaceType: "shared", AppType: "analytics", OwnerID: "user-002", OwnerName: "john.smith", TargetPath: "my-tenant (abc-123)/shared/HR (space-002)/analytics/HR Report (app-002)"},
		},
	}
	results := []sync.Result{
		{ResourceID: "app-001", Status: "synced"},
		{ResourceID: "app-002", Status: "skipped"},
	}

	index := sync.BuildIndex(prep, results)
	if index.Tenant != "my-tenant" {
		t.Errorf("tenant = %q, want my-tenant", index.Tenant)
	}
	if index.AppCount != 2 {
		t.Errorf("appCount = %d, want 2", index.AppCount)
	}
	if _, ok := index.Apps["app-001"]; !ok {
		t.Fatal("app-001 missing")
	}
	if index.Apps["app-001"].Space != "Finance" {
		t.Errorf("space = %q, want Finance", index.Apps["app-001"].Space)
	}
}

func TestWriteIndex(t *testing.T) {
	dir := t.TempDir()
	index := sync.Index{Tenant: "test", TenantID: "123", AppCount: 1, Apps: map[string]sync.IndexEntry{"app-001": {Name: "Test App", Path: "test/path"}}}
	if err := sync.WriteIndex(dir, index); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got sync.Index
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.AppCount != 1 {
		t.Errorf("appCount = %d, want 1", got.AppCount)
	}
}

func TestMergeIndex(t *testing.T) {
	existing := sync.Index{Tenant: "test", AppCount: 1, Apps: map[string]sync.IndexEntry{"old-app": {Name: "Old App"}}}
	newIdx := sync.Index{Tenant: "test", AppCount: 1, Apps: map[string]sync.IndexEntry{"new-app": {Name: "New App"}}}
	merged := sync.MergeIndex(existing, newIdx)
	if merged.AppCount != 2 {
		t.Errorf("appCount = %d, want 2", merged.AppCount)
	}
	if _, ok := merged.Apps["old-app"]; !ok {
		t.Error("old-app missing")
	}
	if _, ok := merged.Apps["new-app"]; !ok {
		t.Error("new-app missing")
	}
}
