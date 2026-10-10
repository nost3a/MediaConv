#!/usr/bin/env bash
# install-termux.sh: Termux 下安装
set -e

echo "==> 安装 ffmpeg..."
pkg install -y ffmpeg

ARCH=$(uname -m)
case "$ARCH" in
    aarch64)  BIN=mediaconv-linux-arm64 ;;
    x86_64)   BIN=mediaconv-linux-amd64 ;;
    i386|i686) BIN=mediaconv-linux-386 ;;
    *) echo "[!] 不支持的架构: $ARCH"; exit 1 ;;
esac
echo "==> 架构: $ARCH -> $BIN"

SCRIPT_DIR=$(cd "$(dirname "$0")/.." && pwd)
SRC="$SCRIPT_DIR/bin/$BIN"
DST="$HOME/mediaconv"
cp "$SRC" "$DST"
chmod +x "$DST"

echo "==> 安装完成: $DST"
echo ""
echo "运行方式:"
echo "  $DST"
echo "然后在 Android WebView 壳或手机浏览器打开 http://127.0.0.1:8090"
