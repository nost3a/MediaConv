#!/usr/bin/env bash
# build-linux-webview.sh: 在 Linux 上编 webview GUI 版
# 依赖: gcc, pkg-config, libwebkit2gtk-4.1-dev (Debian/Ubuntu: sudo apt install)
set -e
cd "$(dirname "$0")/.."

echo "==> 检查 webkit2gtk-4.1..."
if ! pkg-config --exists webkit2gtk-4.1; then
    echo "[!] 缺 webkit2gtk-4.1，先装: sudo apt install libwebkit2gtk-4.1-dev"
    exit 1
fi

echo "==> 编译 linux/amd64 GUI 版"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags "-s -w" \
    -o bin/mediaconv-linux-amd64-gui .

echo "==> 完成: bin/mediaconv-linux-amd64-gui"
ldd bin/mediaconv-linux-amd64-gui | grep -i webkit && echo "已链接 webkit2gtk" || echo "[!] 没找到 webkit 链接"
