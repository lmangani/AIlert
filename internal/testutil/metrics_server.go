package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MetricsServer is a minimal HTTP server that serves Prometheus text exposition at /metrics.
// Use for integration tests without a real Prometheus.
type MetricsServer struct {
	mu     sync.Mutex
	body   string
	Server *httptest.Server
}

// NewMetricsServer starts a test server serving the given body at /metrics.
func NewMetricsServer(metricsBody string) *MetricsServer {
	m := &MetricsServer{body: metricsBody}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.NotFound(w, r)
			return
		}
		m.mu.Lock()
		b := m.body
		m.mu.Unlock()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprint(w, b)
	}))
	return m
}

// URL returns the full URL to /metrics.
func (m *MetricsServer) URL() string {
	return m.Server.URL + "/metrics"
}

// Close shuts down the server.
func (m *MetricsServer) Close() {
	m.Server.Close()
}

// SetBody updates the served body (e.g. for dynamic tests).
func (m *MetricsServer) SetBody(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.body = s
}

// SamplePrometheusMetrics returns a minimal valid Prometheus exposition body.
func SamplePrometheusMetrics() string {
	return `# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",code="200"} 100
http_requests_total{method="POST",code="500"} 2
go_goroutines 8
`
}
