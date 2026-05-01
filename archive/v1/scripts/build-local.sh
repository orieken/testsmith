#!/usr/bin/env bash
set -euo pipefail

echo "══════════════════════════════════════"
echo "  TestSmith — Local Binary Build"
echo "══════════════════════════════════════"

# Ensure build deps are installed
poetry install --with build

# Clean previous builds
rm -rf build/ dist/

# Build
echo ""
echo "Building binary..."
poetry run pyinstaller testsmith.spec --clean --noconfirm

# Verify
echo ""
echo "Verifying..."
./dist/testsmith --version

BINARY_SIZE=$(du -h dist/testsmith | cut -f1)
echo ""
echo "══════════════════════════════════════"
echo "  Build complete!"
echo "  Binary: dist/testsmith"
echo "  Size:   ${BINARY_SIZE}"
echo "══════════════════════════════════════"
