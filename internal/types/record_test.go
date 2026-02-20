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
	tests := []struct {
		level Level
		want  string
	}{
		{LevelError, "ERROR"},
		{LevelWarn, "WARN"},
		{LevelInfo, "INFO"},
		{LevelDebug, "DEBUG"},
		{LevelUnknown, "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}
