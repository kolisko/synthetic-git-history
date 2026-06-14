package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitConfigWritesRequestedFileAndPrintsGuidance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	output := captureStdout(t, func() {
		if err := run([]string{"init-config", "--output", path}); err != nil {
			t.Fatal(err)
		}
	})

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if !strings.Contains(output, "Created config: "+path) {
		t.Fatalf("expected output to include created path, got:\n%s", output)
	}
	if !strings.Contains(output, `"repository"`) {
		t.Fatalf("expected output to include config contents, got:\n%s", output)
	}
	if !strings.Contains(output, "Edit this file") {
		t.Fatalf("expected output to include edit guidance, got:\n%s", output)
	}
	if !strings.Contains(output, "synthgit plan --config "+path) {
		t.Fatalf("expected custom output guidance to include --config, got:\n%s", output)
	}
}

func TestInitConfigRefusesToOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{"init-config", "--output", path})
	if err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite existing file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputPathUsesHomeForDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := resolveOutputPath("~/.synthgit.config.json")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(home, ".synthgit.config.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestPlanUsesDefaultHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".synthgit.config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "repository": {"path": "./repo", "init": true},
  "identity": {"name": "Bot", "email": "bot@example.invalid"},
  "range": {"start": "2020-01-01", "end": "2020-01-01"},
  "volume": {"min_commits_per_day": 1, "max_commits_per_day": 1},
  "content": {"message_templates": ["Commit {sequence}"]}
}`), 0644); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if err := run([]string{"plan"}); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "Planned commits: 1") {
		t.Fatalf("expected default config to be used, got:\n%s", output)
	}
}

func TestInitConfigDefaultOutputPrintsSimpleNextSteps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	output := captureStdout(t, func() {
		if err := run([]string{"init-config"}); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "Created config: "+filepath.Join(home, ".synthgit.config.json")) {
		t.Fatalf("expected default config path, got:\n%s", output)
	}
	if !strings.Contains(output, "Then run:\n  synthgit plan\n  synthgit generate") {
		t.Fatalf("expected simple next steps for default config, got:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	bytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}
