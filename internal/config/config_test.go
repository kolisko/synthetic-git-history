package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExampleConfig(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Repository.Branch != "main" {
		t.Fatalf("branch = %q, want main", cfg.Repository.Branch)
	}
	if cfg.Range.StartRaw != "2010-01-01" {
		t.Fatalf("range.start = %q", cfg.Range.StartRaw)
	}
	if cfg.Volume.MaxCommitsPerDay < cfg.Volume.MinCommitsPerDay {
		t.Fatal("invalid commit volume bounds")
	}
}

func TestRejectsAbsoluteActivityFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "repository": {"path": "./repo"},
  "identity": {"name": "Bot", "email": "bot@example.invalid"},
  "range": {"start": "2020-01-01", "end": "2020-01-01"},
  "content": {"activity_file": "/tmp/activity.log", "message_templates": ["test"]}
}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for absolute activity_file")
	}
}
