package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_MetricsEndpoint(t *testing.T) {
	// Reset counters for deterministic test
	RecordsProcessed.Store(3)
	PatternsNew.Store(2)
	PatternsKnown.Store(1)
	PatternsSuppressed.Store(0)
	AlertsEmitted.Store(5)

	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"ailert_records_processed_total 3",
		"ailert_patterns_new_total 2",
		"ailert_patterns_known_total 1",
		"ailert_patterns_suppressed_total 0",
		"ailert_alerts_emitted_total 5",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\nbody:\n%s", want, body)
		}
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain prefix", ct)
	}
}

func TestHandler_404(t *testing.T) {
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandler_CustomMux(t *testing.T) {
	mux := http.NewServeMux()
	h := Handler(mux)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
