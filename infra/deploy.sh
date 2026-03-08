#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="${SCRIPT_DIR}"
BUILD_SCRIPT="${SCRIPT_DIR}/../backend/build-and-push-image.sh"
BUILD_IMAGE=0
DEPLOY_ARGS=()

usage() {
  cat <<'EOF'
Usage: ./infra/deploy.sh [--build-image] [extra cdk deploy args...]

  --build-image   Build and push a new image before deploying.
  -h, --help      Show this help text.

Without --build-image, the script relies on `infra.go` to reuse the image
currently configured on the deployed Lambda function.
EOF
}

while (($# > 0)); do
  case "$1" in
    --build-image)
      BUILD_IMAGE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      DEPLOY_ARGS+=("$1")
      shift
      ;;
  esac
done

cd "${INFRA_DIR}"

if (( BUILD_IMAGE )); then
  IMAGE_URI="$(${BUILD_SCRIPT})"
  echo "Deploying with ${IMAGE_URI}" >&2
  IMAGE_URI="${IMAGE_URI}" cdk deploy --require-approval never "${DEPLOY_ARGS[@]}"
else
  echo "Deploying with the currently deployed Lambda image" >&2
  cdk deploy --require-approval never "${DEPLOY_ARGS[@]}"
fi
