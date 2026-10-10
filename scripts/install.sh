#!/bin/bash
# MediaConv 一键安装脚本
# One-shot install script for MediaConv
set -e

PREFIX="$HOME/.local/bin"
mkdir -p "$PREFIX"

# 检测发行版 / Detect distro
if [ -r /etc/os-release ]; then
    . /etc/os-release
    ID_LIKE=${ID_LIKE:-}
fi

echo "=== 安装依赖提示 / Dependency hints ==="
case "${ID:-}" in
    debian|ubuntu|linuxmint)
        echo "Debian/Ubuntu: sudo apt install ffmpeg libgl1"
        ;;
    fedora|rhel|centos|rocky|almalinux)
        echo "Fedora/RHEL: sudo dnf install ffmpeg mesa-libGL"
        ;;
    arch|manjaro|endeavouros)
        echo "Arch: sudo pacman -S ffmpeg libgl"
        ;;
    opensuse*)
        echo "openSUSE: sudo zypper install ffmpeg libgl1"
        ;;
    alpine)
        echo "Alpine: sudo apk add ffmpeg mesa-gl"
        ;;
    *)
        echo "未知发行版，请自行安装 ffmpeg 和 libGL。"
        echo "Unknown distro, please install ffmpeg and libGL manually."
        ;;
esac
echo

HERE="$(cd "$(dirname "$0")/.." && pwd)"
for f in mediaconv mconv mediaconv-cli; do
    if [ -x "$HERE/bin/$f" ]; then
        cp "$HERE/bin/$f" "$PREFIX/$f"
        chmod +x "$PREFIX/$f"
        echo "已安装 $f -> $PREFIX/$f"
    fi
done

echo
echo "完成。运行 'mediaconv' 启动 GUI，或 'mconv' 进入交互命令行。"
echo "Done. Run 'mediaconv' for GUI, or 'mconv' for interactive CLI."
