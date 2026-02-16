package source

import (
	"context"

	"github.com/ailert/ailert/internal/types"
)

// Source produces a stream of normalized Records.
// Implementations: file tail, Prometheus scrape, LogQL, etc.
type Source interface {
	// ID returns a stable identifier for this source (e.g. "file:var/log/app.log").
	ID() string
	// Stream sends records on the returned channel until ctx is done or an error occurs.
	// The channel is closed when the source finishes or errors.
	Stream(ctx context.Context) (<-chan types.Record, <-chan error)
}

// FormatMapping normalizes raw content into Record fields (e.g. timestamp layout, level field name).
// Used by source implementations that read structured data.
type FormatMapping struct {
	TimestampLayout string            `yaml:"timestamp_layout"` // e.g. "2006-01-02T15:04:05Z07:00"
	LevelField      string            `yaml:"level_field"`      // JSON field name for level
	MessageField    string            `yaml:"message_field"`   // JSON field name for message
	LabelFields     map[string]string  `yaml:"label_fields"`    // raw field name -> label name
}
