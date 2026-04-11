package cmd

import "testing"

func TestSetupCmdRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "setup" { found = true; break }
	}
	if !found { t.Error("setup command not registered on root") }
}
