#!/bin/bash
set -euo pipefail

# Build script for cross-platform autospec binaries
# Usage: ./scripts/build-all.sh [version]

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Module path
MODULE_PATH="github.com/ariel-frischer/autospec"

LDFLAGS="-X ${MODULE_PATH}/internal/cli.Version=${VERSION} \
         -X ${MODULE_PATH}/internal/cli.Commit=${COMMIT} \
         -X ${MODULE_PATH}/internal/cli.BuildDate=${BUILD_DATE} \
         -s -w"

echo "Building autospec ${VERSION} (commit: ${COMMIT})"
echo "Build date: ${BUILD_DATE}"
echo ""

# Create dist directory
mkdir -p dist

# Linux builds
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-linux-amd64 ./cmd/autospec
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/autospec-linux-arm64 ./cmd/autospec

# macOS builds
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-darwin-amd64 ./cmd/autospec
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/autospec-darwin-arm64 ./cmd/autospec

# Windows builds
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-windows-amd64.exe ./cmd/autospec

echo ""
echo "Build complete! Binaries in dist/:"
ls -lh dist/

echo ""
echo "Checking binary sizes (target: <15MB)..."
for file in dist/*; do
    size=$(stat --printf='%s' "$file" 2>/dev/null || stat -f '%z' "$file")
    size_human=$(numfmt --to=iec "$size" 2>/dev/null || echo "$((size / 1024 / 1024))M")
    echo "  $(basename "$file"): $size_human"
done
