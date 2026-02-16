package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ailert/ailert/internal/config"
	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
)

// TestFileSource runs the pipeline with a file source and asserts new/known counts.
func TestFileSource(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	content := `ERROR connection refused from 10.0.0.1
ERROR connection refused from 10.0.0.2
WARN timeout after 5000 ms
INFO server started
`
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	cfgYaml := fmt.Sprintf(`
sources:
  - id: test
    type: file
    path: %s
`, logPath)
	if err := os.WriteFile(cfgPath, []byte(cfgYaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg
	st := store.New("")
	eng := engine.New(st)
	ctx := context.Background()
	var src source.Source = &source.FileSource{Path: logPath, SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	var newCount, knownCount int
	go func() {
		if err := <-errCh; err != nil {
			t.Error("source error:", err)
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
	if newCount != 3 {
		t.Errorf("new count = %d, want 3", newCount)
	}
	if knownCount != 1 {
		t.Errorf("known count = %d, want 1", knownCount)
	}
	list := st.ListSeen()
	if len(list) < 3 {
		t.Errorf("expected at least 3 patterns in store, got %d", len(list))
	}
}
