package sync

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseVersion extracts major, minor, patch from qlik version output.
// Expected format: "version: X.Y.Z\tcommit: ...\tdate: ..."
func ParseVersion(raw string) (major, minor, patch int, err error) {
	field := raw
	if i := strings.IndexByte(raw, '\t'); i >= 0 {
		field = raw[:i]
	}

	ver, found := strings.CutPrefix(field, "version: ")
	if !found {
		return 0, 0, 0, fmt.Errorf("unexpected version output: %q", raw)
	}

	parts := strings.Split(ver, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected major.minor.patch, got %q", ver)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return major, minor, patch, nil
}
