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
