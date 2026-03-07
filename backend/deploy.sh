#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="${SCRIPT_DIR}/infra"
BUILD_SCRIPT="${SCRIPT_DIR}/build-and-push-image.sh"
BUILD_IMAGE=0
DEPLOY_ARGS=()

usage() {
  cat <<'EOF'
Usage: ./backend/deploy.sh [--build-image] [extra cdk deploy args...]

  --build-image   Build and push a new image before deploying.
  -h, --help      Show this help text.

Without --build-image, the script reuses the image currently configured on the
deployed Lambda function.
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

if (( BUILD_IMAGE )); then
  IMAGE_URI="$(${BUILD_SCRIPT})"
else
  FUNCTION_NAME="$(aws cloudformation describe-stacks \
    --stack-name InfraStack \
    --query 'Stacks[0].Outputs[?OutputKey==`LambdaFunctionName`].OutputValue' \
    --output text)"

  if [[ -z "${FUNCTION_NAME}" || "${FUNCTION_NAME}" == "None" ]]; then
    echo "Could not determine Lambda function name from CloudFormation stack InfraStack." >&2
    exit 1
  fi

  IMAGE_URI="$(aws lambda get-function \
    --function-name "${FUNCTION_NAME}" \
    --query 'Code.ImageUri' \
    --output text)"

  if [[ -z "${IMAGE_URI}" || "${IMAGE_URI}" == "None" ]]; then
    echo "Could not determine currently deployed image for Lambda ${FUNCTION_NAME}." >&2
    exit 1
  fi
fi

echo "Deploying with ${IMAGE_URI}" >&2

cd "${INFRA_DIR}"
IMAGE_URI="${IMAGE_URI}" cdk deploy --require-approval never "${DEPLOY_ARGS[@]}"
