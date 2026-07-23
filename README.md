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

Prints the generated schedule from the default config file without touching a repository.

```bash
synthgit generate
```

Creates commits locally according to the default config file and prints progress as commits are created.

```bash
synthgit fill
```

Inspects the configured repository and fills empty active days from its earliest commit through today. Existing commits are preserved. Days intentionally left inactive by `active_day_probability`, `skip_weekends`, and `weekend_multiplier` remain empty.

```bash
synthgit fill --from 2025-01-01 --to 2025-12-31
```

Fills empty active days only inside an explicit inclusive range. `--to today` is also accepted.

```bash
synthgit fill --after-last --to 2026-12-31
```

Continues after the latest existing commit day without inspecting earlier gaps. When the repository has no commits, generation starts at `range.start` from the config.

```bash
synthgit fill --dry-run
```

Prints the existing history summary and missing commit schedule without changing the repository. Add `--push` to push after filling; the config must also set `repository.push` to `true`.

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

Writes a starter JSON config to the default user config directory, prints the created config, and explains where to edit it.

The default config path follows the XDG-style config directory on macOS and Linux:

- Linux: `$XDG_CONFIG_HOME/synthgit/config.json` or `~/.config/synthgit/config.json`
- macOS: `$XDG_CONFIG_HOME/synthgit/config.json` or `~/.config/synthgit/config.json`
- Windows: `%AppData%\synthgit\config.json`

```bash
synthgit init-config --output ./synthgit.config.json
```

Writes the starter config to a custom path instead.

## macOS Gatekeeper

The macOS release binaries are unsigned and not notarized. If you download a macOS binary in a browser, macOS may attach a quarantine attribute and the terminal can show only:

```text
zsh: killed
```

Remove the quarantine attribute once after download:

```bash
xattr -d com.apple.quarantine ./synthgit-v0.1.3-darwin-arm64
chmod +x ./synthgit-v0.1.3-darwin-arm64
./synthgit-v0.1.3-darwin-arm64 help
```

Use `synthgit-v0.1.3-darwin-amd64` instead on Intel Macs.

## Configuration

See [config.example.json](config.example.json).

Important fields:

- `repository.path`: target repository path; relative paths are resolved from the directory where you run `synthgit`.
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

## Filling Existing Repositories

`fill` reads author dates from every commit reachable from the configured branch. A day is considered present when at least one existing commit has that author date. It then builds the configured synthetic schedule for the selected range and creates commits only for scheduled days that are completely empty.

The operation does not rewrite or force-push history. When an older gap is filled, the new commit is appended at the branch tip with an older author and committer date. Its displayed date belongs to the gap, but its parent is still the previous branch tip. Inserting commits into the middle of Git ancestry would require rewriting the repository and is intentionally outside the scope of `fill`.

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
