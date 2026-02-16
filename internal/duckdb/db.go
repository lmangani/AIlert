package duckdb

import (
	"database/sql"
	"sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB is an open DuckDB database with migrations applied.
// Use Open to create; call Close when done.
type DB struct {
	sql  *sql.DB
	path string
	mu   sync.Mutex
}

// Open opens or creates a DuckDB database at path and runs migrations.
// Path can be a file path (e.g. "ailert.duckdb") or "" for in-memory.
func Open(path string) (*DB, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	}
	sqlDB, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}
	db := &DB{sql: sqlDB, path: path}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database.
func (db *DB) Close() error {
	return db.sql.Close()
}

// SQL returns the underlying *sql.DB for use by sources (e.g. DuckDB source query).
func (db *DB) SQL() *sql.DB {
	return db.sql
}

func (db *DB) migrate() error {
	// records: append-only log for primary datasource queries
	_, err := db.sql.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			timestamp TIMESTAMP NOT NULL,
			level VARCHAR NOT NULL,
			message VARCHAR NOT NULL,
			labels VARCHAR,
			source_id VARCHAR NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	// patterns: level, hash, sample, count (upsert by level+hash)
	_, err = db.sql.Exec(`
		CREATE TABLE IF NOT EXISTS patterns (
			level INTEGER NOT NULL,
			hash VARCHAR NOT NULL,
			sample VARCHAR NOT NULL,
			count BIGINT NOT NULL DEFAULT 1,
			PRIMARY KEY (level, hash)
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.sql.Exec(`
		CREATE TABLE IF NOT EXISTS suppressions (
			hash VARCHAR PRIMARY KEY,
			reason VARCHAR NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.sql.Exec(`
		CREATE TABLE IF NOT EXISTS snapshots (
			id BIGINT PRIMARY KEY,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.sql.Exec(`
		CREATE TABLE IF NOT EXISTS snapshot_patterns (
			snapshot_id BIGINT NOT NULL,
			level INTEGER NOT NULL,
			hash VARCHAR NOT NULL,
			sample VARCHAR NOT NULL,
			count BIGINT NOT NULL,
			PRIMARY KEY (snapshot_id, level, hash)
		)
	`)
	if err != nil {
		return err
	}
	return nil
}
