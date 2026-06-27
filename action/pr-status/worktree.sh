#!/usr/bin/env bash
# Prepare a detached worktree at the PR head ref for status checks.
# Reads pull_request context from GITHUB_EVENT_PATH.
set -euo pipefail

event="${GITHUB_EVENT_PATH:?GITHUB_EVENT_PATH is required}"
repo_root="$(git rev-parse --show-toplevel)"

pr_number=$(jq -r '.pull_request.number // empty' "$event")
head_ref=$(jq -r '.pull_request.head.ref // empty' "$event")
head_repo=$(jq -r '.pull_request.head.repo.clone_url // empty' "$event")
head_sha=$(jq -r '.pull_request.head.sha // empty' "$event")
base_ref=$(jq -r '.pull_request.base.ref // empty' "$event")

if [[ -z "$pr_number" || -z "$head_ref" || -z "$head_repo" || -z "$head_sha" ]]; then
  echo "This action requires a pull_request or pull_request_target event." >&2
  exit 1
fi

suffix="${pr_number}-$(uuidgen | tr '[:upper:]' '[:lower:]')"
base_local="refs/helmver-action/base/${suffix}"
head_local="refs/helmver-action/head/${suffix}"
worktree_dir="$(mktemp -d -t helmver-action-pr-status-XXXXXX)"

is_shallow=false
if git -C "$repo_root" rev-parse --is-shallow-repository | grep -q true; then
  is_shallow=true
fi

fetch_ref() {
  local remote=$1 refspec=$2
  if $is_shallow; then
    git -C "$repo_root" fetch --no-tags --depth=1 "$remote" "$refspec"
  else
    git -C "$repo_root" fetch --no-tags "$remote" "$refspec"
  fi
}

deepen=50
while true; do
  fetch_ref origin "refs/heads/${base_ref}:${base_local}"
  fetch_ref "$head_repo" "refs/heads/${head_ref}:${head_local}"

  git -C "$repo_root" worktree add --detach "$worktree_dir" "$head_local"

  if git -C "$worktree_dir" merge-base "$base_local" HEAD >/dev/null 2>&1; then
    break
  fi

  git -C "$repo_root" worktree remove --force "$worktree_dir" 2>/dev/null || true
  git -C "$repo_root" update-ref -d "$head_local" 2>/dev/null || true
  git -C "$repo_root" update-ref -d "$base_local" 2>/dev/null || true

  if ! $is_shallow; then
    echo "Failed to find merge base between PR head and base ref." >&2
    exit 1
  fi

  git -C "$repo_root" fetch --no-tags --deepen="$deepen" origin "refs/heads/${base_ref}:${base_local}"
  git -C "$repo_root" fetch --no-tags --deepen="$deepen" "$head_repo" "refs/heads/${head_ref}:${head_local}"
done

echo "worktree=${worktree_dir}" >> "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
echo "commit=${head_sha}" >> "$GITHUB_OUTPUT"
