# helmver GitHub Actions

Consumable GitHub Actions for checking Helm chart versions and commenting on pull requests — similar to [changesets/action](https://github.com/changesets/action).

## Actions

| Action | Description |
|--------|-------------|
| [`setup`](setup/action.yml) | Install helmver from GitHub Releases |
| [`check`](check/action.yml) | Fail CI when chart versions are stale |
| [`pr-status`](pr-status/action.yml) | Generate a markdown comment body for PR status |
| [`pr-comment`](pr-comment/action.yml) | Create or update a PR comment |

## Quick start

### PR check (CI gate)

Fails the workflow when charts have changes without a version bump or `.helmver/` changeset.

```yaml
name: Chart version check
on: pull_request

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: jordan-simonovski/helmver/action/check@v1
        with:
          dir: charts/
          require-changeset: true
```

### PR comment (changeset bot)

Posts a comment on every PR showing which charts need bumps and whether changesets are present.

```yaml
name: Comment helmver status on PRs
on:
  pull_request_target:

permissions: {}

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  pr-status:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      comment-body: ${{ steps.status.outputs.comment-body }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - id: status
        uses: jordan-simonovski/helmver/action/pr-status@v1
        with:
          dir: charts/
          require-changeset: true

  pr-comment:
    needs: pr-status
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
    steps:
      - uses: jordan-simonovski/helmver/action/pr-comment@v1
        with:
          body: ${{ needs.pr-status.outputs.comment-body }}
```

> **Note:** `pr-status` uses `pull_request_target` by default so it works for PRs from forks. It only checks out and reads code — it does not execute untrusted code from the fork. If you prefer `pull_request` (same-repo PRs only), add:
>
> ```yaml
> if: github.event.pull_request.head.repo.full_name == github.repository
> ```

## Permissions

| Job | Required permissions |
|-----|---------------------|
| `check` | `contents: read` (default) |
| `pr-status` | `contents: read` |
| `pr-comment` | `pull-requests: write` |

For `pr-comment`, the repository's default `GITHUB_TOKEN` must be allowed to write pull request comments. If your org uses restricted workflow permissions, set **Read and write** under **Settings → Actions → General → Workflow permissions**, or declare permissions explicitly in the workflow as shown above.

No additional secrets are required — the default `GITHUB_TOKEN` is sufficient.

## Inputs

### `check`

| Input | Default | Description |
|-------|---------|-------------|
| `dir` | `.` | Directory to scan for `Chart.yaml` files |
| `base` | _(auto)_ | Base git ref to compare against |
| `require-changeset` | `true` | Accept `.helmver/` changesets as valid bump intent |
| `exclude` | | Comma-separated glob patterns to skip |
| `version` | `latest` | helmver release to install |

### `pr-status`

Same inputs as `check`, plus outputs:

| Output | Description |
|--------|-------------|
| `comment-body` | Markdown for use with `pr-comment` |
| `commit-sha` | PR head commit SHA |

### `pr-comment`

| Input | Default | Description |
|-------|---------|-------------|
| `body` | _(required)_ | Comment body |
| `github-token` | `${{ github.token }}` | Token with `pull-requests: write` |
| `update-id` | `helmver-action-pr-comment` | Marker for idempotent comment updates |

## Version pinning

Pin to a major version tag for automatic updates within that major release:

```yaml
uses: jordan-simonovski/helmver/action/check@v1
```

Or pin to an exact release:

```yaml
uses: jordan-simonovski/helmver/action/check@v1.0.0
```

The `version` input controls which helmver **binary** is installed, independent of the action tag.

## See also

- [CI setup guide](../docs/ci-github-actions.md) — apply-on-merge workflows
- [helmver README](../README.md) — CLI usage and changeset file format
