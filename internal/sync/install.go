package sync

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

func ResolveQlikPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine qs location: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("cannot resolve qs symlink: %w", err)
	}
	dir := filepath.Dir(exe)
	name := "qlik"
	if runtime.GOOS == "windows" {
		name = "qlik.exe"
	}
	return filepath.Join(dir, name), nil
}

func DownloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func EnsureQlikCLI(ctx context.Context, targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		out, err := RunQlikCmd(ctx, targetPath, "version")
		if err == nil {
			if verErr := CheckVersion(strings.TrimSpace(string(out))); verErr == nil {
				fmt.Printf("qlik-cli found at %s\n", targetPath)
				return nil
			}
		}
		fmt.Printf("Replacing incompatible qlik-cli at %s\n", targetPath)
	}

	fmt.Printf("Installing qlik-cli %s to %s\n", QlikCLIVersion, targetPath)

	asset := DetectAssetName()
	archiveURL := BuildDownloadURL(QlikCLIVersion, asset)
	checksumsURL := BuildChecksumsURL(QlikCLIVersion)

	archive, err := DownloadFile(ctx, archiveURL)
	if err != nil {
		return fmt.Errorf("downloading qlik-cli: %w", err)
	}

	checksums, err := DownloadFile(ctx, checksumsURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	if err := VerifyChecksum(archive, checksums, asset); err != nil {
		return fmt.Errorf("verifying qlik-cli: %w", err)
	}

	binary, err := ExtractBinary(archive, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("extracting qlik-cli: %w", err)
	}

	if err := os.WriteFile(targetPath, binary, 0755); err != nil {
		dir := filepath.Dir(targetPath)
		return fmt.Errorf("cannot install qlik-cli to %s — check permissions or run with sudo", dir)
	}

	out, err := RunQlikCmd(ctx, targetPath, "version")
	if err != nil {
		return fmt.Errorf("verifying installed qlik-cli: %w", err)
	}
	if err := CheckVersion(strings.TrimSpace(string(out))); err != nil {
		return fmt.Errorf("installed qlik-cli version mismatch: %w", err)
	}

	fmt.Printf("qlik-cli %s installed successfully\n", QlikCLIVersion)
	return nil
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
