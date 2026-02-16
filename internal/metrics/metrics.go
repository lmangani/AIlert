package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
)

// Counters for pipeline observability. Optional; zero if not used.
var (
	RecordsProcessed atomic.Int64
	PatternsNew      atomic.Int64
	PatternsKnown    atomic.Int64
	PatternsSuppressed atomic.Int64
	AlertsEmitted    atomic.Int64
)

// Handler returns an http.Handler that serves Prometheus text exposition for the counters.
// Serves on the given mux or nil to use http.DefaultServeMux.
func Handler(mux *http.ServeMux) http.Handler {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte("# HELP ailert_records_processed_total Records processed by the pipeline\n"))
		w.Write([]byte("# TYPE ailert_records_processed_total counter\n"))
		w.Write([]byte("ailert_records_processed_total " + strconv.FormatInt(RecordsProcessed.Load(), 10) + "\n"))
		w.Write([]byte("# HELP ailert_patterns_new_total Patterns seen for the first time\n"))
		w.Write([]byte("# TYPE ailert_patterns_new_total counter\n"))
		w.Write([]byte("ailert_patterns_new_total " + strconv.FormatInt(PatternsNew.Load(), 10) + "\n"))
		w.Write([]byte("# HELP ailert_patterns_known_total Patterns already seen\n"))
		w.Write([]byte("# TYPE ailert_patterns_known_total counter\n"))
		w.Write([]byte("ailert_patterns_known_total " + strconv.FormatInt(PatternsKnown.Load(), 10) + "\n"))
		w.Write([]byte("# HELP ailert_patterns_suppressed_total Records suppressed\n"))
		w.Write([]byte("# TYPE ailert_patterns_suppressed_total counter\n"))
		w.Write([]byte("ailert_patterns_suppressed_total " + strconv.FormatInt(PatternsSuppressed.Load(), 10) + "\n"))
		w.Write([]byte("# HELP ailert_alerts_emitted_total Alerts sent to Alertmanager\n"))
		w.Write([]byte("# TYPE ailert_alerts_emitted_total counter\n"))
		w.Write([]byte("ailert_alerts_emitted_total " + strconv.FormatInt(AlertsEmitted.Load(), 10) + "\n"))
	})
}

var (
	metricsServerOnce sync.Once
	metricsServer     *http.Server
)

// Serve starts an HTTP server on addr (e.g. ":9090") serving /metrics. Non-blocking.
func Serve(addr string) {
	metricsServerOnce.Do(func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", Handler(mux))
		metricsServer = &http.Server{Addr: addr, Handler: mux}
		go metricsServer.ListenAndServe()
	})
}
