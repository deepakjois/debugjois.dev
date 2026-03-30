#!/usr/bin/env bash
set -euo pipefail

# Creates or updates a GitHub issue with major version upgrade results.
# Expects UPGRADE_BODY env var to contain the issue body markdown.

title="go: major version upgrades available"
existing=$(gh issue list --state open --search "in:title $title" --json number --jq '.[0].number // empty')

if [ -n "$existing" ]; then
  gh issue edit "$existing" --body "$UPGRADE_BODY"
  echo "Updated issue #$existing"
else
  gh issue create --title "$title" --body "$UPGRADE_BODY" --label dependencies
  echo "Created new issue"
fi
