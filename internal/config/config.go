package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for ailert.
type Config struct {
	StorePath       string       `yaml:"store_path"`        // optional; load/save pattern store (JSON). Ignored when DuckDBPath is set.
	DuckDBPath      string       `yaml:"duckdb_path"`       // optional; use DuckDB for store, records, snapshots. When set, store_path/snapshot_dir are ignored for persistence.
	AlertmanagerURL string       `yaml:"alertmanager_url"`  // optional; emit alerts / create silences
	SnapshotDir     string       `yaml:"snapshot_dir"`     // optional; directory for file snapshots (used only when DuckDBPath is empty)
	Sources         []SourceSpec `yaml:"sources"`
}

// SourceSpec describes one data source (file, prometheus, duckdb, etc.).
type SourceSpec struct {
	ID    string `yaml:"id"`
	Type  string `yaml:"type"`  // "file", "prometheus", "http", "duckdb", ...
	Path  string `yaml:"path"` // for type=file; for type=duckdb optional DB path (else use config duckdb_path)
	URL   string `yaml:"url"`   // for type=prometheus, http
	Query string `yaml:"query"` // for type=duckdb optional SQL query (default: SELECT from records)
}

// Load reads config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
