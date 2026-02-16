package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ailert/ailert/internal/types"
)

// PrometheusSource scrapes a Prometheus /metrics endpoint and emits each non-empty,
// non-comment line as a Record (message = line). Used for ingestion and pattern tests.
type PrometheusSource struct {
	URL      string
	SourceID string
	Client   *http.Client
}

// ID implements Source.
func (p *PrometheusSource) ID() string {
	if p.SourceID != "" {
		return p.SourceID
	}
	return "prometheus:" + p.URL
}

// Stream implements Source. It performs one scrape and then closes the channel.
func (p *PrometheusSource) Stream(ctx context.Context) (<-chan types.Record, <-chan error) {
	recCh := make(chan types.Record, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(recCh)
		defer close(errCh)
		client := p.Client
		if client == nil {
			client = &http.Client{Timeout: 10 * time.Second}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
		if err != nil {
			errCh <- err
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("GET %s: %s", p.URL, resp.Status)
			return
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errCh <- err
			return
		}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			recCh <- types.Record{
				Timestamp: time.Now(),
				Level:     types.LevelUnknown,
				Message:   line,
				SourceID:  p.ID(),
			}
		}
	}()
	return recCh, errCh
}
