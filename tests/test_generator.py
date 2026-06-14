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


class GeneratorTests(unittest.TestCase):
    def test_schedule_is_deterministic_with_seed(self):
        config = _config(seed=123)

        first = build_schedule(config)
        second = build_schedule(config)

        self.assertEqual(first, second)
        self.assertEqual(len(first), 6)

    def test_weekends_can_be_skipped(self):
        config = _config(seed=1, skip_weekends=True)

        schedule = build_schedule(config)

        self.assertTrue(schedule)
        self.assertTrue(all(spec.timestamp.weekday() < 5 for spec in schedule))


def _config(seed=None, skip_weekends=False):
    return AppConfig(
        repository=RepositoryConfig(
            path=Path("/tmp/repo"),
            init=True,
            branch="main",
            remote="",
            push=False,
            allow_dirty=False,
        ),
        identity=IdentityConfig(name="Bot", email="bot@example.invalid"),
        date_range=RangeConfig(
            start=date(2020, 1, 1),
            end=date(2020, 1, 3),
            timezone="+00:00",
            skip_weekends=skip_weekends,
        ),
        volume=VolumeConfig(
            min_commits_per_day=2,
            max_commits_per_day=2,
            active_day_probability=1,
            weekend_multiplier=1,
        ),
        time=TimeConfig(start=time(9, 0), end=time(10, 0)),
        content=ContentConfig(
            activity_file="activity.log",
            line_template="{date} {time} #{sequence}",
            message_templates=("Commit {sequence}",),
        ),
        seed=seed,
    )


if __name__ == "__main__":
    unittest.main()

