package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncSpace   string
	syncStream  string
	syncApp     string
	syncID      string
	syncTenant  string
	syncForce   bool
	syncThreads int
	syncRetries int
)

// ErrPartialSync indicates some apps failed to sync.
var ErrPartialSync = fmt.Errorf("partial sync failure")

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Qlik apps to local files",
	Long:  "Pull Qlik Sense cloud and on-prem apps into the local qlik/ directory.",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncSpace, "space", "", "filter by space name (cloud)")
	syncCmd.Flags().StringVar(&syncStream, "stream", "", "filter by stream name (on-prem)")
	syncCmd.Flags().StringVar(&syncApp, "app", "", "regex filter on app name")
	syncCmd.Flags().StringVar(&syncID, "id", "", "exact app ID")
	syncCmd.Flags().StringVar(&syncTenant, "tenant", "", "filter by tenant context")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip cache, re-sync all")
	syncCmd.Flags().IntVar(&syncThreads, "threads", 0, "concurrent syncs (overrides config)")
	syncCmd.Flags().IntVar(&syncRetries, "retries", 0, "retry count per app (overrides config)")
	rootCmd.AddCommand(syncCmd)
}

const cacheTTL = 5 * time.Minute

func runSync(cmd *cobra.Command, args []string) error {
	if err := qsync.CheckPrerequisites(skipVersionCheck); err != nil {
		return err
	}

	cfg, err := config.Read(configDir)
	if err != nil {
		return fmt.Errorf("reading config: %w\n  Run: qs setup", err)
	}

	var flagThreads, flagRetries *int
	if cmd.Flags().Changed("threads") {
		flagThreads = &syncThreads
	}
	if cmd.Flags().Changed("retries") {
		flagRetries = &syncRetries
	}
	resolved := config.Resolve(cfg, flagThreads, flagRetries)

	tenants := config.FilterTenants(resolved.Tenants, syncTenant)
	if len(tenants) == 0 {
		return fmt.Errorf("no tenants found for context %q\n  Run: qs setup", syncTenant)
	}

	ctx := context.Background()
	filters := qsync.Filters{Space: syncSpace, Stream: syncStream, App: syncApp, ID: syncID}

	hadErrors := false
	for _, tenant := range tenants {
		if tenant.Type != "cloud" {
			fmt.Fprintf(os.Stderr, "Skipping on-prem tenant %q (not yet supported)\n", tenant.Context)
			continue
		}

		apps, spaces, err := prepTenant(ctx, tenant, filters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing %q: %v\n", tenant.Context, err)
			hadErrors = true
			continue
		}

		_ = spaces // used during prep

		apps = qsync.ResolveOwnerNames(ctx, apps, "qlik")

		apps, err = qsync.ApplyFilters(apps, filters)
		if err != nil {
			return err
		}

		if syncForce {
			apps = qsync.MarkSkippedForce(apps)
		} else {
			apps = qsync.MarkSkipped(apps, configDir)
		}

		fmt.Printf("Syncing %s (%d apps, %d threads)...\n", tenant.Context, len(apps), resolved.Threads)

		results := qsync.RunParallel(ctx, apps, configDir, resolved.Threads, resolved.Retries, qsync.CloudSyncApp)

		prep := qsync.PrepOutput{Tenant: tenant.Context, TenantID: "", Context: tenant.Context, Server: tenant.Server, Apps: apps}
		index := qsync.BuildIndex(prep, results)
		existing, _ := qsync.ReadIndex(configDir)
		merged := qsync.MergeIndex(existing, index)
		if err := qsync.WriteIndex(configDir, merged); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing index: %v\n", err)
		}

		synced, skipped, syncErrors := qsync.Summarize(results)
		fmt.Printf("%d synced, %d skipped, %d errors\n", synced, skipped, syncErrors)
		if syncErrors > 0 {
			hadErrors = true
		}
	}

	if hadErrors {
		return ErrPartialSync
	}
	return nil
}

// prepTenant fetches spaces and apps for a tenant, using cache when available.
func prepTenant(ctx context.Context, tenant config.Tenant, filters qsync.Filters) ([]qsync.App, map[string]qsync.SpaceInfo, error) {
	cwd, _ := os.Getwd()
	cacheKey := qsync.BuildCacheKey(tenant.Context, filters.Space, filters.Stream, filters.App, cwd)
	tmpDir := os.TempDir()

	// Try cache (unless --force)
	if !syncForce {
		if cached, err := qsync.CacheRead(tmpDir, cacheKey, cacheTTL); err == nil && cached != nil {
			var prep qsync.PrepOutput
			if json.Unmarshal(cached, &prep) == nil {
				spaces := make(map[string]qsync.SpaceInfo)
				return prep.Apps, spaces, nil
			}
		}
	}

	spacesOut, err := qsync.RunQlikCmd(ctx, "qlik", qsync.BuildSpaceListArgs()...)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching spaces: %w", err)
	}
	spaces, err := qsync.ParseCloudSpaces(spacesOut)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing spaces: %w", err)
	}

	spaceID := ""
	if filters.Space != "" {
		for _, s := range spaces {
			if s.Name == filters.Space {
				spaceID = s.ID
				break
			}
		}
	}

	appsOut, err := qsync.RunQlikCmd(ctx, "qlik", qsync.BuildAppListArgs(spaceID)...)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching apps: %w", err)
	}

	apps, err := qsync.ParseCloudApps(appsOut, spaces, tenant.Context, "")
	if err != nil {
		return nil, nil, fmt.Errorf("parsing apps: %w", err)
	}

	// Write cache
	prep := qsync.PrepOutput{Tenant: tenant.Context, Context: tenant.Context, Server: tenant.Server, Apps: apps}
	if data, err := json.Marshal(prep); err == nil {
		_ = qsync.CacheWrite(tmpDir, cacheKey, data)
	}

	return apps, spaces, nil
}
