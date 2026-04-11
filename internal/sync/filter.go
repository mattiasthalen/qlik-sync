package sync

import (
	"fmt"
	"regexp"
	"strings"
)

// FilterBySpace returns apps whose SpaceName matches the given space string exactly.
func FilterBySpace(apps []App, space string) []App {
	result := make([]App, 0, len(apps))
	for _, a := range apps {
		if a.SpaceName == space {
			result = append(result, a)
		}
	}
	return result
}

// FilterByApp returns apps whose Name matches the given regex pattern.
func FilterByApp(apps []App, pattern string) []App {
	re := regexp.MustCompile(pattern)
	result := make([]App, 0, len(apps))
	for _, a := range apps {
		if re.MatchString(a.Name) {
			result = append(result, a)
		}
	}
	return result
}

// FilterByID returns apps whose ResourceID matches the given id exactly.
func FilterByID(apps []App, id string) []App {
	result := make([]App, 0, len(apps))
	for _, a := range apps {
		if a.ResourceID == id {
			result = append(result, a)
		}
	}
	return result
}

// ApplyFilters chains Space and App filters; if ID is set it short-circuits and returns only ID match.
func ApplyFilters(apps []App, f Filters) []App {
	if f.ID != "" {
		return FilterByID(apps, f.ID)
	}
	result := apps
	if f.Space != "" {
		result = FilterBySpace(result, f.Space)
	}
	if f.App != "" {
		result = FilterByApp(result, f.App)
	}
	return result
}

// Sanitize replaces filesystem-unsafe characters /\:*?"<>| with underscore.
func Sanitize(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		`\`, "_",
		":", "_",
		"*", "_",
		"?", "_",
		`"`, "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// BuildTargetPath builds the target path for an app.
// Format: tenant (tenantId)/spaceType/spaceName (spaceId)/appType/appName (resourceId)
// For personal space: tenant (tenantId)/personal/ownerName (ownerId)/appType/appName (resourceId)
// The appType level is omitted when AppType is empty.
func BuildTargetPath(app App) string {
	tenant := fmt.Sprintf("%s (%s)", Sanitize(app.Tenant), Sanitize(app.TenantID))
	appName := fmt.Sprintf("%s (%s)", Sanitize(app.Name), Sanitize(app.ResourceID))

	var space string
	if app.SpaceID == "" {
		// Personal space: use owner name and ID
		ownerName := app.SpaceName
		if ownerName == "" || ownerName == "personal" {
			ownerName = app.OwnerID
		}
		space = fmt.Sprintf("%s (%s)", Sanitize(ownerName), Sanitize(app.OwnerID))
	} else {
		space = fmt.Sprintf("%s (%s)", Sanitize(app.SpaceName), Sanitize(app.SpaceID))
	}

	if app.AppType == "" {
		return fmt.Sprintf("%s/%s/%s/%s", tenant, Sanitize(app.SpaceType), space, appName)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", tenant, Sanitize(app.SpaceType), space, Sanitize(app.AppType), appName)
}
