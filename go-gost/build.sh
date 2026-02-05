#!/bin/bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
OUT_DIR=${1:-"$ROOT_DIR/dist"}

mkdir -p "$OUT_DIR"

echo "🔧 构建 gost-amd64..."
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/gost-amd64" .

echo "🔧 构建 gost-arm64..."
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/gost-arm64" .

echo "✅ 构建完成:"
ls -la "$OUT_DIR" | awk '{print $9}'
