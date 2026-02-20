package alertmanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAlertJSON(t *testing.T) {
	a := Alert{
		Labels:      map[string]string{"alertname": "Test", "pattern_hash": "abc"},
		Annotations: map[string]string{"summary": "test alert"},
		StartsAt:    time.Now(),
		EndsAt:      time.Time{},
	}
	data, err := json.Marshal([]Alert{a})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty json")
	}
	var out []Alert
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out[0].Labels["alertname"] != "Test" {
		t.Errorf("labels: %v", out[0].Labels)
	}
}

func TestSilenceJSON(t *testing.T) {
	s := Silence{
		Matchers:  []Matcher{{Name: "alertname", Value: "ailert", IsRegex: false}},
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Hour),
		CreatedBy: "test",
		Comment:   "suppress",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var out Silence
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Matchers) != 1 || out.Matchers[0].Name != "alertname" {
		t.Errorf("matchers: %v", out.Matchers)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:9093")
	if c.BaseURL != "http://localhost:9093" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestPostAlerts_Empty(t *testing.T) {
	c := NewClient("http://localhost:9093")
	if err := c.PostAlerts(nil); err != nil {
		t.Errorf("PostAlerts(nil) should be no-op, got %v", err)
	}
}

func TestPostAlerts_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/alerts" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	a := Alert{Labels: map[string]string{"alertname": "test"}, StartsAt: time.Now()}
	if err := c.PostAlerts([]Alert{a}); err != nil {
		t.Errorf("PostAlerts: %v", err)
	}
}

func TestPostAlerts_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	a := Alert{Labels: map[string]string{"alertname": "test"}, StartsAt: time.Now()}
	if err := c.PostAlerts([]Alert{a}); err == nil {
		t.Error("expected error on 500")
	}
}

func TestPostSilence_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/silences" {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"silenceID":"abc-123"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	s := Silence{
		Matchers:  []Matcher{{Name: "pattern_hash", Value: "h1"}},
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Hour),
		CreatedBy: "test",
		Comment:   "noise",
	}
	id, err := c.PostSilence(s)
	if err != nil {
		t.Fatalf("PostSilence: %v", err)
	}
	if id != "abc-123" {
		t.Errorf("silenceID = %q, want abc-123", id)
	}
}

func TestPostSilence_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	s := Silence{Matchers: []Matcher{{Name: "h", Value: "v"}}, StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}
	_, err := c.PostSilence(s)
	if err == nil {
		t.Error("expected error on 400")
	}
}

func TestPostSilence_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	s := Silence{Matchers: []Matcher{{Name: "h", Value: "v"}}, StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}
	_, err := c.PostSilence(s)
	if err == nil {
		t.Error("expected error on bad JSON response from PostSilence")
	}
}

func TestGetAlerts_Success(t *testing.T) {
	alerts := []Alert{
		{Labels: map[string]string{"alertname": "ailert", "level": "ERROR"}, StartsAt: time.Now()},
	}
	body, _ := json.Marshal(alerts)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	got, err := c.GetAlerts(nil)
	if err != nil {
		t.Fatalf("GetAlerts: %v", err)
	}
	if len(got) != 1 || got[0].Labels["alertname"] != "ailert" {
		t.Errorf("got alerts: %+v", got)
	}
}

func TestGetAlerts_WithActiveFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("active") != "true" {
			http.Error(w, "missing filter", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	active := true
	got, err := c.GetAlerts(&active)
	if err != nil {
		t.Fatalf("GetAlerts(active=true): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(got))
	}
}

func TestGetAlerts_ActiveFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("active") != "false" {
			http.Error(w, "missing filter", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	active := false
	got, err := c.GetAlerts(&active)
	if err != nil {
		t.Fatalf("GetAlerts(active=false): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(got))
	}
}

func TestGetAlerts_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "err", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetAlerts(nil)
	if err == nil {
		t.Error("expected error on 500")
	}
}

func TestGetAlerts_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetAlerts(nil)
	if err == nil {
		t.Error("expected error on bad JSON response")
	}
}

// TestAlertmanager_PostAlerts_Integration runs when ALERTMANAGER_URL is set (CI or local).
func TestAlertmanager_PostAlerts_Integration(t *testing.T) {
	url := os.Getenv("ALERTMANAGER_URL")
	if url == "" {
		t.Skip("ALERTMANAGER_URL not set")
	}
	client := NewClient(url)
	a := Alert{
		Labels: map[string]string{
			"alertname":    "ailert_integration",
			"pattern_hash": "integration_test_hash",
			"level":        "ERROR",
			"source":       "test",
		},
		Annotations: map[string]string{
			"summary":     "AIlert integration test",
			"description": "Sample log line",
		},
		StartsAt: time.Now(),
	}
	if err := client.PostAlerts([]Alert{a}); err != nil {
		t.Fatal(err)
	}
	alerts, err := client.GetAlerts(nil)
	if err != nil {
		t.Log("GetAlerts (optional):", err)
		return
	}
	for _, b := range alerts {
		if b.Labels["alertname"] == "ailert_integration" && b.Labels["pattern_hash"] == "integration_test_hash" {
			t.Log("alert visible in GET")
			return
		}
	}
	t.Log("alert may not appear in GET immediately")
}
