#!/usr/bin/env bash
set -euo pipefail

MODULES=("site")

body=""
# Captures gomajor stderr for each module
err_file=$(mktemp)

for module in "${MODULES[@]}"; do
  echo "Checking major version upgrades in ${module}..."
  # -major limits output to new major versions; without it gomajor also lists
  # minor and patch updates, which go-dep-upgrades already handles.
  # gomajor exits 0 even when proxy lookups fail and only reports them on
  # stderr, so fail here rather than treat a failed lookup as "no upgrades"
  # (which would close the tracking issue).
  output=$(cd "$module" && GOWORK=off gomajor list -major 2>"$err_file")
  if grep -q 'failed:' "$err_file"; then
    cat "$err_file" >&2
    exit 1
  fi

  if [ -n "$output" ]; then
    body+="### \`${module}\`"$'\n\n'
    body+='```'$'\n'
    body+="${output}"$'\n'
    body+='```'$'\n\n'
  fi
done

if [ -z "$body" ]; then
  echo "has_upgrades=false" >> "$GITHUB_OUTPUT"
  echo "No major version upgrades available."
  exit 0
fi

echo "has_upgrades=true" >> "$GITHUB_OUTPUT"

EOF_MARKER=$(dd if=/dev/urandom bs=15 count=1 status=none | base64)
{
  echo "body<<$EOF_MARKER"
  echo "## Go Major Version Upgrades"
  echo ""
  echo "The following Go dependencies have new major versions available."
  echo "Major upgrades may include breaking API changes and require code modifications."
  echo ""
  echo "$body"
  echo "$EOF_MARKER"
} >> "$GITHUB_OUTPUT"
