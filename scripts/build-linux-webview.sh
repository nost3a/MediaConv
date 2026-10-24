#!/usr/bin/env bash
# build-linux-webview.sh: 在 Linux 上编译 fyne/OpenGL 原生 GUI 版
#
# 注意: 本仓库 gui_fyne.go 用的是 fyne（go-gl/glfw + OpenGL），
#       不是 webkit。需要的是 X11 / OpenGL / Wayland 开发库，而非 webkit2gtk。
#
# Debian/Ubuntu 依赖:
#   sudo apt install build-essential pkg-config gcc \
#     libgl1-mesa-dev libglu1-mesa-dev \
#     libx11-dev libxext-dev libxrandr-dev libxinerama-dev \
#     libxcursor-dev libxi-dev libxxf86vm-dev \
#     libwayland-dev libxkbcommon-dev
set -e
cd "$(dirname "$0")/.."

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X main.version=${VERSION}"

# 软性检查：缺 gcc / pkg-config 直接报错；缺 X11/GL 开发库则给提示但不阻断
if ! command -v gcc >/dev/null || ! command -v pkg-config >/dev/null; then
    echo "[!] 缺 gcc 或 pkg-config，先装: sudo apt install build-essential pkg-config"
    exit 1
fi
if ! pkg-config --exists glfw3 2>/dev/null && ! ldconfig -p | grep -q libGL.so; then
    echo "[*] 提示: 若链接期报 cannot find -lXxx / -lGL，请先装上面的 X11/OpenGL 开发库。"
fi

echo "==> 编译 linux/amd64 GUI 版 (version=${VERSION})"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags "${LDFLAGS}" \
    -o bin/mediaconv-linux-amd64-gui .

echo "==> 完成: bin/mediaconv-linux-amd64-gui"
ldd bin/mediaconv-linux-amd64-gui | grep -iE "libGL|libX11|glfw" | head
