package sync

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

func ExtractBinary(archive []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		return extractFromZip(archive)
	}
	return extractFromTarGz(archive)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompressing archive: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if hdr.Name == "qlik" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("qlik binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}
	for _, f := range r.File {
		if f.Name == "qlik.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening qlik.exe in zip: %w", err)
			}
			defer func() { _ = rc.Close() }()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("qlik.exe not found in archive")
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
