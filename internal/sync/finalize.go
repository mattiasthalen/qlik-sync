package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// BuildIndex constructs an Index from a PrepOutput and the sync results.
// Only apps with status "synced" or "skipped" are included; errored apps are excluded.
func BuildIndex(prep PrepOutput, results []Result) Index {
	// Build set of non-errored app IDs
	okIDs := make(map[string]bool, len(results))
	for _, r := range results {
		if r.Status == "synced" || r.Status == "skipped" {
			okIDs[r.ResourceID] = true
		}
	}

	apps := make(map[string]IndexEntry, len(prep.Apps))
	for _, a := range prep.Apps {
		if !okIDs[a.ResourceID] {
			continue
		}
		apps[a.ResourceID] = IndexEntry{
			Name:           a.Name,
			Space:          a.SpaceName,
			SpaceID:        a.SpaceID,
			SpaceType:      a.SpaceType,
			AppType:        a.AppType,
			Owner:          a.OwnerID,
			OwnerName:      a.OwnerName,
			Description:    a.Description,
			Tags:           a.Tags,
			Published:      a.Published,
			LastReloadTime: a.LastReloadTime,
			Path:           a.TargetPath,
		}
	}
	return Index{
		LastSync: time.Now().UTC().Format(time.RFC3339),
		Context:  prep.Context,
		Server:   prep.Server,
		Tenant:   prep.Tenant,
		TenantID: prep.TenantID,
		AppCount: len(prep.Apps),
		Apps:     apps,
	}
}

// WriteIndex marshals index to JSON and writes it to dir/index.json.
func WriteIndex(dir string, index Index) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.json"), data, 0644)
}

// ReadIndex reads dir/index.json; returns an empty Index if the file is missing.
func ReadIndex(dir string) (Index, error) {
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if os.IsNotExist(err) {
		return Index{Apps: map[string]IndexEntry{}}, nil
	}
	if err != nil {
		return Index{}, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// MergeIndex merges existing entries into the new index for any keys not already present,
// then updates AppCount to reflect the combined set.
func MergeIndex(existing, new Index) Index {
	merged := Index{
		LastSync: new.LastSync,
		Context:  new.Context,
		Server:   new.Server,
		Tenant:   new.Tenant,
		TenantID: new.TenantID,
		Apps:     make(map[string]IndexEntry, len(new.Apps)+len(existing.Apps)),
	}
	for k, v := range new.Apps {
		merged.Apps[k] = v
	}
	for k, v := range existing.Apps {
		if _, ok := merged.Apps[k]; !ok {
			merged.Apps[k] = v
		}
	}
	merged.AppCount = len(merged.Apps)
	return merged
}
