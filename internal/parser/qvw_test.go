package parser_test

import (
	"bytes"
	"compress/zlib"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func TestExtractQVW(t *testing.T) {
	script := "///\r\nLOAD * FROM data.qvd;\r\n"
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	_, _ = w.Write([]byte(script))
	_ = w.Close()

	var qvw bytes.Buffer
	header := make([]byte, 23)
	qvw.Write(header)
	qvw.Write(compressed.Bytes())

	result, err := parser.ExtractQVW(bytes.NewReader(qvw.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "LOAD * FROM data.qvd;\r\n" {
		t.Errorf("script = %q, want %q", result.Script, "LOAD * FROM data.qvd;\r\n")
	}
}

func TestExtractQVW_NoScript(t *testing.T) {
	content := []byte("just some random data without script marker")
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	_, _ = w.Write(content)
	_ = w.Close()

	var qvw bytes.Buffer
	header := make([]byte, 23)
	qvw.Write(header)
	qvw.Write(compressed.Bytes())

	result, err := parser.ExtractQVW(bytes.NewReader(qvw.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "" {
		t.Errorf("expected empty script, got %q", result.Script)
	}
}
