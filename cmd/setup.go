package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattiasthalen/qlik-sync/internal/config"
	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Qlik tenant connection",
	Long:  "Interactive setup for connecting to a Qlik Cloud or on-prem tenant.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	qlikPath, err := qsync.ResolveQlikPath()
	if err != nil {
		return err
	}

	if err := qsync.EnsureQlikCLI(cmd.Context(), qlikPath); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	// List existing contexts
	out, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "context", "ls")
	if err == nil {
		fmt.Printf("Existing qlik contexts:\n%s\n", string(out))
	}

	fmt.Print("Enter qlik context name (existing or new): ")
	contextName, _ := reader.ReadString('\n')
	contextName = strings.TrimSpace(contextName)

	fmt.Print("Enter server URL (e.g., https://tenant.qlikcloud.com): ")
	server, _ := reader.ReadString('\n')
	server = strings.TrimSpace(server)

	tenantType := config.DetectTenantType(server)
	fmt.Printf("Detected tenant type: %s\n", tenantType)

	// Check if context exists, create if not
	checkCmd := exec.Command(qlikPath, "context", "get", contextName)
	if err := checkCmd.Run(); err != nil {
		fmt.Print("Enter API key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		createArgs := []string{"context", "create", contextName, "--server", server, "--api-key", apiKey}
		if tenantType == "on-prem" {
			createArgs = append(createArgs, "--server-type", "Windows", "--insecure")
		}
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, createArgs...); err != nil {
			return fmt.Errorf("creating context: %w", err)
		}
		fmt.Println("Context created.")
	}

	// Set active context
	if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "context", "use", contextName); err != nil {
		return fmt.Errorf("setting active context: %w", err)
	}

	// Test connectivity
	fmt.Println("Testing connectivity...")
	if tenantType == "cloud" {
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "app", "ls", "--limit", "1"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	} else {
		if _, err := qsync.RunQlikCmd(cmd.Context(), qlikPath, "qrs", "app", "count"); err != nil {
			return fmt.Errorf("connectivity test failed: %w\n  Check your API key and server URL", err)
		}
	}
	fmt.Println("Connected successfully.")

	// Read or create config
	cfg, err := config.Read(configDir)
	if err != nil {
		cfg = &config.Config{Version: "0.2.0"}
	}

	// Add tenant (or update existing)
	found := false
	for i, t := range cfg.Tenants {
		if t.Context == contextName {
			cfg.Tenants[i].Server = server
			cfg.Tenants[i].Type = tenantType
			found = true
			break
		}
	}
	if !found {
		cfg.Tenants = append(cfg.Tenants, config.Tenant{Context: contextName, Server: server, Type: tenantType})
	}

	if err := config.Write(configDir, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nSetup complete. Run: qs sync\n")
	return nil
}
