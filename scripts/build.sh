#!/bin/bash
set -e

VERSION="${1:-0.1.0}"
OUTPUT_DIR="dist"
BINARY_NAME="apipod-cli"
MODULE="./cmd/apipod-cli/"

PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

echo "Building apipod-cli v${VERSION}..."

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  OUTPUT_NAME="${BINARY_NAME}-${GOOS}-${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    OUTPUT_NAME="${OUTPUT_NAME}.exe"
  fi

  echo "  Building ${GOOS}/${GOARCH}..."
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
    "$MODULE"
done

# Create checksums
cd "$OUTPUT_DIR"
shasum -a 256 * > checksums.txt
cd ..

echo ""
echo "Build complete! Binaries in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR/"
