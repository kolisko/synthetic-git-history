package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kolisko/synthetic-git-history/internal/schedule"
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

func TestDefaultConfigPathUsesUserConfigDir(t *testing.T) {
	configHome := t.TempDir()
	wantBase := setTestConfigHome(t, configHome)

	path, err := defaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(wantBase, "synthgit", "config.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestPlanUsesDefaultConfig(t *testing.T) {
	configHome := t.TempDir()
	wantBase := setTestConfigHome(t, configHome)
	configPath := filepath.Join(wantBase, "synthgit", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatal(err)
	}
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
	configHome := t.TempDir()
	wantBase := setTestConfigHome(t, configHome)

	output := captureStdout(t, func() {
		if err := run([]string{"init-config"}); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "Created config: "+filepath.Join(wantBase, "synthgit", "config.json")) {
		t.Fatalf("expected default config path, got:\n%s", output)
	}
	if !strings.Contains(output, "Then run:\n  synthgit plan\n  synthgit generate") {
		t.Fatalf("expected simple next steps for default config, got:\n%s", output)
	}
}

func TestPrintGenerateProgress(t *testing.T) {
	spec := schedule.CommitSpec{
		Timestamp: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
		Timezone:  "+00:00",
		Message:   "Commit 3",
	}

	output := captureStdout(t, func() {
		printGenerateProgress(3, 10, spec)
	})

	want := "[3/10 30%] created 2020-01-02T03:04:05 +00:00 | Commit 3\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
}

func TestFilterMissingDaysKeepsOnlyDaysWithoutExistingCommits(t *testing.T) {
	specs := []schedule.CommitSpec{
		{Timestamp: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)},
		{Timestamp: time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)},
		{Timestamp: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		{Timestamp: time.Date(2026, 1, 3, 9, 0, 0, 0, time.UTC)},
	}

	missing := filterMissingDays(specs, map[string]int{
		"2026-01-01": 2,
		"2026-01-03": 1,
	})

	if len(missing) != 2 {
		t.Fatalf("missing commits = %d, want 2", len(missing))
	}
	for _, spec := range missing {
		if got := spec.Timestamp.Format("2006-01-02"); got != "2026-01-02" {
			t.Fatalf("unexpected missing day %s", got)
		}
	}
}

func TestFillAddsOnlyMissingActiveDays(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.json")

	writeFillTestConfig(t, configPath, repo, "2026-01-01", "2026-01-01")
	captureStdout(t, func() {
		if err := run([]string{"generate", "--config", configPath}); err != nil {
			t.Fatal(err)
		}
	})

	writeFillTestConfig(t, configPath, repo, "2026-01-03", "2026-01-03")
	captureStdout(t, func() {
		if err := run([]string{"generate", "--config", configPath}); err != nil {
			t.Fatal(err)
		}
	})

	output := captureStdout(t, func() {
		if err := run([]string{
			"fill",
			"--config", configPath,
			"--from", "2026-01-01",
			"--to", "2026-01-03",
		}); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "missing: 1") {
		t.Fatalf("expected one missing commit, got:\n%s", output)
	}
	if !strings.Contains(output, "Created 1 missing commits") {
		t.Fatalf("expected one created commit, got:\n%s", output)
	}

	history := gitLog(t, repo, "--format=%ad", "--date=short")
	counts := make(map[string]int)
	for _, day := range strings.Fields(history) {
		counts[day]++
	}
	for _, day := range []string{"2026-01-01", "2026-01-02", "2026-01-03"} {
		if counts[day] != 1 {
			t.Fatalf("commits on %s = %d, want 1; history:\n%s", day, counts[day], history)
		}
	}
}

func TestFillAfterLastRejectsFrom(t *testing.T) {
	configHome := t.TempDir()
	setTestConfigHome(t, configHome)

	err := run([]string{"fill", "--after-last", "--from", "2026-01-01"})
	if err == nil || !strings.Contains(err.Error(), "--after-last cannot be combined with --from") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFillTestConfig(t *testing.T, path, repo, start, end string) {
	t.Helper()
	content := fmt.Sprintf(`{
  "seed": 42,
  "repository": {
    "path": %q,
    "init": true,
    "branch": "main",
    "remote": "",
    "push": false,
    "allow_dirty": false
  },
  "identity": {"name": "Synthetic Test Bot", "email": "synthetic-test@example.invalid"},
  "range": {"start": %q, "end": %q, "timezone": "+00:00"},
  "volume": {
    "min_commits_per_day": 1,
    "max_commits_per_day": 1,
    "active_day_probability": 1,
    "weekend_multiplier": 1
  },
  "time": {"start": "09:00", "end": "17:00"},
  "content": {
    "activity_file": "activity.log",
    "line_template": "{date} {time} synthetic event #{sequence}",
    "message_templates": ["Commit {date}"]
  }
}`, repo, start, end)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitLog(t *testing.T, repo string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", repo, "log"}, args...)
	cmd := exec.Command("git", commandArgs...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %s", strings.TrimSpace(string(bytes)))
	}
	return string(bytes)
}

func setTestConfigHome(t *testing.T, configHome string) string {
	t.Helper()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", configHome)
		return configHome
	default:
		t.Setenv("XDG_CONFIG_HOME", configHome)
		return configHome
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
