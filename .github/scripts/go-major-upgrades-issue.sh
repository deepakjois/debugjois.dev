#!/usr/bin/env bash
set -euo pipefail

# Keeps a single GitHub issue in sync with major version upgrade results:
# creates or updates it when upgrades exist, and closes it when none remain.
# Expects HAS_UPGRADES ("true"/"false") and UPGRADE_BODY (issue body markdown).

title="go: major version upgrades available"
# Title search is fuzzy, so match the exact title before touching any issue
existing=$(gh issue list --state open --search "in:title $title" --json number,title \
  --jq "map(select(.title == \"$title\")) | .[0].number // empty")

if [ "$HAS_UPGRADES" != "true" ]; then
  if [ -n "$existing" ]; then
    gh issue close "$existing" --comment "No Go major version upgrades remain."
    echo "Closed issue #$existing"
  fi
  exit 0
fi

if [ -n "$existing" ]; then
  gh issue edit "$existing" --body "$UPGRADE_BODY"
  echo "Updated issue #$existing"
else
  gh issue create --title "$title" --body "$UPGRADE_BODY" --label dependencies
  echo "Created new issue"
fi
