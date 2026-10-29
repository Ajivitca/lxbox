#!/bin/bash
# Build script for LxBox router binaries
# Target: Keenetic Hopper (KN-3810, MediaTek MT7621 / EcoNet EN7528DU, MIPS 1004Kc, mipsel softfloat)
# Also supports: Keenetic Titan / Hero / Hopper SE (AArch64 / ARM64)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SINGBOX_DIR="${SCRIPT_DIR}/../sing-box-lx"
BIN_DIR="${SCRIPT_DIR}/bin"

ARCH="${1:-mipsle}"

echo "=========================================="
echo " Building LxBox for Target: ${ARCH}"
echo "=========================================="

mkdir -p "${BIN_DIR}/${ARCH}"

case "$ARCH" in
    mipsle|mipsel)
        GOOS=linux
        GOARCH=mipsle
        GOMIPS=softfloat
        ;;
    arm64|aarch64)
        GOOS=linux
        GOARCH=arm64
        GOMIPS=""
        ;;
    armv7|arm)
        GOOS=linux
        GOARCH=arm
        GOARM=7
        GOMIPS=""
        ;;
    amd64|x86_64)
        GOOS=linux
        GOARCH=amd64
        GOMIPS=""
        ;;
    *)
        echo "Usage: $0 [mipsle|arm64|armv7|amd64]"
        exit 1
        ;;
esac

echo "[*] Building lxbox-helper..."
env GOOS=$GOOS GOARCH=$GOARCH GOMIPS=$GOMIPS CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "${BIN_DIR}/${ARCH}/lxbox-helper" "${SCRIPT_DIR}/cmd/lxbox-helper"
echo "[+] lxbox-helper built: ${BIN_DIR}/${ARCH}/lxbox-helper"

if [ -d "$SINGBOX_DIR" ]; then
    echo "[*] Building sing-box-lx from source..."
    cd "$SINGBOX_DIR"
    LX_TAGS=$(make -f Makefile.lx lx-print-tags)
    LX_LDFLAGS="-X 'github.com/sagernet/sing-box/constant.Version=1.14.4-lx.25' -checklinkname=0 -s -w -buildid="
    
    env GOOS=$GOOS GOARCH=$GOARCH GOMIPS=$GOMIPS CGO_ENABLED=0 \
        go build -v -trimpath -tags "$LX_TAGS" -ldflags "$LX_LDFLAGS" -o "${BIN_DIR}/${ARCH}/sing-box" ./cmd/sing-box
    echo "[+] sing-box built: ${BIN_DIR}/${ARCH}/sing-box"
else
    echo "[!] sing-box-lx source tree not found at ${SINGBOX_DIR}, skipping core build."
fi

echo "=========================================="
echo " Build Completed successfully!"
echo "=========================================="
