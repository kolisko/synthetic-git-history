from __future__ import annotations

import random
from dataclasses import dataclass
from datetime import date, datetime, time, timedelta

from .config import AppConfig


@dataclass(frozen=True)
class CommitSpec:
    timestamp: datetime
    timezone: str
    message: str
    line: str
    sequence: int
    day_index: int
    daily_total: int

    @property
    def git_date(self) -> str:
        return f"{self.timestamp.strftime('%Y-%m-%dT%H:%M:%S')} {self.timezone}"


def build_schedule(config: AppConfig) -> list[CommitSpec]:
    rng = random.Random(config.seed)
    specs: list[CommitSpec] = []
    sequence = 1

    for current_day in _days(config.date_range.start, config.date_range.end):
        if config.date_range.skip_weekends and current_day.weekday() >= 5:
            continue

        probability = config.volume.active_day_probability
        if current_day.weekday() >= 5:
            probability *= config.volume.weekend_multiplier
        if rng.random() > probability:
            continue

        daily_total = rng.randint(
            config.volume.min_commits_per_day,
            config.volume.max_commits_per_day,
        )
        timestamps = sorted(_random_times_for_day(rng, current_day, config.time.start, config.time.end, daily_total))

        for day_index, timestamp in enumerate(timestamps, start=1):
            context = _template_context(timestamp, sequence, day_index, daily_total)
            message_template = rng.choice(config.content.message_templates)
            specs.append(
                CommitSpec(
                    timestamp=timestamp,
                    timezone=config.date_range.timezone,
                    message=message_template.format(**context),
                    line=config.content.line_template.format(**context),
                    sequence=sequence,
                    day_index=day_index,
                    daily_total=daily_total,
                )
            )
            sequence += 1

    return specs


def _days(start: date, end: date):
    current = start
    while current <= end:
        yield current
        current += timedelta(days=1)


def _random_times_for_day(rng: random.Random, day: date, start: time, end: time, count: int) -> list[datetime]:
    start_dt = datetime.combine(day, start)
    end_dt = datetime.combine(day, end)
    span_seconds = int((end_dt - start_dt).total_seconds())
    return [start_dt + timedelta(seconds=rng.randint(0, span_seconds)) for _ in range(count)]


def _template_context(timestamp: datetime, sequence: int, index: int, daily_total: int) -> dict[str, str | int]:
    return {
        "date": timestamp.strftime("%Y-%m-%d"),
        "time": timestamp.strftime("%H:%M:%S"),
        "index": index,
        "daily_total": daily_total,
        "sequence": sequence,
    }

