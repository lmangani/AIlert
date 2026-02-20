package changes

import (
	"testing"
	"time"

	"github.com/ailert/ailert/internal/snapshot"
	"github.com/ailert/ailert/internal/types"
)

func TestDetect_NoPrevious(t *testing.T) {
	cur := []snapshot.PatternEnt{
		{Level: types.LevelError, Hash: "h1", Count: 1},
	}
	ch := Detect(cur, nil)
	if len(ch.NewPatterns) != 1 {
		t.Fatalf("NewPatterns len = %d", len(ch.NewPatterns))
	}
	if len(ch.GonePatterns) != 0 {
		t.Errorf("GonePatterns should be empty")
	}
}

func TestDetect_WithPrevious(t *testing.T) {
	prev := &snapshot.Snapshot{
		Timestamp: time.Now(),
		Patterns: []snapshot.PatternEnt{
			{Level: types.LevelError, Hash: "h1", Sample: "e", Count: 1},
			{Level: types.LevelWarn, Hash: "h2", Sample: "w", Count: 2},
		},
	}
	cur := []snapshot.PatternEnt{
		{Level: types.LevelError, Hash: "h1", Sample: "e", Count: 5},
		{Level: types.LevelInfo, Hash: "h3", Sample: "i", Count: 1},
	}
	ch := Detect(cur, prev)
	if len(ch.NewPatterns) != 1 || ch.NewPatterns[0].Hash != "h3" {
		t.Errorf("NewPatterns: %+v", ch.NewPatterns)
	}
	if len(ch.GonePatterns) != 1 || ch.GonePatterns[0].Hash != "h2" {
		t.Errorf("GonePatterns: %+v", ch.GonePatterns)
	}
	if len(ch.CountDeltas) != 1 || ch.CountDeltas[0].Hash != "h1" || ch.CountDeltas[0].OldCount != 1 || ch.CountDeltas[0].NewCount != 5 {
		t.Errorf("CountDeltas: %+v", ch.CountDeltas)
	}
}

func TestSuggestRules(t *testing.T) {
	ch := &Changes{
		NewPatterns: []PatternDelta{
			{Level: types.LevelError, Hash: "e1", Sample: "err", Count: 1},
			{Level: types.LevelInfo, Hash: "i1", Sample: "info", Count: 10},
		},
	}
	rules := SuggestRules(ch, 5)
	var alert, suppress int
	for _, r := range rules {
		if r.Action == "alert" {
			alert++
		}
		if r.Action == "suppress" {
			suppress++
		}
	}
	if alert != 1 {
		t.Errorf("expected 1 alert suggestion, got %d", alert)
	}
	if suppress != 1 {
		t.Errorf("expected 1 suppress suggestion (INFO count 10 >= 5), got %d", suppress)
	}
}

func TestSuggestRules_Warn(t *testing.T) {
	ch := &Changes{
		NewPatterns: []PatternDelta{
			{Level: types.LevelWarn, Hash: "w1", Sample: "warn msg", Count: 1},
		},
	}
	rules := SuggestRules(ch, 5)
	if len(rules) != 1 || rules[0].Action != "alert" {
		t.Errorf("WARN new pattern should produce alert suggestion: %+v", rules)
	}
}

func TestSuggestRules_DebugBelowThreshold(t *testing.T) {
	ch := &Changes{
		NewPatterns: []PatternDelta{
			{Level: types.LevelDebug, Hash: "d1", Sample: "debug msg", Count: 2},
		},
	}
	rules := SuggestRules(ch, 5)
	if len(rules) != 0 {
		t.Errorf("DEBUG below threshold should produce no suggestion, got %+v", rules)
	}
}

func TestSuggestRules_UnknownLevel(t *testing.T) {
	ch := &Changes{
		NewPatterns: []PatternDelta{
			{Level: types.LevelUnknown, Hash: "u1", Sample: "metric line", Count: 1},
		},
	}
	rules := SuggestRules(ch, 5)
	if len(rules) != 0 {
		t.Errorf("UNKNOWN level new pattern should produce no suggestion, got %+v", rules)
	}
}

func TestSuggestRules_CountSpike(t *testing.T) {
	ch := &Changes{
		CountDeltas: []CountDelta{
			{Level: types.LevelError, Hash: "e1", Sample: "err", OldCount: 4, NewCount: 20},
		},
	}
	rules := SuggestRules(ch, 5)
	if len(rules) != 1 || rules[0].Action != "alert" || rules[0].Hash != "e1" {
		t.Errorf("count spike should produce alert suggestion: %+v", rules)
	}
}

func TestSuggestRules_CountNoSpike(t *testing.T) {
	ch := &Changes{
		CountDeltas: []CountDelta{
			{Level: types.LevelError, Hash: "e1", Sample: "err", OldCount: 10, NewCount: 11},
		},
	}
	rules := SuggestRules(ch, 5)
	if len(rules) != 0 {
		t.Errorf("small count delta should produce no suggestion, got %+v", rules)
	}
}
