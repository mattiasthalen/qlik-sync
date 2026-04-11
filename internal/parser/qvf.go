package parser

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
)

type QVFResult struct {
	Script     string            `json:"script,omitempty"`
	Measures   []json.RawMessage `json:"measures,omitempty"`
	Dimensions []json.RawMessage `json:"dimensions,omitempty"`
	Variables  []json.RawMessage `json:"variables,omitempty"`
}

var zlibMarkers = []byte{0x01, 0x5E, 0x9C, 0xDA}

func ExtractQVF(r io.Reader) (*QVFResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading QVF data: %w", err)
	}

	result := &QVFResult{}
	blocks := findZlibBlocks(data)
	for _, block := range blocks {
		decompressed, err := decompressZlib(block)
		if err != nil {
			continue
		}
		parseQVFBlock(decompressed, result)
	}
	return result, nil
}

func findZlibBlocks(data []byte) [][]byte {
	var blocks [][]byte
	for i := 0; i < len(data)-1; i++ {
		if data[i] != 0x78 {
			continue
		}
		for _, marker := range zlibMarkers {
			if data[i+1] == marker {
				blocks = append(blocks, data[i:])
				break
			}
		}
	}
	return blocks
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func parseQVFBlock(data []byte, result *QVFResult) {
	var scriptBlock struct {
		QScript string `json:"qScript"`
	}
	if json.Unmarshal(data, &scriptBlock) == nil && scriptBlock.QScript != "" {
		result.Script = scriptBlock.QScript
	}

	var measuresBlock struct {
		QMeasureList []json.RawMessage `json:"qMeasureList"`
	}
	if json.Unmarshal(data, &measuresBlock) == nil && len(measuresBlock.QMeasureList) > 0 {
		result.Measures = measuresBlock.QMeasureList
	}

	var dimensionsBlock struct {
		QDimensionList []json.RawMessage `json:"qDimensionList"`
	}
	if json.Unmarshal(data, &dimensionsBlock) == nil && len(dimensionsBlock.QDimensionList) > 0 {
		result.Dimensions = dimensionsBlock.QDimensionList
	}

	var variablesBlock struct {
		QVariableList []json.RawMessage `json:"qVariableList"`
	}
	if json.Unmarshal(data, &variablesBlock) == nil && len(variablesBlock.QVariableList) > 0 {
		result.Variables = variablesBlock.QVariableList
	}
}
