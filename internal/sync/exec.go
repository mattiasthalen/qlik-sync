package sync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func CheckPrerequisites(qlikPath string, skipVersionCheck bool) error {
	if _, err := os.Stat(qlikPath); err != nil {
		return fmt.Errorf("qlik-cli not found at %s\n\n  Run \"qs setup\" to install it automatically.", qlikPath)
	}

	if skipVersionCheck {
		return nil
	}

	out, err := RunQlikCmd(context.Background(), qlikPath, "version")
	if err != nil {
		return fmt.Errorf("cannot determine qlik-cli version: %w", err)
	}

	if err := CheckVersion(strings.TrimSpace(string(out))); err != nil {
		return fmt.Errorf("%w\n\n  Run \"qs setup\" to install the correct version.", err)
	}

	return nil
}

func RunQlikCmd(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("qlik %v failed: %s", args, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("qlik %v failed: %w", args, err)
	}
	return out, nil
}

func BuildSpaceListArgs() []string {
	return []string{"space", "ls", "--json"}
}

func BuildAppListArgs(spaceID string) []string {
	args := []string{"app", "ls", "--json", "--limit", "1000"}
	if spaceID != "" {
		args = append(args, "--spaceId", spaceID)
	}
	return args
}

func BuildUnbuildArgs(resourceID, targetDir string) []string {
	return []string{"app", "unbuild", "--app", resourceID, "--dir", targetDir}
}

func CloudSyncApp(ctx context.Context, app App, configDir string, qlikBinary string) error {
	targetDir := fmt.Sprintf("%s/%s", configDir, app.TargetPath)
	args := BuildUnbuildArgs(app.ResourceID, targetDir)
	_, err := RunQlikCmd(ctx, qlikBinary, args...)
	return err
}
