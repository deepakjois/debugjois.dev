#!/usr/bin/env bash
set -euo pipefail

MODULES=("site" "backend/api" "infra")

# Snapshot go.mod files before upgrade
for module in "${MODULES[@]}"; do
  label="${module//\//-}"
  cp "${module}/go.mod" "/tmp/${label}-go.mod.before"
done

# Upgrade each module
for module in "${MODULES[@]}"; do
  echo "Upgrading ${module}..."
  (cd "$module" && go get -u -t ./... && go mod tidy)
done

# Sync workspace
go work sync

# Check for changes
if git diff --quiet; then
  echo "has_updates=false" >> "$GITHUB_OUTPUT"
  echo "No dependency updates available."
  exit 0
fi

echo "has_updates=true" >> "$GITHUB_OUTPUT"

# Build PR body with per-module diffs
EOF_MARKER=$(dd if=/dev/urandom bs=15 count=1 status=none | base64)
{
  echo "body<<$EOF_MARKER"
  echo "## Go Dependency Upgrades"
  echo ""
  for module in "${MODULES[@]}"; do
    label="${module//\//-}"
    MODULE_DIFF=$(diff "/tmp/${label}-go.mod.before" "${module}/go.mod" || true)
    if [ -n "$MODULE_DIFF" ]; then
      echo "### \`${module}\`"
      echo ""
      echo '```diff'
      echo "$MODULE_DIFF"
      echo '```'
      echo ""
    fi
  done
  echo "$EOF_MARKER"
} >> "$GITHUB_OUTPUT"
