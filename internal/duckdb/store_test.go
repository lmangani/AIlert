package duckdb

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ailert/ailert/internal/snapshot"
	"github.com/ailert/ailert/internal/types"
)

func TestStore_Seen_ListSeen_Suppress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.duckdb")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := NewStore(db)
	if err := st.Load(); err != nil {
		t.Fatal(err)
	}

	// New pattern
	if !st.Seen(types.LevelError, "h1", "sample one") {
		t.Error("expected first Seen to be new")
	}
	if st.Seen(types.LevelError, "h1", "sample one") {
		t.Error("expected second Seen to be known")
	}
	if c := st.GetCount(types.LevelError, "h1"); c != 2 {
		t.Errorf("expected count 2, got %d", c)
	}

	list := st.ListSeen()
	if len(list) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(list))
	}
	if list[0].Hash != "h1" || list[0].Count != 2 {
		t.Errorf("expected h1 count 2, got %s %d", list[0].Hash, list[0].Count)
	}

	st.Suppress("h1", "test")
	if !st.IsSuppressed("h1") {
		t.Error("expected h1 to be suppressed")
	}
}

func TestStore_Persist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.duckdb")
	db1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st1 := NewStore(db1)
	st1.Seen(types.LevelWarn, "w1", "warn sample")
	st1.Suppress("w1", "noise")
	db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	st2 := NewStore(db2)
	if !st2.IsSuppressed("w1") {
		t.Error("expected w1 still suppressed after reopen")
	}
	list := st2.ListSeen()
	if len(list) != 1 || list[0].Hash != "w1" {
		t.Errorf("expected 1 pattern w1 after reopen, got %v", list)
	}
}

func TestSnapshot_SaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.duckdb")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	patternEnts := []snapshot.PatternEnt{
		{Level: types.LevelError, Hash: "e1", Sample: "err", Count: 1},
		{Level: types.LevelInfo, Hash: "i1", Sample: "info", Count: 5},
	}
	id, err := db.SaveSnapshot(patternEnts)
	if err != nil {
		t.Fatal(err)
	}
	if id < 1 {
		t.Errorf("expected snapshot id >= 1, got %d", id)
	}
	snap, err := db.LoadLatestSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snap.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(snap.Patterns))
	}
}

func TestSnapshot_LoadLatest_Empty(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	snap, err := db.LoadLatestSnapshot()
	if err != nil {
		t.Fatalf("LoadLatestSnapshot on empty DB: %v", err)
	}
	if snap != nil {
		t.Error("expected nil snapshot on empty DB")
	}
}

func TestStore_SaveLoad_NoOp(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := NewStore(db)
	if err := st.Load(); err != nil {
		t.Errorf("Load should be no-op, got %v", err)
	}
	if err := st.Save(); err != nil {
		t.Errorf("Save should be no-op, got %v", err)
	}
}

func TestDB_SQL(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.SQL() == nil {
		t.Error("SQL() should return non-nil *sql.DB")
	}
}

func TestDB_AppendRecord(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := &types.Record{
		Timestamp: time.Now(),
		Level:     types.LevelError,
		Message:   "test error message",
		Labels:    map[string]string{"env": "test"},
		SourceID:  "test-source",
	}
	if err := db.AppendRecord(rec); err != nil {
		t.Errorf("AppendRecord: %v", err)
	}

	// Record with no labels
	rec2 := &types.Record{
		Timestamp: time.Now(),
		Level:     types.LevelInfo,
		Message:   "info line",
		SourceID:  "test-source",
	}
	if err := db.AppendRecord(rec2); err != nil {
		t.Errorf("AppendRecord (no labels): %v", err)
	}
}
