package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
)

func TestPipeline_HTTPSource_IngestsLines(t *testing.T) {
	body := "ERROR connection refused\nWARN timeout\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	st := store.New("")
	eng := engine.New(st)
	src := &source.HTTPSource{URL: srv.URL, SourceID: "http-test"}
	recCh, errCh := src.Stream(ctx)
	go func() {
		for range errCh {
		}
	}()
	var newCount int
	for rec := range recCh {
		res := eng.Process(&rec)
		if res.IsNew {
			newCount++
		}
	}
	if newCount != 2 {
		t.Errorf("expected 2 new patterns, got %d", newCount)
	}
	list := st.ListSeen()
	if len(list) != 2 {
		t.Errorf("expected 2 patterns in store, got %d", len(list))
	}
}
