package parser_test

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"testing"

	"github.com/mattiasthalen/qlik-sync/internal/parser"
)

func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractScript(t *testing.T) {
	scriptJSON := map[string]string{"qScript": "LOAD * FROM data.qvd;"}
	data, _ := json.Marshal(scriptJSON)
	compressed := zlibCompress(data)

	var input bytes.Buffer
	input.Write([]byte{0x00, 0x00}) // padding
	input.Write(compressed)
	input.Write([]byte{0x00, 0x00}) // padding

	result, err := parser.ExtractQVF(bytes.NewReader(input.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Script != "LOAD * FROM data.qvd;" {
		t.Errorf("script = %q, want %q", result.Script, "LOAD * FROM data.qvd;")
	}
}

func TestExtractMeasures(t *testing.T) {
	measures := []map[string]interface{}{
		{"qMeasure": map[string]interface{}{"qLabel": "Total Sales", "qDef": "Sum(Sales)"}},
	}
	block := map[string]interface{}{"qMeasureList": measures}
	data, _ := json.Marshal(block)
	compressed := zlibCompress(data)

	var input bytes.Buffer
	input.Write(compressed)

	result, err := parser.ExtractQVF(bytes.NewReader(input.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Measures) != 1 {
		t.Fatalf("measures count = %d, want 1", len(result.Measures))
	}
}
