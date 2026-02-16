package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Missing file
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(missing) should error")
	}

	// Valid config
	content := `
store_path: ".ailert/store.json"
sources:
  - id: src1
    type: file
    path: /var/log/app.log
  - id: metrics
    type: prometheus
    url: http://localhost:9090/metrics
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StorePath != ".ailert/store.json" {
		t.Errorf("StorePath = %q", cfg.StorePath)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("Sources len = %d", len(cfg.Sources))
	}
	if cfg.Sources[0].ID != "src1" || cfg.Sources[0].Type != "file" || cfg.Sources[0].Path != "/var/log/app.log" {
		t.Errorf("Sources[0] = %+v", cfg.Sources[0])
	}
	if cfg.Sources[1].Type != "prometheus" || cfg.Sources[1].URL != "http://localhost:9090/metrics" {
		t.Errorf("Sources[1] = %+v", cfg.Sources[1])
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("sources:\n  - id: [broken"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(invalid YAML) should error")
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StorePath != "" || len(cfg.Sources) != 0 {
		t.Errorf("empty config: %+v", cfg)
	}
}
