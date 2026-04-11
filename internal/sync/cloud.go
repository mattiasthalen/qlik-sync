package sync

import (
	"encoding/json"
	"strings"
)

// SpaceInfo holds the id, name, and type of a Qlik Cloud space.
type SpaceInfo struct {
	ID   string
	Name string
	Type string
}

// cloudSpace is the raw JSON shape returned by qlik-cli for spaces.
type cloudSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParseCloudSpaces decodes a JSON array of spaces and returns a map keyed by space ID.
func ParseCloudSpaces(data []byte) (map[string]SpaceInfo, error) {
	var raw []cloudSpace
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make(map[string]SpaceInfo, len(raw))
	for _, s := range raw {
		result[s.ID] = SpaceInfo(s)
	}
	return result, nil
}

// NormalizeAppType converts an uppercase underscore-separated usage string
// (e.g. "DATAFLOW_PREP") to its lowercase hyphenated form (e.g. "dataflow-prep").
func NormalizeAppType(usage string) string {
	return strings.ToLower(strings.ReplaceAll(usage, "_", "-"))
}

// cloudAppAttributes is the nested resourceAttributes object in the apps JSON.
type cloudAppAttributes struct {
	Usage          string `json:"usage"`
	LastReloadTime string `json:"lastReloadTime"`
}

// cloudApp is the raw JSON shape returned by qlik-cli for items/apps.
type cloudApp struct {
	ResourceID         string             `json:"resourceId"`
	Name               string             `json:"name"`
	SpaceID            string             `json:"spaceId"`
	OwnerID            string             `json:"ownerId"`
	Description        string             `json:"description"`
	TenantID           string             `json:"tenantId"`
	ResourceAttributes cloudAppAttributes `json:"resourceAttributes"`
}

// ParseCloudApps decodes a JSON array of cloud apps, resolves space names/types
// from the provided spaces map, normalizes app types, and builds a TargetPath
// for each app using the supplied tenant name and ID.
func ParseCloudApps(data []byte, spaces map[string]SpaceInfo, tenant, tenantID string) ([]App, error) {
	var raw []cloudApp
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	apps := make([]App, 0, len(raw))
	for _, r := range raw {
		space := spaces[r.SpaceID]
		appType := NormalizeAppType(r.ResourceAttributes.Usage)

		// Use tenantId from API response if available, fall back to provided value
		tid := tenantID
		if r.TenantID != "" {
			tid = r.TenantID
		}

		// Handle personal space (no spaceId)
		spaceName := space.Name
		spaceType := space.Type
		if r.SpaceID == "" {
			spaceName = "personal"
			spaceType = "personal"
		}

		a := App{
			ResourceID:     r.ResourceID,
			Name:           r.Name,
			SpaceID:        r.SpaceID,
			SpaceName:      spaceName,
			SpaceType:      spaceType,
			AppType:        appType,
			OwnerID:        r.OwnerID,
			Description:    r.Description,
			LastReloadTime: r.ResourceAttributes.LastReloadTime,
			Tenant:         tenant,
			TenantID:       tid,
		}
		a.TargetPath = BuildTargetPath(a)
		apps = append(apps, a)
	}
	return apps, nil
}
