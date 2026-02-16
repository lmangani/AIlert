package types

import "time"

// Record is the normalized log/metric record after format mapping.
// All sources produce Records so the pattern engine and downstream are source-agnostic.
type Record struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     Level             `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string  `json:"labels,omitempty"`
	SourceID  string            `json:"source_id"`
}

// Level represents log severity (and optionally metric alert severity).
type Level int

const (
	LevelUnknown Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel returns Level from a string (case-insensitive).
func ParseLevel(s string) Level {
	switch len(s) {
	case 4:
		if (s[0] == 'I' || s[0] == 'i') && (s[1] == 'N' || s[1] == 'n') && (s[2] == 'F' || s[2] == 'f') && (s[3] == 'O' || s[3] == 'o') {
			return LevelInfo
		}
		if (s[0] == 'W' || s[0] == 'w') && (s[1] == 'A' || s[1] == 'a') && (s[2] == 'R' || s[2] == 'r') && (s[3] == 'N' || s[3] == 'n') {
			return LevelWarn
		}
	case 5:
		if (s[0] == 'D' || s[0] == 'd') && (s[1] == 'E' || s[1] == 'e') && (s[2] == 'B' || s[2] == 'b') && (s[3] == 'U' || s[3] == 'u') && (s[4] == 'G' || s[4] == 'g') {
			return LevelDebug
		}
		if (s[0] == 'E' || s[0] == 'e') && (s[1] == 'R' || s[1] == 'r') && (s[2] == 'R' || s[2] == 'r') && (s[3] == 'O' || s[3] == 'o') && (s[4] == 'R' || s[4] == 'r') {
			return LevelError
		}
	}
	return LevelUnknown
}
