#!/usr/bin/env bash
set -euo pipefail

# Sync daily notes from Google Drive
./debugjois-site sync-notes-obsidian

# Commit any changes; capture output to detect whether a commit was made
commit_output=$(./debugjois-site commit-notes --skip-ci 2>&1)
echo "$commit_output"

if echo "$commit_output" | grep -q "No changes to commit."; then
  echo "No changes detected, skipping build and upload."
  exit 0
fi

git push
./debugjois-site build
./debugjois-site upload
