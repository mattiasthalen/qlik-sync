// cmd/version.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print qs version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("qs %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
