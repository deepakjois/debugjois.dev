#!/usr/bin/env bash
set -euo pipefail

MODULES=("site" "backend/api" "infra")

# Capture upgrade reports and apply upgrades
for module in "${MODULES[@]}"; do
  label="${module//\//-}"
  echo "Checking upgrades in ${module}..."
  (cd "$module" && GOWORK=off go-mod-upgrade --list) > "/tmp/${label}-upgrades.txt" 2>/dev/null || true
  echo "Upgrading ${module}..."
  (cd "$module" && GOWORK=off go-mod-upgrade --force && go mod tidy)
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

# Clean terminal control characters from reports
clean_output() {
  sed -e 's/\x1b\[[0-9;]*m//g' -e 's/.*\r//' "$1" | sed '/^[[:space:]]*$/d'
}

# Build PR body with per-module upgrade reports
EOF_MARKER=$(dd if=/dev/urandom bs=15 count=1 status=none | base64)
{
  echo "body<<$EOF_MARKER"
  echo "## Go Dependency Upgrades"
  echo ""
  for module in "${MODULES[@]}"; do
    label="${module//\//-}"
    REPORT=$(clean_output "/tmp/${label}-upgrades.txt" | grep -v 'All modules are up to date')
    if [ -n "$REPORT" ]; then
      echo "### \`${module}\`"
      echo ""
      echo '```'
      echo "$REPORT"
      echo '```'
      echo ""
    fi
  done
  echo "$EOF_MARKER"
} >> "$GITHUB_OUTPUT"
