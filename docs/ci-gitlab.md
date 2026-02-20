# GitLab CI

## MR check

```yaml
check-charts:
  stage: test
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
    - helmver check --require-changeset --dir charts/
```

Drop `--require-changeset` if your team bumps versions directly in MRs instead of using changeset files.

## Apply on merge

GitLab CI runners authenticate with `CI_JOB_TOKEN`, which is **read-only** for git pushes by default. You need a project or group access token with write access.

### Option A: Project access token

**Setup:**

1. Go to **Settings > Access tokens**.
2. Create a project access token with the `write_repository` scope and `Maintainer` role.
3. Store the token as a CI/CD variable named `CI_PUSH_TOKEN` (**masked**; uncheck "protected" if you want it available on all branches).
4. If the default branch is protected, go to **Settings > Repository > Protected branches** and add the token's associated bot user to **Allowed to push and merge**.

**Pipeline:**

```yaml
apply-changesets:
  stage: deploy
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      changes:
        - .helmver/**
  script:
    - git config user.name "helmver-bot"
    - git config user.email "helmver-bot@noreply"
    - git remote set-url origin "https://oauth2:${CI_PUSH_TOKEN}@${CI_SERVER_HOST}/${CI_PROJECT_PATH}.git"
    - git checkout ${CI_COMMIT_BRANCH}
    - helmver apply --dir charts/
    - git add -A
    - git commit -m "chore: apply helmver changesets [ci skip]" || true
    - git push origin ${CI_COMMIT_BRANCH}
```

`[ci skip]` in the commit message prevents GitLab from triggering a new pipeline.

The `git checkout` is required because GitLab CI checks out a **detached HEAD** by default. Without it, `git push` has no branch to target.

### Option B: Deploy key (SSH)

Alternative if you prefer SSH over HTTPS tokens.

**Setup:**

1. Generate a key pair: `ssh-keygen -t ed25519 -C "helmver-deploy-key" -f helmver_deploy`
2. Add the **public** key under **Settings > Repository > Deploy keys** with **write access** enabled.
3. Add the **private** key as a **file-type** CI/CD variable named `DEPLOY_SSH_KEY`.

**Pipeline:**

```yaml
apply-changesets:
  stage: deploy
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      changes:
        - .helmver/**
  before_script:
    - eval $(ssh-agent -s)
    - echo "${DEPLOY_SSH_KEY}" | ssh-add -
    - mkdir -p ~/.ssh && ssh-keyscan ${CI_SERVER_HOST} >> ~/.ssh/known_hosts
    - git remote set-url origin "git@${CI_SERVER_HOST}:${CI_PROJECT_PATH}.git"
  script:
    - git config user.name "helmver-bot"
    - git config user.email "helmver-bot@noreply"
    - git checkout ${CI_COMMIT_BRANCH}
    - helmver apply --dir charts/
    - git add -A
    - git commit -m "chore: apply helmver changesets [ci skip]" || true
    - git push origin ${CI_COMMIT_BRANCH}
```

### Protected branches

If the default branch has **"Allowed to push and merge"** restrictions:

- **Project access token**: add the token's bot user to the allowed list.
- **Deploy key**: deploy keys with write access can push to protected branches by default in GitLab.

## Which option to pick

| Scenario | Option |
|----------|--------|
| Simple, HTTPS | A (project access token) |
| Prefer SSH, key-based auth | B (deploy key) |
