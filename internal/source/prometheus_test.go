package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPrometheusSource_Scrape(t *testing.T) {
	body := `# HELP foo
# TYPE foo counter
foo 1
bar{label="x"} 2
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &PrometheusSource{URL: srv.URL + "/metrics", SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	var recs []string
	for rec := range recCh {
		recs = append(recs, rec.Message)
	}
	go func() { <-errCh }()
	if len(recs) != 2 {
		t.Fatalf("expected 2 data lines (foo, bar), got %d: %v", len(recs), recs)
	}
}

func TestPrometheusSource_404(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := &PrometheusSource{URL: srv.URL + "/metrics", SourceID: "test"}
	recCh, errCh := src.Stream(ctx)
	for range recCh {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestPrometheusSource_ID(t *testing.T) {
	src := &PrometheusSource{URL: "http://localhost:9090/metrics", SourceID: "metrics"}
	if got := src.ID(); got != "metrics" {
		t.Errorf("ID() = %q", got)
	}
	src2 := &PrometheusSource{URL: "http://localhost:9090/metrics"}
	if got := src2.ID(); got != "prometheus:http://localhost:9090/metrics" {
		t.Errorf("ID() = %q", got)
	}
}
