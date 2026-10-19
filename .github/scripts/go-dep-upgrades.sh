#!/usr/bin/env bash
set -euo pipefail

MODULES=("site")

# Major version bumps change import paths and APIs, so they are left to the
# separate go-major-upgrades workflow, which files an issue instead.
upgrade_module() {
  (cd "$1" && git checkout -- go.mod go.sum && GOWORK=off go-mod-upgrade --force --no-major && go mod tidy)
}

# Retry a command up to 3 times. The Go module proxy occasionally resets
# connections mid-request (HTTP/2 INTERNAL_ERROR), which is transient.
retry() {
  local attempt
  for attempt in 1 2 3; do
    "$@" && return 0
    echo "Attempt ${attempt} of $* failed." >&2
    [ "$attempt" -lt 3 ] && sleep $((attempt * 15))
  done
  return 1
}

# Capture upgrade reports and apply upgrades
for module in "${MODULES[@]}"; do
  label="${module//\//-}"
  echo "Checking upgrades in ${module}..."
  (cd "$module" && GOWORK=off go-mod-upgrade --list --no-major) > "/tmp/${label}-upgrades.txt" 2>/dev/null || true
  echo "Upgrading ${module}..."
  retry upgrade_module "$module"
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
