package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ailert/ailert/internal/types"
)

func TestStoreSeen(t *testing.T) {
	st := New("")
	isNew := st.Seen(types.LevelError, "abc123", "sample line")
	if !isNew {
		t.Error("first Seen should be new")
	}
	isNew = st.Seen(types.LevelError, "abc123", "another")
	if isNew {
		t.Error("second Seen should not be new")
	}
	if c := st.GetCount(types.LevelError, "abc123"); c != 2 {
		t.Errorf("GetCount = %d, want 2", c)
	}
}

func TestStoreSuppress(t *testing.T) {
	st := New("")
	st.Seen(types.LevelWarn, "xyz", "warn sample")
	st.Suppress("xyz", "noise")
	if !st.IsSuppressed("xyz") {
		t.Error("IsSuppressed should be true")
	}
	if st.IsSuppressed("other") {
		t.Error("IsSuppressed(other) should be false")
	}
}

func TestStorePersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	st := New(path)
	st.Seen(types.LevelError, "h1", "sample1")
	st.Suppress("h1", "test")
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	st2 := New(path)
	if err := st2.Load(); err != nil {
		t.Fatal(err)
	}
	if c := st2.GetCount(types.LevelError, "h1"); c != 1 {
		t.Errorf("after Load GetCount = %d, want 1", c)
	}
	if !st2.IsSuppressed("h1") {
		t.Error("after Load IsSuppressed should be true")
	}
	os.Remove(path)
}

func TestStoreListSeen_MultipleLevels(t *testing.T) {
	st := New("")
	st.Seen(types.LevelError, "h1", "e1")
	st.Seen(types.LevelWarn, "h2", "w1")
	st.Seen(types.LevelInfo, "h3", "i1")
	list := st.ListSeen()
	if len(list) != 3 {
		t.Fatalf("ListSeen len = %d, want 3", len(list))
	}
	byLevel := make(map[types.Level]int)
	for _, p := range list {
		byLevel[p.Level]++
	}
	if byLevel[types.LevelError] != 1 || byLevel[types.LevelWarn] != 1 || byLevel[types.LevelInfo] != 1 {
		t.Errorf("ListSeen by level: %v", byLevel)
	}
}

func TestStore_Load_NonExistentFile(t *testing.T) {
	st := New(filepath.Join(t.TempDir(), "does_not_exist.json"))
	if err := st.Load(); err != nil {
		t.Errorf("Load of non-existent file should return nil, got %v", err)
	}
}

func TestStore_Load_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	st := New(path)
	if err := st.Load(); err == nil {
		t.Error("Load of invalid JSON should return error")
	}
}

func TestStore_Save_NoPath(t *testing.T) {
	st := New("")
	st.Seen(types.LevelError, "h1", "s1")
	if err := st.Save(); err != nil {
		t.Errorf("Save with no path should be no-op, got %v", err)
	}
}

func TestStore_Save_BadPath(t *testing.T) {
	// Use a path where parent is a file (can't mkdir)
	dir := t.TempDir()
	// Make a file where a directory would need to be
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	st := New(filepath.Join(blocker, "subdir", "store.json"))
	st.Seen(types.LevelError, "h1", "s1")
	if err := st.Save(); err == nil {
		t.Error("Save to bad path should return error")
	}
}

func TestStore_Seen_UpdatesSample(t *testing.T) {
	st := New("")
	st.Seen(types.LevelError, "h1", "")
	// second call with non-empty sample should keep original empty sample but update count
	st.Seen(types.LevelError, "h1", "new sample")
	if c := st.GetCount(types.LevelError, "h1"); c != 2 {
		t.Errorf("count = %d, want 2", c)
	}
}
