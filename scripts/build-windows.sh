#!/usr/bin/env bash
# build-windows.sh: 在 Linux 上交叉编 Windows GUI 版 exe
# 依赖: mingw-w64 (x86_64-w64-mingw32-gcc), webview2
# MSYS2 里: pacman -S mingw-w64-x86_64-gcc
set -e
cd "$(dirname "$0")/.."

if ! command -v x86_64-w64-mingw32-gcc >/dev/null; then
    echo "[!] 缺 x86_64-w64-mingw32-gcc。Ubuntu: sudo apt install mingw-w64"
    exit 1
fi

echo "==> 编译 windows/amd64 GUI 版 (无控制台)"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
    CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    go build -trimpath \
    -ldflags "-H=windowsgui -s -w" \
    -o bin/mediaconv-windows-amd64-gui.exe .

echo "==> 完成: bin/mediaconv-windows-amd64-gui.exe"
file bin/mediaconv-windows-amd64-gui.exe | grep -i "gui\|windows"
