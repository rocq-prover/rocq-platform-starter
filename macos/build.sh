#!/usr/bin/env bash
#
# build.sh — Build rocq-platform-starter macOS app bundle and DMG
#
# Prerequisites:
#   - macOS with Xcode Command Line Tools
#   - Go 1.22+
#   - go mod tidy (run once)
#
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==> Syncing embedded assets..."
cp -f ../manifest/latest.json embedded/manifest/latest.json
cp -f ../templates/test.v embedded/templates/test.v
cp -f ../templates/main.v embedded/templates/main.v
cp -f ../templates/_RocqProject embedded/templates/_RocqProject

echo "==> Building rocq-platform-starter (macOS arm64)..."
CGO_ENABLED=1 \
GOOS=darwin \
GOARCH=arm64 \
  go build \
    -ldflags="-s -w" \
    -o rocq-platform-starter \
    ./cmd/rocq-platform-starter/

echo "==> Done: $(ls -lh rocq-platform-starter | awk '{print $5, $NF}')"
file rocq-platform-starter

echo ""
echo "To create the .app bundle and DMG, run: make app-bundle dmg"
