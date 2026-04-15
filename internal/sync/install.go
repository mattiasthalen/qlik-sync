package sync

import (
	"fmt"
	"runtime"
	"strings"
)

const QlikCLIVersion = "3.0.0"

func DetectAssetName() string {
	osName := strings.ToTitle(runtime.GOOS[:1]) + runtime.GOOS[1:]
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("qlik-%s-x86_64.%s", osName, ext)
}

func BuildDownloadURL(version, asset string) string {
	return fmt.Sprintf("https://github.com/qlik-oss/qlik-cli/releases/download/v%s/%s", version, asset)
}

func BuildChecksumsURL(version string) string {
	return fmt.Sprintf("https://github.com/qlik-oss/qlik-cli/releases/download/v%s/qlik_%s_checksums.txt", version, version)
}
