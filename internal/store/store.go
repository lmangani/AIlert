package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/ailert/ailert/internal/types"
)

// PatternStore is the interface used by the engine for pattern/suppression state.
// Implementations: in-memory Store (with optional JSON persist) or DuckDB-backed store.
type PatternStore interface {
	Seen(level types.Level, hash string, sample string) (isNew bool)
	GetCount(level types.Level, hash string) int64
	Suppress(hash string, reason string)
	IsSuppressed(hash string) bool
	ListSeen() []PatternInfo
	Load() error
	Save() error
}

// Store holds seen pattern hashes and optional suppression list.
// Safe for concurrent use. Implements PatternStore.
type Store struct {
	mu          sync.RWMutex
	seen        map[patternKey]patternStat
	suppressed  map[string]string // hash -> reason
	persistPath string
}

type patternKey struct {
	Level types.Level
	Hash  string
}

type patternStat struct {
	Sample   string
	Count    int64
}

// New returns an in-memory store. If persistPath is non-empty, Load/Save will use it.
func New(persistPath string) *Store {
	s := &Store{
		seen:       make(map[patternKey]patternStat),
		suppressed: make(map[string]string),
		persistPath: persistPath,
	}
	return s
}

// Seen returns whether this (level, hash) was seen before and updates the count.
// Returns true if this is the first time (new pattern).
func (s *Store) Seen(level types.Level, hash string, sample string) (isNew bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := patternKey{Level: level, Hash: hash}
	stat, ok := s.seen[key]
	if !ok {
		s.seen[key] = patternStat{Sample: sample, Count: 1}
		return true
	}
	stat.Count++
	if stat.Sample == "" && sample != "" {
		stat.Sample = sample
	}
	s.seen[key] = stat
	return false
}

// GetCount returns the count for a (level, hash). Returns 0 if not seen.
func (s *Store) GetCount(level types.Level, hash string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seen[patternKey{Level: level, Hash: hash}].Count
}

// Suppress marks a pattern hash as suppressed with an optional reason.
func (s *Store) Suppress(hash string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppressed[hash] = reason
}

// IsSuppressed returns whether the pattern hash is suppressed.
func (s *Store) IsSuppressed(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.suppressed[hash]
	return ok
}

// ListSeen returns a snapshot of seen patterns (for CLI/summary).
func (s *Store) ListSeen() []PatternInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PatternInfo, 0, len(s.seen))
	for k, v := range s.seen {
		out = append(out, PatternInfo{
			Level:  k.Level,
			Hash:   k.Hash,
			Sample: v.Sample,
			Count:  v.Count,
		})
	}
	return out
}

// PatternInfo is a read-only view of a stored pattern.
type PatternInfo struct {
	Level  types.Level
	Hash   string
	Sample string
	Count  int64
}

// persistState is the on-disk shape (optional JSON).
type persistState struct {
	Seen       []patternStatPersist `json:"seen"`
	Suppressed map[string]string    `json:"suppressed"`
}

type patternStatPersist struct {
	Level  types.Level `json:"level"`
	Hash   string      `json:"hash"`
	Sample string      `json:"sample"`
	Count  int64       `json:"count"`
}

// Load restores state from persistPath if set and file exists.
func (s *Store) Load() error {
	if s.persistPath == "" {
		return nil
	}
	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state persistState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range state.Seen {
		s.seen[patternKey{Level: v.Level, Hash: v.Hash}] = patternStat{Sample: v.Sample, Count: v.Count}
	}
	for hash, reason := range state.Suppressed {
		s.suppressed[hash] = reason
	}
	return nil
}

// Save writes state to persistPath if set.
func (s *Store) Save() error {
	if s.persistPath == "" {
		return nil
	}
	s.mu.RLock()
	state := persistState{
		Seen:       make([]patternStatPersist, 0, len(s.seen)),
		Suppressed: make(map[string]string),
	}
	for k, v := range s.seen {
		state.Seen = append(state.Seen, patternStatPersist{Level: k.Level, Hash: k.Hash, Sample: v.Sample, Count: v.Count})
	}
	for k, v := range s.suppressed {
		state.Suppressed[k] = v
	}
	s.mu.RUnlock()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.persistPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(s.persistPath, data, 0644)
}
