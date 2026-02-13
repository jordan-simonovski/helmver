# helmver

Version management and changelog generation for Helm charts. Detects stale chart versions via git history and provides an interactive TUI for bumping versions and writing changelog entries.

Designed for both single-chart repos and monorepos with many charts under one roof.

## The problem

Helm charts have a `version` field in `Chart.yaml` that must be bumped on every change. In practice this gets forgotten, leading to charts deployed with the wrong version, missing changelog entries, and CI pipelines that can't tell what changed.

helmver solves this by:

- **Detecting stale versions** -- comparing git history to find charts with changes since their last version bump.
- **Interactive changesets** -- a TUI that walks you through selecting charts, choosing bump types, and writing changelog messages.
- **Updating files in place** -- bumps `version` in `Chart.yaml` (preserving comments and formatting) and prepends entries to `CHANGELOG.md`.
- **CI integration** -- `helmver check` exits non-zero when charts are stale, suitable for CI gates and git hooks.

## Installation

### Homebrew

```bash
brew tap jordan-simonovski/helmver
brew install helmver
```

### Go

```bash
go install github.com/jordan-simonovski/helmver@latest
```

### Docker

```bash
docker pull ghcr.io/jordan-simonovski/helmver:latest

# Run against current directory
docker run --rm -v "$(pwd):/work" -w /work ghcr.io/jordan-simonovski/helmver check
```

### Binary

Download the latest binary from [GitHub Releases](https://github.com/jordan-simonovski/helmver/releases) and place it in your `PATH`.

Available for Linux, macOS, and Windows on both amd64 and arm64.

## Usage

### Check for stale chart versions

```bash
# Single chart in current directory
helmver check

# Monorepo with charts in a subdirectory
helmver check --dir charts/
```

`helmver check` scans for `Chart.yaml` files, compares each chart directory against git history, and reports which charts have changes since their last version bump.

- Exit code `0` -- all charts are up to date.
- Exit code `1` -- one or more charts need a version bump.

Example output:

```
2 chart(s) need a version bump:

  api                            1.2.3  (/path/to/charts/api)
  worker                         0.5.0  (/path/to/charts/worker)
```

### Create a changeset (interactive)

```bash
# Single chart
helmver changeset

# Monorepo
helmver changeset --dir charts/
```

Launches an interactive TUI that:

1. **Shows all discovered charts** -- changed charts highlighted in blue, unchanged in grey. Both are selectable.
2. **Asks for bump type** -- major, minor, or patch, with a version preview for each option.
3. **Asks for a changelog message** -- multiline text editor. Press `ctrl+d` to submit.
4. **Shows a summary** -- review all changes before applying. Press `y` to apply, `n` to abort.

When confirmed, helmver:

- Updates the `version` field in each selected `Chart.yaml` (preserving YAML comments and structure).
- Prepends a new entry to `CHANGELOG.md` in each chart's directory:

```markdown
## 1.3.0 (2026-02-13)

Added pagination support to the list endpoint.
```

Works outside git repos too -- all charts are shown as "unchanged" but you can still create changesets for them.

### TUI controls

| Key           | Action                          |
|---------------|---------------------------------|
| `j` / `k`    | Navigate up/down                |
| `space`       | Toggle selection                |
| `a`           | Select/deselect all             |
| `enter`       | Confirm selection               |
| `ctrl+d`      | Submit changelog message        |
| `y` / `n`     | Confirm or abort in summary     |
| `q` / `ctrl+c`| Quit                           |

## How staleness detection works

helmver uses git to determine if a chart needs a version bump:

1. Find the most recent commit that modified the `version:` line in `Chart.yaml` (using `git log -G`).
2. Check if any files in the chart directory have changed since that commit (committed or uncommitted).
3. If there are changes, the chart is **stale**.

Edge cases:
- Chart never committed: treated as stale.
- Not a git repo: `helmver check` errors; `helmver changeset` works (all charts shown as unchanged).

## Git hook

Use `helmver check` as a pre-commit hook to prevent commits when chart versions are stale.

### Quick setup

```bash
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### With a custom chart directory

```bash
HELMVER_DIR=charts git commit -m "my changes"
```

Or edit `.git/hooks/pre-commit` and set `HELMVER_DIR`:

```bash
HELMVER_DIR="charts" helmver check
```

### With pre-commit framework

Add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: helmver-check
        name: helmver check
        entry: helmver check --dir charts/
        language: system
        pass_filenames: false
        always_run: true
```

### Skipping the hook

```bash
git commit --no-verify -m "wip: skip version check"
```

## CI usage

### GitHub Actions

```yaml
- name: Check chart versions
  run: helmver check --dir charts/
```

Or using the Docker image:

```yaml
- name: Check chart versions
  run: |
    docker run --rm -v "${{ github.workspace }}:/work" -w /work \
      ghcr.io/jordan-simonovski/helmver:latest check --dir charts/
```

`helmver check` returns exit code 1 when charts are stale, failing the CI step.

## Monorepo support

helmver recursively discovers all `Chart.yaml` and `Chart.yml` files under the given directory. Each chart directory is evaluated independently for staleness.

```
my-repo/
  charts/
    api/
      Chart.yaml      # version: 1.2.3
      values.yaml
      templates/
    web/
      Chart.yaml      # version: 2.0.0
      values.yaml
    worker/
      Chart.yaml      # version: 0.5.0
      values.yaml
```

```bash
# Check all charts
helmver check --dir charts/

# Create changesets for stale charts
helmver changeset --dir charts/
```

Each chart gets its own `CHANGELOG.md` in its directory.

## Subchart support

Subcharts (charts nested under `charts/` within a parent chart) are discovered and versioned independently:

```
parent-app/
  Chart.yaml          # version: 3.0.0
  charts/
    redis/
      Chart.yaml      # version: 0.1.0
```

Running `helmver check` in `parent-app/` will check both the parent chart and the redis subchart separately.

## YAML preservation

helmver uses `gopkg.in/yaml.v3` with AST-level node manipulation. When bumping a version, only the `version` field value is changed. Comments, key ordering, formatting, and all other fields are preserved exactly as they were.

## Development

```bash
# Build
make build

# Run all tests (unit + e2e + acceptance)
make test

# Run only unit tests
make test-unit

# Run only e2e tests
make test-e2e

# Run acceptance tests against fixture charts
make test-acceptance

# Lint
make lint

# Format
make fmt

# Cross-compile for all platforms
make cross-compile

# Show all targets
make help
```

## License

MIT. See [LICENSE](LICENSE).
