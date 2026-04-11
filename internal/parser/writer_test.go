package parser_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func TestWriteArtifacts_QVF(t *testing.T) {
	dir := t.TempDir()
	result := &parser.QVFResult{
		Script:   "LOAD * FROM data.qvd;",
		Measures: []json.RawMessage{[]byte(`{"qMeasure":{"qLabel":"Sales"}}`)},
	}
	if err := parser.WriteQVFArtifacts(dir, result); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	script, err := os.ReadFile(filepath.Join(dir, "script.qvs"))
	if err != nil {
		t.Fatalf("script.qvs not found: %v", err)
	}
	if string(script) != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q", string(script))
	}

	measures, err := os.ReadFile(filepath.Join(dir, "measures.json"))
	if err != nil {
		t.Fatalf("measures.json not found: %v", err)
	}
	if len(measures) == 0 {
		t.Error("measures.json is empty")
	}
}

func TestWriteArtifacts_QVW(t *testing.T) {
	dir := t.TempDir()
	result := &parser.QVWResult{Script: "LOAD * FROM data.qvd;"}
	if err := parser.WriteQVWArtifacts(dir, result); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	script, err := os.ReadFile(filepath.Join(dir, "script.qvs"))
	if err != nil {
		t.Fatalf("script.qvs not found: %v", err)
	}
	if string(script) != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q", string(script))
	}
}
