package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSource_Get(t *testing.T) {
	body := "line1\nline2\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &HTTPSource{URL: srv.URL, SourceID: "http-test"}
	recCh, errCh := src.Stream(ctx)
	var recs []string
	for rec := range recCh {
		recs = append(recs, rec.Message)
	}
	go func() { <-errCh }()
	if len(recs) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(recs), recs)
	}
}

func TestHTTPSource_ID(t *testing.T) {
	src := &HTTPSource{URL: "http://example.com/log", SourceID: "my-http"}
	if got := src.ID(); got != "my-http" {
		t.Errorf("ID() = %q", got)
	}
	src2 := &HTTPSource{URL: "http://example.com/log"}
	if got := src2.ID(); got != "http:http://example.com/log" {
		t.Errorf("ID() = %q", got)
	}
}

func TestHTTPSource_Non200(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &HTTPSource{URL: srv.URL + "/missing", SourceID: "http-404"}
	recCh, errCh := src.Stream(ctx)
	for range recCh {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestHTTPSource_BadURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &HTTPSource{URL: "http://127.0.0.1:0/logs", SourceID: "bad-url"}
	recCh, errCh := src.Stream(ctx)
	for range recCh {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected connection error for unreachable URL")
	}
}

func TestHTTPSource_ContextCancel(t *testing.T) {
	// Server that blocks until handler is closed
	ready := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(ready)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	src := &HTTPSource{URL: srv.URL, SourceID: "cancel-test"}
	recCh, errCh := src.Stream(ctx)
	<-ready
	cancel()
	// drain channels; must not hang
	for range recCh {
	}
	for range errCh {
	}
}
