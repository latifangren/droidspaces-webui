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

cd dist/module
if command -v zip >/dev/null 2>&1; then
  zip -r ../droidspaces-unified-v1.0.0.zip ./* > /dev/null
elif command -v python >/dev/null 2>&1; then
  python -c "import shutil; shutil.make_archive('../droidspaces-unified-v1.0.0', 'zip', '.')"
elif command -v powershell >/dev/null 2>&1; then
  powershell -Command "Compress-Archive -Path * -DestinationPath ../droidspaces-unified-v1.0.0.zip -Force"
fi
cd "$DIR"

echo "=================================================="
echo "✔ Unified Build complete!"
echo "  - Core + WebUI Module: dist/droidspaces-unified-v1.0.0.zip"
echo "  - Host binary:         bin/dsweb"
echo "  - Android WebUI:       bin/dsweb_arm64"
echo "=================================================="
