package sync_test

import (
	"testing"

	qsync "github.com/mattiasthalen/qlik-sync/internal/sync"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMaj int
		wantMin int
		wantPat int
		wantErr bool
	}{
		{"valid 3.0.0", "version: 3.0.0\tcommit: abc\tdate: 2026-01-01", 3, 0, 0, false},
		{"valid 3.0.5", "version: 3.0.5\tcommit: abc\tdate: 2026-01-01", 3, 0, 5, false},
		{"valid 2.9.1", "version: 2.9.1\tcommit: abc\tdate: 2026-01-01", 2, 9, 1, false},
		{"no tabs", "version: 3.0.0", 3, 0, 0, false},
		{"malformed no prefix", "3.0.0\tcommit: abc", 0, 0, 0, true},
		{"malformed empty", "", 0, 0, 0, true},
		{"malformed two parts", "version: 3.0\tcommit: abc", 0, 0, 0, true},
		{"malformed non-numeric", "version: a.b.c\tcommit: abc", 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maj, min, pat, err := qsync.ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if maj != tt.wantMaj || min != tt.wantMin || pat != tt.wantPat {
				t.Errorf("got %d.%d.%d, want %d.%d.%d", maj, min, pat, tt.wantMaj, tt.wantMin, tt.wantPat)
			}
		})
	}
}

func TestCheckVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"compatible 3.0.0", "version: 3.0.0\tcommit: abc\tdate: 2026-01-01", false, ""},
		{"compatible 3.0.5", "version: 3.0.5\tcommit: abc\tdate: 2026-01-01", false, ""},
		{"major too low", "version: 2.9.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 2.9.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"major too high", "version: 4.0.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 4.0.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"minor too high", "version: 3.1.0\tcommit: abc\tdate: 2026-01-01", true, "qlik-cli version 3.1.0 is not compatible; requires >= 3.0.0, < 3.1.0"},
		{"malformed", "garbage", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qsync.CheckVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
				t.Errorf("err = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}
