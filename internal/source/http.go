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

// HTTPSource fetches a URL (GET) and emits each non-empty line as a Record.
// Useful for log URLs or simple text endpoints.
type HTTPSource struct {
	URL      string
	SourceID string
	Client   *http.Client
}

// ID implements Source.
func (h *HTTPSource) ID() string {
	if h.SourceID != "" {
		return h.SourceID
	}
	return "http:" + h.URL
}

// Stream implements Source. One fetch, then closes.
func (h *HTTPSource) Stream(ctx context.Context) (<-chan types.Record, <-chan error) {
	recCh := make(chan types.Record, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(recCh)
		defer close(errCh)
		client := h.Client
		if client == nil {
			client = &http.Client{Timeout: 15 * time.Second}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.URL, nil)
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
			errCh <- fmt.Errorf("GET %s: %s", h.URL, resp.Status)
			return
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errCh <- err
			return
		}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
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
				SourceID:  h.ID(),
			}
		}
	}()
	return recCh, errCh
}
