package duckdb

import (
	"database/sql"
	"time"

	"github.com/ailert/ailert/internal/snapshot"
	"github.com/ailert/ailert/internal/types"
)

// SaveSnapshot writes the current pattern list as a new snapshot. Returns the new snapshot ID.
func (db *DB) SaveSnapshot(patterns []snapshot.PatternEnt) (int64, error) {
	var nextID int64
	err := db.sql.QueryRow(`SELECT COALESCE(MAX(id), 0) + 1 FROM snapshots`).Scan(&nextID)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	_, err = db.sql.Exec(`INSERT INTO snapshots (id, created_at) VALUES (?, ?)`, nextID, now)
	if err != nil {
		return 0, err
	}
	for _, p := range patterns {
		_, err = db.sql.Exec(
			`INSERT INTO snapshot_patterns (snapshot_id, level, hash, sample, count) VALUES (?, ?, ?, ?, ?)`,
			nextID, int(p.Level), p.Hash, p.Sample, p.Count,
		)
		if err != nil {
			return 0, err
		}
	}
	return nextID, nil
}

// LoadLatestSnapshot returns the most recent snapshot, or nil if none exist.
func (db *DB) LoadLatestSnapshot() (*snapshot.Snapshot, error) {
	var id int64
	var createdAt time.Time
	err := db.sql.QueryRow(`SELECT id, created_at FROM snapshots ORDER BY id DESC LIMIT 1`).Scan(&id, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := db.sql.Query(
		`SELECT level, hash, sample, count FROM snapshot_patterns WHERE snapshot_id = ? ORDER BY level, hash`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var patterns []snapshot.PatternEnt
	for rows.Next() {
		var level int
		var hash, sample string
		var count int64
		if err := rows.Scan(&level, &hash, &sample, &count); err != nil {
			return nil, err
		}
		patterns = append(patterns, snapshot.PatternEnt{
			Level:  types.Level(level),
			Hash:   hash,
			Sample: sample,
			Count:  count,
		})
	}
	return &snapshot.Snapshot{Timestamp: createdAt, Patterns: patterns}, nil
}
