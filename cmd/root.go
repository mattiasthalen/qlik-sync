// cmd/root.go
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	logLevel         string
	configDir        string
	skipVersionCheck bool
)

var rootCmd = &cobra.Command{
	Use:   "qs",
	Short: "Qlik Sync — sync Qlik apps to local files",
	Long:  "qs syncs Qlik Sense cloud and on-prem apps to a local qlik/ directory for version control and offline inspection.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "disabled", "log level (debug|info|warn|error|disabled)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "qlik", "config and sync directory")
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "skip qlik-cli version compatibility check")
}
