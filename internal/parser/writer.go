package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteQVFArtifacts(dir string, result *QVFResult) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	if result.Script != "" {
		if err := os.WriteFile(filepath.Join(dir, "script.qvs"), []byte(result.Script), 0644); err != nil {
			return fmt.Errorf("writing script: %w", err)
		}
	}
	if len(result.Measures) > 0 {
		if err := writeJSON(filepath.Join(dir, "measures.json"), result.Measures); err != nil {
			return fmt.Errorf("writing measures: %w", err)
		}
	}
	if len(result.Dimensions) > 0 {
		if err := writeJSON(filepath.Join(dir, "dimensions.json"), result.Dimensions); err != nil {
			return fmt.Errorf("writing dimensions: %w", err)
		}
	}
	if len(result.Variables) > 0 {
		if err := writeJSON(filepath.Join(dir, "variables.json"), result.Variables); err != nil {
			return fmt.Errorf("writing variables: %w", err)
		}
	}
	return nil
}

func WriteQVWArtifacts(dir string, result *QVWResult) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	if result.Script != "" {
		if err := os.WriteFile(filepath.Join(dir, "script.qvs"), []byte(result.Script), 0644); err != nil {
			return fmt.Errorf("writing script: %w", err)
		}
	}
	return nil
}

func writeJSON(path string, data interface{}) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
