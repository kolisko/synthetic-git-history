from __future__ import annotations

import json
import re
import tomllib
from dataclasses import dataclass
from datetime import date, time
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class RepositoryConfig:
    path: Path
    init: bool
    branch: str
    remote: str
    push: bool
    allow_dirty: bool


@dataclass(frozen=True)
class IdentityConfig:
    name: str
    email: str


@dataclass(frozen=True)
class RangeConfig:
    start: date
    end: date
    timezone: str
    skip_weekends: bool


@dataclass(frozen=True)
class VolumeConfig:
    min_commits_per_day: int
    max_commits_per_day: int
    active_day_probability: float
    weekend_multiplier: float


@dataclass(frozen=True)
class TimeConfig:
    start: time
    end: time


@dataclass(frozen=True)
class ContentConfig:
    activity_file: str
    line_template: str
    message_templates: tuple[str, ...]


@dataclass(frozen=True)
class AppConfig:
    repository: RepositoryConfig
    identity: IdentityConfig
    date_range: RangeConfig
    volume: VolumeConfig
    time: TimeConfig
    content: ContentConfig
    seed: int | None = None


def load_config(path: Path) -> AppConfig:
    raw = _load_raw(path)
    base_dir = path.resolve().parent

    repository = raw.get("repository", {})
    identity = raw.get("identity", {})
    date_range = raw.get("range", {})
    volume = raw.get("volume", {})
    time_config = raw.get("time", {})
    content = raw.get("content", {})

    repo_path = Path(_required(repository, "path"))
    if not repo_path.is_absolute():
        repo_path = base_dir / repo_path

    config = AppConfig(
        repository=RepositoryConfig(
            path=repo_path,
            init=bool(repository.get("init", False)),
            branch=str(repository.get("branch", "main")),
            remote=str(repository.get("remote", "")),
            push=bool(repository.get("push", False)),
            allow_dirty=bool(repository.get("allow_dirty", False)),
        ),
        identity=IdentityConfig(
            name=str(_required(identity, "name")),
            email=str(_required(identity, "email")),
        ),
        date_range=RangeConfig(
            start=_parse_date(_required(date_range, "start"), "range.start"),
            end=_parse_date(_required(date_range, "end"), "range.end"),
            timezone=str(date_range.get("timezone", "+00:00")),
            skip_weekends=bool(date_range.get("skip_weekends", False)),
        ),
        volume=VolumeConfig(
            min_commits_per_day=int(volume.get("min_commits_per_day", 1)),
            max_commits_per_day=int(volume.get("max_commits_per_day", 1)),
            active_day_probability=float(volume.get("active_day_probability", 1.0)),
            weekend_multiplier=float(volume.get("weekend_multiplier", 1.0)),
        ),
        time=TimeConfig(
            start=_parse_time(time_config.get("start", "09:00"), "time.start"),
            end=_parse_time(time_config.get("end", "17:00"), "time.end"),
        ),
        content=ContentConfig(
            activity_file=str(content.get("activity_file", "activity.log")),
            line_template=str(content.get("line_template", "{date} {time} synthetic event #{sequence}")),
            message_templates=tuple(content.get("message_templates", ("Synthetic commit {date} #{index}",))),
        ),
        seed=int(raw["seed"]) if "seed" in raw and raw["seed"] is not None else None,
    )

    _validate(config)
    return config


def _load_raw(path: Path) -> dict[str, Any]:
    if not path.exists():
        raise ConfigError(f"Config file does not exist: {path}")
    if path.suffix.lower() == ".json":
        with path.open("r", encoding="utf-8") as handle:
            return json.load(handle)
    with path.open("rb") as handle:
        return tomllib.load(handle)


def _required(section: dict[str, Any], key: str) -> Any:
    if key not in section:
        raise ConfigError(f"Missing required config key: {key}")
    return section[key]


def _parse_date(value: Any, field: str) -> date:
    if isinstance(value, date):
        return value
    try:
        return date.fromisoformat(str(value))
    except ValueError as exc:
        raise ConfigError(f"{field} must be an ISO date like 2010-01-31") from exc


def _parse_time(value: Any, field: str) -> time:
    if isinstance(value, time):
        return value
    try:
        return time.fromisoformat(str(value))
    except ValueError as exc:
        raise ConfigError(f"{field} must be a time like 09:30") from exc


def _validate(config: AppConfig) -> None:
    if config.date_range.end < config.date_range.start:
        raise ConfigError("range.end must be on or after range.start")
    if not re.fullmatch(r"[+-]\d{2}:\d{2}", config.date_range.timezone):
        raise ConfigError("range.timezone must use +HH:MM or -HH:MM format")
    if config.volume.min_commits_per_day < 0:
        raise ConfigError("volume.min_commits_per_day must be >= 0")
    if config.volume.max_commits_per_day < config.volume.min_commits_per_day:
        raise ConfigError("volume.max_commits_per_day must be >= min_commits_per_day")
    if not 0 <= config.volume.active_day_probability <= 1:
        raise ConfigError("volume.active_day_probability must be between 0 and 1")
    if config.volume.weekend_multiplier < 0:
        raise ConfigError("volume.weekend_multiplier must be >= 0")
    if config.time.end <= config.time.start:
        raise ConfigError("time.end must be after time.start")
    if not config.content.message_templates:
        raise ConfigError("content.message_templates must contain at least one template")
    if Path(config.content.activity_file).is_absolute() or ".." in Path(config.content.activity_file).parts:
        raise ConfigError("content.activity_file must be a relative path inside the target repository")


class ConfigError(ValueError):
    pass

