package sync

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func VerifyChecksum(data, checksumsBody []byte, assetName string) error {
	expected := ""
	scanner := bufio.NewScanner(bytes.NewReader(checksumsBody))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "<hash>  <filename>"
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 && parts[1] == assetName {
			expected = parts[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
}
