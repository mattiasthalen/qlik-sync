package sync_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestDetectAssetName(t *testing.T) {
	name := qsync.DetectAssetName()

	switch runtime.GOOS {
	case "linux":
		if name != "qlik-Linux-x86_64.tar.gz" {
			t.Errorf("got %q, want qlik-Linux-x86_64.tar.gz", name)
		}
	case "darwin":
		if name != "qlik-Darwin-x86_64.tar.gz" {
			t.Errorf("got %q, want qlik-Darwin-x86_64.tar.gz", name)
		}
	case "windows":
		if name != "qlik-Windows-x86_64.zip" {
			t.Errorf("got %q, want qlik-Windows-x86_64.zip", name)
		}
	default:
		t.Fatalf("untested GOOS: %s", runtime.GOOS)
	}
}

func TestBuildDownloadURL(t *testing.T) {
	got := qsync.BuildDownloadURL("3.0.0", "qlik-Linux-x86_64.tar.gz")
	want := "https://github.com/qlik-oss/qlik-cli/releases/download/v3.0.0/qlik-Linux-x86_64.tar.gz"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildChecksumsURL(t *testing.T) {
	got := qsync.BuildChecksumsURL("3.0.0")
	want := "https://github.com/qlik-oss/qlik-cli/releases/download/v3.0.0/qlik_3.0.0_checksums.txt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	content := []byte("fake-qlik-binary")
	archive := buildTarGz(t, "qlik", content)

	got, err := qsync.ExtractBinary(archive, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	content := []byte("fake-qlik-binary")
	archive := buildZip(t, "qlik.exe", content)

	got, err := qsync.ExtractBinary(archive, "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinary_NoBinary(t *testing.T) {
	archive := buildTarGz(t, "README.md", []byte("not a binary"))

	_, err := qsync.ExtractBinary(archive, "linux")
	if err == nil {
		t.Error("expected error for missing qlik binary in archive")
	}
}

func buildTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0755}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestResolveQlikPath(t *testing.T) {
	got, err := qsync.ResolveQlikPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should end with /qlik (or \qlik.exe on windows)
	base := filepath.Base(got)
	if runtime.GOOS == "windows" {
		if base != "qlik.exe" {
			t.Errorf("base = %q, want qlik.exe", base)
		}
	} else {
		if base != "qlik" {
			t.Errorf("base = %q, want qlik", base)
		}
	}
	// Should be an absolute path
	if !filepath.IsAbs(got) {
		t.Errorf("path %q is not absolute", got)
	}
}

func TestDownloadFile(t *testing.T) {
	content := []byte("file-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	got, err := qsync.DownloadFile(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestDownloadFile_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := qsync.DownloadFile(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world")
	// SHA256 of "hello world"
	checksumsBody := []byte(
		"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  hello.tar.gz\n" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.tar.gz\n",
	)

	t.Run("valid checksum", func(t *testing.T) {
		err := qsync.VerifyChecksum(data, checksumsBody, "hello.tar.gz")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid checksum", func(t *testing.T) {
		bad := []byte("wrong data")
		err := qsync.VerifyChecksum(bad, checksumsBody, "hello.tar.gz")
		if err == nil {
			t.Error("expected error for mismatched checksum")
		}
	})

	t.Run("asset not found in checksums", func(t *testing.T) {
		err := qsync.VerifyChecksum(data, checksumsBody, "missing.tar.gz")
		if err == nil {
			t.Error("expected error for missing asset")
		}
	})
}
