package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kolisko/synthetic-git-history/internal/config"
	"github.com/kolisko/synthetic-git-history/internal/schedule"
)

func EnsureRepository(cfg config.Config) error {
	repo := cfg.Repository.Path
	info, err := os.Stat(repo)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if !cfg.Repository.Init {
			return fmt.Errorf("target repository does not exist: %s", repo)
		}
		if err := os.MkdirAll(repo, 0755); err != nil {
			return err
		}
		if err := git(repo, nil, "init"); err != nil {
			return err
		}
	} else if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", repo)
	} else if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		if !cfg.Repository.Init {
			return fmt.Errorf("target path exists but is not a Git repository: %s", repo)
		}
		if err := git(repo, nil, "init"); err != nil {
			return err
		}
	}

	if !cfg.Repository.AllowDirty {
		dirty, err := isDirty(repo)
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("target repository has uncommitted changes; set allow_dirty to true to continue")
		}
	}

	hasHead, err := hasCommits(repo)
	if err != nil {
		return err
	}
	if hasHead {
		exists, err := branchExists(repo, cfg.Repository.Branch)
		if err != nil {
			return err
		}
		if exists {
			if err := git(repo, nil, "checkout", cfg.Repository.Branch); err != nil {
				return err
			}
		} else if err := git(repo, nil, "checkout", "-b", cfg.Repository.Branch); err != nil {
			return err
		}
	} else if err := git(repo, nil, "checkout", "-B", cfg.Repository.Branch); err != nil {
		return err
	}

	if cfg.Repository.Remote != "" {
		remotes, err := gitOutput(repo, nil, "remote")
		if err != nil {
			return err
		}
		if !containsLine(remotes, "origin") {
			if err := git(repo, nil, "remote", "add", "origin", cfg.Repository.Remote); err != nil {
				return err
			}
		}
	}

	return nil
}

func ApplyCommit(cfg config.Config, spec schedule.CommitSpec) error {
	repo := cfg.Repository.Path
	activityPath := filepath.Join(repo, cfg.Content.ActivityFile)
	if err := os.MkdirAll(filepath.Dir(activityPath), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(activityPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(file, spec.Line); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := git(repo, nil, "add", cfg.Content.ActivityFile); err != nil {
		return err
	}

	env := []string{
		"GIT_AUTHOR_NAME=" + cfg.Identity.Name,
		"GIT_AUTHOR_EMAIL=" + cfg.Identity.Email,
		"GIT_COMMITTER_NAME=" + cfg.Identity.Name,
		"GIT_COMMITTER_EMAIL=" + cfg.Identity.Email,
		"GIT_AUTHOR_DATE=" + spec.GitDate(),
		"GIT_COMMITTER_DATE=" + spec.GitDate(),
	}
	return git(repo, env, "commit", "-m", spec.Message)
}

func Push(cfg config.Config) error {
	if cfg.Repository.Remote == "" {
		return fmt.Errorf("cannot push: repository.remote is empty")
	}
	return git(cfg.Repository.Path, nil, "push", "-u", "origin", cfg.Repository.Branch)
}

func hasCommits(repo string) (bool, error) {
	err := git(repo, nil, "rev-parse", "--verify", "HEAD")
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "needed a single revision") || strings.Contains(err.Error(), "unknown revision") {
		return false, nil
	}
	return false, nil
}

func branchExists(repo, branch string) (bool, error) {
	err := git(repo, nil, "rev-parse", "--verify", branch)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func isDirty(repo string) (bool, error) {
	out, err := gitOutput(repo, nil, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func git(repo string, env []string, args ...string) error {
	_, err := runGit(repo, env, args...)
	return err
}

func gitOutput(repo string, env []string, args ...string) (string, error) {
	return runGit(repo, env, args...)
}

func runGit(repo string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func containsLine(text, want string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}
