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

echo "[4/4] Packaging Magisk/KernelSU Module Zip..."
rm -rf dist
mkdir -p dist/module/bin
cp -r deploy/magisk/* dist/module/
cp bin/dsweb_arm64 dist/module/bin/
cd dist/module
zip -r ../droidspaces-webui-v1.0.0.zip ./* > /dev/null
cd "$DIR"

echo "=================================================="
echo "✔ Build complete!"
echo "  - Host binary:       bin/dsweb"
echo "  - Android binary:    bin/dsweb_arm64"
echo "  - Flashable module:  dist/droidspaces-webui-v1.0.0.zip"
echo "=================================================="
