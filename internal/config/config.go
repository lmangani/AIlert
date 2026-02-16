package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for ailert.
type Config struct {
	StorePath        string       `yaml:"store_path"`         // optional; load/save pattern store
	AlertmanagerURL  string       `yaml:"alertmanager_url"`    // optional; emit alerts / create silences
	SnapshotDir      string       `yaml:"snapshot_dir"`       // optional; directory for snapshots (change detection)
	Sources          []SourceSpec `yaml:"sources"`
}

// SourceSpec describes one data source (file, prometheus, etc.).
type SourceSpec struct {
	ID     string `yaml:"id"`
	Type   string `yaml:"type"` // "file", "prometheus", "logql", ...
	Path   string `yaml:"path"` // for type=file
	URL    string `yaml:"url"`  // for type=prometheus, logql
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
