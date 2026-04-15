package sync_test

import (
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
