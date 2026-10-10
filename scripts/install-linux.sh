#!/usr/bin/env bash
# install-linux.sh: 普通 Linux 一键安装
set -e

echo "==> 检测 ffmpeg..."
if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "[!] 未检测到 ffmpeg，请先运行: sudo apt install ffmpeg"
    exit 1
fi

# 选架构
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)   BIN=mediaconv-linux-amd64 ;;
    i386|i686) BIN=mediaconv-linux-386 ;;
    aarch64)  BIN=mediaconv-linux-arm64 ;;
    *) echo "[!] 不支持的架构: $ARCH"; exit 1 ;;
esac
echo "==> 架构: $ARCH -> $BIN"

SCRIPT_DIR=$(cd "$(dirname "$0")/.." && pwd)
SRC="$SCRIPT_DIR/bin/$BIN"
if [ ! -f "$SRC" ]; then
    echo "[!] 找不到 $SRC"
    exit 1
fi

# 装到 ~/.local/bin（不需要 root）
DST="$HOME/.local/bin"
mkdir -p "$DST"
cp "$SRC" "$DST/mediaconv"
chmod +x "$DST/mediaconv"

echo "==> 安装完成: $DST/mediaconv"
echo ""
echo "运行方式:"
echo "  mediaconv"
echo "然后浏览器打开 http://127.0.0.1:8090"
echo "（端口可改: MEDIACONV_PORT=9000 mediaconv）"
