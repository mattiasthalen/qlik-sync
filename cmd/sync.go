package cmd

import (
	"context"
	"fmt"
	"os"

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

func runSync(cmd *cobra.Command, args []string) error {
	if err := qsync.CheckPrerequisites(); err != nil {
		return err
	}

	cfg, err := config.Read(configDir)
	if err != nil {
		return fmt.Errorf("reading config: %w\n  Run: qs setup", err)
	}

	var flagThreads, flagRetries *int
	if cmd.Flags().Changed("threads") { flagThreads = &syncThreads }
	if cmd.Flags().Changed("retries") { flagRetries = &syncRetries }
	resolved := config.Resolve(cfg, flagThreads, flagRetries)

	tenants := config.FilterTenants(resolved.Tenants, syncTenant)
	if len(tenants) == 0 {
		return fmt.Errorf("no tenants found for context %q\n  Run: qs setup", syncTenant)
	}

	ctx := context.Background()
	filters := qsync.Filters{Space: syncSpace, Stream: syncStream, App: syncApp, ID: syncID}

	exitCode := 0
	for _, tenant := range tenants {
		if tenant.Type != "cloud" {
			fmt.Fprintf(os.Stderr, "Skipping on-prem tenant %q (not yet supported)\n", tenant.Context)
			continue
		}

		spacesOut, err := qsync.RunQlikCmd(ctx, "qlik", qsync.BuildSpaceListArgs()...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching spaces for %q: %v\n", tenant.Context, err)
			exitCode = 2; continue
		}
		spaces, err := qsync.ParseCloudSpaces(spacesOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing spaces for %q: %v\n", tenant.Context, err)
			exitCode = 2; continue
		}

		spaceID := ""
		if syncSpace != "" {
			for _, s := range spaces {
				if s.Name == syncSpace { spaceID = s.ID; break }
			}
		}

		appsOut, err := qsync.RunQlikCmd(ctx, "qlik", qsync.BuildAppListArgs(spaceID)...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching apps for %q: %v\n", tenant.Context, err)
			exitCode = 2; continue
		}

		apps, err := qsync.ParseCloudApps(appsOut, spaces, tenant.Context, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing apps for %q: %v\n", tenant.Context, err)
			exitCode = 2; continue
		}

		apps = qsync.ResolveOwnerNames(ctx, apps, "qlik")
		apps = qsync.ApplyFilters(apps, filters)
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

		synced, skipped, errors := qsync.Summarize(results)
		fmt.Printf("%d synced, %d skipped, %d errors\n", synced, skipped, errors)
		if errors > 0 { exitCode = 2 }
	}

	if exitCode != 0 { os.Exit(exitCode) }
	return nil
}
