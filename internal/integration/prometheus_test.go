package integration

import (
	"context"
	"testing"

	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/testutil"
)

func TestPrometheusSource(t *testing.T) {
	srv := testutil.NewMetricsServer(testutil.SamplePrometheusMetrics())
	defer srv.Close()

	st := store.New("")
	eng := engine.New(st)
	ctx := context.Background()
	src := &source.PrometheusSource{URL: srv.URL(), SourceID: "metrics"}
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
	// Sample has 3 data lines (2 with labels, 1 without)
	if count < 3 {
		t.Errorf("expected at least 3 metric lines, got %d", count)
	}
	list := st.ListSeen()
	if len(list) == 0 {
		t.Error("expected at least one pattern from metrics")
	}
}
