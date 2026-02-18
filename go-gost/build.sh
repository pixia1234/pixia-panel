#!/bin/bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
if [[ $# -gt 0 ]]; then
  case "$1" in
    /*) OUT_DIR="$1" ;;
    *) OUT_DIR="$(pwd)/$1" ;;
  esac
else
  OUT_DIR="$ROOT_DIR/dist"
fi

mkdir -p "$OUT_DIR"

cd "$ROOT_DIR"

export GOCACHE=${GOCACHE:-/tmp/go-build-gost}
export CGO_ENABLED=0

DEFAULT_VERSION="0.3.1"
BUILD_VERSION=${GOST_VERSION:-${GITHUB_REF_NAME:-$DEFAULT_VERSION}}
LDFLAGS="-s -w -X main.version=${BUILD_VERSION}"

echo "🔧 构建 gost-amd64 (CGO_ENABLED=${CGO_ENABLED})..."
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/gost-amd64" .

echo "🔧 构建 gost-arm64 (CGO_ENABLED=${CGO_ENABLED})..."
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/gost-arm64" .

if command -v upx >/dev/null 2>&1; then
  echo "🧰 UPX 压缩..."
  upx --best --lzma "$OUT_DIR/gost-amd64" "$OUT_DIR/gost-arm64"
else
  echo "⚠️ 未检测到 upx，跳过压缩"
fi

echo "✅ 构建完成:"
ls -la "$OUT_DIR" | awk '{print $9}'
