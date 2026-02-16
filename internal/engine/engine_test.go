package engine

import (
	"testing"

	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/types"
)

func TestEngineProcess(t *testing.T) {
	st := store.New("")
	eng := New(st)
	rec := &types.Record{Message: "ERROR connection refused from 10.0.0.1", SourceID: "test"}
	res := eng.Process(rec)
	if !res.IsNew {
		t.Error("first Process should be new")
	}
	if res.Level != types.LevelError {
		t.Errorf("Level = %v, want Error", res.Level)
	}
	if res.Hash == "" {
		t.Error("Hash should be set")
	}
	res2 := eng.Process(&types.Record{Message: "ERROR connection refused from 10.0.0.2", SourceID: "test"})
	if res2.IsNew {
		t.Error("second Process (same pattern) should not be new")
	}
	if res2.Count != 2 {
		t.Errorf("Count = %d, want 2", res2.Count)
	}
}

func TestEngineSuppressed(t *testing.T) {
	st := store.New("")
	eng := New(st)
	rec := &types.Record{Message: "WARN something", SourceID: "test"}
	res := eng.Process(rec)
	st.Suppress(res.Hash, "noise")
	res2 := eng.Process(rec)
	if !res2.Suppressed {
		t.Error("expected Suppressed after Suppress()")
	}
}

func TestEngineProcess_DifferentLevelsSameTemplate(t *testing.T) {
	st := store.New("")
	eng := New(st)
	// Same message text but we force different levels via content
	rec1 := &types.Record{Message: "ERROR connection refused", SourceID: "test"}
	rec2 := &types.Record{Message: "WARN connection refused", SourceID: "test"}
	r1 := eng.Process(rec1)
	r2 := eng.Process(rec2)
	if r1.Level != types.LevelError || r2.Level != types.LevelWarn {
		t.Errorf("levels: %v, %v", r1.Level, r2.Level)
	}
	// Different level => different pattern key in engine, so both are "new"
	if !r1.IsNew || !r2.IsNew {
		t.Errorf("both should be new (different level keys): r1.IsNew=%v r2.IsNew=%v", r1.IsNew, r2.IsNew)
	}
}
