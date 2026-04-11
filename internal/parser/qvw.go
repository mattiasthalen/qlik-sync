package parser

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strings"
)

const qvwHeaderSize = 23
const scriptMarker = "///"

type QVWResult struct {
	Script string `json:"script,omitempty"`
}

func ExtractQVW(r io.Reader) (*QVWResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading QVW data: %w", err)
	}
	if len(data) <= qvwHeaderSize {
		return nil, fmt.Errorf("QVW file too small: %d bytes", len(data))
	}

	compressed := data[qvwHeaderSize:]
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("decompressing QVW: %w", err)
	}
	defer func() { _ = zr.Close() }()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("reading decompressed QVW: %w", err)
	}

	content := string(decompressed)
	result := &QVWResult{}

	idx := strings.Index(content, scriptMarker)
	if idx == -1 {
		return result, nil
	}

	script := content[idx+len(scriptMarker):]
	script = strings.TrimLeft(script, "\r\n")
	if nullIdx := strings.Index(script, "\x00\x00"); nullIdx != -1 {
		script = script[:nullIdx]
	}
	result.Script = script
	return result, nil
}
