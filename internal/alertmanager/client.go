package alertmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client talks to Alertmanager API v2.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client for the given Alertmanager base URL (e.g. http://localhost:9093).
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Alert is the API v2 alert payload.
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations,omitempty"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	GeneratorURL string          `json:"generatorURL,omitempty"`
}

// PostAlerts sends alerts to POST /api/v2/alerts. Returns error on non-2xx.
func (c *Client) PostAlerts(alerts []Alert) error {
	if len(alerts) == 0 {
		return nil
	}
	body, err := json.Marshal(alerts)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v2/alerts", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alertmanager POST /api/v2/alerts: %s", resp.Status)
	}
	return nil
}

// Silence is the API v2 silence payload (create).
type Silence struct {
	Matchers  []Matcher   `json:"matchers"`
	StartsAt  time.Time   `json:"startsAt"`
	EndsAt    time.Time   `json:"endsAt"`
	CreatedBy string      `json:"createdBy"`
	Comment   string      `json:"comment"`
}

// Matcher is a label matcher.
type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

// SilenceResponse is the response from POST /api/v2/silences.
type SilenceResponse struct {
	Data struct {
		SilenceID string `json:"silenceID"`
	} `json:"data"`
}

// PostSilence creates a silence. Returns the silence ID or error.
func (c *Client) PostSilence(s Silence) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v2/silences", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("alertmanager POST /api/v2/silences: %s", resp.Status)
	}
	var out SilenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Data.SilenceID, nil
}

// GetAlerts returns GET /api/v2/alerts (optional filter by active=true).
func (c *Client) GetAlerts(active *bool) ([]Alert, error) {
	url := c.BaseURL + "/api/v2/alerts"
	if active != nil {
		v := "false"
		if *active {
			v = "true"
		}
		url += "?active=" + v
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alertmanager GET /api/v2/alerts: %s", resp.Status)
	}
	var out []Alert
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
