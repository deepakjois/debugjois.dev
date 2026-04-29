#!/usr/bin/env bash
set -euo pipefail

MODULES=("site")

body=""

for module in "${MODULES[@]}"; do
  echo "Checking major version upgrades in ${module}..."
  output=$(cd "$module" && GOWORK=off gomajor list 2>/dev/null) || true

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
