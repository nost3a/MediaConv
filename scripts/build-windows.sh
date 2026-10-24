#!/usr/bin/env bash
# build-windows.sh: 在 Linux 上交叉编 Windows GUI 版 exe
# 依赖: mingw-w64 (x86_64-w64-mingw32-gcc), webview2
# MSYS2 里: pacman -S mingw-w64-x86_64-gcc
set -e
cd "$(dirname "$0")/.."

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-H=windowsgui -s -w -X main.version=${VERSION}"

if ! command -v x86_64-w64-mingw32-gcc >/dev/null; then
    echo "[!] 缺 x86_64-w64-mingw32-gcc。Ubuntu: sudo apt install mingw-w64"
    exit 1
fi

# Windows 构建需要 internal/ffembed/ffmpeg.exe（//go:embed 必需）
if [ ! -s internal/ffembed/ffmpeg.exe ]; then
    echo "[*] 未找到内嵌 ffmpeg.exe，尝试下载..."
    bash scripts/fetch-ffmpeg.sh
fi

echo "==> 编译 windows/amd64 GUI 版 (无控制台, version=${VERSION})"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
    CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    go build -trimpath \
    -ldflags "${LDFLAGS}" \
    -o bin/mediaconv-windows-amd64-gui.exe .

echo "==> 完成: bin/mediaconv-windows-amd64-gui.exe"
file bin/mediaconv-windows-amd64-gui.exe | grep -i "gui\|windows"
