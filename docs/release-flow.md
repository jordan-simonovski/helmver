# Release Flow

This project uses Release Please + GoReleaser for semver releases, Docker image
publishing to GHCR, and vendored release artifacts.

## Pipeline Overview

1. `main` receives new commits.
2. `release-please.yml` opens/updates a release PR:
   - bumps version
   - updates `CHANGELOG.md`
3. Merge release PR:
   - creates GitHub release + `vX.Y.Z` tag
4. `release.yml` triggers on `v*` tags:
   - runs GoReleaser
   - publishes binaries/archives/checksums
   - publishes Docker image to GHCR
5. `release.yml` then runs a vendoring job:
   - downloads GoReleaser `dist/` artifact from the previous job
   - writes GHCR image refs
   - uploads everything as a reusable workflow artifact
6. `release.yml` updates the floating `v1` Git tag used by `uses: .../helmver/action/check@v1`.
7. `vendor-artifacts.yml` remains available as a manual fallback workflow.

> Important: to allow downstream tag-triggered workflows (`release.yml`) after
> Release Please creates a tag/release, configure repository secret
> `RELEASE_PLEASE_TOKEN` (PAT). Using only `GITHUB_TOKEN` may suppress those
> downstream workflow triggers.

## Files Involved

- `.github/workflows/release-please.yml`
- `.github/workflows/release.yml`
- `.github/workflows/vendor-artifacts.yml`
- `.goreleaser.yml`
- `release-please-config.json`
- `.release-please-manifest.json`
- `CHANGELOG.md`

## Required Repository Secrets

- `RELEASE_PLEASE_TOKEN` (recommended): personal access token used by
  `release-please.yml` so generated tag/release events can trigger downstream
  workflows reliably.

## Commit Convention and Version Bumps

Release Please infers semver bump type from Conventional Commits:

- `fix:` -> patch
- `feat:` -> minor
- `!` or `BREAKING CHANGE:` -> major

Examples:

- `fix(check): handle missing base ref`
- `feat(action): add PR comment bot`
- `feat!: change default changeset workflow`

CI validates Conventional Commit PR titles on pull requests when code files are
changed. This aligns with squash-merge workflows where the PR title becomes the
merge commit on `main`.

## Tagging Convention

This repository uses a single semver convention everywhere:

- Git tag: `vX.Y.Z` (for example `v0.6.0`)
- GitHub release: `vX.Y.Z`
- GHCR image tag: `vX.Y.Z`
- GoReleaser archive: `helmver_X.Y.Z_<os>_<arch>.tar.gz`

Release Please is configured with:

- `include-v-in-tag: true`
- `include-component-in-tag: false`

## First Release After Setup

Release Please is bootstrapped at `0.5.0` (the last manually cut release). After
merging a `feat:` or `fix:` commit to `main`:

1. Wait for `release-please.yml` to open a release PR.
2. Merge the release PR to cut `v0.6.0` (or `v0.5.1` for fixes only).
3. `release.yml` publishes binaries and the Docker image automatically.

## Publishing to GitHub Marketplace

The root `action.yml` is the Marketplace-listed action. Sub-actions under
`action/` can still be referenced by path (`jordan-simonovski/helmver/action/check@v1`)
but only the root action is listed on the Marketplace.

To publish after a release:

1. Open the release for tag `vX.Y.Z` on GitHub.
2. Check **Publish this Action to the GitHub Marketplace** (visible when `action.yml`
   at the repo root has `name`, `description`, and `branding`).

## Manual Vendoring

`vendor-artifacts.yml` supports manual dispatch for backfill/retry:

- Input `version` expects bare semver (for example `0.6.0`, without `v`).

## Published Artifacts

From GoReleaser:

- Multi-OS binaries and archives (linux/darwin/windows, amd64/arm64)
- `checksums.txt`
- Docker image:
  - `ghcr.io/<owner>/helmver:vX.Y.Z`
  - `ghcr.io/<owner>/helmver:latest`

From vendor-artifacts workflow:

- Uploaded workflow artifact named `helmver-vX.Y.Z`
- Contains downloaded release assets and `image-refs.txt`
