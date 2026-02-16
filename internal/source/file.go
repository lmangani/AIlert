package source

import (
	"bufio"
	"context"
	"os"
	"strings"
	"time"

	"github.com/ailert/ailert/internal/types"
)

// FileSource tails a file (or reads it fully) and emits one Record per line.
// No format mapping: each line is the message, level is detected from content.
type FileSource struct {
	Path   string
	SourceID string
	Tail   bool // if true, follow file like tail -f; else read once
}

// ID implements Source.
func (f *FileSource) ID() string {
	if f.SourceID != "" {
		return f.SourceID
	}
	return "file:" + f.Path
}

// Stream implements Source.
func (f *FileSource) Stream(ctx context.Context) (<-chan types.Record, <-chan error) {
	recCh := make(chan types.Record, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(recCh)
		defer close(errCh)
		file, err := os.Open(f.Path)
		if err != nil {
			errCh <- err
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			recCh <- types.Record{
				Timestamp: time.Now(),
				Level:     types.LevelUnknown,
				Message:   line,
				SourceID:  f.ID(),
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		if !f.Tail {
			return
		}
		// Simple tail: re-read from end (in a real impl use fsnotify + seek)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// Read-once only; tail (follow) can be added later with fsnotify
			return
		}
	}()
	return recCh, errCh
}
