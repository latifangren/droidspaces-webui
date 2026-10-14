#!/usr/bin/env bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "[1/4] Building Svelte Frontend with Bun..."
cd web
bun run build
cd ..

echo "[2/4] Compiling Host Binary (x86_64)..."
mkdir -p bin
go build -ldflags="-s -w" -o bin/dsweb ./cmd/dsweb

echo "[3/4] Compiling Android ARM64 Binary..."
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/dsweb_arm64 ./cmd/dsweb

echo "[4/4] Packaging Unified Magisk/KernelSU Module Zip..."
rm -rf dist
mkdir -p dist/module/bin
cp -r deploy/magisk/* dist/module/

# Copy binaries into module bin
cp bin/dsweb_arm64 dist/module/bin/
if [ -f bin/droidspaces_arm64 ]; then
  cp bin/droidspaces_arm64 dist/module/bin/droidspaces
fi
if [ -f bin/busybox_arm64 ]; then
  cp bin/busybox_arm64 dist/module/bin/busybox
fi

VERSION=$(grep '^version=' deploy/magisk/module.prop | cut -d= -f2)
[ -z "$VERSION" ] && VERSION="v7.0.0"
ZIP_NAME="droidspaces-unified-${VERSION}.zip"

cd dist/module
if command -v zip >/dev/null 2>&1; then
  zip -r "../${ZIP_NAME}" ./* > /dev/null
elif command -v python >/dev/null 2>&1; then
  python -c "import shutil; shutil.make_archive('../${ZIP_NAME%.zip}', 'zip', '.')"
elif command -v powershell >/dev/null 2>&1; then
  powershell -Command "Compress-Archive -Path * -DestinationPath ../${ZIP_NAME} -Force"
fi
cp "../${ZIP_NAME}" "../droidspaces-unified-v7.0.0.zip" 2>/dev/null || true
cd "$DIR"
echo "=================================================="
echo "✔ Unified Build complete!"
echo "  - Core + WebUI Module: dist/${ZIP_NAME}"
echo "  - Host binary:         bin/dsweb"
echo "  - Android WebUI:       bin/dsweb_arm64"
echo "=================================================="
