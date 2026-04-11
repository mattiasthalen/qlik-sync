package cmd

import (
	"testing"
)

func TestSyncCmdFlags(t *testing.T) {
	cmd := syncCmd
	flags := []struct {
		name     string
		defValue string
	}{
		{"space", ""},
		{"stream", ""},
		{"app", ""},
		{"id", ""},
		{"tenant", ""},
		{"threads", "0"},
		{"retries", "0"},
	}
	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil { t.Errorf("flag %q not found", f.name); continue }
		if flag.DefValue != f.defValue { t.Errorf("flag %q default = %q, want %q", f.name, flag.DefValue, f.defValue) }
	}
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil { t.Error("flag 'force' not found") }
}

func TestSyncCmdRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "sync" { found = true; break }
	}
	if !found { t.Error("sync command not registered on root") }
}
