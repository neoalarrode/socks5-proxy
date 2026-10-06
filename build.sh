#!/bin/bash
set -e

VERSION="1.0.0"
OUT_DIR="dist"
MODULE="socks5-proxy"

mkdir -p "$OUT_DIR"

echo "Building SOCKS5 Proxy v${VERSION}"
echo "================================"

echo "[1/2] Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${OUT_DIR}/socks5-proxy-linux-amd64" .

echo "[2/2] Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "${OUT_DIR}/socks5-proxy-windows-amd64.exe" .

echo ""
echo "Build complete:"
ls -lh "${OUT_DIR}/"
