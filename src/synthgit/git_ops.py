from __future__ import annotations

import os
import subprocess
from pathlib import Path

from .config import AppConfig
from .generator import CommitSpec


def ensure_repository(config: AppConfig) -> None:
    repo = config.repository.path
    if not repo.exists():
        if not config.repository.init:
            raise GitError(f"Target repository does not exist: {repo}")
        repo.mkdir(parents=True)
        _git(repo, "init", "-b", config.repository.branch)
    elif not (repo / ".git").exists():
        if not config.repository.init:
            raise GitError(f"Target path exists but is not a Git repository: {repo}")
        _git(repo, "init", "-b", config.repository.branch)

    if not config.repository.allow_dirty and _is_dirty(repo):
        raise GitError("Target repository has uncommitted changes. Set allow_dirty = true to continue.")

    if _has_commits(repo):
        if _branch_exists(repo, config.repository.branch):
            _git(repo, "checkout", config.repository.branch)
        else:
            _git(repo, "checkout", "-b", config.repository.branch)
    else:
        _git(repo, "checkout", "-B", config.repository.branch)

    if config.repository.remote:
        remotes = _git(repo, "remote", capture=True).stdout.splitlines()
        if "origin" not in remotes:
            _git(repo, "remote", "add", "origin", config.repository.remote)


def apply_commit(config: AppConfig, spec: CommitSpec) -> None:
    repo = config.repository.path
    activity_path = repo / config.content.activity_file
    activity_path.parent.mkdir(parents=True, exist_ok=True)
    with activity_path.open("a", encoding="utf-8") as handle:
        handle.write(spec.line + "\n")

    relative_file = str(activity_path.relative_to(repo))
    _git(repo, "add", relative_file)

    env = os.environ.copy()
    env.update(
        {
            "GIT_AUTHOR_NAME": config.identity.name,
            "GIT_AUTHOR_EMAIL": config.identity.email,
            "GIT_COMMITTER_NAME": config.identity.name,
            "GIT_COMMITTER_EMAIL": config.identity.email,
            "GIT_AUTHOR_DATE": spec.git_date,
            "GIT_COMMITTER_DATE": spec.git_date,
        }
    )
    _git(repo, "commit", "-m", spec.message, env=env)


def push(config: AppConfig) -> None:
    if not config.repository.remote:
        raise GitError("Cannot push: repository.remote is empty")
    _git(config.repository.path, "push", "-u", "origin", config.repository.branch)


def _has_commits(repo: Path) -> bool:
    result = subprocess.run(
        ["git", "rev-parse", "--verify", "HEAD"],
        cwd=repo,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    return result.returncode == 0


def _branch_exists(repo: Path, branch: str) -> bool:
    result = subprocess.run(
        ["git", "rev-parse", "--verify", branch],
        cwd=repo,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    return result.returncode == 0


def _is_dirty(repo: Path) -> bool:
    return bool(_git(repo, "status", "--porcelain", capture=True).stdout.strip())


def _git(repo: Path, *args: str, env: dict[str, str] | None = None, capture: bool = False) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            ["git", *args],
            cwd=repo,
            env=env,
            check=True,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
    except subprocess.CalledProcessError as exc:
        detail = exc.stderr.strip() if exc.stderr else str(exc)
        raise GitError(f"git {' '.join(args)} failed: {detail}") from exc


class GitError(RuntimeError):
    pass
