package pattern

import (
	"regexp"
	"strings"

	"github.com/ailert/ailert/internal/types"
)

// DetectLevel tries to infer level from the log line (keywords, common prefixes).
func DetectLevel(line string) types.Level {
	lower := strings.ToLower(line)
	// Order matters: check more specific first
	if strings.Contains(lower, "error") || strings.Contains(lower, "exception") || strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") {
		return types.LevelError
	}
	if strings.Contains(lower, "warn") || strings.Contains(lower, "warning") {
		return types.LevelWarn
	}
	if strings.Contains(lower, "debug") {
		return types.LevelDebug
	}
	if strings.Contains(lower, "info") {
		return types.LevelInfo
	}
	// Common prefixes
	if levelPrefix.MatchString(line) {
		switch line[0] {
		case 'E', 'e':
			return types.LevelError
		case 'W', 'w':
			return types.LevelWarn
		case 'I', 'i':
			return types.LevelInfo
		case 'D', 'd':
			return types.LevelDebug
		}
	}
	return types.LevelUnknown
}

// levelPrefix matches lines like "ERROR ", "WARN [", "INFO  " etc.
var levelPrefix = regexp.MustCompile(`(?i)^(ERROR|WARN|WARNING|INFO|DEBUG)\s+`)
