package sync_test

import (
	"os"
	"testing"

	sync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestParseCloudSpaces(t *testing.T) {
	data, err := os.ReadFile("../../testdata/cloud/spaces.json")
	if err != nil {
		t.Fatal(err)
	}
	spaces, err := sync.ParseCloudSpaces(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spaces) != 2 {
		t.Fatalf("count = %d, want 2", len(spaces))
	}
	if spaces["space-001"].Name != "Finance Prod" {
		t.Errorf("name = %q, want %q", spaces["space-001"].Name, "Finance Prod")
	}
}

func TestNormalizeAppType(t *testing.T) {
	tests := []struct{ input, want string }{
		{"ANALYTICS", "analytics"},
		{"DATAFLOW_PREP", "dataflow-prep"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sync.NormalizeAppType(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeAppType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseCloudApps(t *testing.T) {
	spacesData, _ := os.ReadFile("../../testdata/cloud/spaces.json")
	spaces, _ := sync.ParseCloudSpaces(spacesData)
	appsData, _ := os.ReadFile("../../testdata/cloud/apps.json")

	apps, err := sync.ParseCloudApps(appsData, spaces, "my-tenant", "tenant-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("count = %d, want 2", len(apps))
	}
	app := apps[0]
	if app.ResourceID != "app-001" {
		t.Errorf("resourceId = %q, want app-001", app.ResourceID)
	}
	if app.SpaceName != "Finance Prod" {
		t.Errorf("spaceName = %q, want Finance Prod", app.SpaceName)
	}
	if app.SpaceType != "managed" {
		t.Errorf("spaceType = %q, want managed", app.SpaceType)
	}
	if app.AppType != "analytics" {
		t.Errorf("appType = %q, want analytics", app.AppType)
	}
}

func TestParseCloudApps_TenantIDFromAPI(t *testing.T) {
	spacesData, _ := os.ReadFile("../../testdata/cloud/spaces.json")
	spaces, _ := sync.ParseCloudSpaces(spacesData)

	appsJSON := `[{
		"resourceId": "app-001", "name": "Test",
		"spaceId": "space-001", "ownerId": "user-001",
		"tenantId": "from-api-123",
		"resourceAttributes": {"usage": "ANALYTICS"}
	}]`

	apps, err := sync.ParseCloudApps([]byte(appsJSON), spaces, "my-tenant", "")
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].TenantID != "from-api-123" {
		t.Errorf("tenantID = %q, want %q", apps[0].TenantID, "from-api-123")
	}
}

func TestParseCloudApps_PersonalSpace(t *testing.T) {
	spaces := make(map[string]sync.SpaceInfo)

	appsJSON := `[{
		"resourceId": "app-001", "name": "My Personal App",
		"spaceId": "", "ownerId": "user-001",
		"tenantId": "tid-123",
		"resourceAttributes": {"usage": "ANALYTICS"}
	}]`

	apps, err := sync.ParseCloudApps([]byte(appsJSON), spaces, "my-tenant", "")
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].SpaceName != "personal" {
		t.Errorf("spaceName = %q, want %q", apps[0].SpaceName, "personal")
	}
	if apps[0].SpaceType != "personal" {
		t.Errorf("spaceType = %q, want %q", apps[0].SpaceType, "personal")
	}
}
