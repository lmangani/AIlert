package types

import (
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		s      string
		expect Level
	}{
		{"ERROR", LevelError},
		{"error", LevelError},
		{"WARN", LevelWarn},
		{"INFO", LevelInfo},
		{"DEBUG", LevelDebug},
		{"unknown", LevelUnknown},
		{"", LevelUnknown},
	}
	for _, tt := range tests {
		got := ParseLevel(tt.s)
		if got != tt.expect {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.s, got, tt.expect)
		}
	}
}

func TestLevelString(t *testing.T) {
	if LevelError.String() != "ERROR" {
		t.Errorf("LevelError.String() = %s", LevelError.String())
	}
}
