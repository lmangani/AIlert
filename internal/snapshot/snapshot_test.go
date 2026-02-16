package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ailert/ailert/internal/types"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	ents := []PatternEnt{
		{Level: types.LevelError, Hash: "h1", Sample: "err", Count: 2},
		{Level: types.LevelWarn, Hash: "h2", Sample: "warn", Count: 1},
	}
	if err := Save(path, ents); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}
	if len(loaded.Patterns) != 2 {
		t.Fatalf("len(Patterns) = %d", len(loaded.Patterns))
	}
	byHash := make(map[string]PatternEnt)
	for _, p := range loaded.Patterns {
		byHash[p.Hash] = p
	}
	if len(byHash) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(byHash))
	}
	if byHash["h1"].Count != 2 || byHash["h2"].Count != 1 {
		t.Errorf("Patterns: %+v", loaded.Patterns)
	}
	if loaded.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestLoadMissing(t *testing.T) {
	loaded, err := Load("/nonexistent/snap.json")
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("Load(missing) should return nil snapshot")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "snap.json")
	ents := []PatternEnt{{Level: types.LevelInfo, Hash: "x", Count: 1}}
	if err := Save(path, ents); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

var _ = time.Time{} // use time in tests if needed
