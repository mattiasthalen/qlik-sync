package sync

import (
	"os"
	"path/filepath"
)

// MarkSkipped returns a copy of apps with Skip=true for any app whose
// TargetPath already contains a config.yml or script.qvs file under configDir.
func MarkSkipped(apps []App, configDir string) []App {
	out := make([]App, len(apps))
	copy(out, apps)
	for i := range out {
		targetDir := filepath.Join(configDir, out[i].TargetPath)
		if fileExists(filepath.Join(targetDir, "config.yml")) || fileExists(filepath.Join(targetDir, "script.qvs")) {
			out[i].Skip = true
			out[i].SkipReason = "already synced"
		}
	}
	return out
}

// MarkSkippedForce returns a copy of apps with all Skip flags cleared,
// allowing force-mode to re-sync apps that already exist on disk.
func MarkSkippedForce(apps []App) []App {
	out := make([]App, len(apps))
	copy(out, apps)
	for i := range out {
		out[i].Skip = false
		out[i].SkipReason = ""
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
