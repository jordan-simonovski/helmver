# GitHub Actions

## PR check

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

      - name: Install helmver
        run: go install github.com/jordan-simonovski/helmver@latest

      - name: Check chart versions
        run: helmver check --require-changeset --dir charts/
```

Drop `--require-changeset` if your team bumps versions directly in PRs instead of using changeset files.

## Apply on merge

### Option A: Default token

Simplest setup. Works if the default branch is **not** protected with "Require a pull request before merging."

The built-in `GITHUB_TOKEN` can push commits but cannot trigger further workflow runs and cannot bypass PR-required branch protection.

```yaml
name: Apply changesets
on:
  push:
    branches: [main]
    paths: ['.helmver/**']

permissions:
  contents: write

jobs:
  apply:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install helmver
        run: go install github.com/jordan-simonovski/helmver@latest

      - name: Apply pending changesets
        run: helmver apply --dir charts/

      - name: Commit and push
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git commit -m "chore: apply helmver changesets [skip ci]" || exit 0
          git push
```

`[skip ci]` in the commit message prevents GitHub Actions from triggering another workflow run on the version bump commit.

### Option B: GitHub App token

Required when branch protection rules block the default token from pushing to main. Preferred over deploy keys because commits are attributed to the app (cleaner audit trail) and permissions are scoped per-repository.

**Setup:**

1. Create a GitHub App (or use an existing one) with **Contents: Read & write** permission.
2. Install the app on the repository.
3. Store the App ID and private key as repository secrets (`APP_ID`, `APP_PRIVATE_KEY`).
4. Add a branch protection bypass rule for the app under **Settings > Branches > Branch protection rules > Allow specified actors to bypass required pull requests**.

**Workflow:**

```yaml
name: Apply changesets
on:
  push:
    branches: [main]
    paths: ['.helmver/**']

jobs:
  apply:
    runs-on: ubuntu-latest
    steps:
      - name: Generate token
        id: token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ steps.token.outputs.token }}

      - name: Install helmver
        run: go install github.com/jordan-simonovski/helmver@latest

      - name: Apply pending changesets
        run: helmver apply --dir charts/

      - name: Commit and push
        run: |
          git config user.name "helmver[bot]"
          git config user.email "helmver[bot]@users.noreply.github.com"
          git add -A
          git commit -m "chore: apply helmver changesets [skip ci]" || exit 0
          git push
```

Passing `token` to `actions/checkout` configures the git remote to authenticate with that token on push. `[skip ci]` is still needed -- GitHub App tokens can trigger workflows by default.

### Option C: Deploy key

Alternative to a GitHub App. Simpler to set up but less flexible (one key per repo, no audit attribution).

**Setup:**

1. Generate a key pair: `ssh-keygen -t ed25519 -C "helmver-deploy-key" -f helmver_deploy`
2. Add the **public** key as a deploy key on the repository (**Settings > Deploy keys**) with **Allow write access** checked.
3. Add the **private** key as a repository secret (`DEPLOY_KEY`).

**Workflow change** (replace the checkout step):

```yaml
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ssh-key: ${{ secrets.DEPLOY_KEY }}
```

Deploy keys bypass "Restrict who can push to matching branches" but **not** "Require a pull request before merging" unless the key is added to the bypass list.

## Which option to pick

| Scenario | Option |
|----------|--------|
| No branch protection | A (default token) |
| Branch protection, need audit trail | B (GitHub App) |
| Branch protection, minimal setup | C (deploy key) |
