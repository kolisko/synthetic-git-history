package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kolisko/synthetic-git-history/internal/config"
	"github.com/kolisko/synthetic-git-history/internal/schedule"
)

func TestCreatesCommitWithConfiguredDate(t *testing.T) {
	requireGit(t)

	cfg := testConfig(t.TempDir())
	specs := schedule.Build(cfg)

	if err := EnsureRepository(cfg); err != nil {
		t.Fatal(err)
	}
	if err := ApplyCommit(cfg, specs[0]); err != nil {
		t.Fatal(err)
	}

	out := run(t, cfg.Repository.Path, "git", "log", "-1", "--format=%aI|%s")
	if !strings.HasPrefix(strings.TrimSpace(out), "2010-01-01T") {
		t.Fatalf("unexpected commit date: %s", out)
	}
	if !strings.Contains(out, "|Commit 1") {
		t.Fatalf("unexpected commit subject: %s", out)
	}
}

func TestCreatesMissingBranchInExistingRepository(t *testing.T) {
	requireGit(t)

	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "init")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "add", "README.md")
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Base",
		"GIT_AUTHOR_EMAIL=base@example.invalid",
		"GIT_COMMITTER_NAME=Base",
		"GIT_COMMITTER_EMAIL=base@example.invalid",
	)
	runWithEnv(t, repo, env, "git", "commit", "-m", "Base")

	cfg := testConfigWithRepo(repo)
	cfg.Repository.Branch = "synthetic-history"

	if err := EnsureRepository(cfg); err != nil {
		t.Fatal(err)
	}

	branch := strings.TrimSpace(run(t, repo, "git", "branch", "--show-current"))
	if branch != "synthetic-history" {
		t.Fatalf("branch = %q, want synthetic-history", branch)
	}
}

func testConfig(base string) config.Config {
	return testConfigWithRepo(filepath.Join(base, "repo"))
}

func testConfigWithRepo(repo string) config.Config {
	seed := int64(99)
	cfg, err := config.Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil {
		panic(err)
	}
	cfg.Seed = &seed
	cfg.Repository.Path = repo
	cfg.Repository.Init = true
	cfg.Repository.Remote = ""
	cfg.Repository.Push = false
	cfg.Range.StartRaw = "2010-01-01"
	cfg.Range.EndRaw = "2010-01-01"
	cfg.Range.End = cfg.Range.Start
	cfg.Volume.MinCommitsPerDay = 1
	cfg.Volume.MaxCommitsPerDay = 1
	cfg.Volume.ActiveDayProb = 1
	cfg.Volume.WeekendMultiplier = 1
	cfg.Content.MessageTemplates = []string{"Commit {sequence}"}
	return cfg
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is required")
	}
}

func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	return runWithEnv(t, dir, os.Environ(), name, args...)
}

func runWithEnv(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %s", name, strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out)
}
