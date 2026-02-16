package pattern

import (
	"testing"

	"github.com/ailert/ailert/internal/types"
)

func TestDetectLevel(t *testing.T) {
	tests := []struct {
		line   string
		expect types.Level
	}{
		{"ERROR something failed", types.LevelError},
		{"WARN deprecated", types.LevelWarn},
		{"INFO started", types.LevelInfo},
		{"DEBUG trace", types.LevelDebug},
		{"something error in message", types.LevelError},
		{"warning: low disk", types.LevelWarn},
		{"no level here", types.LevelUnknown},
	}
	for _, tt := range tests {
		got := DetectLevel(tt.line)
		if got != tt.expect {
			t.Errorf("DetectLevel(%q) = %v, want %v", tt.line, got, tt.expect)
		}
	}
}
