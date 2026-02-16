package alertmanager

import (
	"encoding/json"
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
	// Marshal/unmarshal roundtrip
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

// TestPostAlertsAgainstRealAM runs only when ALERTMANAGER_URL is set (CI or local).
func TestPostAlertsAgainstRealAM(t *testing.T) {
	url := os.Getenv("ALERTMANAGER_URL")
	if url == "" {
		t.Skip("ALERTMANAGER_URL not set")
	}
	client := NewClient(url)
	a := Alert{
		Labels: map[string]string{
			"alertname":    "ailert_smoke",
			"pattern_hash": "smoke_test_hash",
			"level":        "ERROR",
			"source":       "test",
		},
		Annotations: map[string]string{
			"summary":     "AIlert smoke test",
			"description": "Sample log line",
		},
		StartsAt: time.Now(),
	}
	if err := client.PostAlerts([]Alert{a}); err != nil {
		t.Fatal(err)
	}
	// Optionally GET to verify (API may return different shape; don't fail test)
	alerts, err := client.GetAlerts(nil)
	if err != nil {
		t.Log("GetAlerts (optional):", err)
		return
	}
	for _, b := range alerts {
		if b.Labels["alertname"] == "ailert_smoke" && b.Labels["pattern_hash"] == "smoke_test_hash" {
			t.Log("alert visible in GET")
			return
		}
	}
	t.Log("alert may not appear in GET immediately")
}
