package source

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ailert/ailert/internal/testutil"
)

func TestFileSource_ReadFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	lines := []string{
		"ERROR first line",
		"WARN second line",
	}
	if err := testutil.WriteLogLines(logPath, lines); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &FileSource{Path: logPath, SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	var recs []string
	for rec := range recCh {
		recs = append(recs, rec.Message)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0] != "ERROR first line" || recs[1] != "WARN second line" {
		t.Errorf("recs = %v", recs)
	}
}

func TestFileSource_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")
	if err := testutil.WriteLogLines(logPath, nil); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &FileSource{Path: logPath, SourceID: "test"}
	recCh, _ := src.Stream(ctx)
	count := 0
	for range recCh {
		count++
	}
	if count != 0 {
		t.Errorf("empty file should yield 0 records, got %d", count)
	}
}

func TestFileSource_MissingFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &FileSource{Path: "/nonexistent/path.log", SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	<-recCh // drain so goroutine can exit
	err := <-errCh
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileSource_ID(t *testing.T) {
	src := &FileSource{Path: "/var/log/app.log", SourceID: "my-id"}
	if got := src.ID(); got != "my-id" {
		t.Errorf("ID() = %q, want my-id", got)
	}
	src2 := &FileSource{Path: "/var/log/app.log"}
	if got := src2.ID(); got != "file:/var/log/app.log" {
		t.Errorf("ID() = %q, want file:/var/log/app.log", got)
	}
}

func TestFileSource_SkipsEmptyLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "mixed.log")
	lines := []string{"a", "", "b", "  ", "c"}
	if err := testutil.WriteLogLines(logPath, lines); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &FileSource{Path: logPath, SourceID: "test"}
	recCh, _ := src.Stream(ctx)
	var recs []string
	for rec := range recCh {
		recs = append(recs, rec.Message)
	}
	// We trim space; "  " becomes "" and is skipped
	if len(recs) != 3 {
		t.Fatalf("expected 3 non-empty lines, got %d: %v", len(recs), recs)
	}
}
