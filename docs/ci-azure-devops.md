# Azure DevOps Pipelines

## PR check

```yaml
trigger: none
pr:
  branches:
    include: ['*']

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0

  - script: helmver check --require-changeset --dir charts/
    displayName: Check chart versions
```

Drop `--require-changeset` if your team bumps versions directly in PRs instead of using changeset files.

## Apply on merge

### Option A: Build Service identity

Uses the built-in `System.AccessToken` with elevated permissions. No external secrets needed.

**Setup:**

1. Go to **Project Settings > Repositories** and select your repository.
2. Under **Security**, find the **\<Project Name\> Build Service (\<Org Name\>)** identity.
3. Set **Contribute** to **Allow**.
4. If the branch has policies, set **Bypass policies when pushing** to **Allow**.

**Pipeline:**

```yaml
trigger:
  branches:
    include: [main]
  paths:
    include: ['.helmver/*']

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0
    persistCredentials: true

  - script: helmver apply --dir charts/
    displayName: Apply changesets

  - script: |
      git config user.name "helmver-bot"
      git config user.email "helmver-bot@noreply"
      git add -A
      git commit -m "chore: apply helmver changesets ***NO_CI***" || exit 0
      git push origin HEAD:$(Build.SourceBranchName)
    displayName: Commit and push
```

`persistCredentials: true` on the checkout step configures git to authenticate with `System.AccessToken` on push.

Azure DevOps uses `***NO_CI***` or `[skip ci]` in the commit message to prevent triggering another pipeline run.

### Option B: Personal access token (PAT)

Use when you cannot grant the Build Service identity sufficient permissions, or when you need the commit attributed to a specific user/service account.

**Setup:**

1. Create a PAT under **User settings > Personal access tokens** with **Code: Read & write** scope.
2. Store the PAT as a pipeline secret variable named `HELMVER_PAT`.

**Pipeline change** (replace the push step):

```yaml
  - script: |
      git config user.name "helmver-bot"
      git config user.email "helmver-bot@noreply"
      git remote set-url origin "https://pat:$(HELMVER_PAT)@dev.azure.com/$(System.CollectionUri)/$(System.TeamProject)/_git/$(Build.Repository.Name)"
      git add -A
      git commit -m "chore: apply helmver changesets [skip ci]" || exit 0
      git push origin HEAD:$(Build.SourceBranchName)
    displayName: Commit and push
```

### Branch policies

If the default branch has policies:

- **Build Service identity**: needs **Bypass policies when pushing** set to **Allow** in the repository security settings.
- **PAT**: the user whose PAT is used must have the **Bypass policies when pushing** permission.

## Which option to pick

| Scenario | Option |
|----------|--------|
| Can grant Build Service permissions | A (Build Service identity) |
| Need specific user attribution or can't modify Build Service perms | B (PAT) |
