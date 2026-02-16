package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ailert/ailert/internal/duckdb"
	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/types"
)

func TestPipeline_DuckDBSource_QueryRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "src.duckdb")
	db, err := duckdb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert records as if from a previous run
	now := time.Now()
	for _, msg := range []string{"ERROR db connection failed", "WARN retry attempt 1", "INFO started"} {
		rec := types.Record{
			Timestamp: now,
			Level:     types.LevelUnknown,
			Message:   msg,
			SourceID:  "ingest",
		}
		if err := db.AppendRecord(&rec); err != nil {
			t.Fatal(err)
		}
	}

	st := duckdb.NewStore(db)
	eng := engine.New(st)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	src := &source.DuckDBSource{DB: db.SQL(), Query: "", SourceID: "duckdb-test"}
	recCh, errCh := src.Stream(ctx)
	go func() {
		for range errCh {
		}
	}()
	var count int
	for rec := range recCh {
		_ = eng.Process(&rec)
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}
	list := st.ListSeen()
	if len(list) != 3 {
		t.Errorf("expected 3 patterns in store, got %d", len(list))
	}
}
