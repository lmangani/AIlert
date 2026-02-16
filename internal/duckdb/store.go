package duckdb

import (
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/types"
)

// Store is a DuckDB-backed PatternStore. Safe for concurrent use.
type Store struct {
	db *DB
	mu sync.RWMutex
}

// NewStore returns a PatternStore that uses the given DB for patterns and suppressions.
func NewStore(db *DB) *Store {
	return &Store{db: db}
}

// Seen records the pattern and returns true if it was new.
func (s *Store) Seen(level types.Level, hash string, sample string) (isNew bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int64
	err := s.db.sql.QueryRow(
		`SELECT count FROM patterns WHERE level = ? AND hash = ?`,
		int(level), hash,
	).Scan(&count)
	if err == sql.ErrNoRows {
		_, err = s.db.sql.Exec(
			`INSERT INTO patterns (level, hash, sample, count) VALUES (?, ?, ?, 1)`,
			int(level), hash, sample,
		)
		if err != nil {
			return true // treat as new on insert error
		}
		return true
	}
	if err != nil {
		return true
	}
	count++
	_, _ = s.db.sql.Exec(
		`UPDATE patterns SET count = ?, sample = COALESCE(NULLIF(TRIM(sample), ''), ?) WHERE level = ? AND hash = ?`,
		count, sample, int(level), hash,
	)
	return false
}

// GetCount returns the count for (level, hash). Returns 0 if not seen.
func (s *Store) GetCount(level types.Level, hash string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	err := s.db.sql.QueryRow(
		`SELECT count FROM patterns WHERE level = ? AND hash = ?`,
		int(level), hash,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// Suppress marks the pattern hash as suppressed.
func (s *Store) Suppress(hash string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.sql.Exec(
		`INSERT INTO suppressions (hash, reason) VALUES (?, ?) ON CONFLICT (hash) DO UPDATE SET reason = excluded.reason`,
		hash, reason,
	)
}

// IsSuppressed returns whether the hash is suppressed.
func (s *Store) IsSuppressed(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var x int
	err := s.db.sql.QueryRow(`SELECT 1 FROM suppressions WHERE hash = ?`, hash).Scan(&x)
	return err == nil
}

// ListSeen returns all stored patterns.
func (s *Store) ListSeen() []store.PatternInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.sql.Query(`SELECT level, hash, sample, count FROM patterns ORDER BY level, hash`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []store.PatternInfo
	for rows.Next() {
		var level int
		var hash, sample string
		var count int64
		if err := rows.Scan(&level, &hash, &sample, &count); err != nil {
			continue
		}
		out = append(out, store.PatternInfo{
			Level:  types.Level(level),
			Hash:   hash,
			Sample: sample,
			Count:  count,
		})
	}
	return out
}

// Load is a no-op for DuckDB (state is already in DB).
func (s *Store) Load() error {
	return nil
}

// Save is a no-op for DuckDB (writes are immediate).
func (s *Store) Save() error {
	return nil
}

// AppendRecord inserts one record into the records table (for run command).
func (db *DB) AppendRecord(rec *types.Record) error {
	labelsJSON := ""
	if len(rec.Labels) > 0 {
		b, _ := json.Marshal(rec.Labels)
		labelsJSON = string(b)
	}
	_, err := db.sql.Exec(
		`INSERT INTO records (timestamp, level, message, labels, source_id) VALUES (?, ?, ?, ?, ?)`,
		rec.Timestamp, rec.Level.String(), rec.Message, labelsJSON, rec.SourceID,
	)
	return err
}
