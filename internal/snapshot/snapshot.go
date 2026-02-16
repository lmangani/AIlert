package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ailert/ailert/internal/types"
)

// Snapshot is a point-in-time view of seen patterns (for change detection).
type Snapshot struct {
	Timestamp time.Time    `json:"timestamp"`
	Patterns  []PatternEnt `json:"patterns"`
}

// PatternEnt is one pattern in a snapshot.
type PatternEnt struct {
	Level  types.Level `json:"level"`
	Hash   string      `json:"hash"`
	Sample string      `json:"sample"`
	Count  int64       `json:"count"`
}

// Save writes a snapshot to path (JSON). Creates parent dirs.
func Save(path string, patterns []PatternEnt) error {
	s := Snapshot{Timestamp: time.Now(), Patterns: patterns}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads a snapshot from path. Returns nil if file does not exist.
func Load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
