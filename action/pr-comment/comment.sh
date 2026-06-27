#!/usr/bin/env bash
# Create or update a PR comment with a hidden marker for idempotent updates.
set -euo pipefail

event="${GITHUB_EVENT_PATH:?GITHUB_EVENT_PATH is required}"
token="${INPUT_GITHUB_TOKEN:?github-token is required}"
body="${INPUT_BODY:?body is required}"
update_id="${INPUT_UPDATE_ID:-helmver-action-pr-comment}"

owner=$(jq -r '.pull_request.base.repo.owner.login // empty' "$event")
repo=$(jq -r '.pull_request.base.repo.name // empty' "$event")
issue_number=$(jq -r '.pull_request.number // empty' "$event")

if [[ -z "$owner" || -z "$repo" || -z "$issue_number" ]]; then
  echo "This action requires a pull_request or pull_request_target event." >&2
  exit 1
fi

marker="<!-- ${update_id} -->"
comment_body="${marker}

${body}"

api() {
  curl -fsSL \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "$@"
}

api_json() {
  curl -fsSL \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/vnd.github+json" \
    -H "Content-Type: application/json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "$@"
}

existing_id=$(api "https://api.github.com/repos/${owner}/${repo}/issues/${issue_number}/comments?per_page=100" \
  | jq --arg marker "$marker" '[.[] | select(.body | contains($marker))][0].id // empty')

payload=$(jq -n --arg body "$comment_body" '{body: $body}')

if [[ -n "$existing_id" ]]; then
  echo "Updating comment ${existing_id}..."
  api_json -X PATCH -d "$payload" \
    "https://api.github.com/repos/${owner}/${repo}/issues/comments/${existing_id}" >/dev/null
  echo "comment-id=${existing_id}" >> "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
else
  echo "Creating comment..."
  new_id=$(api_json -X POST -d "$payload" \
    "https://api.github.com/repos/${owner}/${repo}/issues/${issue_number}/comments" \
    | jq -r '.id')
  echo "comment-id=${new_id}" >> "$GITHUB_OUTPUT"
fi

echo "Done."
