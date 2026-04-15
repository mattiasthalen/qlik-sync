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
