package sync

import (
	"testing"
)

var testApps = []App{
	{ResourceID: "1", Name: "Sales Dashboard", SpaceName: "Finance", SpaceID: "sp1", SpaceType: "shared", AppType: "qvf", Tenant: "mytenant", TenantID: "tid1"},
	{ResourceID: "2", Name: "Sales Pipeline", SpaceName: "Finance", SpaceID: "sp1", SpaceType: "shared", AppType: "qvf", Tenant: "mytenant", TenantID: "tid1"},
	{ResourceID: "3", Name: "HR Overview", SpaceName: "HR", SpaceID: "sp2", SpaceType: "managed", AppType: "qvf", Tenant: "mytenant", TenantID: "tid1"},
}

func TestFilterBySpace(t *testing.T) {
	result := FilterBySpace(testApps, "Finance")
	if len(result) != 2 {
		t.Errorf("expected 2 apps, got %d", len(result))
	}
}

func TestFilterByApp(t *testing.T) {
	result, err := FilterByApp(testApps, "Sales")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 apps, got %d", len(result))
	}
}

func TestFilterByApp_InvalidRegex(t *testing.T) {
	_, err := FilterByApp(testApps, "[invalid")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestFilterByID(t *testing.T) {
	apps := []App{
		{ResourceID: "abc123", Name: "App One"},
		{ResourceID: "xyz789", Name: "App Two"},
	}
	result := FilterByID(apps, "abc123")
	if len(result) != 1 {
		t.Errorf("expected 1 app, got %d", len(result))
	}
	if result[0].ResourceID != "abc123" {
		t.Errorf("expected resourceId abc123, got %s", result[0].ResourceID)
	}
}

func TestApplyFilters(t *testing.T) {
	f := Filters{Space: "Finance", App: "Pipeline"}
	result, err := ApplyFilters(testApps, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 app, got %d", len(result))
	}
	if result[0].Name != "Sales Pipeline" {
		t.Errorf("expected 'Sales Pipeline', got %s", result[0].Name)
	}
}

func TestSanitize(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Sales/Revenue", "Sales_Revenue"},
		{`My App: "Test"`, "My App_ _Test_"},
		{"Normal Name", "Normal Name"},
	}
	for _, c := range cases {
		got := Sanitize(c.input)
		if got != c.expected {
			t.Errorf("Sanitize(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestBuildTargetPath(t *testing.T) {
	appWithType := App{
		ResourceID: "r1",
		Name:       "Sales Dashboard",
		SpaceID:    "sp1",
		SpaceName:  "Finance",
		SpaceType:  "shared",
		AppType:    "qvf",
		Tenant:     "mytenant",
		TenantID:   "tid1",
	}
	got := BuildTargetPath(appWithType)
	want := "mytenant (tid1)/shared/Finance (sp1)/qvf/Sales Dashboard (r1)"
	if got != want {
		t.Errorf("BuildTargetPath with appType: got %q, want %q", got, want)
	}

	appNoType := App{
		ResourceID: "r2",
		Name:       "HR Overview",
		SpaceID:    "sp2",
		SpaceName:  "HR",
		SpaceType:  "managed",
		AppType:    "",
		Tenant:     "mytenant",
		TenantID:   "tid1",
	}
	got2 := BuildTargetPath(appNoType)
	want2 := "mytenant (tid1)/managed/HR (sp2)/HR Overview (r2)"
	if got2 != want2 {
		t.Errorf("BuildTargetPath without appType: got %q, want %q", got2, want2)
	}

	personalApp := App{
		ResourceID: "r3",
		Name:       "My App",
		SpaceID:    "",
		SpaceName:  "Jane Doe",
		SpaceType:  "personal",
		AppType:    "analytics",
		OwnerID:    "user-001",
		Tenant:     "mytenant",
		TenantID:   "tid1",
	}
	got3 := BuildTargetPath(personalApp)
	want3 := "mytenant (tid1)/personal/Jane Doe (user-001)/analytics/My App (r3)"
	if got3 != want3 {
		t.Errorf("BuildTargetPath personal: got %q, want %q", got3, want3)
	}
}
