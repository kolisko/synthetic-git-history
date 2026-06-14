# Synthetic Git History

Synthetic Git History is a small, dependency-free CLI for generating configurable Git commit histories in test repositories. It is intended for testing GitHub, Git hosting, analytics, CI, reporting, and import integrations that need realistic commit timelines.

The default workflow is local and explicit:

- commit volume is configured per day,
- commit dates can be backdated for test ranges,
- output is deterministic when `seed` is set,
- push is disabled unless `--push` and config push settings are both enabled,
- dirty repositories are rejected unless `allow_dirty = true`.

Use this only in repositories that are clearly synthetic or dedicated to testing.

## Quick Start

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install -e .

synthgit plan --config config.example.toml
synthgit generate --config config.example.toml
```

Without installation:

```bash
PYTHONPATH=src python3 -m synthgit plan --config config.example.toml
PYTHONPATH=src python3 -m synthgit generate --config config.example.toml
```

## Commands

```bash
synthgit plan --config config.example.toml
```

Prints the generated schedule without touching a repository.

```bash
synthgit generate --config config.example.toml
```

Creates commits locally according to the config.

```bash
synthgit generate --config config.example.toml --dry-run
```

Shows what would be committed without changing files.

```bash
synthgit generate --config config.example.toml --push
```

Pushes after generation only when `[repository].push = true` and `[repository].remote` is configured.

## Configuration

See [config.example.toml](config.example.toml).

Important fields:

- `[repository].path`: target repository path.
- `[repository].init`: initialize the repository when it does not exist.
- `[repository].branch`: branch to create/use.
- `[repository].push`: allows pushing when the CLI also receives `--push`.
- `[range].start` / `[range].end`: inclusive synthetic commit date range.
- `[volume].min_commits_per_day` / `[volume].max_commits_per_day`: daily volume bounds.
- `[volume].active_day_probability`: probability that a day receives commits.
- `[volume].weekend_multiplier`: lowers or raises weekend activity.
- `[identity]`: author/committer identity used for generated commits.
- `[content].message_templates`: commit message templates.

Template variables:

- `{date}`: date in `YYYY-MM-DD`.
- `{time}`: local time in `HH:MM:SS`.
- `{index}`: commit index within the day.
- `{daily_total}`: total commits for the day.
- `{sequence}`: global commit sequence.

## Notes

GitHub contribution graphs and analytics can depend on repository visibility, default branch, email ownership, branch membership, and GitHub-side processing. This tool creates ordinary Git commits with controlled timestamps; it does not attempt to bypass or manipulate GitHub behavior.

## Tests

```bash
PYTHONPATH=src python3 -m unittest discover -s tests
```
