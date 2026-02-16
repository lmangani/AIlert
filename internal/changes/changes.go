package changes

import (
	"fmt"

	"github.com/ailert/ailert/internal/snapshot"
	"github.com/ailert/ailert/internal/types"
)

// Changes is the result of comparing current state to a previous snapshot.
type Changes struct {
	NewPatterns  []PatternDelta  // in current, not in previous
	GonePatterns []PatternDelta  // in previous, not in current
	CountDeltas  []CountDelta    // in both, count changed
}

// PatternDelta describes one pattern (new or gone).
type PatternDelta struct {
	Level  types.Level
	Hash   string
	Sample string
	Count  int64
}

// CountDelta describes a count change for an existing pattern.
type CountDelta struct {
	Level    types.Level
	Hash     string
	Sample   string
	OldCount int64
	NewCount int64
}

// Detect compares current patterns to a previous snapshot.
// If previous is nil, all current patterns are considered "new".
func Detect(current []snapshot.PatternEnt, previous *snapshot.Snapshot) *Changes {
	out := &Changes{}
	prevByKey := make(map[string]snapshot.PatternEnt)
	if previous != nil {
		for _, p := range previous.Patterns {
			prevByKey[key(p.Level, p.Hash)] = p
		}
	}
	curByKey := make(map[string]snapshot.PatternEnt)
	for _, p := range current {
		curByKey[key(p.Level, p.Hash)] = p
	}
	for k, p := range curByKey {
		prev, ok := prevByKey[k]
		if !ok {
			out.NewPatterns = append(out.NewPatterns, PatternDelta{Level: p.Level, Hash: p.Hash, Sample: p.Sample, Count: p.Count})
			continue
		}
		if prev.Count != p.Count {
			out.CountDeltas = append(out.CountDeltas, CountDelta{
				Level: p.Level, Hash: p.Hash, Sample: p.Sample,
				OldCount: prev.Count, NewCount: p.Count,
			})
		}
	}
	if previous != nil {
		for k, p := range prevByKey {
			if _, ok := curByKey[k]; !ok {
				out.GonePatterns = append(out.GonePatterns, PatternDelta{Level: p.Level, Hash: p.Hash, Sample: p.Sample, Count: p.Count})
			}
		}
	}
	return out
}

func key(level types.Level, hash string) string {
	return level.String() + ":" + hash
}

// SuggestedRule is a heuristic suggestion (no LLM).
type SuggestedRule struct {
	Action  string       // "suppress" or "alert"
	Hash    string
	Level   types.Level
	Sample  string
	Reason  string
}

// SuggestRules returns rule suggestions from a change set using simple heuristics:
// new ERROR/WARN -> suggest alert; new INFO/DEBUG with count above threshold -> suggest suppress.
func SuggestRules(ch *Changes, suppressCountThreshold int64) []SuggestedRule {
	var out []SuggestedRule
	for _, p := range ch.NewPatterns {
		switch p.Level {
		case types.LevelError, types.LevelWarn:
			out = append(out, SuggestedRule{
				Action: "alert",
				Hash:   p.Hash,
				Level:  p.Level,
				Sample: p.Sample,
				Reason: "new " + p.Level.String() + " pattern",
			})
		case types.LevelInfo, types.LevelDebug:
			if p.Count >= suppressCountThreshold {
				out = append(out, SuggestedRule{
					Action: "suppress",
					Hash:   p.Hash,
					Level:  p.Level,
					Sample: p.Sample,
					Reason: fmt.Sprintf("new %s pattern, count %d", p.Level.String(), p.Count),
				})
			}
		}
	}
	// Count deltas: large increase could suggest alert
	for _, d := range ch.CountDeltas {
		if d.NewCount > d.OldCount*2 && d.NewCount >= 10 {
			out = append(out, SuggestedRule{
				Action: "alert",
				Hash:   d.Hash,
				Level:  d.Level,
				Sample: d.Sample,
				Reason: "count spike",
			})
		}
	}
	return out
}

