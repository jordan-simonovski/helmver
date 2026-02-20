# Bitbucket Pipelines

## PR check

```yaml
pipelines:
  pull-requests:
    '**':
      - step:
          name: Check chart versions
          script:
            - helmver check --require-changeset --dir charts/
```

Drop `--require-changeset` if your team bumps versions directly in PRs instead of using changeset files.

## Apply on merge

### Option A: Built-in SSH key

Simplest setup. Bitbucket Pipelines provides a managed SSH key that can push to the repository.

**Setup:**

1. Go to **Repository settings > Pipelines > SSH keys**.
2. Click **Generate keys** (or add your own key pair). Bitbucket automatically registers the public key as a repository access key.
3. Under **Known hosts**, add `bitbucket.org` (click **Fetch** to auto-populate the fingerprint).

**Pipeline:**

```yaml
pipelines:
  branches:
    main:
      - step:
          name: Apply changesets
          script:
            - helmver apply --dir charts/
            - git config user.name "helmver-bot"
            - git config user.email "helmver-bot@noreply"
            - git add -A
            - git commit -m "chore: apply helmver changesets [skip ci]" || true
            - git push
```

`[skip ci]` in the commit message prevents Bitbucket from triggering a new pipeline.

### Option B: App password (for restricted branches)

If **branch permissions** restrict who can push to main, the built-in SSH key's associated user may not have access. Use an app password instead.

**Setup:**

1. Create a Bitbucket **app password** under **Personal settings > App passwords** with `Repositories: Write` scope.
2. Store the username and app password as **repository variables** (`BB_USER`, `BB_APP_PASSWORD`). Mark `BB_APP_PASSWORD` as **secured**.

**Pipeline:**

```yaml
pipelines:
  branches:
    main:
      - step:
          name: Apply changesets
          script:
            - helmver apply --dir charts/
            - git config user.name "helmver-bot"
            - git config user.email "helmver-bot@noreply"
            - git remote set-url origin "https://${BB_USER}:${BB_APP_PASSWORD}@bitbucket.org/${BITBUCKET_REPO_FULL_NAME}.git"
            - git add -A
            - git commit -m "chore: apply helmver changesets [skip ci]" || true
            - git push
```

### Branch permissions

If the repository has branch permissions on main:

- **SSH key**: Go to **Repository settings > Branch permissions** and allow the SSH key's associated user to push.
- **App password**: The user whose app password is used must have push access in the branch permissions.

## Which option to pick

| Scenario | Option |
|----------|--------|
| No branch restrictions | A (built-in SSH key) |
| Branch permissions restrict pushes | B (app password) |
