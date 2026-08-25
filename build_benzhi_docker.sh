#!/usr/bin/env bash
# 评测镜像构建脚本（同源 benzhi.Dockerfile）。
# usage: bash build_benzhi_docker.sh <镜像名> <平台>
#   平台示例: linux/amd64, linux/arm64, linux/amd64,linux/arm64
set -euo pipefail

IMAGE_NAME="${1:-my-project}"
PLATFORM="${2:-linux/amd64}"

if [[ -z "${IMAGE_NAME}" ]]; then
  echo "usage: bash build_benzhi_docker.sh <镜像名> <平台>" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

docker buildx build \
  --file benzhi.Dockerfile \
  --platform "${PLATFORM}" \
  --tag "${IMAGE_NAME}" \
  --load \
  .

echo "built ${IMAGE_NAME} for ${PLATFORM}"
