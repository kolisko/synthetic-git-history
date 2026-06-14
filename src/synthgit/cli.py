from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .config import ConfigError, load_config
from .generator import CommitSpec, build_schedule
from .git_ops import GitError, apply_commit, ensure_repository, push


def main(argv: list[str] | None = None) -> int:
    parser = _build_parser()
    args = parser.parse_args(argv)

    try:
        config = load_config(args.config)
        schedule = build_schedule(config)

        if args.command == "plan":
            _print_plan(schedule)
            return 0

        if args.dry_run:
            _print_plan(schedule)
            print(f"\nDry run: no files changed. Planned commits: {len(schedule)}")
            return 0

        ensure_repository(config)
        for spec in schedule:
            apply_commit(config, spec)

        print(f"Created {len(schedule)} commits in {config.repository.path}")

        if args.push:
            if not config.repository.push:
                raise ConfigError("Refusing to push because repository.push is false in config")
            push(config)
            print(f"Pushed branch {config.repository.branch} to origin")
        elif config.repository.push:
            print("Config allows push, but CLI --push was not provided. Nothing was pushed.")

        return 0
    except (ConfigError, GitError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="synthgit",
        description="Generate configurable synthetic Git commit histories for test repositories.",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    plan = subparsers.add_parser("plan", help="Print the generated commit schedule without changing files.")
    plan.add_argument("--config", required=True, type=Path)

    generate = subparsers.add_parser("generate", help="Generate commits in the configured repository.")
    generate.add_argument("--config", required=True, type=Path)
    generate.add_argument("--dry-run", action="store_true", help="Print the schedule without changing files.")
    generate.add_argument("--push", action="store_true", help="Push after generation when config also allows it.")

    return parser


def _print_plan(schedule: list[CommitSpec]) -> None:
    if not schedule:
        print("No commits planned.")
        return
    for spec in schedule:
        print(f"{spec.git_date} | {spec.message}")
    print(f"\nPlanned commits: {len(schedule)}")


if __name__ == "__main__":
    raise SystemExit(main())

