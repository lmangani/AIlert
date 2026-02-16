// Package integration tests the full pipeline (source → engine → store) with
// simulated log datasets. See testutil.Datasets() for the list of scenarios:
// MixedLevels, SamePatternRepeated, AllDistinct, OnlyErrors, WithEmptyAndWhitespace,
// LevelInMiddle, UUIDAndHexDropped, JavaStyleStacktrace, SingleLine, Empty.
package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/pattern"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/testutil"
)

// TestPipeline_FileSource_PatternDetection_SimulatedDatasets runs file → engine → store
// for each simulated dataset and asserts new/known counts and pattern diversity.
func TestPipeline_FileSource_PatternDetection_SimulatedDatasets(t *testing.T) {
	for _, ds := range testutil.Datasets() {
		t.Run(ds.Name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "log.txt")
			if len(ds.Lines) > 0 {
				if err := testutil.WriteLogLines(logPath, ds.Lines); err != nil {
					t.Fatal(err)
				}
			} else {
				// Empty dataset: create empty file
				if err := testutil.WriteLogLines(logPath, nil); err != nil {
					t.Fatal(err)
				}
			}
			st := store.New("")
			eng := engine.New(st)
			ctx := context.Background()
			src := &source.FileSource{Path: logPath, SourceID: "test"}
			recCh, errCh := src.Stream(ctx)
			var newCount, knownCount int
			go func() {
				for range errCh {
				}
			}()
			for rec := range recCh {
				res := eng.Process(&rec)
				if res.Suppressed {
					continue
				}
				if res.IsNew {
					newCount++
				} else {
					knownCount++
				}
			}
			if newCount != ds.WantNew {
				t.Errorf("new: got %d, want %d", newCount, ds.WantNew)
			}
			if knownCount != ds.WantKnown {
				t.Errorf("known: got %d, want %d", knownCount, ds.WantKnown)
			}
			list := st.ListSeen()
			if len(list) < ds.WantPatterns {
				t.Errorf("patterns: got %d, want at least %d", len(list), ds.WantPatterns)
			}
		})
	}
}

// TestPipeline_Suppression_PreSuppressedPatternNotCounted verifies that pre-suppressed
// patterns are not counted as new/known and only non-suppressed patterns appear in the store.
func TestPipeline_Suppression_PreSuppressedPatternNotCounted(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "log.txt")
	lines := []string{
		"ERROR noisy message 1",
		"ERROR noisy message 2",
		"ERROR important message",
	}
	if err := testutil.WriteLogLines(logPath, lines); err != nil {
		t.Fatal(err)
	}
	st := store.New("")
	eng := engine.New(st)
	// Suppress the "noisy" pattern (same template for both "1" and "2" after number stripping)
	noisyPat := pattern.New("ERROR noisy message 1")
	st.Suppress(noisyPat.Hash(), "noise")
	ctx := context.Background()
	src := &source.FileSource{Path: logPath, SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	var newCount, knownCount, suppressedCount int
	go func() {
		for range errCh {
		}
	}()
	for rec := range recCh {
		res := eng.Process(&rec)
		if res.Suppressed {
			suppressedCount++
			continue
		}
		if res.IsNew {
			newCount++
		} else {
			knownCount++
		}
	}
	if suppressedCount != 2 {
		t.Errorf("suppressed: got %d, want 2", suppressedCount)
	}
	if newCount != 1 {
		t.Errorf("new (only important): got %d, want 1", newCount)
	}
	if knownCount != 0 {
		t.Errorf("known: got %d, want 0", knownCount)
	}
	list := st.ListSeen()
	// Suppressed pattern might still be in "seen" store; we just don't alert. So we could have 2 patterns (noisy + important) or 1 (only important if we don't call Seen for suppressed). Currently we do NOT call Seen for suppressed, so only "important" is in store.
	if len(list) != 1 {
		t.Errorf("patterns in store: got %d, want 1 (only non-suppressed)", len(list))
	}
}
