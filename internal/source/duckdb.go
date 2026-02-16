package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ailert/ailert/internal/types"
)

// DefaultDuckDBQuery is the query used when no custom query is set for the DuckDB source.
// Columns must be: timestamp, level, message, labels (optional, JSON string), source_id (optional).
const DefaultDuckDBQuery = `SELECT timestamp, level, message, COALESCE(labels, '{}') AS labels, COALESCE(source_id, '') AS source_id FROM records ORDER BY timestamp`

// DuckDBSource reads records from a DuckDB database by running a query.
// The query must return columns: timestamp (TIMESTAMP), level (VARCHAR), message (VARCHAR), labels (VARCHAR JSON, optional), source_id (VARCHAR, optional).
type DuckDBSource struct {
	DB      *sql.DB
	Query   string
	SourceID string
}

// ID implements Source.
func (d *DuckDBSource) ID() string {
	if d.SourceID != "" {
		return d.SourceID
	}
	return "duckdb"
}

// Stream implements Source. Runs the query once and streams rows as Records.
func (d *DuckDBSource) Stream(ctx context.Context) (<-chan types.Record, <-chan error) {
	recCh := make(chan types.Record, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(recCh)
		defer close(errCh)
		query := d.Query
		if query == "" {
			query = DefaultDuckDBQuery
		}
		rows, err := d.DB.QueryContext(ctx, query)
		if err != nil {
			errCh <- err
			return
		}
		defer rows.Close()
		for rows.Next() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			var ts time.Time
			var levelStr, message, labelsJSON, sourceID string
			if err := rows.Scan(&ts, &levelStr, &message, &labelsJSON, &sourceID); err != nil {
				errCh <- err
				return
			}
			level := types.ParseLevel(levelStr)
			labels := make(map[string]string)
			if labelsJSON != "" && labelsJSON != "{}" {
				_ = json.Unmarshal([]byte(labelsJSON), &labels)
			}
			if sourceID == "" {
				sourceID = d.SourceID
			}
			if sourceID == "" {
				sourceID = "duckdb"
			}
			recCh <- types.Record{
				Timestamp: ts,
				Level:    level,
				Message:   message,
				Labels:    labels,
				SourceID:  sourceID,
			}
		}
		if err := rows.Err(); err != nil {
			errCh <- err
		}
	}()
	return recCh, errCh
}
