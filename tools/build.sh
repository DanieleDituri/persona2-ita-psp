#!/usr/bin/env bash
# Build the installer for every platform into dist/. Needs Go >= 1.21 (GO env var to override).
set -euo pipefail
cd "$(dirname "$0")/../installer"
GO="${GO:-go}"
mkdir -p ../dist
for t in windows/amd64 windows/arm64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os=${t%/*}; arch=${t#*/}
  name=$os; [ "$os" = darwin ] && name=macos
  ext=""; [ "$os" = windows ] && ext=.exe
  echo "build $os/$arch"
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch "$GO" build -trimpath -ldflags "-s -w" -o "../dist/installa_persona2_ita_${name}_${arch}${ext}" .
done
ls -la ../dist
