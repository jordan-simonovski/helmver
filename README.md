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

# Accept pending changeset files as valid intent to bump
helmver check --require-changeset
```

`helmver check` scans for `Chart.yaml` files, compares each chart directory against git history, and reports which charts have changes since their last version bump.

- Exit code `0` -- all charts are up to date (or have pending changesets when `--require-changeset` is set).
- Exit code `1` -- one or more charts need a version bump (and have no pending changeset).

The `--require-changeset` flag tells helmver to look in `.helmver/` for pending changeset files. A stale chart that has a corresponding changeset is not flagged -- the changeset is a valid intent to bump that will be applied later via `helmver apply`.

When run outside a git repository, `helmver check` lists all discovered charts with their current versions and exits 0, since staleness cannot be determined without git history.

Example output:

```
2 chart(s) need a version bump:

  api                            1.2.3  (/path/to/charts/api)
  worker                         0.5.0  (/path/to/charts/worker)
```

### Create a changeset (interactive)

```bash
# Apply immediately (default)
helmver changeset

# Write a .helmver/ changeset file instead
helmver changeset --write

# Monorepo
helmver changeset --write --dir charts/
```

Launches an interactive TUI that:

1. **Shows all discovered charts** -- changed charts highlighted in blue, unchanged in grey. Both are selectable.
2. **Asks for bump type** -- major, minor, or patch, with a version preview for each option.
3. **Asks for a changelog message** -- multiline text editor. Press `ctrl+d` to submit.
4. **Shows a summary** -- review all changes before applying. Press `y` to apply, `n` to abort.

**Without `--write`** (default), helmver applies immediately:

- Updates the `version` field in each selected `Chart.yaml` (preserving YAML comments and structure).
- Prepends a new entry to `CHANGELOG.md` in each chart's directory.

**With `--write`**, helmver creates changeset files in `.helmver/` instead of modifying Chart.yaml directly. These files accumulate across PRs and are consumed later via `helmver apply`. See [Changeset files](#changeset-files) below.

Works outside git repos too -- all charts are shown as "unchanged" but you can still create changesets for them.

### Apply pending changesets

```bash
helmver apply
helmver apply --dir charts/
```

Reads all `.helmver/*.md` changeset files, computes the aggregate version bump per chart (highest bump wins when multiple changesets target the same chart), applies the bumps to `Chart.yaml`, writes changelogs, and deletes the consumed changeset files.

Example output:

```
  api: 1.2.3 -> 1.3.0 (minor)
  worker: 0.5.0 -> 0.5.1 (patch)

2 chart(s) updated, 3 changeset(s) consumed
```

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

## Changeset files

Changeset files let teams decouple "deciding to bump" from "applying the bump." A developer records intent in their PR; a CI pipeline applies all pending changesets on merge to the main branch.

### File format

Changeset files live in `.helmver/` and use markdown with YAML front matter:

```markdown
---
"api": minor
---

Added horizontal pod autoscaling support
```

The front matter maps chart names to bump types (`patch`, `minor`, `major`). The body is the changelog message. One file per chart is typical, but a single file can reference multiple charts for a shared change:

```markdown
---
"api": minor
"worker": patch
---

Migrated shared config loading to use structured types
```

### Creating changeset files

```bash
helmver changeset --write
```

The TUI walks you through chart selection, bump type, and message -- then writes `.helmver/<random-id>.md` instead of touching Chart.yaml.

### Aggregation rules

When multiple changeset files target the same chart, `helmver apply` picks the **highest** bump type:

| Changesets for `api`        | Result |
|-----------------------------|--------|
| patch + patch               | patch  |
| patch + minor               | minor  |
| patch + minor + major       | major  |

All changelog messages are concatenated in the order they're discovered.

### CI workflow with changeset files

The recommended flow for teams using changesets:

```
developer creates PR
  └─ runs: helmver changeset --write
  └─ commits .helmver/*.md file(s)

CI on PR (check)
  └─ runs: helmver check --require-changeset
  └─ passes if changed charts have either a version bump or a pending changeset

CI on merge to main (apply)
  └─ runs: helmver apply
  └─ commits updated Chart.yaml + CHANGELOG.md
  └─ pushes with [skip ci] to avoid retriggering
```

The apply step needs write access to push the version bump commit back to the branch. Each CI provider handles this differently (tokens, deploy keys, branch protection bypass). See the setup guides:

- [GitHub Actions](docs/ci-github-actions.md) -- default token, GitHub App, or deploy key
- [GitLab CI](docs/ci-gitlab.md) -- project access token or deploy key
- [Bitbucket Pipelines](docs/ci-bitbucket.md) -- built-in SSH key or app password
- [Azure DevOps](docs/ci-azure-devops.md) -- Build Service identity or PAT

**Quick start** (GitHub Actions, unprotected branch):

```yaml
# .github/workflows/check.yml
name: Chart version check
on: pull_request
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: go install github.com/jordan-simonovski/helmver@latest
      - run: helmver check --require-changeset --dir charts/
```

```yaml
# .github/workflows/apply.yml
name: Apply changesets
on:
  push:
    branches: [main]
    paths: ['.helmver/**']
permissions: { contents: write }
jobs:
  apply:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: go install github.com/jordan-simonovski/helmver@latest
      - run: helmver apply --dir charts/
      - run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git commit -m "chore: apply helmver changesets [skip ci]" || exit 0
          git push
```

#### Skip-CI commit message reference

| Provider | Token | Notes |
|----------|-------|-------|
| GitHub Actions | `[skip ci]` or `[ci skip]` | Anywhere in commit message |
| GitLab CI | `[ci skip]` or `[skip ci]` | Anywhere in commit message |
| Bitbucket Pipelines | `[skip ci]` or `[ci skip]` | Anywhere in commit message |
| Azure DevOps | `***NO_CI***` or `[skip ci]` | `[skip ci]` supported since 2022 |

## How staleness detection works

helmver uses git to determine if a chart needs a version bump:

1. Resolve the base ref (CI env vars > remote HEAD > `origin/main`; override with `--base`).
2. Run `git diff --name-only <base>...HEAD -- <chartDir>` to find changed files.
3. Compare the `version` field in `Chart.yaml` at the base ref vs HEAD.
4. If files changed but the version did not, the chart is **stale**.

Base ref resolution order:

| Priority | Source | Example |
|----------|--------|---------|
| 1 | `--base` flag | `--base origin/develop` |
| 2 | `GITHUB_BASE_REF` | GitHub Actions PR events |
| 3 | `BITBUCKET_PR_DESTINATION_BRANCH` | Bitbucket Pipelines |
| 4 | `CI_MERGE_REQUEST_TARGET_BRANCH_NAME` | GitLab merge requests |
| 5 | `CI_DEFAULT_BRANCH` | GitLab fallback |
| 6 | `git symbolic-ref refs/remotes/origin/HEAD` | Remote default branch |
| 7 | `origin/main` | Final fallback |

Edge cases:

- New chart (not in base ref): never stale.
- Not a git repo: `helmver check` lists charts and exits 0; `helmver changeset` works with all charts shown as unchanged.

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

Add `helmver check` to your CI pipeline to gate on stale chart versions. Use `--require-changeset` if your team uses changeset files, or drop it for direct version bumps.

```bash
# Direct bumps -- fail if charts have changes without a version bump
helmver check --dir charts/

# Changeset workflow -- also accept pending .helmver/ files
helmver check --require-changeset --dir charts/
```

For the full apply-on-merge workflow (apply + commit + push back), see [CI workflow with changeset files](#ci-workflow-with-changeset-files) and the provider-specific setup guides:

- [GitHub Actions](docs/ci-github-actions.md)
- [GitLab CI](docs/ci-gitlab.md)
- [Bitbucket Pipelines](docs/ci-bitbucket.md)
- [Azure DevOps](docs/ci-azure-devops.md)

### Docker

Any of the CI examples can use the Docker image instead of installing the binary:

```bash
docker run --rm -v "$(pwd):/work" -w /work \
  ghcr.io/jordan-simonovski/helmver:latest check --dir charts/
```

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
