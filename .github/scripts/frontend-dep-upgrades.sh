#!/usr/bin/env bash
set -euo pipefail

# Capture human-readable upgrade report (shows package old → new)
REPORT=$(npx npm-check-updates 2>/dev/null || true)

# Check if any upgrades are available (returns {} if none)
UPGRADES=$(npx npm-check-updates --jsonUpgraded 2>/dev/null || echo '{}')
if [ "$UPGRADES" = '{}' ] || [ -z "$UPGRADES" ]; then
  echo "has_updates=false" >> "$GITHUB_OUTPUT"
  echo "No dependency updates available."
  exit 0
fi

# Apply upgrades to package.json and install
npx npm-check-updates -u
npm install

echo "has_updates=true" >> "$GITHUB_OUTPUT"

# Build PR body with the human-readable report
EOF_MARKER=$(dd if=/dev/urandom bs=15 count=1 status=none | base64)
{
  echo "body<<$EOF_MARKER"
  echo "## Frontend Dependency Upgrades"
  echo ""
  echo "The following packages were upgraded:"
  echo ""
  echo '```'
  echo "$REPORT"
  echo '```'
  echo "$EOF_MARKER"
} >> "$GITHUB_OUTPUT"
