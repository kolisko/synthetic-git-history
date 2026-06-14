# Synthetic Git History

Synthetic Git History is a small Go CLI for generating configurable synthetic Git commit histories in test repositories. It is intended for testing GitHub, Git hosting, analytics, CI, reporting, and import integrations that need realistic commit timelines.

Runtime requirements:

- `git`
- no Python, Node, pip, npm, or external runtime

The default workflow is local and explicit:

- commit volume is configured per day,
- commit dates can be backdated for test ranges,
- output is deterministic when `seed` is set,
- push is disabled unless `--push` and config push settings are both enabled,
- dirty repositories are rejected unless `allow_dirty` is `true`.

Use this only in repositories that are clearly synthetic or dedicated to testing.

## Quick Start

From a release binary:

```bash
synthgit init-config
synthgit plan
synthgit generate
```

From source:

```bash
go build -o synthgit ./cmd/synthgit
./synthgit init-config
./synthgit plan
./synthgit generate
```

## Commands

```bash
synthgit plan
```

Prints the generated schedule from `~/.synthgit.config.json` without touching a repository.

```bash
synthgit generate
```

Creates commits locally according to `~/.synthgit.config.json`.

```bash
synthgit plan --config ./config.example.json
```

Uses a custom config path.

```bash
synthgit generate --config config.example.json --dry-run
```

Shows what would be committed without changing files.

```bash
synthgit generate --config config.example.json --push
```

Pushes after generation only when `repository.push` is `true` and `repository.remote` is configured.

```bash
synthgit init-config
```

Writes a starter JSON config to `~/.synthgit.config.json`, prints the created config, and explains where to edit it.

```bash
synthgit init-config --output ./synthgit.config.json
```

Writes the starter config to a custom path instead.

## Configuration

See [config.example.json](config.example.json).

Important fields:

- `repository.path`: target repository path.
- `repository.init`: initialize the repository when it does not exist.
- `repository.branch`: branch to create/use.
- `repository.push`: allows pushing when the CLI also receives `--push`.
- `range.start` / `range.end`: inclusive synthetic commit date range.
- `volume.min_commits_per_day` / `volume.max_commits_per_day`: daily volume bounds.
- `volume.active_day_probability`: probability that a day receives commits.
- `volume.weekend_multiplier`: lowers or raises weekend activity.
- `identity`: author/committer identity used for generated commits.
- `content.message_templates`: commit message templates.

Template variables:

- `{date}`: date in `YYYY-MM-DD`.
- `{time}`: local time in `HH:MM:SS`.
- `{index}`: commit index within the day.
- `{daily_total}`: total commits for the day.
- `{sequence}`: global commit sequence.

## Releases

The release workflow builds binaries for:

- Linux `amd64` and `arm64`
- macOS `amd64` and `arm64`
- Windows `amd64`

Create a release by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions will attach platform binaries and `.sha256` checksum files to the release.

## Notes

GitHub contribution graphs and analytics can depend on repository visibility, default branch, email ownership, branch membership, and GitHub-side processing. This tool creates ordinary Git commits with controlled timestamps; it does not attempt to bypass or manipulate GitHub behavior.

## Tests

```bash
go test ./...
```
