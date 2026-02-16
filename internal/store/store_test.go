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
