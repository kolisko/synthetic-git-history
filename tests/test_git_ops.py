import os
import shutil
import subprocess
import tempfile
import unittest
from datetime import date, time
from pathlib import Path

from synthgit.config import (
    AppConfig,
    ContentConfig,
    IdentityConfig,
    RangeConfig,
    RepositoryConfig,
    TimeConfig,
    VolumeConfig,
)
from synthgit.generator import build_schedule
from synthgit.git_ops import apply_commit, ensure_repository


@unittest.skipUnless(shutil.which("git"), "git executable is required")
class GitOpsTests(unittest.TestCase):
    def test_creates_commit_with_configured_date(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            config = AppConfig(
                repository=RepositoryConfig(
                    path=repo,
                    init=True,
                    branch="main",
                    remote="",
                    push=False,
                    allow_dirty=False,
                ),
                identity=IdentityConfig(name="Bot", email="bot@example.invalid"),
                date_range=RangeConfig(
                    start=date(2010, 1, 1),
                    end=date(2010, 1, 1),
                    timezone="+00:00",
                    skip_weekends=False,
                ),
                volume=VolumeConfig(
                    min_commits_per_day=1,
                    max_commits_per_day=1,
                    active_day_probability=1,
                    weekend_multiplier=1,
                ),
                time=TimeConfig(start=time(9, 0), end=time(9, 1)),
                content=ContentConfig(
                    activity_file="activity.log",
                    line_template="{date} {time} #{sequence}",
                    message_templates=("Commit {sequence}",),
                ),
                seed=99,
            )

            schedule = build_schedule(config)
            ensure_repository(config)
            apply_commit(config, schedule[0])

            log = subprocess.run(
                ["git", "log", "-1", "--format=%aI|%s"],
                cwd=repo,
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()

            self.assertTrue(log.startswith("2010-01-01T09:"))
            self.assertTrue(log.endswith("|Commit 1"))

    def test_creates_missing_branch_in_existing_repository(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            repo.mkdir()
            subprocess.run(["git", "init", "-b", "main"], cwd=repo, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            (repo / "README.md").write_text("base\n", encoding="utf-8")
            subprocess.run(["git", "add", "README.md"], cwd=repo, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            subprocess.run(
                ["git", "commit", "-m", "Base"],
                cwd=repo,
                check=True,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                env={
                    **os.environ,
                    "GIT_AUTHOR_NAME": "Base",
                    "GIT_AUTHOR_EMAIL": "base@example.invalid",
                    "GIT_COMMITTER_NAME": "Base",
                    "GIT_COMMITTER_EMAIL": "base@example.invalid",
                },
            )
            config = _app_config(repo, branch="synthetic-history")

            ensure_repository(config)

            branch = subprocess.run(
                ["git", "branch", "--show-current"],
                cwd=repo,
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
            self.assertEqual(branch, "synthetic-history")


def _app_config(repo: Path, branch: str = "main") -> AppConfig:
    return AppConfig(
        repository=RepositoryConfig(
            path=repo,
            init=True,
            branch=branch,
            remote="",
            push=False,
            allow_dirty=False,
        ),
        identity=IdentityConfig(name="Bot", email="bot@example.invalid"),
        date_range=RangeConfig(
            start=date(2010, 1, 1),
            end=date(2010, 1, 1),
            timezone="+00:00",
            skip_weekends=False,
        ),
        volume=VolumeConfig(
            min_commits_per_day=1,
            max_commits_per_day=1,
            active_day_probability=1,
            weekend_multiplier=1,
        ),
        time=TimeConfig(start=time(9, 0), end=time(9, 1)),
        content=ContentConfig(
            activity_file="activity.log",
            line_template="{date} {time} #{sequence}",
            message_templates=("Commit {sequence}",),
        ),
        seed=99,
    )


if __name__ == "__main__":
    unittest.main()
