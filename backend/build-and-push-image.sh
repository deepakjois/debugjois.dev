#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="${SCRIPT_DIR}/api"
REPOSITORY_NAME="debugjois-dev"
IMAGE_TAG="debugjois-dev-backend-api-$(date +%Y%m%d-%H%M%S)"

AWS_REGION="${AWS_REGION:-${AWS_DEFAULT_REGION:-$(aws configure get region 2>/dev/null || true)}}"

if [[ -z "${AWS_REGION}" ]]; then
  echo "AWS region is not configured. Set AWS_REGION or AWS_DEFAULT_REGION, or configure a default AWS CLI region." >&2
  exit 1
fi

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"

REPOSITORY_URI="$(aws ecr describe-repositories --repository-names "${REPOSITORY_NAME}" --region "${AWS_REGION}" --query 'repositories[0].repositoryUri' --output text)"
IMAGE_REF="${REPOSITORY_URI}:${IMAGE_TAG}"

aws ecr get-login-password --region "${AWS_REGION}" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com" >/dev/null

docker buildx build \
  --platform linux/amd64 \
  --provenance=false \
  --push \
  -t "${IMAGE_REF}" \
  "${API_DIR}" >/dev/null

IMAGE_DIGEST="$(aws ecr describe-images --repository-name "${REPOSITORY_NAME}" --region "${AWS_REGION}" --image-ids imageTag="${IMAGE_TAG}" --query 'imageDetails[0].imageDigest' --output text)"

if [[ -z "${IMAGE_DIGEST}" || "${IMAGE_DIGEST}" == "None" ]]; then
  echo "Failed to resolve digest for pushed image ${IMAGE_REF}." >&2
  exit 1
fi

printf '%s@%s\n' "${REPOSITORY_URI}" "${IMAGE_DIGEST}"
